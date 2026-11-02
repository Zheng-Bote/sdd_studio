package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

// initNext is the step after `sdd init`, as the CLI names it.
const initNext = "thresholds.layers und thresholds.feature_artifacts_required_when.touched_paths in sdd.config.yaml an das Projekt anpassen — solange beide leer sind, verlangt SDD-020/030 nie ein PRD."

// toolInit is `sdd init [--git]` as a tool: preview without apply, apply only with --allow-write. Git preconditions are
// checked before anything is written.
func (s *Server) toolInit(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Profile string `json:"profile"`
		NoCI    bool   `json:"no_ci"`
		Git     bool   `json:"git"`
		Apply   bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Apply && !s.AllowWrite {
		return nil, errors.New("Schreiben ist nicht freigegeben: der Server läuft ohne --allow-write; ohne apply gibt es die Vorschau")
	}
	if a.Profile == "" {
		a.Profile = "standard"
	}
	project := "Projekt"
	if abs, err := filepath.Abs(s.Root); err == nil {
		project = filepath.Base(abs)
	}
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: a.Profile, Project: project, CI: !a.NoCI})
	if err != nil {
		return nil, err
	}
	agents, _, err := scaffold.AgentFilesPlan(s.Root, scaffold.DefaultConfig(a.Profile))
	if err != nil {
		return nil, err
	}
	plan.Ops = append(plan.Ops, agents.Ops...)
	mode, gitNote := scaffold.GitSkip, "ohne Git"
	if a.Git {
		if mode, err = scaffold.CheckGitInit(s.Root); err != nil {
			return nil, err
		}
		gitNote = scaffold.GitSkipNote
		if mode == scaffold.GitCreate {
			gitNote = fmt.Sprintf("git init (Branch %s) und ein Commit mit den angelegten Dateien", scaffold.InitBranch)
		}
	}

	if !a.Apply {
		preview, err := plan.Preview(s.Root)
		if err != nil {
			return nil, err
		}
		ops := make([]writeOp, 0, len(plan.Ops))
		for i, op := range plan.Ops {
			p := writeOp{Path: op.Path, Action: "create", Content: string(op.Content)}
			switch {
			case preview[i].Status == scaffold.StatusSkipped:
				p = writeOp{Path: op.Path, Action: "skip", Reason: preview[i].Reason}
			case op.Prepend:
				p = writeOp{Path: op.Path, Action: "update", Diff: scaffold.UnifiedDiff(op.Path, op.Expect, append(append([]byte{}, op.Content...), op.Expect...))}
			}
			ops = append(ops, p)
		}
		return resultOf(map[string]any{"applied": false, "ops": ops, "git": gitNote}), nil
	}

	results, err := plan.Apply(s.Root)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"applied": true, "results": results, "git": gitNote, "next": initNext}
	if a.Git && mode == scaffold.GitCreate {
		hash, err := scaffold.CommitInit(s.Root, results)
		if err != nil {
			return nil, fmt.Errorf("Dateien angelegt, Git fehlgeschlagen: %w", err)
		}
		out["commit"] = hash
	}
	return resultOf(out), nil
}
