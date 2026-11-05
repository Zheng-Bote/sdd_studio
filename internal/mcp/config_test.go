package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// configProject is project(t) with a second PRD whose AC has no test, so that lint reports SDD-022 for it.
func configProject(t *testing.T) (root, cfgText string) {
	t.Helper()
	root = project(t)
	write(t, root, "docs/prd/neu.md", strings.ReplaceAll(searchPRD, "search", "neu"))
	return root, readFileT(t, root, "sdd.config.yaml")
}

func configCall(t *testing.T, s *Server, content, base string, apply bool) map[string]any {
	t.Helper()
	return call(t, s, "sdd_config", map[string]any{"content": content, "base_sha256": base, "apply": apply})
}

// desk-config-critical AC-1: sdd_config exists only with --allow-critical, changes only an existing configuration with
// the current sha256 and a content that loads; otherwise it writes nothing and names the reason.
func TestConfigToolNeedsFlag(t *testing.T) {
	root, cfgText := configProject(t)
	for _, n := range toolNames(t, newServer(root, true)) {
		if n == "sdd_config" {
			t.Fatal("sdd_config offered without --allow-critical")
		}
	}
	res := exchange(t, newServer(root, true), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"sdd_config","arguments":{"content":"x","base_sha256":"`+shaOf(cfgText)+`"}}}`)
	if res[0]["error"] == nil {
		t.Errorf("sdd_config callable without --allow-critical: %v", res)
	}
}

// desk-config-critical AC-1: invalid content, a stale or missing sha256 and a missing configuration are refused.
func TestConfigToolRefusesInvalid(t *testing.T) {
	root, cfgText := configProject(t)
	s := criticalServer(root)
	bad := strings.Replace(cfgText, "profile: standard", "profile: bogus", 1)
	cases := map[string]map[string]any{
		"invalid": configCall(t, s, bad, shaOf(cfgText), true),
		"stale":   configCall(t, s, cfgText+"\n", shaOf("anders"), true),
		"missing": configCall(t, s, cfgText+"\n", "", true),
	}
	for name, r := range cases {
		if r["isError"] != true {
			t.Errorf("%s: accepted: %v", name, r)
		}
	}
	if !strings.Contains(errText(cases["invalid"]), "bogus") || !strings.Contains(errText(cases["stale"]), shaOf(cfgText)) {
		t.Errorf("reasons: %q / %q", errText(cases["invalid"]), errText(cases["stale"]))
	}
	if readFileT(t, root, "sdd.config.yaml") != cfgText {
		t.Error("configuration changed")
	}
	empty := t.TempDir()
	if r := configCall(t, criticalServer(empty), "version: 1\n", "", false); r["isError"] != true || !strings.Contains(errText(r), "sdd_init") {
		t.Errorf("without configuration: %v", r)
	}
}

// desk-config-critical AC-2: the preview shows the diff and, per check run, the findings that disappear or appear.
func TestConfigToolPreview(t *testing.T) {
	root, cfgText := configProject(t)
	s := criticalServer(root)
	excluded := strings.Replace(cfgText, "exclude:\n", "exclude:\n  - docs/prd/neu.md\n", 1)
	if excluded == cfgText {
		t.Fatal("test setup: no exclude list in the configuration")
	}
	prev := structured(t, configCall(t, s, excluded, shaOf(cfgText), false))
	if prev["applied"] != false || prev["critical"] != true {
		t.Fatalf("preview: %v", prev)
	}
	if op := prev["ops"].([]any)[0].(map[string]any); !strings.Contains(op["diff"].(string), "+  - docs/prd/neu.md") {
		t.Errorf("diff: %v", op)
	}
	data, _ := json.Marshal(prev["findings_delta"])
	var deltas []runDelta
	if err := json.Unmarshal(data, &deltas); err != nil {
		t.Fatal(err)
	}
	var lint *runDelta
	for i := range deltas {
		if deltas[i].Gate == "lint" {
			lint = &deltas[i]
		}
	}
	if lint == nil || len(lint.Dropped) == 0 || len(lint.Added) != 0 {
		t.Fatalf("lint delta: %+v", deltas)
	}
	for _, f := range lint.Dropped {
		if f.Path != "docs/prd/neu.md" {
			t.Errorf("dropped finding outside the excluded path: %+v", f)
		}
	}
	if !strings.Contains(prev["summary"].(string), "exclude") || !strings.Contains(prev["summary"].(string), "−") {
		t.Errorf("summary: %v", prev["summary"])
	}
	if readFileT(t, root, "sdd.config.yaml") != cfgText {
		t.Error("preview wrote the configuration")
	}
}

// desk-config-critical AC-3: the summary names a profile change with both values and every gate override.
func TestConfigToolSummary(t *testing.T) {
	root, cfgText := configProject(t)
	s := criticalServer(root)
	changed := strings.Replace(cfgText, "profile: standard", "profile: minimal", 1) + "\ngates:\n  pre_merge:\n    override: [SDD-001]\n"
	r := configCall(t, s, changed, shaOf(cfgText), false)
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	sum := structured(t, r)["summary"].(string)
	for _, want := range []string{"profile standard → minimal", "gates.pre_merge.override"} {
		if !strings.Contains(sum, want) {
			t.Errorf("summary %q lacks %q", sum, want)
		}
	}
}

// desk-config-critical AC-4: apply replaces the file and returns sha256 and diff of what was written.
func TestConfigToolApply(t *testing.T) {
	root, cfgText := configProject(t)
	s := criticalServer(root)
	changed := strings.Replace(cfgText, "exclude:\n", "exclude:\n  - docs/prd/neu.md\n", 1)
	r := configCall(t, s, changed, shaOf(cfgText), true)
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	res := structured(t, r)
	if res["applied"] != true || res["sha256"] != shaOf(changed) || !strings.Contains(res["diff"].(string), "docs/prd/neu.md") {
		t.Errorf("result: %v", res)
	}
	if readFileT(t, root, "sdd.config.yaml") != changed {
		t.Error("configuration not written")
	}
}

// desk-config-critical AC-5: sdd_write refuses sdd.config.yaml, as preview and with apply, and points to sdd_config.
func TestWriteRefusesConfig(t *testing.T) {
	root, cfgText := configProject(t)
	s := criticalServer(root)
	for _, apply := range []bool{false, true} {
		r := call(t, s, "sdd_write", map[string]any{"path": "sdd.config.yaml", "content": cfgText + "\n", "base_sha256": shaOf(cfgText), "apply": apply})
		if r["isError"] != true || !strings.Contains(errText(r), "sdd_config") {
			t.Errorf("apply %v: %v", apply, r)
		}
	}
	if readFileT(t, root, "sdd.config.yaml") != cfgText {
		t.Error("configuration changed")
	}
	if r := readTool(t, s, "sdd.config.yaml"); r["isError"] == true {
		t.Errorf("sdd_read must still read the configuration: %v", r)
	}
}

// desk-config-critical AC-7: prompts and server instructions name sdd_config for the configuration and treat it as
// critical; sdd-tests extends tests.include with sdd_config. No text tells the model to change it with sdd_write.
func TestPromptsConfigTool(t *testing.T) {
	s := criticalServer(project(t))
	init := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	instr, _ := init[0]["result"].(map[string]any)["instructions"].(string)
	texts := map[string]string{"instructions": instr}
	for _, name := range []string{"sdd-idea", "sdd-start", "sdd-tests", "sdd-constitution"} {
		texts[name] = promptText(t, s, name, `{"feature":"billing","title":"X"}`)
	}
	viaWrite := regexp.MustCompile(`sdd\.config\.yaml[^.;]*sdd_write|tests\.include mit sdd_write`)
	for name, text := range texts {
		if !strings.Contains(text, "sdd_config") {
			t.Errorf("%s does not name sdd_config", name)
		}
		if viaWrite.MatchString(text) {
			t.Errorf("%s still changes the configuration with sdd_write", name)
		}
	}
	if !strings.Contains(texts["sdd-tests"], "tests.include mit sdd_config") || !strings.Contains(texts["sdd-idea"], "einzelnen Bestätigung") {
		t.Errorf("sdd-tests or common lacks the critical config rule")
	}
}
