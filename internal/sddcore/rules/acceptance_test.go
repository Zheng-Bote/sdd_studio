package rules

import (
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

// AC-8: every acceptance criterion in an active PRD carries a stable ID.
func TestSDD021(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/good.md", `---
feature: good
status: active
---
# PRD

Fließtext mit AC-ähnlichem Wort ist kein Kriterium.

## Akzeptanzkriterien

- **AC-1** Erstes Kriterium.
  Fortsetzungszeile ohne Kennung.
- ~~AC-2~~ entfallen.
- [x] AC-3 erledigt.

### Unterabschnitt

- AC-4 im Unterabschnitt.

## Nicht-Ziele

- Keine Kennung nötig.
`)
	writeFile(t, root, "docs/prd/bad.md", "---\nfeature: bad\n---\n"+
		"## Acceptance Criteria\n\n"+
		"- Missing ID.\n"+ // line 6
		"1. AC-01 leading zero.\n"+ // line 7
		"```\n- inside code fence\n```\n"+
		"## Rollout\n\n"+
		"- [ ] checkbox anywhere\n"+ // line 13
		"- AC-x bullet with AC- prefix\n"+ // line 14
		"- plain bullet outside section\n")
	writeFile(t, root, "docs/prd/old.md", "---\nfeature: old\nstatus: archived\n---\n## Akzeptanzkriterien\n- no id\n")

	ctx := &Context{RepoRoot: root, Config: featureConfig()}
	findings, err := checkSDD021(ctx, catalog.Rule{ID: "SDD-021"})
	if err != nil {
		t.Fatal(err)
	}
	wantLines := []int{6, 7, 13, 14}
	if len(findings) != len(wantLines) {
		t.Fatalf("expected %d findings, got %+v", len(wantLines), findings)
	}
	for i, line := range wantLines {
		if findings[i].Path != "docs/prd/bad.md" || findings[i].Line != line {
			t.Errorf("finding %d: want docs/prd/bad.md:%d, got %s:%d (%s)", i, line, findings[i].Path, findings[i].Line, findings[i].Message)
		}
	}
}
