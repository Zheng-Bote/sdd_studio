package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

// configTool changes sdd.config.yaml. Every change is critical (ADR 0004): the configuration decides what sdd checks,
// so one line can switch off rules that the baseline may not suppress (profile: minimal) or whole paths (exclude).
var configTool = tool{Name: "sdd_config", Title: "Konfiguration ändern",
	Description: "KRITISCH (ADR 0004). Ersetzt den Inhalt einer bestehenden sdd.config.yaml; base_sha256 aus sdd_read ist Pflicht, der neue Inhalt muss sich laden lassen. Ohne apply nur Vorschau: Diff und je Prüflauf die Befunde, die wegfallen oder hinzukommen. Ausgeführt wird nur nach der einzelnen Bestätigung des Menschen.",
	InputSchema: obj(map[string]any{
		"content":     str("vollständiger neuer Inhalt von sdd.config.yaml"),
		"base_sha256": str("sha256 des aktuellen Inhalts aus sdd_read"),
		"apply":       map[string]any{"type": "boolean", "description": "wirklich schreiben"},
	}, "content", "base_sha256"), Annotations: criticalHints}

// configRuns are the check runs compared in the preview: lint and every gate.
var configRuns = []string{"", "pre-implementation", "pre-merge", "pre-release"}

type findingRef struct {
	RuleID  string `json:"rule_id"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type runDelta struct {
	Gate    string       `json:"gate"` // "lint" for the run without gate
	Dropped []findingRef `json:"dropped"`
	Added   []findingRef `json:"added"`
}

type uncheckedRun struct {
	Gate  string `json:"gate"`
	Error string `json:"error"`
}

func gateLabel(g string) string {
	if g == "" {
		return "lint"
	}
	return g
}

func (s *Server) toolConfig(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Content    string `json:"content"`
		BaseSHA256 string `json:"base_sha256"`
		Apply      bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if len(a.Content) > maxWriteBytes {
		return nil, fmt.Errorf("Inhalt zu groß: %d Bytes (höchstens %d)", len(a.Content), maxWriteBytes)
	}
	const rel = config.DefaultConfigFileName
	old, exists, err := readExisting(s.Root, rel)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("keine Konfiguration: sdd.config.yaml fehlt — das Projekt erst mit sdd_init anlegen")
	}
	if strings.TrimSpace(a.BaseSHA256) != sha256Hex(old) {
		return nil, fmt.Errorf("base_sha256 fehlt oder ist veraltet; aktueller Stand von %s: %s — erst mit sdd_read lesen", rel, sha256Hex(old))
	}
	oldCfg, err := config.Parse(old)
	if err != nil {
		return nil, err
	}
	newCfg, err := config.Parse([]byte(a.Content))
	if err != nil {
		return nil, fmt.Errorf("sdd.config.yaml nicht geschrieben: %w", err)
	}

	diff := scaffold.UnifiedDiff(rel, old, []byte(a.Content))
	if diff == "" {
		return resultOf(map[string]any{"applied": false, "critical": true, "summary": "sdd.config.yaml unverändert",
			"ops": []writeOp{{Path: rel, Action: "unchanged"}}}), nil
	}
	deltas, unchecked, err := s.findingDeltas(oldCfg, newCfg)
	if err != nil {
		return nil, err
	}
	keys, err := changedKeys(old, []byte(a.Content), oldCfg, newCfg)
	if err != nil {
		return nil, err
	}
	dropped, added := 0, 0
	for _, d := range deltas {
		dropped += len(d.Dropped)
		added += len(d.Added)
	}
	summary := fmt.Sprintf("sdd.config.yaml ändern: %s (−%d / +%d Befunde)", strings.Join(keys, ", "), dropped, added)

	if !a.Apply {
		return resultOf(map[string]any{"applied": false, "critical": true, "summary": summary,
			"ops":            []writeOp{{Path: rel, Action: "update", Diff: diff}},
			"findings_delta": deltas, "unchecked": unchecked}), nil
	}
	results, err := scaffold.Plan{Ops: []scaffold.Op{{Path: rel, Content: []byte(a.Content), Replace: true, Expect: old}}}.Apply(s.Root)
	if err != nil {
		return nil, err
	}
	if results[0].Status == scaffold.StatusSkipped {
		return nil, fmt.Errorf("%s nicht geschrieben: %s", rel, results[0].Reason)
	}
	written, _, err := readExisting(s.Root, rel)
	if err != nil {
		return nil, err
	}
	return resultOf(map[string]any{"applied": true, "critical": true, "summary": summary, "path": rel, "status": results[0].Status,
		"sha256": sha256Hex(written), "lines": lineCount(written), "diff": diff, "findings_delta": deltas}), nil
}

// findingDeltas runs every check run with the old and the new configuration (baseline applied, as CI sees it) and
// returns the findings that disappear or appear. A run that already fails with the old configuration is reported as
// unchecked; one that fails only with the new configuration fails the preview.
func (s *Server) findingDeltas(oldCfg, newCfg *config.Config) ([]runDelta, []uncheckedRun, error) {
	deltas := []runDelta{}
	unchecked := []uncheckedRun{}
	for _, g := range configRuns {
		before, err := sddcore.RunCheckWithConfig(sddcore.RunOptions{RepoRoot: s.Root, Gate: g}, oldCfg)
		if err != nil {
			unchecked = append(unchecked, uncheckedRun{Gate: gateLabel(g), Error: err.Error()})
			continue
		}
		after, err := sddcore.RunCheckWithConfig(sddcore.RunOptions{RepoRoot: s.Root, Gate: g}, newCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("Prüflauf %s scheitert mit der neuen Konfiguration: %w", gateLabel(g), err)
		}
		d := runDelta{Gate: gateLabel(g), Dropped: minus(before.Findings, after.Findings), Added: minus(after.Findings, before.Findings)}
		if len(d.Dropped) > 0 || len(d.Added) > 0 {
			deltas = append(deltas, d)
		}
	}
	return deltas, unchecked, nil
}

// minus returns the findings of a without a counterpart in b. A configuration change edits no file, so rule, path,
// line and ordinal identify a finding across both runs.
func minus(a, b []report.Finding) []findingRef {
	type key struct {
		rule, path    string
		line, ordinal int
	}
	in := map[key]bool{}
	for _, f := range b {
		in[key{f.RuleID, f.Path, f.Line, f.Ordinal}] = true
	}
	out := []findingRef{}
	for _, f := range a {
		if !in[key{f.RuleID, f.Path, f.Line, f.Ordinal}] {
			out = append(out, findingRef{RuleID: f.RuleID, Path: f.Path, Line: f.Line, Message: f.Message})
		}
	}
	return out
}

// changedKeys names what a change touches: top-level keys, under gates one level deeper (gates.pre_merge.override).
// A profile change reads "profile <old> → <new>".
func changedKeys(oldData, newData []byte, oldCfg, newCfg *config.Config) ([]string, error) {
	var o, n map[string]any
	if err := yaml.Unmarshal(oldData, &o); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(newData, &n); err != nil {
		return nil, err
	}
	var keys []string
	for _, k := range unionKeys(o, n) {
		if reflect.DeepEqual(o[k], n[k]) {
			continue
		}
		switch k {
		case "profile":
			keys = append(keys, fmt.Sprintf("profile %s → %s", oldCfg.Profile, newCfg.Profile))
		case "gates":
			og, _ := o[k].(map[string]any)
			ng, _ := n[k].(map[string]any)
			for _, g := range unionKeys(og, ng) {
				ogg, _ := og[g].(map[string]any)
				ngg, _ := ng[g].(map[string]any)
				for _, f := range unionKeys(ogg, ngg) {
					if !reflect.DeepEqual(ogg[f], ngg[f]) {
						keys = append(keys, "gates."+g+"."+f)
					}
				}
				if len(ogg) == 0 && len(ngg) == 0 && !reflect.DeepEqual(og[g], ng[g]) {
					keys = append(keys, "gates."+g)
				}
			}
		default:
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		keys = []string{"Kommentare/Formatierung"}
	}
	return keys, nil
}

func unionKeys(a, b map[string]any) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
