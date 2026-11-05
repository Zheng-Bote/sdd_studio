package guide_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

const (
	billingPRD    = "---\nfeature: billing\nstatus: active\narchive-after: 2025-12-31\ntouches:\n  - api/**\n---\n\n# PRD\n\n## Nicht-Ziele\n\n- nichts\n\n## Akzeptanzkriterien\n\n- **AC-1** Export\n- **AC-2** Import\n"
	billingDesign = "---\nfeature: billing\nstatus: active\n---\n\n# Tech Spec\n\n| AC-1 | Export |\n| AC-2 | Import |\n"
	billingPlan   = "---\nfeature: billing\n---\n\n- [ ] AC-1 Export bauen\n"
	billingTests  = "// billing AC-1: Export\n// billing AC-2: Import\n"
)

// billingProject has the feature billing with PRD, Tech Spec, Plan (without status) and tests for both criteria.
func billingProject(t *testing.T) string {
	t.Helper()
	root := project(t)
	write(t, root, "docs/prd/billing.md", billingPRD)
	write(t, root, "docs/design/billing.md", billingDesign)
	write(t, root, "docs/plans/billing.md", billingPlan)
	write(t, root, "api/billing_test.go", billingTests)
	return root
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func gone(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return errors.Is(err, os.ErrNotExist)
}

// sdd-archive AC-1: every artifact moves to <archive>/<type folder>/…, with status: archived and release, rest unchanged.
// sdd-archive AC-3: the preview names every move and writes nothing.
func TestArchivePlanMoves(t *testing.T) {
	root := billingProject(t)
	plan, err := guide.Archive(root, "billing", "v1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	moves := map[string]string{}
	for _, op := range plan.Ops {
		moves[op.MoveFrom] = op.Path
	}
	want := map[string]string{"docs/prd/billing.md": "docs/archive/prd/billing.md", "docs/design/billing.md": "docs/archive/design/billing.md", "docs/plans/billing.md": "docs/archive/plans/billing.md"}
	if len(moves) != len(want) {
		t.Fatalf("moves %v", moves)
	}
	for from, to := range want {
		if moves[from] != to {
			t.Errorf("%s -> %q, want %q", from, moves[from], to)
		}
	}
	preview, err := plan.Preview(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range preview {
		if r.Status != scaffold.StatusMoved {
			t.Errorf("preview %+v", r)
		}
	}
	if gone(root, "docs/prd/billing.md") || !gone(root, "docs/archive/prd/billing.md") {
		t.Fatalf("preview changed files")
	}

	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	wantContent := map[string]string{
		"docs/archive/prd/billing.md":    strings.Replace(billingPRD, "status: active\n", "status: archived\n", 1),
		"docs/archive/design/billing.md": strings.Replace(billingDesign, "status: active\n", "status: archived\n", 1),
		"docs/archive/plans/billing.md":  strings.Replace(billingPlan, "feature: billing\n", "feature: billing\nstatus: archived\n", 1),
	}
	for rel, w := range wantContent {
		w = strings.Replace(w, "\n---\n", "\nrelease: v1.2.0\n---\n", 1)
		if got := read(t, root, rel); got != w {
			t.Errorf("%s:\n%s\nwant:\n%s", rel, got, w)
		}
	}
	for from := range want {
		if !gone(root, from) {
			t.Errorf("%s still exists", from)
		}
	}
}

// sdd-archive AC-2: untested criteria, an existing target, no artifacts or no archive folder stop it; nothing moves.
func TestArchiveRefuses(t *testing.T) {
	root := billingProject(t)
	write(t, root, "api/billing_test.go", "// billing AC-1: Export\n")
	_, err := guide.Archive(root, "billing", "")
	if !errors.Is(err, scaffold.ErrInvalidInput) || !strings.Contains(err.Error(), "AC-2 ohne Testverweis") || strings.Contains(err.Error(), "AC-1,") {
		t.Errorf("untested: %v", err)
	}

	write(t, root, "api/billing_test.go", billingTests)
	write(t, root, "docs/archive/design/billing.md", "alt\n")
	if _, err := guide.Archive(root, "billing", ""); !errors.Is(err, scaffold.ErrInvalidInput) || !strings.Contains(err.Error(), "Ziel existiert") {
		t.Errorf("target exists: %v", err)
	}
	if _, err := guide.Archive(root, "unknown", ""); !errors.Is(err, scaffold.ErrInvalidInput) || !strings.Contains(err.Error(), "keine Artefakte") {
		t.Errorf("no artifacts: %v", err)
	}

	cfgPath := filepath.Join(root, "sdd.config.yaml")
	cfg := read(t, root, "sdd.config.yaml")
	if !strings.Contains(cfg, `archive: "docs/archive"`) {
		t.Fatalf("setup: archive line not found in %s", cfg)
	}
	if err := os.WriteFile(cfgPath, []byte(strings.Replace(cfg, `archive: "docs/archive"`, `archive: "docs/archive/*.md"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := guide.Archive(root, "billing", ""); !errors.Is(err, scaffold.ErrInvalidInput) || !strings.Contains(err.Error(), "Datei-Glob") {
		t.Errorf("glob archive: %v", err)
	}
	for _, rel := range []string{"docs/prd/billing.md", "docs/design/billing.md", "docs/plans/billing.md"} {
		if gone(root, rel) {
			t.Errorf("%s moved despite the refusal", rel)
		}
	}
}

// sdd-archive AC-5: after archiving, SDD-050 is gone for the feature and its test references still count.
func TestArchiveThenLint(t *testing.T) {
	root := billingProject(t)
	write(t, root, "docs/prd/other.md", "---\nfeature: other\nstatus: active\n---\n\n## Nicht-Ziele\n\n- nichts\n\n## Akzeptanzkriterien\n\n- **AC-1** Anderes\n")
	write(t, root, "api/other_test.go", "// other AC-1: Anderes\n")
	findings := func() map[string]int {
		rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root})
		if err != nil {
			t.Fatal(err)
		}
		n := map[string]int{}
		for _, f := range rep.Findings {
			n[f.RuleID]++
		}
		return n
	}
	if n := findings(); n["SDD-050"] == 0 {
		t.Fatalf("setup: billing must be overdue (SDD-050): %v", n)
	}
	plan, err := guide.Archive(root, "billing", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	if n := findings(); n["SDD-050"] != 0 || n["SDD-022"] != 0 {
		t.Errorf("after archive: %v", n)
	}
}

// archive-not-draft AC-1, AC-3: a feature whose PRD is still a draft is not archived, even with every criterion tested;
// active, implemented and no status archive as before.
func TestArchiveRefusesDraft(t *testing.T) {
	root := billingProject(t)
	write(t, root, "docs/prd/billing.md", strings.Replace(billingPRD, "status: active", "status: draft", 1))
	_, err := guide.Archive(root, "billing", "v1.2.0")
	if !errors.Is(err, scaffold.ErrInvalidInput) || !strings.Contains(err.Error(), "docs/prd/billing.md hat status: draft") {
		t.Fatalf("draft: %v", err)
	}
	if gone(root, "docs/prd/billing.md") {
		t.Fatal("draft PRD moved")
	}
	for _, statusLine := range []string{"status: active\n", "status: implemented\n", ""} { // "" = no status
		write(t, root, "docs/prd/billing.md", strings.Replace(billingPRD, "status: active\n", statusLine, 1))
		if _, err := guide.Archive(root, "billing", "v1.2.0"); err != nil {
			t.Errorf("%q: %v", statusLine, err)
		}
	}
}

// archive-not-draft AC-4: handbook and README name the draft condition of sdd archive.
func TestArchiveDocsNameDraft(t *testing.T) {
	for _, f := range []string{"../../../handbuch/03-mechanismen.md", "../../../handbuch/04-agenten.md", "../../../README.md"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "`status: draft` trägt") {
			t.Errorf("%s does not name the draft condition of sdd archive", f)
		}
	}
}
