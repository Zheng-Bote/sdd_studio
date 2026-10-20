package rules

import (
	"strings"
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

// sdd-rules-v1-1 AC-2: every AC of the PRD appears in a Tech Spec of the same feature.
func TestSDD031(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/billing.md", "---\nfeature: billing\n---\n## Akzeptanzkriterien\n- AC-1 a\n- AC-2 b\n- ~~AC-3~~ entfallen\n- AC-4 d\n")
	writeFile(t, root, "docs/design/billing.md", "---\nfeature: billing\n---\n| AC-1 | x |\n```\nAC-2 im Codeblock zählt nicht\n```\n")
	writeFile(t, root, "docs/design/billing-api.md", "---\nfeature: billing\n---\nSetzt AC-4 um, ~~AC-2~~ nicht.\n")
	writeFile(t, root, "docs/prd/search.md", "---\nfeature: search\n---\n## Akzeptanzkriterien\n- AC-1 a\n") // no Tech Spec: SDD-030
	writeFile(t, root, "docs/design/other.md", "---\nfeature: other\n---\nAC-1\n")

	findings, err := checkSDD031(&Context{RepoRoot: root, Config: featureConfig()}, catalog.Rule{ID: "SDD-031"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Path != "docs/design/billing-api.md" || !strings.Contains(findings[0].Message, "AC-2 aus docs/prd/billing.md") {
		t.Fatalf("expected one finding for billing AC-2, got %+v", findings)
	}
}
