package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitT(t *testing.T, root string, a ...string) string {
	t.Helper()
	cmd := exec.Command("git", a...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", a, err, out)
	}
	return strings.TrimSpace(string(out))
}

// featureRepo is gitProject on a feature branch with a changed PRD.
func featureRepo(t *testing.T) string {
	t.Helper()
	root := gitProject(t)
	gitT(t, root, "switch", "--quiet", "-c", "feat/x")
	write(t, root, "docs/prd/billing.md", readFileT(t, root, "docs/prd/billing.md")+"\nNeu.\n")
	return root
}

func commitCall(t *testing.T, root string, paths []string, msg string, apply bool) map[string]any {
	t.Helper()
	return call(t, criticalServer(root), "sdd_commit", map[string]any{"paths": paths, "message": msg, "apply": apply})
}

// sdd-commit AC-1: sdd_commit exists only with --allow-critical and is marked destructive.
func TestCommitToolNeedsFlag(t *testing.T) {
	root := project(t)
	for _, n := range toolNames(t, newServer(root, true)) {
		if n == "sdd_commit" {
			t.Fatal("sdd_commit offered without --allow-critical")
		}
	}
	for _, tl := range toolList(t, criticalServer(root)) {
		if tl["name"] == "sdd_commit" && tl["annotations"].(map[string]any)["destructiveHint"] != true {
			t.Errorf("no destructiveHint: %v", tl)
		}
	}
}

// sdd-commit AC-2: only single files of the allowed classes; everything else is refused and nothing is committed.
func TestCommitRefusesPaths(t *testing.T) {
	root := featureRepo(t)
	head := gitT(t, root, "rev-parse", "HEAD")
	write(t, root, "src/main.go", "package main\n")
	write(t, root, "README.md", "x\n")
	outside := t.TempDir()
	write(t, outside, "x_test.go", "geheim\n")
	if err := os.Symlink(outside, filepath.Join(root, "api/out")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	for _, p := range []string{"src/main.go", "README.md", ".git/config", "../x.md", "/etc/passwd", "docs/prd/*.md", ":(glob)**/*_test.go", ":/docs/prd/billing.md", "api/out/x_test.go", "docs/prd/fehlt.md", "docs/design/billing.md"} {
		for _, apply := range []bool{false, true} {
			if r := commitCall(t, root, []string{"docs/prd/billing.md", p}, "m", apply); r["isError"] != true {
				t.Errorf("%s accepted (apply %v): %v", p, apply, r)
			}
		}
	}
	if r := commitCall(t, root, nil, "m", true); r["isError"] != true {
		t.Errorf("empty paths: %v", r)
	}
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "  ", true); r["isError"] != true {
		t.Errorf("empty message: %v", r)
	}
	if got := gitT(t, root, "rev-parse", "HEAD"); got != head {
		t.Error("something was committed")
	}
}

// sdd-commit AC-3: no commit on main, master, base_ref or a detached HEAD.
func TestCommitRefusesBranches(t *testing.T) {
	root := gitProject(t)
	write(t, root, "docs/prd/billing.md", readFileT(t, root, "docs/prd/billing.md")+"\nNeu.\n")
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "m", false); r["isError"] != true || !strings.Contains(errText(r), "Feature-Branch") {
		t.Errorf("main: %v", r)
	}
	gitT(t, root, "switch", "--quiet", "-c", "master")
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "m", true); r["isError"] != true {
		t.Errorf("master: %v", r)
	}
	gitT(t, root, "switch", "--quiet", "--detach")
	t.Setenv("GITHUB_HEAD_REF", "feat/aus-ci") // must not hide the detached HEAD
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "m", true); r["isError"] != true || !strings.Contains(errText(r), "detached") {
		t.Errorf("detached: %v", r)
	}
	gitT(t, root, "switch", "--quiet", "-c", "entwicklung")
	cfg := readFileT(t, root, "sdd.config.yaml")
	write(t, root, "sdd.config.yaml", strings.Replace(cfg, `base_ref: ""`, `base_ref: "origin/entwicklung"`, 1))
	if readFileT(t, root, "sdd.config.yaml") == cfg {
		t.Fatal("test setup: base_ref not set")
	}
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "m", true); r["isError"] != true || !strings.Contains(errText(r), "entwicklung") {
		t.Errorf("base_ref: %v", r)
	}
}

// sdd-commit AC-4: the preview names branch, message, action and diff per file and changes nothing.
func TestCommitPreview(t *testing.T) {
	root := featureRepo(t)
	write(t, root, "docs/notes/neu.md", "---\nstatus: active\n---\n# Neu\n")
	before := gitT(t, root, "status", "--porcelain")
	r := commitCall(t, root, []string{"docs/prd/billing.md", "docs/notes/neu.md"}, "docs: Neu", false)
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	p := structured(t, r)
	if p["applied"] != false || p["critical"] != true || p["branch"] != "feat/x" || p["message"] != "docs: Neu" || p["summary"] != "2 Datei(en) auf feat/x committen" {
		t.Errorf("preview: %v", p)
	}
	ops := p["ops"].([]any)
	if u := ops[0].(map[string]any); u["action"] != "update" || !strings.Contains(u["diff"].(string), "+Neu.") {
		t.Errorf("update: %v", u)
	}
	if c := ops[1].(map[string]any); c["action"] != "create" || !strings.Contains(c["content"].(string), "# Neu") {
		t.Errorf("create: %v", c)
	}
	if got := gitT(t, root, "status", "--porcelain"); got != before {
		t.Errorf("preview changed Git: %q", got)
	}
}

// sdd-commit AC-5: apply commits exactly the named paths, also a deletion and check state; other staged work stays out;
// a path without a change is refused.
func TestCommitApply(t *testing.T) {
	root := featureRepo(t)
	if err := os.Remove(filepath.Join(root, "docs/plans/billing.md")); err != nil {
		t.Fatal(err)
	}
	write(t, root, ".sdd/adr-lock.json", "{}\n")
	write(t, root, "src/fremd.go", "package fremd\n")
	gitT(t, root, "add", "src/fremd.go")
	r := commitCall(t, root, []string{"docs/prd/billing.md", "docs/plans/billing.md", ".sdd/adr-lock.json"}, "docs: Stand", true)
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	res := structured(t, r)
	if res["applied"] != true || res["commit"] != gitT(t, root, "rev-parse", "--short", "HEAD") {
		t.Errorf("result: %v", res)
	}
	files := gitT(t, root, "show", "--name-status", "--format=", "HEAD")
	for _, want := range []string{"M\tdocs/prd/billing.md", "D\tdocs/plans/billing.md", ".sdd/adr-lock.json"} {
		if !strings.Contains(files, want) {
			t.Errorf("commit lacks %q: %s", want, files)
		}
	}
	if strings.Contains(files, "fremd") {
		t.Errorf("staged foreign file committed: %s", files)
	}
	if got := gitT(t, root, "status", "--porcelain"); !strings.Contains(got, "A  src/fremd.go") {
		t.Errorf("foreign file no longer staged: %q", got)
	}
	if r := commitCall(t, root, []string{"docs/prd/billing.md"}, "m", true); r["isError"] != true || !strings.Contains(errText(r), "keine Änderung") {
		t.Errorf("unchanged: %v", r)
	}
}

// sdd-commit AC-7: prompts and server instructions treat sdd_commit as critical.
func TestPromptsCommitTool(t *testing.T) {
	s := criticalServer(project(t))
	init := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	if instr, _ := init[0]["result"].(map[string]any)["instructions"].(string); !strings.Contains(instr, "sdd_commit") {
		t.Error("instructions do not name sdd_commit")
	}
	if text := promptText(t, s, "sdd-idea", `{}`); !strings.Contains(text, "oder sdd_commit nur auf") || !strings.Contains(text, "nie auf dem Standard-Branch") {
		t.Error("common rule does not treat sdd_commit as critical")
	}
}

// sdd-commit AC-8: handbook, README, architecture documentation and CHANGELOG describe the tool.
func TestCommitDocumented(t *testing.T) {
	for _, f := range []string{"../../handbuch/04-agenten.md", "../../README.md", "../../docs/arc42.md", "../../CHANGELOG.md"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "sdd_commit") {
			t.Errorf("%s does not name sdd_commit", f)
		}
	}
}
