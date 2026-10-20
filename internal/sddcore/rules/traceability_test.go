package rules

import (
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

func findingPaths(t *testing.T, check CheckFunc, ctx *Context, id string) map[string]int {
	t.Helper()
	findings, err := check(ctx, catalog.Rule{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, f := range findings {
		out[f.Path]++
	}
	return out
}

// sdd-rules-v1-1 AC-1: active PRDs need a non-empty non-goals section.
func TestSDD023(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/ok.md", "---\nfeature: ok\n---\n## Nicht-Ziele\n\n- kein Export\n## Akzeptanzkriterien\n")
	writeFile(t, root, "docs/prd/english.md", "# PRD\n### Out of Scope {#oos}\nNothing else.\n")
	writeFile(t, root, "docs/prd/nongoals.md", "## Non-Goals ##\n- x\n")
	writeFile(t, root, "docs/prd/missing.md", "## Ziel\n- x\n")
	writeFile(t, root, "docs/prd/empty.md", "## Nicht-Ziele\n\n## Akzeptanzkriterien\n- AC-1 x\n")
	writeFile(t, root, "docs/prd/subsection-empty.md", "## Nicht-Ziele\n### Details\n- zählt nicht\n")
	writeFile(t, root, "docs/prd/fenced.md", "```\n## Nicht-Ziele\n- x\n```\n")
	writeFile(t, root, "docs/prd/old.md", "---\nfeature: old\nstatus: archived\n---\n## Ziel\n")
	writeFile(t, root, "docs/archive/moved.md", "## Ziel\n")
	cfg := featureConfig()
	cfg.Artifacts.Archive = "docs/archive"

	got := findingPaths(t, checkSDD023, &Context{RepoRoot: root, Config: cfg}, "SDD-023")
	want := map[string]int{"docs/prd/missing.md": 1, "docs/prd/empty.md": 1, "docs/prd/subsection-empty.md": 1, "docs/prd/fenced.md": 1}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for p := range want {
		if got[p] != 1 {
			t.Errorf("expected one finding for %s, got %v", p, got)
		}
	}
}
