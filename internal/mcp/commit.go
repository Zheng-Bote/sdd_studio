package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// commitTool commits explicitly named artifacts, tests, configuration and check state (ADR 0008).
var commitTool = tool{Name: "sdd_commit", Title: "Dateien committen",
	Description: "KRITISCH (ADR 0008). Committet genau die genannten Dateien (Artefakte, Testdateien, sdd.config.yaml, .sdd/baseline.json, .sdd/adr-lock.json) auf dem aktuellen Branch; nie auf dem Standard-Branch, kein Quellcode, keine Platzhalter. Ohne apply nur Vorschau mit Diff. Nur auf ausdrücklichen Wunsch des Menschen.",
	InputSchema: obj(map[string]any{
		"paths":   strs("einzelne Dateien, repository-relativ"),
		"message": str("Commit-Nachricht"),
		"apply":   map[string]any{"type": "boolean", "description": "wirklich committen"},
	}, "paths", "message"), Annotations: criticalHints}

// commitState are the files of the check state sdd_commit may commit besides what sdd_write may write.
var commitState = []string{".sdd/baseline.json", ".sdd/adr-lock.json"}

func (s *Server) toolCommit(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Paths   []string `json:"paths"`
		Message string   `json:"message"`
		Apply   bool     `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Message) == "" {
		return nil, errors.New("Commit-Nachricht fehlt")
	}
	if len(a.Paths) == 0 {
		return nil, errors.New("keine Pfade genannt")
	}
	cfg, err := config.Load(s.Root)
	if err != nil {
		return nil, err
	}
	g := scan.CLIGit{Root: s.Root}
	branch := g.HeadBranch()
	if branch == "" {
		return nil, errors.New("nicht committet: kein Branch ausgecheckt (detached HEAD) — erst einen Feature-Branch anlegen")
	}
	protected := []string{"main", "master", strings.TrimPrefix(strings.TrimSpace(cfg.BaseRef), "origin/")}
	if slices.Contains(protected, branch) {
		return nil, fmt.Errorf("nicht committet: %s ist der Standard-Branch — erst einen Feature-Branch anlegen (git switch -c <name>)", branch)
	}

	ops := make([]writeOp, 0, len(a.Paths))
	paths := make([]string, 0, len(a.Paths))
	for _, p := range a.Paths {
		if strings.HasPrefix(p, ":") || scan.HasWildcard(p) {
			return nil, fmt.Errorf("nicht committet: %s ist keine einzelne Datei", p)
		}
		rel, err := scan.CleanRel(p)
		if err != nil {
			return nil, err
		}
		if slices.Contains(paths, rel) {
			continue
		}
		if !slices.Contains(commitState, rel) {
			class, err := classify(cfg, rel)
			if err != nil {
				return nil, err
			}
			if class == classContext {
				return nil, fmt.Errorf("nicht committet: %s ist eine Kontextdatei", rel)
			}
		}
		data, exists, err := readExisting(s.Root, rel)
		if err != nil {
			return nil, err
		}
		old, tracked, err := g.ShowFile("HEAD", rel)
		if err != nil {
			return nil, err
		}
		op := writeOp{Path: rel}
		switch {
		case !exists && !tracked:
			return nil, fmt.Errorf("nicht committet: %s existiert nicht", rel)
		case !exists:
			op.Action, op.Diff = "delete", scaffold.UnifiedDiff(rel, old, nil)
		case !tracked:
			op.Action, op.Content = "create", string(data)
		case string(old) == string(data):
			return nil, fmt.Errorf("nicht committet: %s hat keine Änderung", rel)
		default:
			op.Action, op.Diff = "update", scaffold.UnifiedDiff(rel, old, data)
		}
		ops, paths = append(ops, op), append(paths, rel)
	}

	summary := fmt.Sprintf("%d Datei(en) auf %s committen", len(paths), branch)
	if !a.Apply {
		return resultOf(map[string]any{"applied": false, "critical": true, "summary": summary, "branch": branch, "message": a.Message, "ops": ops}), nil
	}
	if !g.HasIdentity() {
		return nil, errors.New("nicht committet: Git-Identität fehlt (user.name, user.email)")
	}
	commit, err := g.Commit(a.Message, paths)
	if err != nil {
		return nil, err
	}
	return resultOf(map[string]any{"applied": true, "critical": true, "summary": summary, "branch": branch, "commit": commit, "paths": paths}), nil
}
