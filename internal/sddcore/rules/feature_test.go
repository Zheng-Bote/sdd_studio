package rules

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

type fakeGit struct {
	changed []string
	branch  string
	err     error
}

func (f fakeGit) ChangedFiles(string) ([]string, error) { return f.changed, f.err }
func (f fakeGit) CurrentBranch() (string, error)        { return f.branch, f.err }
func (f fakeGit) ResolveBaseRef(string) (string, error) { return "main", f.err }

func featureConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Profile: "standard",
		Artifacts: config.ArtifactsConfig{
			PRD:    "docs/prd",
			Design: "docs/design",
			Plans:  "docs/plans",
		},
		Thresholds: config.ThresholdsConfig{
			Layers: []string{"internal/a/**", "internal/b/**"},
			FeatureArtifactsRequiredWhen: config.FeatureThresholdTrigger{
				MinLayersTouched: 2,
				TouchedPaths:     []string{"cmd/**", "go.mod"},
			},
		},
		Exclude: []string{"vendor/**"},
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type featureResult struct {
	severity string
	contains string
}

// AC-7: PRD above the size threshold, warn on undeclared intent, fail-closed when unresolved.
func TestSDD020(t *testing.T) {
	root := t.TempDir()
	// billing has a PRD declaring its intent; reports has only a Tech Spec; notes has no frontmatter.
	writeFile(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: active\ntouches:\n  - cmd/**\n---\n# PRD\n")
	writeFile(t, root, "docs/design/reports.md", "---\nfeature: reports\ntouches:\n  - go.mod\n---\n")
	writeFile(t, root, "docs/design/quiet.md", "---\nfeature: quiet\ntouches:\n  - internal/a/**\n---\n")
	writeFile(t, root, "docs/prd/notes.md", "feature: reports\n")

	cases := []struct {
		name    string
		gate    string
		feature string
		paths   []string
		git     fakeGit
		want    []featureResult
	}{
		{name: "lint without gate is not evaluated", gate: "", git: fakeGit{changed: []string{"cmd/x.go"}}},
		{name: "pre-release is not evaluated", gate: "pre-release", git: fakeGit{changed: []string{"cmd/x.go"}}},

		// pre-implementation: declared intent
		{name: "impl: PRD present", gate: "pre-implementation", feature: "billing"},
		{name: "impl: PRD missing, intent from tech spec touches", gate: "pre-implementation", feature: "reports",
			want: []featureResult{{"", "Feature reports"}}},
		{name: "impl: threshold not touched", gate: "pre-implementation", feature: "quiet"},
		{name: "impl: unknown feature without intent warns", gate: "pre-implementation", feature: "fresh",
			want: []featureResult{{"warn", "Feature fresh ohne `touches`"}}},
		{name: "impl: unknown feature with --paths over threshold", gate: "pre-implementation", feature: "fresh", paths: []string{"internal/a/**", "internal/b/x.go"},
			want: []featureResult{{"", "Feature fresh"}}},
		{name: "impl: unresolved without --paths warns", gate: "pre-implementation", git: fakeGit{branch: "main"},
			want: []featureResult{{"warn", "kein Feature aufgelöst"}}},
		{name: "impl: unresolved with --paths over threshold fails closed", gate: "pre-implementation", paths: []string{"go.mod"}, git: fakeGit{branch: "fix/typo"},
			want: []featureResult{{"", "kein Feature auflösbar"}}},
		{name: "impl: unresolved with --paths below threshold", gate: "pre-implementation", paths: []string{"docs/readme.md"}, git: fakeGit{branch: "main"}},
		{name: "impl: feature from branch", gate: "pre-implementation", git: fakeGit{branch: "feat/42-reports"},
			want: []featureResult{{"", "Branch feat/42-reports"}}},

		// pre-merge: diff
		{name: "merge: feature from frontmatter in diff, PRD present", gate: "pre-merge", git: fakeGit{changed: []string{"docs/prd/billing.md", "cmd/x.go"}}},
		{name: "merge: feature from frontmatter in diff, PRD missing", gate: "pre-merge", git: fakeGit{changed: []string{"docs/design/reports.md", "go.mod"}},
			want: []featureResult{{"", "Frontmatter im Diff"}}},
		{name: "merge: file without frontmatter does not resolve", gate: "pre-merge", git: fakeGit{changed: []string{"docs/prd/notes.md", "go.mod"}, branch: "main"},
			want: []featureResult{{"", "kein Feature auflösbar"}}},
		{name: "merge: below threshold", gate: "pre-merge", git: fakeGit{changed: []string{"internal/a/x.go"}, branch: "main"}},
		{name: "merge: excluded paths are ignored", gate: "pre-merge", git: fakeGit{changed: []string{"vendor/cmd/x.go"}, branch: "main"}},
		{name: "merge: two features, one missing", gate: "pre-merge", feature: "billing,reports", git: fakeGit{changed: []string{"cmd/x.go"}},
			want: []featureResult{{"", "Feature reports"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &Context{RepoRoot: root, Config: featureConfig(), Gate: tc.gate, Feature: tc.feature, Paths: tc.paths, Git: tc.git}
			findings, err := checkSDD020(ctx, catalog.Rule{ID: "SDD-020"})
			if err != nil {
				t.Fatalf("checkSDD020 returned error: %v", err)
			}
			if len(findings) != len(tc.want) {
				t.Fatalf("expected %d findings, got %+v", len(tc.want), findings)
			}
			for i, w := range tc.want {
				if findings[i].Severity != w.severity || !strings.Contains(findings[i].Message, w.contains) {
					t.Errorf("finding %d: want severity %q containing %q, got %q %q", i, w.severity, w.contains, findings[i].Severity, findings[i].Message)
				}
				if findings[i].Path != "docs/prd" {
					t.Errorf("finding %d: want path docs/prd, got %q", i, findings[i].Path)
				}
			}
		})
	}
}

// AC-7: Tech Spec above the size threshold.
func TestSDD030UsesDesignDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/billing.md", "---\nfeature: billing\ntouches: [cmd/**]\n---\n")
	writeFile(t, root, "docs/design/reports.md", "---\nfeature: reports\ntouches: [go.mod]\n---\n")

	for slug, wantFinding := range map[string]bool{"billing": true, "reports": false} {
		ctx := &Context{RepoRoot: root, Config: featureConfig(), Gate: "pre-implementation", Feature: slug, Git: fakeGit{}}
		findings, err := checkSDD030(ctx, catalog.Rule{ID: "SDD-030"})
		if err != nil {
			t.Fatal(err)
		}
		if got := len(findings) == 1 && strings.Contains(findings[0].Message, "Tech Spec") && findings[0].Path == "docs/design"; got != wantFinding {
			t.Errorf("%s: want finding=%v, got %+v", slug, wantFinding, findings)
		}
	}
}

func TestFeatureChecksPropagateErrors(t *testing.T) {
	root := t.TempDir()
	gitErr := errors.New("git kaputt")
	ctx := &Context{RepoRoot: root, Config: featureConfig(), Gate: "pre-merge", Git: fakeGit{err: gitErr}}
	if _, err := checkSDD020(ctx, catalog.Rule{ID: "SDD-020"}); !errors.Is(err, gitErr) {
		t.Fatalf("expected git error, got %v", err)
	}
	ctx = &Context{RepoRoot: root, Config: featureConfig(), Gate: "pre-implementation", Feature: "Bad_Slug", Git: fakeGit{}}
	if _, err := checkSDD020(ctx, catalog.Rule{ID: "SDD-020"}); err == nil || !strings.Contains(err.Error(), "ungültiger Feature-Slug") {
		t.Fatalf("expected invalid slug error, got %v", err)
	}
}

func TestRunnerKeepsCheckSeverity(t *testing.T) {
	root := t.TempDir()
	cat := &catalog.Catalog{Rules: []catalog.Rule{{
		ID:          "SDD-020",
		Severity:    catalog.SeveritySpec{ByGate: map[string]string{"pre-implementation": "error"}},
		Profile:     "standard",
		Check:       "feature_prd_present",
		GateDefault: []string{"pre-implementation"},
	}}}
	ctx := &Context{RepoRoot: root, Config: featureConfig(), Catalog: cat, Gate: "pre-implementation", Git: fakeGit{branch: "main"}}
	findings, err := Evaluate(ctx, NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Severity != "warn" || findings[0].RuleID != "SDD-020" {
		t.Fatalf("expected one warn finding for SDD-020, got %+v", findings)
	}
}
