package mcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitProject is project(t) as a Git repository with one commit, isolated from the user's Git configuration.
func gitProject(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte("[user]\n\tname = T\n\temail = t@example.com\n[commit]\n\tgpgsign = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"GITHUB_BASE_REF", "CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "GITHUB_HEAD_REF", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "CI_COMMIT_BRANCH"} {
		t.Setenv(v, "")
	}
	root := project(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
	for _, args := range [][]string{{"init", "--quiet", "--initial-branch=main"}, {"add", "-A"}, {"commit", "--quiet", "-m", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func criticalServer(root string) *Server {
	s := newServer(root, true)
	s.AllowCritical = true
	return s
}

func toolNames(t *testing.T, s *Server) []string {
	t.Helper()
	res := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var names []string
	for _, tl := range res[0]["result"].(map[string]any)["tools"].([]any) {
		names = append(names, tl.(map[string]any)["name"].(string))
	}
	return names
}

// desk-critical-tools AC-1: the critical tools exist only with --allow-critical and carry destructiveHint.
func TestCriticalToolsNeedFlag(t *testing.T) {
	root := project(t)
	plain := toolNames(t, newServer(root, true))
	for _, n := range plain {
		if n == "sdd_baseline" || n == "sdd_adr_accept" || n == "sdd_adopt_apply" || n == "sdd_config" {
			t.Errorf("%s offered without --allow-critical", n)
		}
	}
	for _, n := range []string{"sdd_baseline", "sdd_adr_accept", "sdd_adopt_apply", "sdd_config"} {
		res := exchange(t, newServer(root, true), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"`+n+`","arguments":{}}}`)
		if res[0]["error"] == nil {
			t.Errorf("%s callable without --allow-critical: %v", n, res)
		}
	}
	all := exchange(t, criticalServer(root), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)[0]["result"].(map[string]any)["tools"].([]any)
	if len(all) != len(plain)+4 {
		t.Fatalf("with --allow-critical: %d tools, want %d", len(all), len(plain)+4)
	}
	for _, tl := range all[len(plain):] {
		if tl.(map[string]any)["annotations"].(map[string]any)["destructiveHint"] != true {
			t.Errorf("%v lacks destructiveHint", tl)
		}
	}
}

// desk-critical-tools AC-2: sdd_baseline previews additions, removals and ADR locks, writes like sdd baseline and
// refuses rules with baseline: forbidden.
func TestBaselineTool(t *testing.T) {
	root := gitProject(t) // billing AC-2 has no test: one SDD-022 error; ADR 0001 accepted, not locked
	s := criticalServer(root)
	prev := structured(t, call(t, s, "sdd_baseline", map[string]any{}))
	if prev["applied"] != false || prev["critical"] != true || len(prev["added"].([]any)) == 0 ||
		!strings.Contains(prev["summary"].(string), "SDD-022") || len(prev["locked"].([]any)) != 1 {
		t.Fatalf("preview: %v", prev)
	}
	if _, err := os.Stat(filepath.Join(root, ".sdd/adr-lock.json")); err == nil {
		t.Fatalf("preview wrote the lockfile")
	}
	before := readFileT(t, root, ".sdd/baseline.json")
	res := structured(t, call(t, s, "sdd_baseline", map[string]any{"apply": true}))
	if res["applied"] != true || readFileT(t, root, ".sdd/baseline.json") == before || !strings.Contains(readFileT(t, root, ".sdd/adr-lock.json"), "docs/adr/0001-api.md") {
		t.Errorf("apply: %v", res)
	}
	if r := call(t, s, "sdd_baseline", map[string]any{"accept": "SDD-050", "apply": true}); r["isError"] != true || !strings.Contains(errText(r), "baseline: forbidden") {
		t.Errorf("forbidden rule: %v", r)
	}
}

// desk-critical-tools AC-3: sdd_adr_accept decides only a proposed ADR and changes nothing but the status.
func TestADRAcceptTool(t *testing.T) {
	root := project(t)
	const rel = "docs/adr/0002-neu.md"
	proposed := "---\nstatus: proposed\ndate: 2026-10-20\nsupersedes: []\nreferences: []\n---\n\n# ADR 0002 — Neu\n"
	write(t, root, rel, proposed)
	s := criticalServer(root)
	prev := structured(t, call(t, s, "sdd_adr_accept", map[string]any{"path": rel}))
	if op := prev["ops"].([]any)[0].(map[string]any); prev["critical"] != true || !strings.Contains(op["diff"].(string), "-status: proposed\n+status: accepted") {
		t.Fatalf("preview: %v", prev)
	}
	if r := call(t, s, "sdd_adr_accept", map[string]any{"path": rel, "apply": true}); r["isError"] == true {
		t.Fatalf("apply: %s", errText(r))
	}
	if got := readFileT(t, root, rel); got != strings.Replace(proposed, "status: proposed", "status: accepted", 1) {
		t.Errorf("content %q", got)
	}
	for _, p := range []string{rel, "docs/adr/0001-api.md", "docs/prd/billing.md", "../x.md"} { // now accepted, already accepted, no ADR, outside
		if r := call(t, s, "sdd_adr_accept", map[string]any{"path": p, "apply": true}); r["isError"] != true {
			t.Errorf("%s accepted: %v", p, r)
		}
	}
}

// desk-critical-tools AC-4: sdd_adopt_apply shows the adopt report, then applies it on a new branch.
func TestAdoptApplyTool(t *testing.T) {
	root := gitProject(t)
	s := criticalServer(root)
	prev := structured(t, call(t, s, "sdd_adopt_apply", map[string]any{"branch": "sdd/adopt"}))
	if prev["applied"] != false || prev["critical"] != true || !strings.Contains(prev["summary"].(string), "sdd/adopt") || prev["report"].(string) == "" {
		t.Fatalf("preview: %v", prev)
	}
	r := call(t, s, "sdd_adopt_apply", map[string]any{"branch": "sdd/adopt", "apply": true})
	if r["isError"] == true {
		t.Fatalf("apply: %s", errText(r))
	}
	res := structured(t, r)
	if res["branch"] != "sdd/adopt" || res["from_branch"] != "main" {
		t.Errorf("result: %v", res)
	}
}

// desk-critical-tools AC-7: prompts and instructions allow critical tools only on the human's explicit request, each
// confirmed singly, and no retry after a refusal.
func TestPromptsCriticalRule(t *testing.T) {
	s := criticalServer(project(t))
	init := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	instr, _ := init[0]["result"].(map[string]any)["instructions"].(string)
	for _, want := range []string{"Der Mensch entscheidet sie", "ausdrücklichen Wunsch", "einzeln", "nicht erneut"} {
		if !strings.Contains(instr, want) {
			t.Errorf("instructions miss %q", want)
		}
	}
	for _, name := range []string{"sdd-idea", "sdd-techspec", "sdd-adr", "sdd-plan", "sdd-review"} {
		text := fmt.Sprint(exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"`+name+`","arguments":{"feature":"billing","title":"X"}}}`)[0]["result"])
		for _, want := range []string{"ausdrücklich verlangt", "bestätigt er einzeln", "frage im selben Zug nicht erneut"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}
}

// desk-guided-flow AC-1: sdd_stack reports the languages and the configured tests.include.
func TestStackTool(t *testing.T) {
	root := project(t)
	write(t, root, "go.mod", "module x\n")
	info := structured(t, call(t, newServer(root, false), "sdd_stack", map[string]any{}))
	langs := info["languages"].([]any)
	if len(langs) == 0 || langs[0].(map[string]any)["name"] != "Go" || info["configured"] != true {
		t.Errorf("sdd_stack: %v", info)
	}
}

func promptText(t *testing.T, s *Server, name, args string) string {
	t.Helper()
	r := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"`+name+`","arguments":`+args+`}}`)
	return r[0]["result"].(map[string]any)["messages"].([]any)[0].(map[string]any)["content"].(map[string]any)["text"].(string)
}

// desk-guided-flow AC-2: sdd-idea settles the technology after scope and before criteria and touches.
func TestIdeaAsksTechnologyBeforeTouches(t *testing.T) {
	text := promptText(t, newServer(project(t), false), "sdd-idea", `{}`)
	scope, tech := strings.Index(text, "Nicht-Zielen"), strings.Index(text, "sdd_stack")
	criteria, touches := strings.Index(text, "Akzeptanzkriterien, die ein Test prüfen kann"), strings.Index(text, "(touches)")
	if scope < 0 || tech < 0 || criteria < 0 || touches < 0 || !(scope < tech && tech < criteria && criteria < touches) {
		t.Errorf("order scope %d < technology %d < criteria %d < touches %d violated:\n%s", scope, tech, criteria, touches, text)
	}
	for _, want := range []string{"begründe sie", "als ADR festzuhalten", "nie mit Platzhalter-Endungen"} {
		if !strings.Contains(text, want) {
			t.Errorf("sdd-idea misses %q", want)
		}
	}
}

// desk-guided-flow AC-3: every guided prompt names the process order and that an overdue archive-after before the
// implementation is no reason to archive.
func TestPromptsNameProcessOrder(t *testing.T) {
	s := newServer(project(t), false)
	for _, name := range []string{"sdd-idea", "sdd-techspec", "sdd-adr", "sdd-plan", "sdd-constitution", "sdd-tests", "sdd-conflicts", "sdd-review"} {
		text := promptText(t, s, name, `{"feature":"billing","title":"X"}`)
		for _, want := range []string{"Tests je Kriterium → Umsetzung → Gate pre-merge → Merge → Release → Archiv", "Archiviert wird erst nach dem Release", "nicht archivieren"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}
}

// desk-guided-flow AC-4: sdd-tests walks trace, stack, tests.include, one test per criterion with the slug line, check.
func TestTestsPrompt(t *testing.T) {
	text := promptText(t, newServer(project(t), false), "sdd-tests", `{"feature":"billing"}`)
	last := -1
	for _, step := range []string{"sdd_trace (feature: billing)", "sdd_stack", "tests.include", "<Kommentar> billing AC-<n>:", "mit sdd_write an", "sdd_next"} {
		i := strings.Index(text, step)
		if i <= last {
			t.Errorf("step %q missing or out of order", step)
		}
		last = i
	}
}
