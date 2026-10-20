package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

// sdd-core-v1 AC-8: every acceptance criterion in an active PRD carries a stable ID.
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

// sdd-core-v1 AC-9: every AC ID of an active PRD is referenced in at least one test file.
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

// sdd-core-v1 AC-9: an AC number defined by several PRDs needs the feature slug on the referencing test line.
func TestSDD022AmbiguousNumbersNeedSlug(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/billing.md", "---\nfeature: billing\n---\n## Akzeptanzkriterien\n- AC-1 a\n- AC-2 b\n")
	writeFile(t, root, "docs/prd/billing-v2.md", "---\nfeature: billing-v2\n---\n## Akzeptanzkriterien\n- AC-1 c\n")
	writeFile(t, root, "docs/prd/no-frontmatter.md", "## Akzeptanzkriterien\n- AC-2 d\n")
	// AC-1 of billing-v2 and AC-2 of no-frontmatter (slug from file name) are covered; billing's AC-1 and AC-2 are not:
	// the token "billing" inside "billing-v2" does not count.
	writeFile(t, root, "pkg/v2_test.go", "// billing-v2 AC-1\n")
	writeFile(t, root, "pkg/legacy_test.go", "// no-frontmatter: AC-2\n")
	cfg := featureConfig()
	cfg.Tests.Include = []string{"**/*_test.go"}

	findings, err := checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range findings {
		got[f.Path+" "+strings.Fields(f.Message)[0]] = true
	}
	want := map[string]bool{"docs/prd/billing.md AC-1": true, "docs/prd/billing.md AC-2": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %+v", want, findings)
	}

	// Slug and reference on different lines do not count, even in the same file.
	writeFile(t, root, "pkg/split_test.go", "// billing\n// AC-1 AC-2\n")
	if findings, err = checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"}); err != nil || len(findings) != 2 {
		t.Fatalf("slug on another line must not count, got %+v, %v", findings, err)
	}

	// With several features even a unique number needs the slug on the line.
	writeFile(t, root, "docs/prd/billing-v2.md", "---\nfeature: billing-v2\n---\n## Akzeptanzkriterien\n- AC-3 c\n")
	writeFile(t, root, "pkg/v2_test.go", "// AC-3\n")
	findings, err = checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"})
	flagged := false
	for _, f := range findings {
		flagged = flagged || (f.Path == "docs/prd/billing-v2.md" && strings.HasPrefix(f.Message, "AC-3 "))
	}
	if err != nil || !flagged {
		t.Fatalf("unique number without slug must not count with several features, got %+v, %v", findings, err)
	}
	writeFile(t, root, "pkg/v2_test.go", "// billing-v2 AC-3\n")
	writeFile(t, root, "pkg/billing_test.go", "// billing AC-1, AC-2\n")
	if findings, err = checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"}); err != nil || len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v, %v", findings, err)
	}
}

// sdd-core-v1 AC-9: with more than one known feature (archived ones included) a reference must name the slug,
// even when the number is unique among active PRDs.
func TestSDD022ForeignSlugDoesNotCount(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/new.md", "---\nfeature: new\n---\n## Akzeptanzkriterien\n- AC-1 a\n- AC-2 b\n- AC-3 c\n")
	writeFile(t, root, "docs/archive/prd/old.md", "---\nfeature: old\nstatus: archived\n---\n## Akzeptanzkriterien\n- AC-1 x\n- AC-2 y\n")
	writeFile(t, root, "pkg/a_test.go", "// old AC-1: archived feature\nfunc TestOld(t *testing.T) {}\n// new AC-2\nfunc TestNew(t *testing.T) {}\n// AC-3 without slug\nfunc TestPlain(t *testing.T) {}\n")
	cfg := featureConfig()
	cfg.Artifacts.Archive = "docs/archive"
	cfg.Tests.Include = []string{"**/*_test.go"}

	findings, err := checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range findings {
		got = append(got, strings.Fields(f.Message)[0])
	}
	if want := []string{"AC-1", "AC-3"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v (%+v)", got, want, findings)
	}

	// A single-feature project needs no slug.
	if err := os.RemoveAll(filepath.Join(root, "docs/archive")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "pkg/a_test.go", "// AC-1\n// AC-2\n// AC-3\n")
	if findings, err = checkSDD022(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-022"}); err != nil || len(findings) != 0 {
		t.Fatalf("single feature: %+v, %v", findings, err)
	}
}
