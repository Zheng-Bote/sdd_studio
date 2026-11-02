package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adopt"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// criticalTools are offered only with --allow-critical (ADR 0003, tier "critical"): each execution needs a human's
// single confirmation, which the client (sdd-desk) enforces.
var criticalTools = []tool{
	{Name: "sdd_baseline", Title: "Baseline pflegen", Description: "KRITISCH. Wie `sdd baseline [--gate] [--prune] [--accept <Regel>]`: schreibt .sdd/baseline.json und lockt entschiedene ADRs in .sdd/adr-lock.json; --accept SDD-011 quittiert geänderte ADRs. Nur auf ausdrücklichen Wunsch des Menschen; ohne apply nur Vorschau.",
		InputSchema: obj(map[string]any{
			"gate": gateEnum, "prune": map[string]any{"type": "boolean", "description": "erledigte Einträge entfernen"},
			"accept": str("nur neue Befunde dieser Regel aufnehmen (SDD-011: geänderte ADRs quittieren)"),
			"apply":  map[string]any{"type": "boolean", "description": "wirklich schreiben"},
		}), Annotations: criticalHints},
	{Name: "sdd_adr_accept", Title: "ADR entscheiden", Description: "KRITISCH. Setzt ein ADR mit status: proposed auf status: accepted; danach ist sein Text unveränderlich und wird mit sdd_baseline gelockt. Nur auf ausdrücklichen Wunsch des Menschen; ohne apply nur Vorschau.",
		InputSchema: obj(map[string]any{
			"path":  str("Pfad des ADR, z. B. docs/adr/0004-x.md"),
			"apply": map[string]any{"type": "boolean", "description": "wirklich entscheiden"},
		}, "path"), Annotations: criticalHints},
	{Name: "sdd_adopt_apply", Title: "Übernahme anwenden", Description: "KRITISCH. Wie `sdd adopt --apply`: wendet die Übernahme eines bestehenden Projekts auf einem neuen Branch an, ein Commit je Schritt (ADR 0001). Nur auf ausdrücklichen Wunsch des Menschen; ohne apply nur der Bericht.",
		InputSchema: obj(map[string]any{
			"profile": map[string]any{"type": "string", "enum": []string{"minimal", "standard", "regulated"}, "description": "Profil (Standard: standard)"},
			"branch":  str("neuer Branch (Standard: sdd/adopt)"),
			"apply":   map[string]any{"type": "boolean", "description": "wirklich anwenden"},
		}), Annotations: criticalHints},
}

var criticalHints = map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false, "openWorldHint": false}

func (s *Server) runCritical(name string, raw json.RawMessage) (*toolResult, error) {
	if !s.AllowCritical {
		return nil, fmt.Errorf("%w %q", errUnknownTool, name)
	}
	switch name {
	case "sdd_baseline":
		return s.toolBaseline(raw)
	case "sdd_adr_accept":
		return s.toolADRAccept(raw)
	default:
		return s.toolAdoptApply(raw)
	}
}

func (s *Server) toolBaseline(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Gate   string `json:"gate"`
		Prune  bool   `json:"prune"`
		Accept string `json:"accept"`
		Apply  bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	plan, err := sddcore.PlanBaseline(s.Root, sddcore.BaselineOptions{Gate: a.Gate, Prune: a.Prune, Accept: strings.TrimSpace(a.Accept)})
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"critical": true, "summary": plan.Summary(),
		"added": entriesOrEmpty(plan.Added), "removed": entriesOrEmpty(plan.Removed),
		"locked": stringsOrEmpty(plan.Locked), "reaccepted": stringsOrEmpty(plan.Reaccepted),
	}
	if !a.Apply {
		out["applied"] = false
		return resultOf(out), nil
	}
	if err := plan.Apply(s.Root); err != nil {
		return nil, err
	}
	out["applied"], out["entries"], out["locked_total"] = true, len(plan.File.Entries), len(plan.Lock.Entries)
	return resultOf(out), nil
}

func (s *Server) toolADRAccept(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Path  string `json:"path"`
		Apply bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(s.Root)
	if err != nil {
		return nil, err
	}
	plan, err := scaffold.AcceptADRPlan(s.Root, cfg, a.Path)
	if err != nil {
		return nil, err
	}
	op := plan.Ops[0]
	summary := fmt.Sprintf("ADR %s entscheiden (status: accepted)", op.Path)
	if !a.Apply {
		return resultOf(map[string]any{"applied": false, "critical": true, "summary": summary,
			"ops": []writeOp{{Path: op.Path, Action: "update", Diff: scaffold.UnifiedDiff(op.Path, op.Expect, op.Content)}}}), nil
	}
	results, err := plan.Apply(s.Root)
	if err != nil {
		return nil, err
	}
	if results[0].Status == scaffold.StatusSkipped {
		return nil, fmt.Errorf("%s nicht entschieden: %s", op.Path, results[0].Reason)
	}
	written, _, err := readExisting(s.Root, op.Path)
	if err != nil {
		return nil, err
	}
	return resultOf(map[string]any{"applied": true, "critical": true, "summary": summary, "path": op.Path, "status": results[0].Status,
		"sha256": sha256Hex(written), "next": "Mit sdd_baseline locken, sobald der Mensch es verlangt (SDD-011)."}), nil
}

func (s *Server) toolAdoptApply(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Profile string `json:"profile"`
		Branch  string `json:"branch"`
		Apply   bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Profile == "" {
		a.Profile = "standard"
	}
	if strings.TrimSpace(a.Branch) == "" {
		a.Branch = "sdd/adopt"
	}
	project := "Projekt"
	if abs, err := filepath.Abs(s.Root); err == nil {
		project = filepath.Base(abs)
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	proposal, err := adopt.Propose(s.Root, adopt.Options{Profile: a.Profile, Branch: a.Branch, Project: project, Today: now()})
	if err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Übernahme auf neuem Branch %s anwenden (ein Commit je Schritt)", a.Branch)
	if !a.Apply {
		return resultOf(map[string]any{"applied": false, "critical": true, "summary": summary, "report": adopt.RenderReport(proposal, nil)}), nil
	}
	res, err := adopt.Apply(s.Root, proposal, scan.CLIGit{Root: s.Root})
	if err != nil {
		if res != nil && len(res.Commits) > 0 {
			return nil, fmt.Errorf("%w — Branch %s enthält die Schritte bis „%s“; der Ausgangsbranch %s ist unverändert", err, res.Branch, res.Commits[len(res.Commits)-1].Step, res.FromBranch)
		}
		return nil, err
	}
	return resultOf(map[string]any{"applied": true, "critical": true, "summary": summary, "branch": res.Branch, "from_branch": res.FromBranch, "commits": res.Commits}), nil
}

func entriesOrEmpty(e []baseline.Entry) []baseline.Entry {
	if e == nil {
		return []baseline.Entry{}
	}
	return e
}

func stringsOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
