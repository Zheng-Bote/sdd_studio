package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func errText(r map[string]any) string {
	return r["content"].([]any)[0].(map[string]any)["text"].(string)
}

func shaOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func readFileT(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

const searchPRD = "---\nfeature: search\nstatus: draft\narchive-after:\ntouches:\n  - api/search/**\n---\n\n# PRD — Suche\n\n## Problem\n\nNutzer finden nichts.\n\n## Nicht-Ziele\n\n- Volltext\n\n## Akzeptanzkriterien\n\n- **AC-1** Die Suche findet Titel.\n"

// desk-write-content AC-1: sdd_new writes the checked draft instead of the template, or nothing with the reason.
func TestNewWithContent(t *testing.T) {
	root := project(t)
	s := newServer(root, true)

	prev := structured(t, call(t, s, "sdd_new", map[string]any{"types": []string{"prd"}, "feature": "search", "content": searchPRD}))
	if op := prev["ops"].([]any)[0].(map[string]any); op["content"] != searchPRD || op["path"] != "docs/prd/search.md" {
		t.Fatalf("preview: %v", prev)
	}
	res := structured(t, call(t, s, "sdd_new", map[string]any{"types": []string{"prd"}, "feature": "search", "content": searchPRD, "apply": true}))
	first := res["results"].([]any)[0].(map[string]any)
	if got := readFileT(t, root, "docs/prd/search.md"); got != searchPRD {
		t.Fatalf("written %q", got)
	}
	if first["status"] != "created" || first["sha256"] != shaOf(searchPRD) || first["lines"].(float64) != float64(strings.Count(searchPRD, "\n")) {
		t.Errorf("result %v", first)
	}

	refused := []struct {
		name string
		args map[string]any
		want string
	}{
		{"two types", map[string]any{"feature": "other", "content": searchPRD}, "genau einem Typ"},
		{"wrong feature", map[string]any{"types": []string{"prd"}, "feature": "other", "content": searchPRD}, `erwartet "other"`},
		{"past archive-after", map[string]any{"types": []string{"prd"}, "feature": "search2",
			"content": strings.Replace(strings.ReplaceAll(searchPRD, "search", "search2"), "archive-after:", "archive-after: 2025-12-31", 1)}, "vor dem heutigen Datum"},
		{"rule error", map[string]any{"types": []string{"prd"}, "feature": "search3",
			"content": strings.Replace(strings.ReplaceAll(searchPRD, "search", "search3"), "- **AC-1** Die Suche findet Titel.", "- Die Suche findet Titel.", 1)}, "SDD-021"},
		{"accepted ADR", map[string]any{"types": []string{"adr"}, "title": "Entscheid",
			"content": "---\nstatus: accepted\ndate: 2026-10-20\nsupersedes: []\nreferences: []\n---\n\n# ADR 0002 — Entscheid\n"}, "entscheidet der Mensch"},
	}
	for _, tc := range refused {
		tc.args["apply"] = true
		r := call(t, s, "sdd_new", tc.args)
		if r["isError"] != true || !strings.Contains(errText(r), tc.want) {
			t.Errorf("%s: %v", tc.name, r)
		}
	}
	for _, rel := range []string{"docs/prd/other.md", "docs/prd/search2.md", "docs/prd/search3.md", "docs/adr/0002-entscheid.md"} {
		if exists(root, rel) {
			t.Errorf("%s written despite the error", rel)
		}
	}
}

// desk-write-content AC-2: only artifacts, sdd.config.yaml and test files are writable; nothing else, no links.
func TestWritePathClasses(t *testing.T) {
	root := project(t)
	s := newServer(root, true)
	cfgText := readFileT(t, root, "sdd.config.yaml")
	allowed := map[string]map[string]any{
		"docs/prd/neu.md":     {"content": strings.ReplaceAll(searchPRD, "search", "neu")},
		"docs/notes/notiz.md": {"content": "---\nstatus: active\n---\n\n# Notiz\n"},
		"api/search_test.go":  {"content": "// search AC-1: Titel\n"},
		"sdd.config.yaml":     {"content": cfgText + "\n", "base_sha256": shaOf(cfgText)},
	}
	for rel, a := range allowed {
		a["path"] = rel
		if r := call(t, s, "sdd_write", a); r["isError"] == true {
			t.Errorf("%s refused: %s", rel, errText(r))
		}
	}

	outside := t.TempDir()
	write(t, outside, "secret.md", "geheim")
	write(t, outside, "secret_test.go", "geheim")
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(root, "docs/prd/link.md")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "docs/design/out")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "api/out")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"README.md", "src/main.go", "api/export.go", ".git/config", ".sdd/baseline.json", "../x.md", "/etc/passwd",
		"docs/prd/notes.txt", "node_modules/x/a_test.go", "docs/prd/link.md", "docs/design/out/secret.md",
		".sdd/evil_test.go", ".git/hooks/evil_test.go", "api/out/secret_test.go"} {
		// Neither a preview (it would show the old content in its diff), nor a write, nor a read.
		for _, apply := range []bool{false, true} {
			r := call(t, s, "sdd_write", map[string]any{"path": rel, "content": "x\n", "base_sha256": shaOf("geheim"), "apply": apply})
			if r["isError"] != true {
				t.Errorf("%s writable (apply %v): %v", rel, apply, r)
			}
		}
		if r := readTool(t, s, rel); r["isError"] != true {
			t.Errorf("%s readable: %v", rel, r)
		}
	}
	if got := readFileT(t, outside, "secret.md"); got != "geheim" {
		t.Errorf("file outside the repository changed: %q", got)
	}
}

// desk-write-content AC-3: replacing needs the current sha256; artifacts and configuration are checked before writing.
func TestWriteReplace(t *testing.T) {
	root := project(t)
	s := newServer(root, true)
	const rel = "docs/prd/billing.md"
	if err := os.Chmod(filepath.Join(root, rel), 0o600); err != nil {
		t.Fatal(err)
	}
	old := readFileT(t, root, rel)
	updated := strings.Replace(old, "- AC-2 Import", "- AC-2 Import aus CSV", 1)

	for name, base := range map[string]any{"missing": nil, "stale": shaOf("anders")} {
		a := map[string]any{"path": rel, "content": updated, "apply": true}
		if base != nil {
			a["base_sha256"] = base
		}
		if r := call(t, s, "sdd_write", a); r["isError"] != true || !strings.Contains(errText(r), shaOf(old)) {
			t.Errorf("%s base: %v", name, r)
		}
	}
	broken := strings.Replace(old, "- AC-2 Import", "- Import", 1)
	if r := call(t, s, "sdd_write", map[string]any{"path": rel, "content": broken, "base_sha256": shaOf(old), "apply": true}); r["isError"] != true || !strings.Contains(errText(r), "SDD-021") {
		t.Errorf("rule error must block: %v", r)
	}
	if got := readFileT(t, root, rel); got != old {
		t.Fatalf("file changed by a refused write: %q", got)
	}

	read := structured(t, readTool(t, s, rel))
	if r := call(t, s, "sdd_write", map[string]any{"path": rel, "content": updated, "base_sha256": read["sha256"], "apply": true}); r["isError"] == true {
		t.Fatalf("replace: %s", errText(r))
	}
	info, _ := os.Stat(filepath.Join(root, rel))
	if got := readFileT(t, root, rel); got != updated || info.Mode().Perm() != 0o600 {
		t.Errorf("content %q, mode %v", got, info.Mode().Perm())
	}

	cfgText := readFileT(t, root, "sdd.config.yaml")
	bad := strings.Replace(cfgText, "profile: standard", "profile: bogus", 1)
	if r := call(t, s, "sdd_write", map[string]any{"path": "sdd.config.yaml", "content": bad, "base_sha256": shaOf(cfgText), "apply": true}); r["isError"] != true || !strings.Contains(errText(r), "bogus") {
		t.Errorf("invalid configuration must be refused: %v", r)
	}
	if readFileT(t, root, "sdd.config.yaml") != cfgText {
		t.Errorf("configuration changed")
	}
}

// desk-write-content AC-4: sdd_write never decides an ADR, neither in a new file nor by changing the status.
func TestWriteNeverAcceptsADR(t *testing.T) {
	root := project(t)
	s := newServer(root, true)
	proposed := "---\nstatus: proposed\ndate: 2026-10-20\nsupersedes: []\nreferences: []\n---\n\n# ADR 0002 — Neu\n"
	accepted := strings.Replace(proposed, "status: proposed", "status: accepted", 1)

	if r := call(t, s, "sdd_write", map[string]any{"path": "docs/adr/0002-neu.md", "content": accepted, "apply": true}); r["isError"] != true || !strings.Contains(errText(r), "ADR 0003") {
		t.Errorf("new accepted ADR: %v", r)
	}
	if r := call(t, s, "sdd_write", map[string]any{"path": "docs/adr/0002-neu.md", "content": proposed, "apply": true}); r["isError"] == true {
		t.Fatalf("proposed ADR: %s", errText(r))
	}
	if r := call(t, s, "sdd_write", map[string]any{"path": "docs/adr/0002-neu.md", "content": accepted, "base_sha256": shaOf(proposed), "apply": true}); r["isError"] != true {
		t.Errorf("proposed -> accepted: %v", r)
	}
	if got := readFileT(t, root, "docs/adr/0002-neu.md"); got != proposed {
		t.Errorf("ADR changed: %q", got)
	}
}

// desk-write-content AC-5: the preview shows action and diff or content; the result names what was really written.
func TestWritePreviewAndResult(t *testing.T) {
	root := project(t)
	const rel = "api/export_test.go"
	old := readFileT(t, root, rel)
	updated := old + "func TestImport(t *testing.T) {}\n"

	ro := newServer(root, false)
	prev := structured(t, call(t, ro, "sdd_write", map[string]any{"path": rel, "content": updated, "base_sha256": shaOf(old)}))
	op := prev["ops"].([]any)[0].(map[string]any)
	if prev["applied"] != false || op["action"] != "update" || !strings.Contains(op["diff"].(string), "+func TestImport(t *testing.T) {}") {
		t.Errorf("update preview: %v", prev)
	}
	same := structured(t, call(t, ro, "sdd_write", map[string]any{"path": rel, "content": old, "base_sha256": shaOf(old)}))
	if a := same["ops"].([]any)[0].(map[string]any)["action"]; a != "unchanged" {
		t.Errorf("unchanged preview: %v", same)
	}
	created := structured(t, call(t, ro, "sdd_write", map[string]any{"path": "api/new_test.go", "content": "// neu\n"}))
	if op := created["ops"].([]any)[0].(map[string]any); op["action"] != "create" || op["content"] != "// neu\n" {
		t.Errorf("create preview: %v", created)
	}
	if r := call(t, ro, "sdd_write", map[string]any{"path": rel, "content": updated, "base_sha256": shaOf(old), "apply": true}); r["isError"] != true || readFileT(t, root, rel) != old {
		t.Errorf("apply without --allow-write: %v", r)
	}

	res := structured(t, call(t, newServer(root, true), "sdd_write", map[string]any{"path": rel, "content": updated, "base_sha256": shaOf(old), "apply": true}))
	if res["applied"] != true || res["status"] != "replaced" || res["sha256"] != shaOf(updated) || res["lines"].(float64) != 3 ||
		!strings.Contains(res["diff"].(string), "+func TestImport") {
		t.Errorf("result %v", res)
	}
}

// desk-write-content AC-6: sdd_read reads sdd.config.yaml and test files, always with the sha256 of the whole file.
func TestReadConfigAndTests(t *testing.T) {
	root := project(t)
	s := newServer(root, false)
	for _, rel := range []string{"sdd.config.yaml", "api/export_test.go", "docs/prd/billing.md"} {
		r := structured(t, readTool(t, s, rel))
		if r["path"] != rel || r["sha256"] != shaOf(readFileT(t, root, rel)) {
			t.Errorf("%s: %v", rel, r)
		}
	}
	if list := structured(t, call(t, s, "sdd_read", map[string]any{})); list["config"] != "sdd.config.yaml" {
		t.Errorf("list: %v", list)
	}
}

// desk-write-content AC-8: prompts and instructions demand content, sdd_read/sdd_write and success only from results.
func TestPromptsDemandContent(t *testing.T) {
	s := newServer(project(t), false)
	init := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	instr, _ := init[0]["result"].(map[string]any)["instructions"].(string)
	for _, want := range []string{"sdd_new samt content", "sdd_write", "Werkzeugergebnis"} {
		if !strings.Contains(instr, want) {
			t.Errorf("instructions miss %q: %s", want, instr)
		}
	}
	for _, name := range []string{"sdd-idea", "sdd-techspec", "sdd-adr", "sdd-plan"} {
		res := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"`+name+`","arguments":{"feature":"billing","title":"Entscheid"}}}`)
		text := fmt.Sprint(res[0]["result"])
		for _, want := range []string{"als content, nie das leere Template", "mit sdd_write, mit base_sha256", "Melde nur, was ein Werkzeugergebnis belegt"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
		if strings.Contains(text, "mit deinen eigenen Werkzeugen") {
			t.Errorf("%s still points to the agent's own tools", name)
		}
	}
	for _, name := range []string{"sdd-idea", "sdd-techspec", "sdd-adr", "sdd-plan"} {
		step := fmt.Sprint(exchange(t, s, `{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"`+name+`","arguments":{"feature":"billing","title":"Entscheid"}}}`)[0]["result"])
		if !strings.Contains(step, "content: der geprüfte Entwurf") || strings.Contains(step, "dann ausfüllen") {
			t.Errorf("%s must store the checked draft, not fill a template afterwards", name)
		}
	}
}
