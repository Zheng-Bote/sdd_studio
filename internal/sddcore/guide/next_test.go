package guide_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// sdd-guidance AC-2: the ADR step is done only when the feature links an accepted ADR that names the constitution,
// the condition SDD-010 checks in the gate; an unrelated ADR naming the constitution is not enough.
func TestNextADRStep(t *testing.T) {
	root := project(t)
	const body = "## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- AC-1 x\n"
	prd := func(extra string) {
		write(t, root, "docs/prd/rules.md", "---\nfeature: rules\nstatus: active\ntouches: [constitution.md]\n---\n"+body+extra)
	}
	adr := func(rel, status, refs string) {
		write(t, root, rel, "---\nstatus: "+status+"\nreferences: ["+refs+"]\n---\n# ADR\n")
	}
	stepOf := func() guide.Step {
		t.Helper()
		g, err := guide.Next(root, "rules")
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range g.Features[0].Steps {
			if s.ID == "adr" {
				return s
			}
		}
		t.Fatal("no adr step")
		return guide.Step{}
	}
	link := "\nBegründung: [ADR 0002](../adr/0002-neue-regel.md)\n"

	prd("")
	if s := stepOf(); s.State != guide.StateManual || !strings.Contains(s.Command, "--type adr") {
		t.Errorf("no ADR: %+v", s)
	}

	adr("docs/adr/0001-alt.md", "accepted", "constitution.md")
	adr("docs/adr/0002-neue-regel.md", "proposed", "constitution.md")
	adr("docs/adr/0003-anderes.md", "accepted", "README.md")
	if s := stepOf(); s.State != guide.StateManual || !strings.Contains(s.Detail, "0001-alt.md") || strings.Contains(s.Detail, "0003") {
		t.Errorf("unlinked ADRs must only be suggested: %+v", s)
	}

	prd(link)
	if s := stepOf(); s.State != guide.StateManual || s.Title != "ADR entscheiden" || !strings.Contains(s.Detail, "0002-neue-regel.md") {
		t.Errorf("linked but proposed: %+v", s)
	}

	adr("docs/adr/0002-neue-regel.md", "rejected", "constitution.md")
	if s := stepOf(); s.State == guide.StateDone {
		t.Errorf("a rejected ADR does not justify the change: %+v", s)
	}

	adr("docs/adr/0002-neue-regel.md", "accepted", "constitution.md")
	if s := stepOf(); s.State != guide.StateDone || !strings.Contains(s.Detail, "0002-neue-regel.md (accepted)") {
		t.Errorf("linked and accepted: %+v", s)
	}

	adr("docs/adr/0002-neue-regel.md", "accepted", "README.md")
	if s := stepOf(); s.State == guide.StateDone {
		t.Errorf("a linked ADR that does not name the constitution: %+v", s)
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

// constitution-guide AC-2: placeholders in the constitution put "Verfassung ausfüllen" first; code blocks do not
// count; without placeholders the step is gone.
func TestNextConstitutionStep(t *testing.T) {
	root := project(t) // the init template: three placeholder list items
	g, err := guide.Next(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.Constitution == nil || g.Constitution.ID != "constitution" || !strings.Contains(g.Constitution.Detail, "3 Platzhalter") ||
		!strings.Contains(g.Constitution.Detail, "Branch der Übernahme") || !strings.Contains(g.Constitution.Command, "sdd-constitution") {
		t.Fatalf("constitution step: %+v", g.Constitution)
	}
	cpath := filepath.Join(root, "constitution.md")
	data, _ := os.ReadFile(cpath)
	filled := regexpReplace(string(data))
	write(t, root, "constitution.md", filled+"\n```\n1. <Beispiel im Codeblock>\n```\n")
	g, err = guide.Next(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if g.Constitution != nil {
		t.Errorf("filled constitution still reported: %+v", g.Constitution)
	}
}

// regexpReplace fills every placeholder list item of the template with a real rule.
func regexpReplace(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "1. <") {
			line = "1. Eine echte Regel."
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// flow-closing AC-3: on the unmerged adopt branch, "Übernahme abschließen" names commit, diff and merge; after the
// merge (on main, or back on the adopt branch) and without a report it is gone.
func TestNextAdoptionStep(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte("[user]\n\tname = T\n\temail = t@example.com\n[commit]\n\tgpgsign = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"GITHUB_HEAD_REF", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "CI_COMMIT_BRANCH"} {
		t.Setenv(v, "")
	}
	root := project(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	adoption := func() *guide.Step {
		t.Helper()
		g, err := guide.Next(root, "")
		if err != nil {
			t.Fatal(err)
		}
		return g.Adoption
	}
	git("init", "--quiet", "--initial-branch=main")
	git("add", "-A")
	git("commit", "--quiet", "-m", "init")
	if a := adoption(); a != nil {
		t.Fatalf("no report: %+v", a)
	}
	git("switch", "--quiet", "--create", "sdd/adopt")
	write(t, root, ".sdd/adopt-report.md", "# Adopt-Bericht\n\nAngewendet am 2026-11-02 auf Branch `sdd/adopt` (ausgehend von `main`), Profil `standard`.\n")
	git("add", "-A")
	git("commit", "--quiet", "-m", "sdd adopt: Bericht")
	a := adoption()
	if a == nil || a.ID != "adoption" || !strings.Contains(a.Detail, "committen") || !strings.Contains(a.Detail, "in main mergen") || a.Command != "git diff main...sdd/adopt" {
		t.Fatalf("pending adoption: %+v", a)
	}
	git("switch", "--quiet", "--create", "other")
	if a := adoption(); a != nil {
		t.Errorf("on another branch: %+v", a)
	}
	git("switch", "--quiet", "main")
	git("merge", "--quiet", "--ff-only", "sdd/adopt")
	if a := adoption(); a != nil {
		t.Errorf("after the merge, on main: %+v", a)
	}
	git("switch", "--quiet", "sdd/adopt")
	if a := adoption(); a != nil {
		t.Errorf("after the merge, back on the adopt branch: %+v", a)
	}
}

// The acceptance step counts distinct AC IDs: an AC named again in open points is not another criterion (found in the
// live test on 2026-11-05: "5 Kennungen" for AC-1 to AC-4).
func TestNextCountsDistinctACs(t *testing.T) {
	root := project(t)
	write(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: active\n---\n## Nicht-Ziele\n- nichts\n"+
		"## Akzeptanzkriterien\n- **AC-1** x\n- **AC-2** y\n## Offene Punkte\n- Wohin meldet AC-2? Siehe auch AC-1.\n")
	g, err := guide.Next(root, "billing")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range g.Features[0].Steps {
		if s.ID == "acceptance" && s.Detail != "2 Kennungen" {
			t.Errorf("acceptance detail = %q, want 2 Kennungen", s.Detail)
		}
	}
}

// arc42-docs AC-4: the architecture step opens when a feature was archived with a release and docs/arc42.md is
// missing or names an older release; it disappears when the release matches or nothing was archived with a release.
func TestNextArchitectureStep(t *testing.T) {
	root := project(t)
	step := func() *guide.Step {
		t.Helper()
		g, err := guide.Next(root, "")
		if err != nil {
			t.Fatal(err)
		}
		return g.Architecture
	}
	if s := step(); s != nil {
		t.Fatalf("no archived release: %+v", s)
	}
	write(t, root, "docs/archive/prd/a.md", "---\nfeature: a\nstatus: archived\nrelease: v1.9.0\n---\n# A\n")
	write(t, root, "docs/archive/prd/b.md", "---\nfeature: b\nstatus: archived\nrelease: v1.10.0\n---\n# B\n")
	if s := step(); s == nil || s.Title != "Architekturdoku anlegen" || !strings.Contains(s.Detail, "v1.10.0") {
		t.Fatalf("missing doc: %+v", s)
	}
	write(t, root, "docs/arc42.md", "---\nstatus: active\nrelease: v1.9.0\n---\n# Architektur\n")
	if s := step(); s == nil || s.Title != "Architekturdoku nachführen" || !strings.Contains(s.Detail, "steht auf v1.9.0") {
		t.Fatalf("older release: %+v", s)
	}
	write(t, root, "docs/arc42.md", "---\nstatus: active\nrelease: v1.10.0\n---\n# Architektur\n")
	if s := step(); s != nil {
		t.Fatalf("current doc: %+v", s)
	}
}
