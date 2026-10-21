package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/overview"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

type tool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

func obj(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func strs(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

var readOnly = map[string]any{"readOnlyHint": true, "openWorldHint": false}

var gateEnum = map[string]any{"type": "string", "enum": catalog.AllGates, "description": "Gate"}

// toolList is the complete, fixed set of tools. Nothing here touches the baseline, the ADR lockfile or Git.
func toolList() []tool {
	return []tool{
		{Name: "sdd_next", Title: "Nächster Prozessschritt", Description: "Prozessschritte je Feature mit Zustand und dem nächsten Schritt (wie `sdd next --json`).",
			InputSchema: obj(map[string]any{"feature": str("Feature-Slug; leer = alle aktiven Features")}), Annotations: readOnly},
		{Name: "sdd_lint", Title: "Befunde", Description: "Befunde im Berichtsformat v1 (wie `sdd lint --json`, mit gate wie `sdd gate <gate> --json`).",
			InputSchema: obj(map[string]any{"gate": gateEnum}), Annotations: readOnly},
		{Name: "sdd_gate", Title: "Gate prüfen", Description: "Führt ein Gate aus und meldet, ob es besteht (wie `sdd gate`).",
			InputSchema: obj(map[string]any{"gate": gateEnum, "feature": str("Feature-Slugs, kommagetrennt"), "paths": strs("erklärte Absicht für pre-implementation")}, "gate"), Annotations: readOnly},
		{Name: "sdd_report", Title: "Projektbericht", Description: "Reifegrad, Gates und Schuldenstand (wie `sdd report --format json`).",
			InputSchema: obj(map[string]any{}), Annotations: readOnly},
		{Name: "sdd_explain", Title: "Regel erklären", Description: "Katalogeintrag, Handbuchabschnitt und Prüfvertrag einer Regel.",
			InputSchema: obj(map[string]any{"rule_id": str("z. B. SDD-022")}, "rule_id"), Annotations: readOnly},
		{Name: "sdd_check_intent", Title: "Absicht prüfen", Description: "Vor einer Änderung: Greift die Größenschwelle, ist die Verfassung betroffen, welche Artefakte sind nötig?",
			InputSchema: obj(map[string]any{"paths": strs("Pfade oder Globs, die geändert werden sollen"), "feature": str("Feature-Slug")}, "paths"), Annotations: readOnly},
		{Name: "sdd_context", Title: "Was gilt hier?", Description: "Artefakte, die für eine Datei gelten, nach Präzedenz: Verfassung, ADR, Tech Spec, PRD, Plan.",
			InputSchema: obj(map[string]any{"path": str("Pfad relativ zum Repository")}, "path"), Annotations: readOnly},
		{Name: "sdd_trace", Title: "Rückverfolgung", Description: "Je Akzeptanzkriterium eines Features: Tech Specs, Tests (Datei:Zeile) und Tasks.",
			InputSchema: obj(map[string]any{"feature": str("Feature-Slug")}, "feature"), Annotations: readOnly},
		{Name: "sdd_read", Title: "Artefakt lesen", Description: "Text eines Artefakts (Verfassung, ADR, PRD, Tech Spec, Plan, Archiv) mit Zeilennummern; ohne path die Liste der lesbaren Artefakte. Liest nichts anderes.",
			InputSchema: obj(map[string]any{"path": str("Pfad relativ zum Repository, z. B. docs/design/<feature>.md; leer = Liste")}), Annotations: readOnly},
		{Name: "sdd_validate_draft", Title: "Entwurf prüfen", Description: "Prüft einen noch nicht gespeicherten Artefakt-Entwurf mit den Regeln, als läge er schon unter path.",
			InputSchema: obj(map[string]any{"path": str("Zielpfad relativ zum Repository"), "content": str("vollständiger Dateiinhalt")}, "path", "content"), Annotations: readOnly},
		{Name: "sdd_new", Title: "Artefakt anlegen", Description: "Wie `sdd new`: ohne apply nur Vorschau mit Inhalt; apply legt neue Dateien an (nur wenn der Server mit --allow-write läuft), nie überschreibend.",
			InputSchema: obj(map[string]any{
				"types": strs("prd, design, plan, adr, doc; leer = prd + design"), "feature": str("Feature-Slug"), "title": str("Titel"),
				"slug": str("Slug für adr/doc"), "touches": strs("erklärte Änderungsabsicht"), "archive_after": str("YYYY-MM-DD"),
				"apply": map[string]any{"type": "boolean", "description": "Dateien wirklich anlegen"},
			}), Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}},
	}
}

type toolResult struct {
	Content           []map[string]any `json:"content"`
	StructuredContent any              `json:"structuredContent,omitempty"`
	IsError           bool             `json:"isError,omitempty"`
}

func resultOf(v any) *toolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	sc := v
	if _, isObject := asObject(v); !isObject {
		sc = map[string]any{"result": v} // structuredContent must be an object
	}
	return &toolResult{Content: []map[string]any{{"type": "text", "text": string(data)}}, StructuredContent: sc}
}

func asObject(v any) (map[string]any, bool) {
	data, err := json.Marshal(v)
	if err != nil || len(data) == 0 || data[0] != '{' {
		return nil, false
	}
	var m map[string]any
	return m, json.Unmarshal(data, &m) == nil
}

func toolError(err error) *toolResult {
	return &toolResult{Content: []map[string]any{{"type": "text", "text": err.Error()}}, IsError: true}
}

func args[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 || string(raw) == "null" {
		return v, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("ungültige Argumente: %w", err)
	}
	return v, nil
}

func validGate(g string) bool {
	for _, v := range catalog.AllGates {
		if v == g {
			return true
		}
	}
	return false
}

// callTool runs one tool. Tool failures are results with isError, not protocol errors.
func (s *Server) callTool(name string, raw json.RawMessage) (any, *rpcError) {
	res, err := s.runTool(name, raw)
	if errors.Is(err, errUnknownTool) {
		return nil, &rpcError{codeInvalidParams, err.Error()}
	}
	if err != nil {
		return toolError(err), nil
	}
	return res, nil
}

var errUnknownTool = errors.New("unbekanntes Werkzeug")

func (s *Server) runTool(name string, raw json.RawMessage) (*toolResult, error) {
	switch name {
	case "sdd_next":
		a, err := args[struct {
			Feature string `json:"feature"`
		}](raw)
		if err != nil {
			return nil, err
		}
		g, err := guide.Next(s.Root, strings.TrimSpace(a.Feature))
		if err != nil {
			return nil, err
		}
		return resultOf(g), nil

	case "sdd_read":
		a, err := args[struct {
			Path string `json:"path"`
		}](raw)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(a.Path) == "" {
			paths, _, err := s.artifactPaths()
			if err != nil {
				return nil, err
			}
			return resultOf(map[string]any{"artifacts": paths}), nil
		}
		rel, text, err := s.readArtifact(a.Path)
		if err != nil {
			return nil, err
		}
		return resultOf(numbered(rel, text)), nil

	case "sdd_lint":
		a, err := args[struct {
			Gate string `json:"gate"`
		}](raw)
		if err != nil {
			return nil, err
		}
		if a.Gate != "" && !validGate(a.Gate) {
			return nil, fmt.Errorf("ungültiges Gate %q", a.Gate)
		}
		rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: s.Root, Gate: a.Gate})
		if err != nil {
			return nil, err
		}
		return resultOf(rep), nil

	case "sdd_gate":
		a, err := args[struct {
			Gate    string   `json:"gate"`
			Feature string   `json:"feature"`
			Paths   []string `json:"paths"`
		}](raw)
		if err != nil {
			return nil, err
		}
		if !validGate(a.Gate) {
			return nil, fmt.Errorf("ungültiges Gate %q (erlaubt: %s)", a.Gate, strings.Join(catalog.AllGates, ", "))
		}
		for _, p := range a.Paths {
			if _, err := scan.CleanRel(p); err != nil {
				return nil, err
			}
		}
		rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: s.Root, Gate: a.Gate, Feature: a.Feature, Paths: a.Paths})
		if err != nil {
			return nil, err
		}
		return resultOf(map[string]any{"passed": rep.ExitCode() == 0, "report": rep}), nil

	case "sdd_report":
		if _, err := args[struct{}](raw); err != nil {
			return nil, err
		}
		o, err := overview.Build(s.Root, "", scan.CLIGit{Root: s.Root})
		if err != nil {
			return nil, err
		}
		return resultOf(o), nil

	case "sdd_explain":
		a, err := args[struct {
			RuleID string `json:"rule_id"`
		}](raw)
		if err != nil {
			return nil, err
		}
		e, err := guide.Explain(s.Root, a.RuleID)
		if err != nil {
			return nil, err
		}
		return resultOf(e), nil

	case "sdd_check_intent":
		a, err := args[struct {
			Paths   []string `json:"paths"`
			Feature string   `json:"feature"`
		}](raw)
		if err != nil {
			return nil, err
		}
		in, err := guide.CheckIntent(s.Root, a.Paths, strings.TrimSpace(a.Feature))
		if err != nil {
			return nil, err
		}
		return resultOf(in), nil

	case "sdd_context":
		a, err := args[struct {
			Path string `json:"path"`
		}](raw)
		if err != nil {
			return nil, err
		}
		c, err := guide.Context(s.Root, a.Path)
		if err != nil {
			return nil, err
		}
		return resultOf(map[string]any{"path": a.Path, "governing": c}), nil

	case "sdd_trace":
		a, err := args[struct {
			Feature string `json:"feature"`
		}](raw)
		if err != nil {
			return nil, err
		}
		t, err := guide.Trace(s.Root, a.Feature)
		if err != nil {
			return nil, err
		}
		return resultOf(map[string]any{"feature": a.Feature, "criteria": t}), nil

	case "sdd_validate_draft":
		a, err := args[struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}](raw)
		if err != nil {
			return nil, err
		}
		d, err := guide.ValidateDraft(s.Root, a.Path, a.Content)
		if err != nil {
			return nil, err
		}
		return resultOf(d), nil

	case "sdd_new":
		return s.toolNew(raw)

	default:
		return nil, fmt.Errorf("%w %q", errUnknownTool, name)
	}
}

type opPreview struct {
	Path    string `json:"path"`
	Action  string `json:"action"`
	Reason  string `json:"reason,omitempty"`
	Content string `json:"content,omitempty"`
}

func (s *Server) toolNew(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Types        []string `json:"types"`
		Feature      string   `json:"feature"`
		Title        string   `json:"title"`
		Slug         string   `json:"slug"`
		Touches      []string `json:"touches"`
		ArchiveAfter string   `json:"archive_after"`
		Apply        bool     `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Apply && !s.AllowWrite {
		return nil, errors.New("Schreiben ist nicht freigegeben: der Server läuft ohne --allow-write; ohne apply gibt es die Vorschau")
	}
	cfg, err := config.Load(s.Root)
	if errors.Is(err, config.ErrConfigNotFound) {
		cfg, err = scaffold.DefaultConfig("standard"), nil
	}
	if err != nil {
		return nil, err
	}
	plan, err := scaffold.NewPlan(s.Root, cfg, scaffold.NewOptions{
		Types: a.Types, Feature: strings.TrimSpace(a.Feature), Title: a.Title, Slug: strings.TrimSpace(a.Slug),
		Touches: a.Touches, ArchiveAfter: strings.TrimSpace(a.ArchiveAfter), Today: s.Now(),
	})
	if err != nil {
		return nil, err
	}
	if !a.Apply {
		preview, err := plan.Preview(s.Root)
		if err != nil {
			return nil, err
		}
		ops := make([]opPreview, 0, len(plan.Ops))
		for i, op := range plan.Ops {
			p := opPreview{Path: op.Path, Action: "create", Content: string(op.Content)}
			if preview[i].Status == scaffold.StatusSkipped {
				p.Action, p.Reason, p.Content = "skip", preview[i].Reason, ""
			}
			ops = append(ops, p)
		}
		return resultOf(map[string]any{"applied": false, "ops": ops}), nil
	}
	results, err := plan.Apply(s.Root)
	if err != nil {
		return nil, err
	}
	return resultOf(map[string]any{"applied": true, "results": results}), nil
}

// maxReadBytes bounds the text sdd_read returns, so one artifact cannot fill a model's context.
const maxReadBytes = 256 << 10

type readResult struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
	Text      string `json:"text"`
}

// numbered prefixes every line with its number, so a model can cite file:line; long texts are cut at a line
// boundary.
func numbered(rel, text string) readResult {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	res := readResult{Path: rel, Lines: len(lines)}
	var b strings.Builder
	for i, line := range lines {
		next := fmt.Sprintf("%4d| %s\n", i+1, line)
		if b.Len()+len(next) > maxReadBytes {
			res.Truncated = true
			fmt.Fprintf(&b, "… gekürzt nach Zeile %d von %d (mehr als %d KiB)\n", i, len(lines), maxReadBytes>>10)
			break
		}
		b.WriteString(next)
	}
	res.Text = b.String()
	return res
}
