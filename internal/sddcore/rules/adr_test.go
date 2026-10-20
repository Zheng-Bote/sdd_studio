package rules

import (
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

// sdd-rules-v1-1 AC-5: a constitution change beyond formatting needs an ADR in the same diff that names it.
func TestSDD010(t *testing.T) {
	const base = "# V\n\n## Security\n1. Regel A\n"
	cases := []struct {
		name    string
		gate    string
		current string
		diff    []string
		adr     string // content of docs/adr/0002-x.md, "" = absent
		atBase  bool
		want    bool
	}{
		{name: "unchanged constitution", gate: "pre-merge", current: base, diff: []string{"cmd/x.go"}, atBase: true},
		{name: "formatting only", gate: "pre-merge", current: "# V\r\n\r\n\r\n##   Security\n1. Regel   A  \n", diff: []string{"constitution.md"}, atBase: true},
		{name: "rule changed without ADR", gate: "pre-merge", current: base + "2. Regel B\n", diff: []string{"constitution.md"}, atBase: true, want: true},
		{name: "ADR not in diff", gate: "pre-merge", current: base + "2. Regel B\n", diff: []string{"constitution.md"}, atBase: true,
			adr: "---\nreferences: [constitution.md]\n---\n", want: true},
		{name: "ADR in diff names it", gate: "pre-implementation", current: base + "2. Regel B\n", diff: []string{"constitution.md", "docs/adr/0002-x.md"}, atBase: true,
			adr: "---\nstatus: accepted\nreferences:\n  - constitution.md\n---\n"},
		{name: "ADR names it relatively via adr field", gate: "pre-merge", current: base + "2. Regel B\n", diff: []string{"constitution.md", "docs/adr/0002-x.md"}, atBase: true,
			adr: "---\nadr: ../../constitution.md\n---\n"},
		{name: "ADR in diff names something else", gate: "pre-merge", current: base + "2. Regel B\n", diff: []string{"constitution.md", "docs/adr/0002-x.md"}, atBase: true,
			adr: "---\nreferences: [README.md]\n---\n", want: true},
		{name: "first creation needs no ADR", gate: "pre-merge", current: base, diff: []string{"constitution.md"}},
		{name: "not evaluated in lint", gate: "", current: base + "2. Regel B\n", diff: []string{"constitution.md"}, atBase: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "constitution.md", tc.current)
			if tc.adr != "" {
				writeFile(t, root, "docs/adr/0002-x.md", tc.adr)
			}
			files := map[string]string{}
			if tc.atBase {
				files["main:constitution.md"] = base
			}
			cfg := featureConfig()
			cfg.Artifacts.Constitution = "constitution.md"
			cfg.Artifacts.ADR = "docs/adr"
			ctx := &Context{RepoRoot: root, Config: cfg, Gate: tc.gate, Git: fakeGit{changed: tc.diff, files: files}}
			findings, err := checkSDD010(ctx, catalog.Rule{ID: "SDD-010"})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(findings) == 1; got != tc.want || len(findings) > 1 {
				t.Fatalf("want finding=%v, got %+v", tc.want, findings)
			}
			if tc.want && !strings.Contains(findings[0].Message, "kein ADR im selben Diff") {
				t.Errorf("unexpected message %q", findings[0].Message)
			}
		})
	}
}
