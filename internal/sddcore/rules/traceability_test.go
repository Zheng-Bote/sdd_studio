package rules

import (
	"fmt"
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

// sdd-rules-v1-1 AC-3: every task carries an AC ID in its line or continuation lines.
func TestSDD041(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/plans/billing.md", "---\nfeature: billing\n---\n# Plan\n"+
		"- [ ] AC-1 direkt\n"+ // 5 ok
		"- [x] Fortsetzung\n"+ // 6 ok via continuation
		"  setzt AC-2 um\n"+
		"- [ ] ohne Kennung\n"+ // 8 finding
		"  - Detail ohne Kennung\n"+
		"  - [ ] Unteraufgabe AC-3\n"+ // 10 ok; its AC must not count for line 8
		"* [ ] Stern ohne\n"+ // 11 ok via indented continuation after a blank line
		"\n"+
		"  AC-4 nach Leerzeile, eingerückt\n"+ // continuation of 11 -> ok
		"1. [ ] nummeriert ~~AC-5~~\n"+ // 14 finding: tombstone only
		"Text AC-6 auf gleicher Ebene\n"+
		"- [ ] letzte\n"+ // 16 finding
		"## Nächster Abschnitt\n"+
		"  AC-7 nach Überschrift zählt nicht\n")
	writeFile(t, root, "docs/plans/done.md", "---\nfeature: done\nstatus: archived\n---\n- [ ] alt\n")
	writeFile(t, root, "docs/prd/billing.md", "- [ ] Checkbox im PRD ist kein Task\n")

	findings, err := checkSDD041(&Context{RepoRoot: root, Config: featureConfig()}, catalog.Rule{ID: "SDD-041"})
	if err != nil {
		t.Fatal(err)
	}
	var lines []int
	for _, f := range findings {
		if f.Path != "docs/plans/billing.md" {
			t.Errorf("unexpected path %s", f.Path)
		}
		lines = append(lines, f.Line)
	}
	if want := []int{8, 14, 16}; fmt.Sprint(lines) != fmt.Sprint(want) {
		t.Fatalf("finding lines %v, want %v (%+v)", lines, want, findings)
	}
}

// sdd-rules-v1-1 AC-4: active feature artifacts must not link to archived feature artifacts.
func TestSDD051(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/archive/old.md", "---\nfeature: old\nstatus: archived\n---\n")
	writeFile(t, root, "docs/prd/retired.md", "---\nfeature: retired\nstatus: archived\n---\n")
	writeFile(t, root, "docs/prd/live.md", "---\nfeature: live\nstatus: active\n---\n")
	writeFile(t, root, "docs/prd/new.md", "---\nfeature: new\n---\n"+
		"Siehe [alt](../archive/old.md#ziel) und [live](live.md).\n"+ // 4: finding (archive dir)
		"Ref auf ![Bild](/docs/archive/old.md \"Titel\")\n"+ // 5: finding (root-relative)
		"[r]: retired.md\n"+ // 6: finding (status archived)
		"[extern](https://example.com/docs/archive/old.md) [anker](#x) [raus](../../../etc/passwd)\n"+
		"```\n[im Code](../archive/old.md)\n```\n")
	writeFile(t, root, "docs/design/new.md", "---\nfeature: new\nstatus: superseded\n---\n[alt](../archive/old.md)\n") // inactive source
	writeFile(t, root, "docs/adr/0001-x.md", "[alt](../archive/old.md)\n")                                             // ADRs may cite history
	cfg := featureConfig()
	cfg.Artifacts.Archive = "docs/archive"

	findings, err := checkSDD051(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-051"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range findings {
		got = append(got, fmt.Sprintf("%s:%d", f.Path, f.Line))
	}
	if want := []string{"docs/prd/new.md:4", "docs/prd/new.md:5", "docs/prd/new.md:6"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// sdd-rules-003-040 AC-2: a feature with tasks needs an active Tech Spec.
func TestSDD040(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/plans/billing.md", "---\nfeature: billing\n---\n# Plan\n- [ ] AC-1 erste Task\n- [ ] AC-2 zweite\n") // no Tech Spec -> line 5
	writeFile(t, root, "docs/plans/search.md", "---\nfeature: search\n---\n- [ ] AC-1 x\n")                                       // has Tech Spec
	writeFile(t, root, "docs/design/search.md", "---\nfeature: search\n---\n")
	writeFile(t, root, "docs/plans/notes.md", "---\nfeature: notes\n---\nNoch keine Tasks.\n")       // no tasks
	writeFile(t, root, "docs/plans/old.md", "---\nfeature: old\nstatus: archived\n---\n- [ ] alt\n") // inactive
	writeFile(t, root, "docs/plans/retired.md", "---\nfeature: retired\n---\n- [ ] AC-1 x\n")
	writeFile(t, root, "docs/design/retired.md", "---\nfeature: retired\nstatus: superseded\n---\n") // inactive Tech Spec does not count

	got := map[string]int{}
	findings, err := checkSDD040(&Context{RepoRoot: root, Config: featureConfig()}, catalog.Rule{ID: "SDD-040"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		got[fmt.Sprintf("%s:%d", f.Path, f.Line)]++
	}
	if want := map[string]int{"docs/plans/billing.md:5": 1, "docs/plans/retired.md:4": 1}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Spec Kit: tasks.md without plan.md in the same feature folder.
	kit := t.TempDir()
	writeFile(t, kit, "specs/features/001-a/tasks.md", "- [ ] T1\n")
	writeFile(t, kit, "specs/features/002-b/tasks.md", "- [ ] T1\n")
	writeFile(t, kit, "specs/features/002-b/plan.md", "# Plan\n")
	cfg := featureConfig()
	cfg.Artifacts.PRD, cfg.Artifacts.Design, cfg.Artifacts.Plans = "specs/features/*/spec.md", "specs/features/*/plan.md", "specs/features/*/tasks.md"
	findings, err = checkSDD040(&Context{RepoRoot: kit, Config: cfg}, catalog.Rule{ID: "SDD-040"})
	if err != nil || len(findings) != 1 || findings[0].Path != "specs/features/001-a/tasks.md" {
		t.Fatalf("Spec Kit: %+v, %v", findings, err)
	}
}
