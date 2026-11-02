package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sdd-init-git AC-4: preview with files and the Git step, apply only with --allow-write, then files plus a commit.
// sdd-init-git AC-5: the instructions name sdd_init for a project without configuration.
func TestInitTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte("[user]\n\tname = T\n\temail = t@example.com\n[commit]\n\tgpgsign = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))

	prev := structured(t, call(t, newServer(root, false), "sdd_init", map[string]any{"git": true}))
	if prev["applied"] != false || !strings.Contains(prev["git"].(string), "git init") {
		t.Fatalf("preview: %v", prev)
	}
	found := false
	for _, op := range prev["ops"].([]any) {
		o := op.(map[string]any)
		found = found || (o["path"] == "sdd.config.yaml" && strings.Contains(o["content"].(string), "version: 1"))
	}
	if !found {
		t.Errorf("preview lacks sdd.config.yaml with content: %v", prev["ops"])
	}
	if r := call(t, newServer(root, false), "sdd_init", map[string]any{"git": true, "apply": true}); r["isError"] != true {
		t.Errorf("apply without --allow-write: %v", r)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("written without --allow-write: %v", entries)
	}

	res := structured(t, call(t, newServer(root, true), "sdd_init", map[string]any{"git": true, "no_ci": true, "apply": true}))
	if res["applied"] != true || res["commit"] == nil || res["commit"] == "" || !strings.Contains(res["next"].(string), "thresholds") {
		t.Fatalf("apply: %v", res)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Errorf("no repository: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github/workflows/sdd.yml")); err == nil {
		t.Errorf("no_ci ignored")
	}
	// agent-files AC-4: the agent files are created with init and are part of the first commit.
	cmd := exec.Command("git", "ls-tree", "--name-only", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		if !strings.Contains("\n"+string(out), "\n"+f+"\n") {
			t.Errorf("%s not in the first commit: %s", f, out)
		}
	}
	// The project is usable right away: the tools that failed before init now work.
	if r := call(t, newServer(root, true), "sdd_check_intent", map[string]any{"paths": []string{"src/main.go"}}); r["isError"] == true {
		t.Errorf("sdd_check_intent after init: %v", r)
	}

	init := exchange(t, newServer(root, false), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	if instr, _ := init[0]["result"].(map[string]any)["instructions"].(string); !strings.Contains(instr, "sdd_init") {
		t.Errorf("instructions miss sdd_init: %s", instr)
	}
}
