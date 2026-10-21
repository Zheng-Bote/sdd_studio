package scaffold_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

var today = time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

func initProject(t *testing.T, profile string) string {
	t.Helper()
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: profile, Project: "demo", CI: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func statuses(results []scaffold.Result) map[string]string {
	out := map[string]string{}
	for _, r := range results {
		out[r.Path] = r.Status
	}
	return out
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// sdd-init-new AC-1: init creates a project that lints without errors.
func TestInitCreatesLintCleanProject(t *testing.T) {
	root := initProject(t, "standard")
	for _, rel := range []string{"sdd.config.yaml", "constitution.md", "docs/prd/.gitkeep", "docs/design/.gitkeep", "docs/plans/.gitkeep", "docs/adr/.gitkeep", "docs/archive/.gitkeep", "docs/notes/.gitkeep", scaffold.WorkflowPath, ".sdd/baseline.json"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("expected %s: %v", rel, err)
		}
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("generated config invalid: %v", err)
	}
	if !reflect.DeepEqual(cfg.Artifacts, scaffold.DefaultArtifacts()) {
		t.Errorf("generated artifacts %+v differ from defaults", cfg.Artifacts)
	}
	rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Error != 0 {
		t.Fatalf("expected no errors after init, got %+v", rep.Findings)
	}
}

// sdd-init-new AC-3: only the three profiles are accepted and the profile is written.
func TestInitRejectsUnknownProfile(t *testing.T) {
	if _, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "strict"}); err == nil {
		t.Fatal("expected error for unknown profile")
	}
	for _, p := range []string{"minimal", "standard", "regulated"} {
		cfg, err := config.Load(initProject(t, p))
		if err != nil || cfg.Profile != p {
			t.Errorf("profile %s: got %+v, %v", p, cfg, err)
		}
	}
}

// sdd-init-new AC-2: existing files are never overwritten; Preview does not write.
func TestApplyNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "constitution.md"), []byte("# Eigene Verfassung\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs/prd"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "minimal", Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}

	preview, err := plan.Preview(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "sdd.config.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Preview must not write: %v", err)
	}

	results, err := plan.Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(statuses(preview), statuses(results)) {
		t.Errorf("preview %v differs from apply %v", statuses(preview), statuses(results))
	}
	st := statuses(results)
	if st["constitution.md"] != scaffold.StatusSkipped || st["docs/prd/.gitkeep"] != scaffold.StatusSkipped || st["sdd.config.yaml"] != scaffold.StatusCreated {
		t.Errorf("unexpected statuses %v", st)
	}
	if _, ok := st[scaffold.WorkflowPath]; ok {
		t.Errorf("CI=false must not plan the workflow")
	}
	if got := readFile(t, root, "constitution.md"); got != "# Eigene Verfassung\n" {
		t.Errorf("constitution was overwritten: %q", got)
	}

	again, err := plan.Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range again {
		if r.Status != scaffold.StatusSkipped {
			t.Errorf("second apply must skip everything, got %+v", r)
		}
	}
}

// sdd-init-new AC-1: the CLI passes "." as root; relative roots must work.
func TestApplyWithRelativeRoot(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo", CI: true})
	if err != nil {
		t.Fatal(err)
	}
	results, err := plan.Apply(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Status != scaffold.StatusCreated {
			t.Errorf("expected %s to be created, got %+v", r.Path, r)
		}
	}
}

func TestApplyRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"../evil.md", "/etc/evil.md", "a/../../evil.md"} {
		plan := scaffold.Plan{Ops: []scaffold.Op{{Path: "ok.md"}, {Path: p}}}
		if _, err := plan.Apply(root); !errors.Is(err, scaffold.ErrUnsafePath) {
			t.Errorf("%s: expected ErrUnsafePath, got %v", p, err)
		}
		if _, err := os.Stat(filepath.Join(root, "ok.md")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: nothing may be written before validation fails", p)
		}
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	plan := scaffold.Plan{Ops: []scaffold.Op{{Path: "link/evil.md"}}}
	if _, err := plan.Apply(root); !errors.Is(err, scaffold.ErrUnsafePath) {
		t.Errorf("symlink escape: expected ErrUnsafePath, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "evil.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file written outside the repository")
	}
}

// sdd-init-new AC-4: PRD and Tech Spec with all required frontmatter fields.
func TestNewFeatureCreatesPRDAndTechSpec(t *testing.T) {
	root := initProject(t, "standard")
	cfg, _ := config.Load(root)
	plan, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Feature: "billing-export", Title: "Abrechnungsexport", ArchiveAfter: "2027-01-31", Touches: []string{"./cmd/**", "openapi.yaml"}, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"docs/prd/billing-export.md", "docs/design/billing-export.md"} {
		fm, ok := scan.ParseFrontmatter([]byte(readFile(t, root, rel)))
		if !ok || fm.Feature != "billing-export" || fm.Status != "draft" || fm.ArchiveAfter != "2027-01-31" || !reflect.DeepEqual(fm.Touches, []string{"cmd/**", "openapi.yaml"}) {
			t.Errorf("%s: unexpected frontmatter %+v (ok=%v)", rel, fm, ok)
		}
	}
	if design := readFile(t, root, "docs/design/billing-export.md"); !strings.Contains(design, "docs/prd/billing-export.md") {
		t.Errorf("Tech Spec does not reference its PRD")
	}

	// Without --touches and --archive-after the fields exist but are empty.
	plan, err = scaffold.NewPlan(root, cfg, scaffold.NewOptions{Feature: "bare", Today: today})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	prd := readFile(t, root, "docs/prd/bare.md")
	fm, ok := scan.ParseFrontmatter([]byte(prd))
	if !ok || fm.ArchiveAfter != "" || len(fm.Touches) != 0 || !strings.Contains(prd, "archive-after:") || !strings.Contains(prd, "touches: []") {
		t.Errorf("bare PRD: unexpected frontmatter %+v\n%s", fm, prd)
	}
}

// sdd-init-new AC-2, AC-4: an artifact carrying the slug is kept, whatever its file name.
func TestNewSkipsExistingFeatureArtifact(t *testing.T) {
	root := initProject(t, "standard")
	cfg, _ := config.Load(root)
	own := "---\nfeature: billing-export\nstatus: active\n---\n# Bestehendes PRD\n"
	if err := os.WriteFile(filepath.Join(root, "docs/prd/abrechnung.md"), []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Feature: "billing-export", Today: today})
	if err != nil {
		t.Fatal(err)
	}
	results, err := plan.Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	st := statuses(results)
	if st["docs/prd/abrechnung.md"] != scaffold.StatusSkipped || st["docs/design/billing-export.md"] != scaffold.StatusCreated {
		t.Errorf("unexpected statuses %v", st)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing-export.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("PRD duplicated under the slug file name")
	}
	if got := readFile(t, root, "docs/prd/abrechnung.md"); got != own {
		t.Errorf("existing PRD changed")
	}
	if !strings.Contains(readFile(t, root, "docs/design/billing-export.md"), "docs/prd/abrechnung.md") {
		t.Errorf("Tech Spec should reference the existing PRD")
	}
}

// sdd-init-new AC-5: plan, ADR with sequential number, doc.
func TestNewPlanADRDoc(t *testing.T) {
	root := initProject(t, "standard")
	cfg, _ := config.Load(root)
	if err := os.WriteFile(filepath.Join(root, "docs/adr/0007-alt.md"), []byte("---\nstatus: accepted\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		opts       scaffold.NewOptions
		path, want string
	}{
		{scaffold.NewOptions{Types: []string{"plan"}, Feature: "billing-export"}, "docs/plans/billing-export.md", "feature: billing-export"},
		{scaffold.NewOptions{Types: []string{"adr"}, Title: "Größenschwelle über Globs"}, "docs/adr/0008-groessenschwelle-ueber-globs.md", "status: proposed\ndate: 2026-10-20"},
		{scaffold.NewOptions{Types: []string{"doc"}, Title: "Recherche", Slug: "recherche-ci"}, "docs/notes/recherche-ci.md", "status: active"},
	}
	for _, tc := range cases {
		tc.opts.Today = today
		plan, err := scaffold.NewPlan(root, cfg, tc.opts)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		results, err := plan.Apply(root)
		if err != nil || len(results) != 1 || results[0].Path != tc.path || results[0].Status != scaffold.StatusCreated {
			t.Fatalf("%s: got %+v, %v", tc.path, results, err)
		}
		if !strings.Contains(readFile(t, root, tc.path), tc.want) {
			t.Errorf("%s: missing %q", tc.path, tc.want)
		}
	}
}

// sdd-init-new AC-6: generated artifacts satisfy SDD-021 and SDD-050 without further edits.
func TestNewArtifactsPassRules(t *testing.T) {
	root := initProject(t, "standard")
	cfg, _ := config.Load(root)
	// RunCheck uses the real clock, so archive-after must lie in the future relative to it.
	future := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	for _, opts := range []scaffold.NewOptions{
		{Feature: "billing-export", Types: []string{"prd", "design", "plan"}, ArchiveAfter: future},
		{Types: []string{"adr"}, Title: "Erste Entscheidung"},
		{Types: []string{"doc"}, Title: "Notiz"},
	} {
		opts.Today = today
		plan, err := scaffold.NewPlan(root, cfg, opts)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := plan.Apply(root); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rep.Findings {
		if f.RuleID == "SDD-021" || f.RuleID == "SDD-050" || f.Severity == "error" {
			t.Errorf("unexpected finding %+v", f)
		}
	}
}

// sdd-init-new AC-6: invalid input is rejected before anything is planned.
func TestNewRejectsInvalidInput(t *testing.T) {
	root := initProject(t, "standard")
	cfg, _ := config.Load(root)
	for name, opts := range map[string]scaffold.NewOptions{
		"no feature, no type": {},
		"bad slug":            {Feature: "Billing_Export"},
		"bad date":            {Feature: "x", ArchiveAfter: "v1.2.0"},
		"touches escape":      {Feature: "x", Touches: []string{"../outside/**"}},
		"unknown type":        {Feature: "x", Types: []string{"spec"}},
		"adr without title":   {Types: []string{"adr"}},
		"adr with prd":        {Feature: "x", Types: []string{"adr", "prd"}, Title: "t"},
	} {
		if _, err := scaffold.NewPlan(root, cfg, opts); !errors.Is(err, scaffold.ErrInvalidInput) {
			t.Errorf("%s: expected ErrInvalidInput, got %v", name, err)
		}
	}
	bad := *cfg
	bad.Artifacts.PRD = "../outside"
	if _, err := scaffold.NewPlan(root, &bad, scaffold.NewOptions{Feature: "x", Types: []string{"prd"}}); !errors.Is(err, scaffold.ErrUnsafePath) {
		t.Errorf("artifact dir outside the repository: expected ErrUnsafePath, got %v", err)
	}
}

// sdd-adopt AC-1: sdd new derives targets from Spec Kit globs and recognizes existing artifacts there.
func TestNewWithGlobSpec(t *testing.T) {
	root := t.TempDir()
	cfg := scaffold.DefaultConfig("standard")
	cfg.Artifacts.PRD = "specs/features/*/spec.md"
	cfg.Artifacts.Design = "specs/features/*/plan.md"
	cfg.Artifacts.Plans = "specs/features/*/tasks.md"
	if err := os.MkdirAll(filepath.Join(root, "specs/features/legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "specs/features/legacy/spec.md"), []byte("---\nfeature: billing\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Feature: "billing", Types: []string{"prd", "design", "plan"}, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	results, err := plan.Apply(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"specs/features/legacy/spec.md":   scaffold.StatusSkipped,
		"specs/features/billing/plan.md":  scaffold.StatusCreated,
		"specs/features/billing/tasks.md": scaffold.StatusCreated,
	}
	if got := statuses(results); !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}

	cfg.Artifacts.PRD = "specs/**/spec.md"
	if _, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Feature: "other", Types: []string{"prd"}, Today: today}); !errors.Is(err, scaffold.ErrInvalidInput) {
		t.Errorf("non-derivable glob: expected ErrInvalidInput, got %v", err)
	}
}

// sdd-adopt AC-5: a frontmatter block is prepended only to the unchanged file; nothing is removed.
func TestApplyPrepend(t *testing.T) {
	root := t.TempDir()
	original := []byte("# Spec\n\nInhalt\n")
	target := filepath.Join(root, "specs/a/spec.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, original, 0o640); err != nil {
		t.Fatal(err)
	}
	head := []byte("---\nfeature: a\nstatus: active\n---\n\n")
	plan := scaffold.Plan{Ops: []scaffold.Op{{Path: "specs/a/spec.md", Content: head, Prepend: true, Expect: original}}}
	results, err := plan.Apply(root)
	if err != nil || len(results) != 1 || results[0].Status != scaffold.StatusPrepended {
		t.Fatalf("got %+v, %v", results, err)
	}
	if got := readFile(t, root, "specs/a/spec.md"); got != string(head)+string(original) {
		t.Fatalf("unexpected content %q", got)
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o640 {
		t.Errorf("permissions changed to %v", info.Mode().Perm())
	}
	// The file now differs from Expect: a second apply must skip instead of prepending twice.
	results, err = plan.Apply(root)
	if err != nil || results[0].Status != scaffold.StatusSkipped {
		t.Fatalf("second apply: got %+v, %v", results, err)
	}
}

// sdd-adopt AC-1: generated configuration keeps globs intact (quoted YAML).
func TestRenderConfigQuotesGlobs(t *testing.T) {
	cfg := scaffold.DefaultConfig("standard")
	cfg.Artifacts.PRD = "**/spec.md"
	cfg.Artifacts.Design = "specs/features/*/plan.md"
	content, err := scaffold.RenderConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sdd.config.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(root)
	if err != nil {
		t.Fatalf("rendered config invalid: %v\n%s", err, content)
	}
	if loaded.Artifacts.PRD != "**/spec.md" || loaded.Artifacts.Design != "specs/features/*/plan.md" {
		t.Errorf("globs not preserved: %+v", loaded.Artifacts)
	}
}
