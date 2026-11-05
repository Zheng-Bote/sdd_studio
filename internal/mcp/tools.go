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

// toolList is the fixed set of tools; the critical ones (baseline, ADR lockfile, adopt --apply, deciding an ADR) only
// with --allow-critical.
func (s *Server) toolList() []tool {
	if s.AllowCritical {
		return append(baseTools(), criticalTools...)
	}
	return baseTools()
}

// baseTools are the tools without the critical ones.
func baseTools() []tool {
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
		{Name: "sdd_stack", Title: "Technik erkennen", Description: "Sprachen des Repositorys aus Dateiendungen und Marker-Dateien (go.mod, Cargo.toml, CMakeLists.txt, package.json …), nach Zahl der Quelldateien; je Sprache Testdatei-Muster, Kommentarsyntax und Beispiel für die Testverweis-Zeile; dazu das aktuelle tests.include. Liest keine Dateiinhalte.",
			InputSchema: obj(map[string]any{}), Annotations: readOnly},
		{Name: "sdd_read", Title: "Artefakt lesen", Description: "Text eines Artefakts (Verfassung, ADR, PRD, Tech Spec, Plan, Archiv, Doku), von sdd.config.yaml, einer Testdatei (tests.include) oder einer Kontextdatei (README.md, AGENTS.md, CLAUDE.md, GEMINI.md, .gemini/GEMINI.md, …) mit Zeilennummern und sha256 des Inhalts; ohne path die Liste der lesbaren Artefakte und Kontextdateien. Liest nichts anderes.",
			InputSchema: obj(map[string]any{"path": str("Pfad relativ zum Repository, z. B. docs/design/<feature>.md; leer = Liste")}), Annotations: readOnly},
		{Name: "sdd_validate_draft", Title: "Entwurf prüfen", Description: "Prüft einen noch nicht gespeicherten Artefakt-Entwurf mit den Regeln, als läge er schon unter path.",
			InputSchema: obj(map[string]any{"path": str("Zielpfad relativ zum Repository"), "content": str("vollständiger Dateiinhalt")}, "path", "content"), Annotations: readOnly},
		{Name: "sdd_new", Title: "Artefakt anlegen", Description: "Wie `sdd new`: ohne apply nur Vorschau mit Inhalt; apply legt neue Dateien an (nur wenn der Server mit --allow-write läuft), nie überschreibend. Mit content (genau ein Typ) wird dieser geprüfte Entwurf statt des Templates angelegt; bei einem Regel-Fehler wird nichts geschrieben.",
			InputSchema: obj(map[string]any{
				"types": strs("prd, design, plan, adr, doc; leer = prd + design"), "feature": str("Feature-Slug"), "title": str("Titel"),
				"content": str("vollständiger Inhalt statt des Templates (nur mit genau einem Typ)"),
				"slug":    str("Slug für adr/doc"), "touches": strs("erklärte Änderungsabsicht"), "archive_after": str("YYYY-MM-DD"),
				"apply": map[string]any{"type": "boolean", "description": "Dateien wirklich anlegen"},
			}), Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}},
		{Name: "sdd_write", Title: "Datei schreiben", Description: "Legt ein Artefakt oder eine Testdatei (tests.include) an oder ersetzt ihren Inhalt; sdd.config.yaml ändert nur sdd_config (kritisch). Ohne apply nur Vorschau (Diff bzw. Inhalt); apply nur mit --allow-write. Eine bestehende Datei nur mit base_sha256 aus sdd_read. Artefakte werden vorher mit den Regeln geprüft, ein Fehler verhindert das Schreiben; ein ADR auf accepted setzt dieses Werkzeug nie.",
			InputSchema: obj(map[string]any{
				"path": str("Pfad relativ zum Repository"), "content": str("vollständiger neuer Inhalt"),
				"base_sha256": str("sha256 des aktuellen Inhalts aus sdd_read; Pflicht, wenn die Datei existiert"),
				"apply":       map[string]any{"type": "boolean", "description": "Datei wirklich schreiben"},
			}, "path", "content"), Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true, "openWorldHint": false}},
		{Name: "sdd_init", Title: "Projekt anlegen", Description: "Wie `sdd init [--git]`: Konfiguration, Verfassung, Verzeichnisse, CI-Workflow und leere Baseline, nie überschreibend. Mit git in einem Ordner ohne Git-Repository auch git init (Branch main) und ein erster Commit der angelegten Dateien. Ohne apply nur Vorschau; apply nur mit --allow-write.",
			InputSchema: obj(map[string]any{
				"profile": map[string]any{"type": "string", "enum": []string{"minimal", "standard", "regulated"}, "description": "Profil (Standard: standard)"},
				"no_ci":   map[string]any{"type": "boolean", "description": "keinen GitHub-Actions-Workflow anlegen"},
				"git":     map[string]any{"type": "boolean", "description": "ohne Repository: git init und erster Commit"},
				"apply":   map[string]any{"type": "boolean", "description": "wirklich anlegen"},
			}), Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}},
		{Name: "sdd_archive", Title: "Feature archivieren", Description: "Wie `sdd archive`: verschiebt PRD, Tech Spec und Plan eines Features nach artifacts.archive und setzt status: archived (optional release). Nur wenn kein PRD des Features mehr status: draft trägt und jedes Akzeptanzkriterium einen Testverweis hat; nach dem Release, nicht vor der Umsetzung. Ohne apply nur Vorschau; apply nur mit --allow-write.",
			InputSchema: obj(map[string]any{
				"feature": str("Feature-Slug"), "release": str("Version für release: im Frontmatter (optional)"),
				"apply": map[string]any{"type": "boolean", "description": "wirklich verschieben"},
			}, "feature"), Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false, "openWorldHint": false}},
		conflictsTool,
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

	case "sdd_stack":
		if _, err := args[struct{}](raw); err != nil {
			return nil, err
		}
		info, err := guide.Stack(s.Root)
		if err != nil {
			return nil, err
		}
		return resultOf(info), nil

	case "sdd_conflicts_check":
		return s.toolConflictsCheck(raw)

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
			context := []string{}
			for _, c := range contextFiles {
				if _, ok, err := readExisting(s.Root, c); ok && err == nil {
					context = append(context, c)
				}
			}
			return resultOf(map[string]any{"artifacts": paths, "config": config.DefaultConfigFileName, "context": context}), nil
		}
		rel, text, err := s.readReadable(a.Path)
		if err != nil {
			return nil, err
		}
		res := numbered(rel, text)
		res.SHA256 = sha256Hex([]byte(text))
		return resultOf(res), nil

	case "sdd_lint":
		a, err := args[struct {
			Gate string `json:"gate"`
		}](raw)
		if err != nil {
			return nil, err
		}
		if a.Gate != "" && !validGate(a.Gate) {
			return nil, catalog.InvalidGateError(a.Gate)
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
			return nil, catalog.InvalidGateError(a.Gate)
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

	case "sdd_write":
		return s.toolWrite(raw)

	case "sdd_init":
		return s.toolInit(raw)

	case "sdd_archive":
		return s.toolArchive(raw)

	case "sdd_baseline", "sdd_adr_accept", "sdd_adopt_apply", "sdd_config":
		return s.runCritical(name, raw)

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
		Content      string   `json:"content"`
		Apply        bool     `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Content != "" && len(a.Types) != 1 {
		return nil, fmt.Errorf("%w: content nur mit genau einem Typ (types)", scaffold.ErrInvalidInput)
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
	if a.Content != "" && plan.Ops[0].Skip == "" {
		if err := s.checkNewContent(cfg, a.Types[0], strings.TrimSpace(a.Feature), plan.Ops[0].Path, a.Content); err != nil {
			return nil, err
		}
		plan.Ops[0].Content = []byte(a.Content)
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
	out := make([]newResult, 0, len(results))
	for _, r := range results {
		nr := newResult{Path: r.Path, Status: r.Status, Reason: r.Reason}
		if r.Status == scaffold.StatusCreated {
			if data, ok, err := readExisting(s.Root, r.Path); err == nil && ok {
				nr.SHA256, nr.Lines = sha256Hex(data), lineCount(data)
			}
		}
		out = append(out, nr)
	}
	return resultOf(map[string]any{"applied": true, "results": out}), nil
}

// newResult is one created or skipped file of sdd_new with what was really written.
type newResult struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Lines  int    `json:"lines,omitempty"`
}

// checkNewContent checks a draft given to sdd_new: a feature artifact names its slug, and checkArtifact passes.
func (s *Server) checkNewContent(cfg *config.Config, typ, feature, rel, content string) error {
	if typ == scaffold.TypePRD || typ == scaffold.TypeDesign || typ == scaffold.TypePlan {
		if fm, _ := scan.ParseFrontmatter([]byte(content)); fm.Feature != feature {
			return fmt.Errorf("%w: content trägt feature: %q, erwartet %q", scaffold.ErrInvalidInput, fm.Feature, feature)
		}
	}
	_, err := s.checkArtifact(cfg, rel, nil, false, content)
	return err
}

// maxReadBytes bounds the text sdd_read returns, so one artifact cannot fill a model's context.
const maxReadBytes = 256 << 10

type readResult struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
	SHA256    string `json:"sha256"`
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
