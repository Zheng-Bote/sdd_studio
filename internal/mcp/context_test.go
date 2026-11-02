package mcp

import (
	"strings"
	"testing"
)

// constitution-guide AC-1: context files are readable with sha256 and listed, but never writable.
func TestReadContextFiles(t *testing.T) {
	root := project(t)
	write(t, root, ".gemini/GEMINI.md", "Make sure to use Go only.\n")
	write(t, root, "README.md", "# Demo\n")
	s := newServer(root, true)
	for _, rel := range []string{".gemini/GEMINI.md", "README.md"} {
		r := structured(t, readTool(t, s, rel))
		if r["path"] != rel || r["sha256"] != shaOf(readFileT(t, root, rel)) || !strings.Contains(r["text"].(string), "   1| ") {
			t.Errorf("%s: %v", rel, r)
		}
		if w := call(t, s, "sdd_write", map[string]any{"path": rel, "content": "x\n", "base_sha256": shaOf(readFileT(t, root, rel)), "apply": true}); w["isError"] != true || !strings.Contains(errText(w), "Kontextdatei") {
			t.Errorf("%s writable: %v", rel, w)
		}
	}
	list := structured(t, call(t, s, "sdd_read", map[string]any{}))
	if got := strings.Join(toStrings(list["context"].([]any)), ","); got != "README.md,.gemini/GEMINI.md" {
		t.Errorf("context list %q", got)
	}
	if r := readTool(t, s, "CLAUDE.md"); r["isError"] != true {
		t.Errorf("missing context file readable: %v", r)
	}
}

func toStrings(v []any) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, x.(string))
	}
	return out
}

// constitution-guide AC-3: sdd-constitution reads sources, proposes rules with sources per section, lets the human
// decide rule by rule, validates, writes and checks SDD-010.
// constitution-guide AC-4: sdd-idea starts with sdd_next and suggests sdd-constitution first.
func TestConstitutionPrompt(t *testing.T) {
	s := newServer(project(t), false)
	text := promptText(t, s, "sdd-constitution", `{}`)
	rest := text
	for _, step := range []string{"auf dem Branch der Übernahme", "Kontextdateien", "sdd_stack", "Fundstelle als Datei:Zeile", "SDD-003", "Regel für Regel", "sdd_validate_draft", "sdd_write", "SDD-010", "sdd-adr"} {
		i := strings.Index(rest, step)
		if i < 0 {
			t.Errorf("sdd-constitution: %q missing or out of order", step)
			continue
		}
		rest = rest[i+len(step):]
	}
	idea := promptText(t, s, "sdd-idea", `{}`)
	if i, j := strings.Index(idea, "0. Rufe sdd_next auf"), strings.Index(idea, "1. Kläre das Problem"); i < 0 || j < i || !strings.Contains(idea, "zuerst den Ablauf sdd-constitution") {
		t.Errorf("sdd-idea must start with sdd_next and the constitution hint")
	}
}

// desk-project-start AC-4: sdd-adopt goes report → confirmed apply → constitution on the adopt branch → commit and
// merge; sdd-start goes init with git → technology → constitution → first feature; the instructions name both.
func TestStartPrompts(t *testing.T) {
	s := newServer(project(t), false)
	inOrder := func(name string, steps ...string) {
		t.Helper()
		rest := promptText(t, s, name, `{}`)
		for _, step := range steps {
			i := strings.Index(rest, step)
			if i < 0 {
				t.Errorf("%s: %q missing or out of order", name, step)
				continue
			}
			rest = rest[i+len(step):]
		}
	}
	inOrder("sdd-adopt", "sdd_stack", "sdd_adopt_apply ohne apply", "Agenten-Dateien", "sdd_adopt_apply mit apply", "ich bestätige einzeln", "sdd-constitution", "vor dem Merge", "committen", "mergen")
	inOrder("sdd-start", "sdd_init mit git: true", "sdd_stack", "Begründung", "ADR", "sdd-constitution", "sdd-idea")
	init := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	if instr, _ := init[0]["result"].(map[string]any)["instructions"].(string); !strings.Contains(instr, "sdd-adopt") || !strings.Contains(instr, "sdd-start") {
		t.Errorf("instructions miss sdd-adopt/sdd-start: %s", instr)
	}
}

// flow-closing AC-1: every guided prompt ends with a summary in words, never with a bare tool call; sdd-constitution
// names commit and, after a take-over, review and merge.
func TestPromptsDemandClosing(t *testing.T) {
	s := newServer(project(t), false)
	for _, name := range []string{"sdd-idea", "sdd-techspec", "sdd-adr", "sdd-plan", "sdd-adopt", "sdd-start", "sdd-constitution", "sdd-tests", "sdd-conflicts", "sdd-review"} {
		text := promptText(t, s, name, `{"feature":"billing","title":"X"}`)
		for _, want := range []string{"in eigenen Worten zusammenfassen", "Pfad", "in Klartext", "Ende nie mit einem bloßen Werkzeugaufruf"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}
	text := promptText(t, s, "sdd-constitution", `{}`)
	for _, want := range []string{"committet werden muss", "Übernahme abschließen", "Branch prüfen und mergen"} {
		if !strings.Contains(text, want) {
			t.Errorf("sdd-constitution misses %q", want)
		}
	}
}
