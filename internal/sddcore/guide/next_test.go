package guide_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	// A threshold so that the Tech Spec step depends on touches.
	cfg := filepath.Join(root, "sdd.config.yaml")
	data, _ := os.ReadFile(cfg)
	data = []byte(string(data) + "\n")
	data = []byte(replace(string(data), "    touched_paths: []", "    touched_paths: [\"api/**\"]"))
	if err := os.WriteFile(cfg, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func replace(s, old, new string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	panic("pattern not found: " + old)
}

func nextOf(t *testing.T, root, feature string) (string, map[string]string) {
	t.Helper()
	g, err := guide.Next(root, feature)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Features) != 1 || g.Features[0].Next == nil {
		t.Fatalf("unexpected guide %+v", g)
	}
	states := map[string]string{}
	for _, s := range g.Features[0].Steps {
		states[s.ID] = s.State
	}
	return g.Features[0].Next.ID, states
}

// sdd-guidance AC-1, AC-2: the steps follow the process and name exactly one next step derived from the rules.
func TestNextWalksTheProcess(t *testing.T) {
	root := project(t)
	prd := func(frontmatter, body string) {
		write(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: active\n"+frontmatter+"---\n"+body)
	}
	steps := []struct {
		name   string
		setup  func()
		want   string
		states map[string]string
	}{
		{"nothing yet", func() {}, "prd", map[string]string{"prd": guide.StateOpen, "tests": guide.StateLater}},
		{"PRD without non-goals", func() { prd("", "## Ziel\nx\n") }, "non-goals", nil},
		{"PRD without AC IDs", func() { prd("", "## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- ohne Kennung\n") }, "acceptance", nil},
		{"no declared intent", func() { prd("", "## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- AC-1 x\n") }, "intent", nil},
		{"above threshold without Tech Spec", func() {
			prd("touches: [api/**]\n", "## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- AC-1 x\n")
		}, "design", nil},
		{"Tech Spec without AC reference", func() { write(t, root, "docs/design/billing.md", "---\nfeature: billing\n---\n# Spec\n") }, "design-acs", nil},
		{"no test files at all", func() { write(t, root, "docs/design/billing.md", "---\nfeature: billing\n---\n| AC-1 | x |\n") }, "tests",
			map[string]string{"design": guide.StateDone, "adr": guide.StateNotNeeded, "plan": guide.StateNotNeeded}},
		{"plan with a task without AC", func() {
			write(t, root, "pkg/a_test.go", "// AC-1\nfunc TestA(t *testing.T) {}\n")
			write(t, root, "docs/plans/billing.md", "---\nfeature: billing\n---\n- [ ] ohne Kennung\n")
		}, "plan", map[string]string{"tests": guide.StateDone}},
		{"everything done: finish", func() { write(t, root, "docs/plans/billing.md", "---\nfeature: billing\n---\n- [ ] AC-1 x\n") }, "archive",
			map[string]string{"plan": guide.StateDone, "archive": guide.StateLater}},
		{"implemented: archive before release", func() {
			write(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: implemented\ntouches: [api/**]\n---\n## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- AC-1 x\n")
		}, "archive", map[string]string{"archive": guide.StateOpen}},
	}
	for _, st := range steps {
		st.setup()
		next, states := nextOf(t, root, "billing")
		if next != st.want {
			t.Fatalf("%s: next = %s, want %s (states %v)", st.name, next, st.want, states)
		}
		for id, want := range st.states {
			if states[id] != want {
				t.Errorf("%s: step %s = %s, want %s", st.name, id, states[id], want)
			}
		}
	}
}

// sdd-guidance AC-1: without --feature every active feature is listed; without any feature the entry point is shown.
func TestNextWithoutFeature(t *testing.T) {
	root := project(t)
	g, err := guide.Next(root, "")
	if err != nil || g.Start == nil || g.Start.Command != "sdd new --feature <slug> --type prd" || len(g.Features) != 0 {
		t.Fatalf("empty project: %+v, %v", g, err)
	}
	write(t, root, "docs/prd/a.md", "---\nfeature: a\n---\n")
	write(t, root, "docs/design/b.md", "---\nfeature: b\n---\n")
	write(t, root, "docs/prd/old.md", "---\nfeature: old\nstatus: archived\n---\n")
	g, err = guide.Next(root, "")
	if err != nil || g.Start != nil || len(g.Features) != 2 || g.Features[0].Feature != "a" || g.Features[1].Feature != "b" {
		t.Fatalf("two active features expected: %+v, %v", g, err)
	}
	if g.Features[1].Next.ID != "prd" {
		t.Errorf("feature b has only a Tech Spec; next should be the PRD, got %s", g.Features[1].Next.ID)
	}
	if _, err := guide.Next(root, "Bad Slug"); err == nil {
		t.Errorf("invalid slug must be rejected")
	}
}
