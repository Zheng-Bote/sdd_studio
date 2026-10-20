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

// AC-9: every AC ID of an active PRD is referenced in at least one test file.
func TestSDD022(t *testing.T) {
	prd := "---\nfeature: f\n---\n## Akzeptanzkriterien\n- AC-1 a\n- AC-2 b\n- ~~AC-3~~ entfallen\n- AC-4 d, siehe auch AC-1\n- AC-12 e\n"

	t.Run("references via hyphen, identifier and comment", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/prd/f.md", prd)
		writeFile(t, root, "pkg/a_test.go", "// covers AC-1\nfunc TestAC2_Login(t *testing.T) {}\nfunc TestTRAC4(t *testing.T) {}\nvar x = \"trac4 AC-40\"\n")
		writeFile(t, root, "web/b.spec.ts", "it('ac_12 works', () => {})\n")
		writeFile(t, root, "pkg/helper.go", "// AC-4\n")
		writeFile(t, root, "vendor/x/c_test.go", "// AC-4\n")
		cfg := featureConfig()
		cfg.Tests.Include = []string{"**/*_test.go", "**/*.spec.ts"}

		findings, err := checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"})
		if err != nil {
			t.Fatal(err)
		}
		if len(findings) != 1 || findings[0].Path != "docs/prd/f.md" || findings[0].Line != 8 || findings[0].Severity != "" {
			t.Fatalf("expected one finding for AC-4 at docs/prd/f.md:8, got %+v", findings)
		}
	})

	t.Run("fallback list is used when tests.include is empty", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/prd/f.md", prd)
		writeFile(t, root, "tests/all.txt", "AC-1 AC-2 AC-4 AC-12")
		findings, err := checkSDD022(&Context{RepoRoot: root, Config: featureConfig()}, catalog.Rule{ID: "SDD-022"})
		if err != nil || len(findings) != 0 {
			t.Fatalf("expected no findings, got %+v, %v", findings, err)
		}
	})

	t.Run("no test files yields a single warn", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/prd/f.md", prd)
		findings, err := checkSDD022(&Context{RepoRoot: root, Config: featureConfig()}, catalog.Rule{ID: "SDD-022"})
		if err != nil || len(findings) != 1 || findings[0].Severity != "warn" {
			t.Fatalf("expected one warn finding, got %+v, %v", findings, err)
		}
	})
}
