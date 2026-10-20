package rules

import (
	"fmt"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

// sdd-rules-v1-1 AC-7: in pre-merge every changed implementation file lies within the declared touches.
func TestSDD024(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/billing.md", "---\nfeature: billing\ntouches: [cmd/**]\n---\n")
	writeFile(t, root, "docs/design/billing.md", "---\nfeature: billing\ntouches: [internal/a/**]\n---\n")
	writeFile(t, root, "docs/prd/quiet.md", "---\nfeature: quiet\n---\n")
	cfg := featureConfig()
	cfg.Artifacts.Constitution = "constitution.md"
	cfg.Artifacts.ADR = "docs/adr"
	cfg.Artifacts.Docs = []string{"docs/notes"}
	diff := []string{"cmd/x.go", "internal/a/y.go", "internal/b/z.go", "go.mod", "docs/prd/billing.md", "docs/adr/0001-x.md",
		"docs/notes/n.md", "constitution.md", ".sdd/baseline.json", "vendor/v.go"}

	cases := []struct {
		name, gate, feature string
		paths               []string
		branch              string
		want                []string
	}{
		{name: "files outside touches of PRD and Tech Spec", gate: "pre-merge", feature: "billing", want: []string{"internal/b/z.go", "go.mod"}},
		{name: "--paths extends the intent", gate: "pre-merge", feature: "billing", paths: []string{"go.mod"}, want: []string{"internal/b/z.go"}},
		{name: "feature resolved from frontmatter in diff", gate: "pre-merge", want: []string{"internal/b/z.go", "go.mod"}},
		{name: "no declared intent", gate: "pre-merge", feature: "quiet"},
		{name: "unresolved feature", gate: "pre-merge", feature: "", branch: "main"},
		{name: "only pre-merge", gate: "pre-implementation", feature: "billing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := diff
			if tc.feature == "" && tc.branch != "" {
				changed = []string{"cmd/x.go", "internal/b/z.go"} // no feature artifact in the diff
			}
			ctx := &Context{RepoRoot: root, Config: cfg, Gate: tc.gate, Feature: tc.feature, Paths: tc.paths, Git: fakeGit{changed: changed, branch: tc.branch}}
			findings, err := checkSDD024(ctx, catalog.Rule{ID: "SDD-024"})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range findings {
				got = append(got, f.Path)
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
