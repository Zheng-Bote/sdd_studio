package scan

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"**/*_test.go", "a_test.go", true},
		{"**/*_test.go", "internal/x/a_test.go", true},
		{"**/*_test.go", "internal/x/a.go", false},
		{"cmd/**", "cmd/sdd/main.go", true},
		{"cmd/**", "cmd", true},
		{"cmd/**", "cmdx/main.go", false},
		{"*.sdd", "md-indexer.sdd", true},
		{"*.sdd", "docs/x.sdd", false},
		{"**/*.{test,spec}.{ts,js}", "web/a.spec.ts", true},
		{"**/*.{test,spec}.{ts,js}", "web/a.unit.ts", false},
		{"internal/**/sqlite.go", "internal/sqlite.go", true},
		{"internal/**/sqlite.go", "internal/repo/db/sqlite.go", true},
		{"file?.md", "file1.md", true},
		{"file[0-9].md", "fileA.md", false},
		{"./go.mod", "go.mod", true},
	}
	for _, tc := range cases {
		if got := Match(tc.pattern, tc.path); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestOverlaps(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"go.mod", "go.mod", true},
		{"go.mod", "go.sum", false},
		{"cmd/**", "cmd/sdd/main.go", true},
		{"internal/service/**", "internal/**", true},
		{"internal/service/**", "internal/repository/**", false},
		{"**/*.go", "internal/service/**", true},
		{"*.sdd", "openapi.yaml", false},
	}
	for _, tc := range cases {
		if got := Overlaps(tc.a, tc.b); got != tc.want {
			t.Errorf("Overlaps(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := Overlaps(tc.b, tc.a); got != tc.want {
			t.Errorf("Overlaps(%q, %q) (swapped) = %v, want %v", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestParseFrontmatter(t *testing.T) {
	fm, ok := ParseFrontmatter([]byte("---\r\nfeature: billing-export\r\nstatus: active\r\ntouches:\r\n  - openapi.yaml\r\n---\r\n# PRD\n"))
	if !ok || fm.Feature != "billing-export" || fm.Status != "active" || !reflect.DeepEqual(fm.Touches, []string{"openapi.yaml"}) {
		t.Fatalf("unexpected frontmatter %+v ok=%v", fm, ok)
	}
	for name, input := range map[string]string{
		"no frontmatter": "# PRD\nfeature: x\n",
		"unterminated":   "---\nfeature: x\n",
		"invalid yaml":   "---\nfeature: [x\n---\n",
	} {
		if _, ok := ParseFrontmatter([]byte(input)); ok {
			t.Errorf("%s: expected ok=false", name)
		}
	}
	if _, ok := ParseFrontmatter([]byte("---\n---\n")); !ok {
		t.Errorf("empty frontmatter block should parse")
	}
}

func TestEvaluateThreshold(t *testing.T) {
	th := config.ThresholdsConfig{
		Layers: []string{"internal/a/**", "internal/b/**", "internal/c/**"},
		FeatureArtifactsRequiredWhen: config.FeatureThresholdTrigger{
			MinLayersTouched: 2,
			TouchedPaths:     []string{"cmd/**", "go.mod"},
		},
	}
	cases := []struct {
		name  string
		paths []string
		want  bool
	}{
		{"nothing", nil, false},
		{"one layer", []string{"internal/a/x.go"}, false},
		{"two layers", []string{"internal/a/x.go", "internal/b/y.go"}, true},
		{"same layer twice", []string{"internal/a/x.go", "internal/a/y.go"}, false},
		{"touched path", []string{"go.mod"}, true},
		{"declared glob", []string{"cmd/**"}, true},
		{"broad glob hits two layers", []string{"internal/**"}, true},
		{"docs only", []string{"docs/readme.md"}, false},
	}
	for _, tc := range cases {
		if got := EvaluateThreshold(th, tc.paths); got.Exceeded != tc.want {
			t.Errorf("%s: Exceeded = %v (%+v), want %v", tc.name, got.Exceeded, got, tc.want)
		}
	}
}

func TestListArtifacts(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "docs/prd/a.md"), "---\nfeature: a\n---\n")
	mustWrite(t, filepath.Join(root, "docs/prd/sub/b.md"), "# no frontmatter\n")
	mustWrite(t, filepath.Join(root, "docs/prd/skip/c.md"), "---\nfeature: c\n---\n")
	mustWrite(t, filepath.Join(root, "docs/prd/note.txt"), "---\nfeature: d\n---\n")

	got, err := ListArtifacts(root, "docs/prd/", []string{"docs/prd/skip/**"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "docs/prd/a.md" || got[0].Frontmatter.Feature != "a" || got[1].Path != "docs/prd/sub/b.md" || got[1].HasFrontmatter {
		t.Fatalf("unexpected artifacts %+v", got)
	}
	if missing, err := ListArtifacts(root, "docs/missing", nil); err != nil || len(missing) != 0 {
		t.Fatalf("missing dir: got %v, %v", missing, err)
	}
}

func TestCLIGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	for _, env := range []string{"GITHUB_BASE_REF", "CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "GITHUB_HEAD_REF", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "CI_COMMIT_BRANCH"} {
		t.Setenv(env, "")
	}
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	mustWrite(t, filepath.Join(root, "old.txt"), "x")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "feature/billing-export")
	git("mv", "old.txt", "renamed.txt")
	mustWrite(t, filepath.Join(root, "cmd/main.go"), "package main")
	git("add", ".")
	git("commit", "-q", "-m", "change")

	g := CLIGit{Root: root}
	base, err := g.ResolveBaseRef("")
	if err != nil || base != "main" {
		t.Fatalf("ResolveBaseRef = %q, %v", base, err)
	}
	files, err := g.ChangedFiles(base)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cmd/main.go", "old.txt", "renamed.txt"}; !reflect.DeepEqual(files, want) {
		t.Fatalf("ChangedFiles = %v, want %v", files, want)
	}
	if b, err := g.CurrentBranch(); err != nil || b != "feature/billing-export" {
		t.Fatalf("CurrentBranch = %q, %v", b, err)
	}
	if _, err := g.ResolveBaseRef("does-not-exist"); err == nil {
		t.Fatalf("expected error for unknown base_ref")
	}

	// A pushed feature branch tracks its own remote copy: that is no merge base, main is.
	remote := t.TempDir()
	git("init", "-q", "--bare", remote)
	git("remote", "add", "origin", remote)
	git("push", "-q", "-u", "origin", "main", "feature/billing-export")
	if base, err := g.ResolveBaseRef(""); err != nil || base != "main" {
		t.Fatalf("pushed feature branch: ResolveBaseRef = %q, %v", base, err)
	}
	// A branch tracking another branch keeps that upstream; main keeps origin/main (unpushed commits).
	git("checkout", "-q", "-b", "fix", "--track", "origin/main")
	if base, err := g.ResolveBaseRef(""); err != nil || base != "origin/main" {
		t.Fatalf("branch tracking origin/main: ResolveBaseRef = %q, %v", base, err)
	}
	git("checkout", "-q", "main")
	if base, err := g.ResolveBaseRef(""); err != nil || base != "origin/main" {
		t.Fatalf("main: ResolveBaseRef = %q, %v", base, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalkFiles(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"a_test.go", "pkg/b_test.go", "pkg/b.go", "vendor/x/c_test.go", ".git/d_test.go", "web/e.spec.ts"} {
		mustWrite(t, filepath.Join(root, p), "x")
	}
	got, err := WalkFiles(root, []string{"**/*_test.go", "**/*.{test,spec}.{ts,tsx,js,jsx}"}, []string{"vendor/**"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a_test.go", "pkg/b_test.go", "web/e.spec.ts"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("WalkFiles = %v, want %v", got, want)
	}
}

func TestListArtifactsRejectsPathsOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	mustWrite(t, filepath.Join(parent, "outside/secret.md"), "---\nfeature: x\n---\n")
	mustWrite(t, filepath.Join(root, "keep"), "")
	for _, dir := range []string{"../outside", "/etc", "docs/../../outside"} {
		if _, err := ListArtifacts(root, dir, nil); !errors.Is(err, ErrUnsafePath) {
			t.Errorf("%s: expected ErrUnsafePath, got %v", dir, err)
		}
	}
}

// sdd-adopt AC-1: artifact specs may be directories or file globs.
func TestArtifactSpecGlob(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "specs/features/001-billing/spec.md"), "---\nfeature: billing\n---\n")
	mustWrite(t, filepath.Join(root, "specs/features/001-billing/plan.md"), "")
	mustWrite(t, filepath.Join(root, "specs/features/002-search/spec.md"), "")
	mustWrite(t, filepath.Join(root, "specs/features/002-search/notes/spec.md"), "") // deeper: not matched by the glob

	got, err := ListArtifacts(root, "specs/features/*/spec.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, a := range got {
		paths = append(paths, a.Path)
	}
	if want := []string{"specs/features/001-billing/spec.md", "specs/features/002-search/spec.md"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("ListArtifacts(glob) = %v, want %v", paths, want)
	}
	if _, err := ListArtifacts(root, "../*/spec.md", nil); !errors.Is(err, ErrUnsafePath) {
		t.Errorf("glob outside root: expected ErrUnsafePath, got %v", err)
	}

	if !InSpec("specs/features/x/spec.md", "specs/features/*/spec.md") || InSpec("specs/features/x/plan.md", "specs/features/*/spec.md") || !InSpec("docs/prd/a/b.md", "docs/prd") {
		t.Errorf("InSpec mismatch")
	}
	for spec, want := range map[string]string{
		"specs/features/*/spec.md": "specs/features/billing/spec.md",
		"docs/prd":                 "docs/prd/billing.md",
		"docs/prd/":                "docs/prd/billing.md",
	} {
		if got, err := TargetPath(spec, "billing"); err != nil || got != want {
			t.Errorf("TargetPath(%q) = %q, %v; want %q", spec, got, err, want)
		}
	}
	for _, spec := range []string{"specs/**/spec.md", "specs/*/*/spec.md", "specs/f*/spec.md", "../x"} {
		if _, err := TargetPath(spec, "billing"); err == nil {
			t.Errorf("TargetPath(%q): expected error", spec)
		}
	}
	if got := SlugInSpec("specs/features/001-billing/spec.md", "specs/features/*/spec.md"); got != "001-billing" {
		t.Errorf("SlugInSpec(glob) = %q", got)
	}
	if got := SlugInSpec("docs/prd/billing.md", "docs/prd"); got != "billing" {
		t.Errorf("SlugInSpec(dir) = %q", got)
	}
}
