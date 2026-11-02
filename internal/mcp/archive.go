package mcp

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

type moveOp struct {
	Path   string `json:"path"`
	Action string `json:"action"` // move
	From   string `json:"from"`
	Diff   string `json:"diff"`
}

// toolArchive is `sdd archive` as a tool: preview of every move with its frontmatter diff, apply only with --allow-write.
func (s *Server) toolArchive(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Feature string `json:"feature"`
		Release string `json:"release"`
		Apply   bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Apply && !s.AllowWrite {
		return nil, errors.New("Schreiben ist nicht freigegeben: der Server läuft ohne --allow-write; ohne apply gibt es die Vorschau")
	}
	plan, err := guide.Archive(s.Root, strings.TrimSpace(a.Feature), a.Release)
	if err != nil {
		return nil, err
	}
	if !a.Apply {
		ops := make([]moveOp, 0, len(plan.Ops))
		for _, op := range plan.Ops {
			ops = append(ops, moveOp{Path: op.Path, Action: "move", From: op.MoveFrom, Diff: scaffold.UnifiedDiff(op.Path, op.Expect, op.Content)})
		}
		return resultOf(map[string]any{"applied": false, "ops": ops}), nil
	}
	results, err := plan.Apply(s.Root)
	if err != nil {
		return nil, err
	}
	type moveResult struct {
		Path   string `json:"path"`
		Status string `json:"status"`
		From   string `json:"from"`
	}
	out := make([]moveResult, 0, len(results))
	for i, r := range results {
		out = append(out, moveResult{Path: r.Path, Status: r.Status, From: plan.Ops[i].MoveFrom})
	}
	return resultOf(map[string]any{"applied": true, "results": out}), nil
}
