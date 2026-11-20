package mcp

import (
	"strings"
	"testing"
)

// bmad-import AC-9: the three macro prompts are served, demand a confirmation before writing, and name their tools.
func TestBMADPrompts(t *testing.T) {
	s := newServer(project(t), true)
	r := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"prompts/list","params":{}}`)
	names := map[string]bool{}
	for _, p := range r[0]["result"].(map[string]any)["prompts"].([]any) {
		names[p.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"sdd-ingest-bmad", "sdd-architecture", "sdd-epic-refine"} {
		if !names[want] {
			t.Errorf("prompt %s missing", want)
		}
	}

	cases := map[string]struct {
		args string
		want []string
	}{
		"sdd-ingest-bmad":  {`{}`, []string{"_bmad-output/", "R<n>", "sdd_validate_draft", "sdd_new", "sdd_adr_accept", "SDD-003"}},
		"sdd-architecture": {`{}`, []string{"sdd_stack", "arc42", "sdd_new", "ADR", "SDD-003"}},
		"sdd-epic-refine":  {`{"feature":"epic-plattform"}`, []string{"refined: false", "sdd_check_intent", "sdd_write", "base_sha256", "Bestätigung"}},
	}
	for name, c := range cases {
		text := promptText(t, s, name, c.args)
		for _, want := range c.want {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
		if !strings.Contains(text, "bestätig") {
			t.Errorf("%s does not demand a confirmation", name)
		}
	}

	// The epic prompt needs its slug and rejects a malformed one.
	if r := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"sdd-epic-refine","arguments":{}}}`); r[0]["error"] == nil {
		t.Error("sdd-epic-refine without feature accepted")
	}
}

// bmad-import AC-9: the BMAD ledger is readable for the take-over and can never be written.
func TestBMADSourceReadOnly(t *testing.T) {
	root := project(t)
	write(t, root, "_bmad-output/initiative-x/initiative-x.md", "---\ntype: initiative\n---\n\n- **R1** eins\n")
	s := newServer(root, true)

	read := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"sdd_read","arguments":{"path":"_bmad-output/initiative-x/initiative-x.md"}}}`)
	res, _ := read[0]["result"].(map[string]any)
	if res == nil || res["isError"] == true {
		t.Fatalf("sdd_read refused the ledger: %v", read[0])
	}
	if text, _ := res["content"].([]any)[0].(map[string]any)["text"].(string); !strings.Contains(text, "R1") {
		t.Errorf("ledger content not returned: %s", text)
	}

	wr := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"sdd_write","arguments":{"path":"_bmad-output/initiative-x/initiative-x.md","content":"x","apply":true}}}`)
	if res, _ := wr[0]["result"].(map[string]any); res == nil || res["isError"] != true {
		t.Fatalf("sdd_write accepted the ledger: %v", wr[0])
	}
}
