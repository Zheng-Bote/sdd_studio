package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adrlock"
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

// sdd-rules-v1-1 AC-6: decided ADRs are locked; a changed body is an error, an unlocked ADR a warn,
// frontmatter and proposed ADRs stay free; only Update with accept acknowledges a change.
func TestSDD011(t *testing.T) {
	root := t.TempDir()
	cfg := featureConfig()
	cfg.Artifacts.ADR = "docs/adr"
	writeFile(t, root, "docs/adr/0001-a.md", "---\nstatus: proposed\n---\n# A\nEntscheidung A\n")
	writeFile(t, root, "docs/adr/0002-b.md", "---\nstatus: accepted\n---\n# B\nEntscheidung B\n")
	writeFile(t, root, "docs/adr/notes.md", "kein ADR-Dateiname\n")
	check := func() map[string]string {
		t.Helper()
		findings, err := checkSDD011(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-011"})
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]string{}
		for _, f := range findings {
			sev := f.Severity
			if sev == "" {
				sev = "error"
			}
			out[f.Path] = sev
		}
		return out
	}
	lockNow := func(accept bool) (int, int) {
		t.Helper()
		lock, _, err := adrlock.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		adrs, err := adrlock.Collect(root, cfg)
		if err != nil {
			t.Fatal(err)
		}
		added, updated := adrlock.Update(lock, adrs, accept)
		data, _ := lock.Marshal()
		writeFile(t, root, adrlock.RelPath, string(data))
		return added, updated
	}

	if got := check(); len(got) != 1 || got["docs/adr/0002-b.md"] != "warn" {
		t.Fatalf("before locking: %v", got)
	}
	if added, _ := lockNow(false); added != 1 {
		t.Fatalf("expected one ADR to be locked, got %d", added)
	}
	if got := check(); len(got) != 0 {
		t.Fatalf("after locking: %v", got)
	}

	// Status change and reformatting keep the hash; the proposed ADR may change freely and gets locked once decided.
	writeFile(t, root, "docs/adr/0002-b.md", "---\nstatus: superseded\nsuperseded-by: 0003\n---\n\n#  B\r\nEntscheidung   B\n")
	writeFile(t, root, "docs/adr/0001-a.md", "---\nstatus: proposed\n---\n# A\nganz anders\n")
	if got := check(); len(got) != 0 {
		t.Fatalf("frontmatter/format/proposed changes must not count: %v", got)
	}

	// A changed decision text is an error; plain Update does not acknowledge it, accept does.
	writeFile(t, root, "docs/adr/0002-b.md", "---\nstatus: accepted\n---\n# B\nEntscheidung B, nachträglich geändert\n")
	if got := check(); got["docs/adr/0002-b.md"] != "error" {
		t.Fatalf("changed ADR: %v", got)
	}
	if _, updated := lockNow(false); updated != 0 || check()["docs/adr/0002-b.md"] != "error" {
		t.Fatalf("Update without accept must not acknowledge the change")
	}
	if _, updated := lockNow(true); updated != 1 || len(check()) != 0 {
		t.Fatalf("Update with accept must acknowledge the change: %v", check())
	}

	// Deleted ADRs stay in the lockfile.
	if err := os.Remove(filepath.Join(root, "docs/adr/0002-b.md")); err != nil {
		t.Fatal(err)
	}
	lockNow(false)
	if lock, _, _ := adrlock.Load(root); lock.Entries["docs/adr/0002-b.md"] == "" {
		t.Errorf("deleted ADR removed from lockfile")
	}
}
