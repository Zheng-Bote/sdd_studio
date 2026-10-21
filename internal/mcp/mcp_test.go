package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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
	write(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: active\ntouches: [api/**]\n---\n## Nicht-Ziele\n- nichts\n## Akzeptanzkriterien\n- AC-1 Export\n- AC-2 Import\n")
	write(t, root, "docs/design/billing.md", "---\nfeature: billing\nstatus: active\ntouches: [api/export/**]\n---\n| AC-1 | Export |\n")
	write(t, root, "docs/plans/billing.md", "---\nfeature: billing\n---\n- [ ] AC-1 Export bauen\n")
	write(t, root, "docs/adr/0001-api.md", "---\nstatus: accepted\nreferences: [api/**]\n---\n# API\n")
	write(t, root, "api/export_test.go", "// billing AC-1: export\nfunc TestExport(t *testing.T) {}\n")
	return root
}

func newServer(root string, allowWrite bool) *Server {
	s := New(root, allowWrite, "test", nil)
	s.Now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
	return s
}

// exchange sends messages (one per line) and returns the decoded responses.
func exchange(t *testing.T, s *Server, msgs ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(strings.Join(msgs, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid response line %q: %v", line, err)
		}
		res = append(res, m)
	}
	return res
}

func call(t *testing.T, s *Server, tool string, args any) map[string]any {
	t.Helper()
	a, _ := json.Marshal(args)
	res := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"`+tool+`","arguments":`+string(a)+`}}`)
	if len(res) != 1 || res[0]["result"] == nil {
		t.Fatalf("%s: unexpected response %v", tool, res)
	}
	return res[0]["result"].(map[string]any)
}

func structured(t *testing.T, r map[string]any) map[string]any {
	t.Helper()
	if r["isError"] == true {
		t.Fatalf("tool error: %v", r["content"])
	}
	return r["structuredContent"].(map[string]any)
}

// sdd-guidance AC-3: handshake with version negotiation, ping, errors without aborting, silent notifications.
func TestProtocolHandshakeAndErrors(t *testing.T) {
	s := newServer(project(t), false)
	res := exchange(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`,
		`{"jsonrpc":"2.0","id":"p","method":"ping"}`,
		`kein json`,
		`{"jsonrpc":"2.0","id":3,"method":"gibt/es/nicht"}`,
		`{"jsonrpc":"1.0","id":4,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"sdd_unbekannt"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":"falsch"}`,
	)
	if len(res) != 8 {
		t.Fatalf("expected 8 responses (notification unanswered), got %d: %v", len(res), res)
	}
	init := res[0]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" || init["capabilities"].(map[string]any)["tools"] == nil {
		t.Errorf("initialize: %v", init)
	}
	if res[1]["result"].(map[string]any)["protocolVersion"] != SupportedVersions[0] {
		t.Errorf("unknown version must fall back to the newest: %v", res[1])
	}
	if res[2]["id"] != "p" || res[2]["result"] == nil {
		t.Errorf("ping: %v", res[2])
	}
	for i, code := range map[int]float64{3: codeParseError, 4: codeMethodNotFound, 5: codeInvalidRequest, 6: codeInvalidParams, 7: codeInvalidParams} {
		e, _ := res[i]["error"].(map[string]any)
		if e == nil || e["code"] != code {
			t.Errorf("response %d: want error %v, got %v", i, code, res[i])
		}
	}
}

// sdd-guidance AC-4: tools return what the library functions return.
func TestToolsMatchLibrary(t *testing.T) {
	root := project(t)
	s := newServer(root, false)

	got := structured(t, call(t, s, "sdd_next", map[string]any{"feature": "billing"}))
	want, err := guide.Next(root, "billing")
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	var a, b any
	_ = json.Unmarshal(wantJSON, &a)
	_ = json.Unmarshal(gotJSON, &b)
	if !jsonEqual(a, b) {
		t.Errorf("sdd_next differs from guide.Next:\n%s\n%s", gotJSON, wantJSON)
	}

	lint := structured(t, call(t, s, "sdd_lint", map[string]any{}))
	if lint["schema_version"] != float64(1) {
		t.Errorf("sdd_lint: %v", lint)
	}
	gate := structured(t, call(t, s, "sdd_gate", map[string]any{"gate": "pre-release"}))
	if _, ok := gate["passed"].(bool); !ok {
		t.Errorf("sdd_gate: %v", gate)
	}
	ex := structured(t, call(t, s, "sdd_explain", map[string]any{"rule_id": "SDD-041"}))
	if !strings.Contains(ex["section"].(string), "SDD-041") || !strings.Contains(ex["contract"].(string), "Prüfvertrag") {
		t.Errorf("sdd_explain: %v", ex)
	}
	in := structured(t, call(t, s, "sdd_check_intent", map[string]any{"paths": []string{"constitution.md"}, "feature": "billing"}))
	if in["constitution_touched"] != true || in["has_prd"] != true {
		t.Errorf("sdd_check_intent: %v", in)
	}
	tr := structured(t, call(t, s, "sdd_trace", map[string]any{"feature": "billing"}))
	crit := tr["criteria"].([]any)
	first, second := crit[0].(map[string]any), crit[1].(map[string]any)
	if first["id"] != "AC-1" || len(first["tests"].([]any)) != 1 || len(first["tasks"].([]any)) != 1 || len(first["tech_specs"].([]any)) != 1 ||
		second["id"] != "AC-2" || len(second["tests"].([]any)) != 0 {
		t.Errorf("sdd_trace: %v", tr)
	}
	if r := call(t, s, "sdd_lint", map[string]any{"gate": "bogus"}); r["isError"] != true {
		t.Errorf("invalid gate must be a tool error: %v", r)
	}
}

func jsonEqual(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// sdd-guidance AC-4: sdd_context orders the governing artifacts by precedence.
func TestContextPrecedence(t *testing.T) {
	root := project(t)
	got, err := guide.Context(root, "api/export/csv.go")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, g := range got {
		order = append(order, g.Kind+":"+g.Path)
	}
	want := "Verfassung:constitution.md ADR:docs/adr/0001-api.md Tech Spec:docs/design/billing.md PRD:docs/prd/billing.md"
	if strings.Join(order, " ") != want {
		t.Fatalf("precedence\n got %v\nwant %s", order, want)
	}
	other, _ := guide.Context(root, "README.md")
	if len(other) != 1 {
		t.Errorf("unrelated file: only the constitution applies, got %v", other)
	}
	if _, err := guide.Context(root, "../etc/passwd"); err == nil {
		t.Errorf("path outside the repository accepted")
	}
}

// sdd-guidance AC-4: a draft is checked as if stored, without writing it to the repository.
func TestValidateDraft(t *testing.T) {
	root := project(t)
	s := newServer(root, false)
	d := structured(t, call(t, s, "sdd_validate_draft", map[string]any{
		"path": "docs/prd/search.md", "content": "---\nfeature: search\nstatus: draft\n---\n## Akzeptanzkriterien\n- ohne Kennung\n",
	}))
	rules := map[string]bool{}
	for _, f := range d["findings"].([]any) {
		rules[f.(map[string]any)["rule_id"].(string)] = true
	}
	if !rules["SDD-021"] || !rules["SDD-023"] || rules["SDD-022"] {
		t.Errorf("draft findings %v", d)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/search.md")); err == nil {
		t.Errorf("draft was written to the repository")
	}
	ok := structured(t, call(t, s, "sdd_validate_draft", map[string]any{
		"path": "docs/prd/search.md", "content": "---\nfeature: search\nstatus: draft\n---\n## Nicht-Ziele\n- x\n## Akzeptanzkriterien\n- AC-1 x\n",
	}))
	if ok["errors"] != float64(0) || ok["warnings"] != float64(0) {
		t.Errorf("clean draft: %v", ok)
	}
	if r := call(t, s, "sdd_validate_draft", map[string]any{"path": "../x.md", "content": "x"}); r["isError"] != true {
		t.Errorf("path outside the repository accepted")
	}
}

// sdd-guidance AC-5: without --allow-write sdd_new only previews; with it, files are created via plan/apply.
func TestNewNeedsAllowWrite(t *testing.T) {
	root := project(t)
	ro := newServer(root, false)
	prev := structured(t, call(t, ro, "sdd_new", map[string]any{"feature": "search"}))
	ops := prev["ops"].([]any)
	if prev["applied"] != false || len(ops) != 2 || !strings.Contains(ops[0].(map[string]any)["content"].(string), "feature: search") {
		t.Fatalf("preview %v", prev)
	}
	if r := call(t, ro, "sdd_new", map[string]any{"feature": "search", "apply": true}); r["isError"] != true {
		t.Fatalf("apply without --allow-write must fail: %v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/search.md")); err == nil {
		t.Fatalf("file written without --allow-write")
	}
	rw := newServer(root, true)
	structured(t, call(t, rw, "sdd_new", map[string]any{"feature": "search", "apply": true}))
	if _, err := os.Stat(filepath.Join(root, "docs/prd/search.md")); err != nil {
		t.Fatalf("file not written with --allow-write: %v", err)
	}
	again := structured(t, call(t, rw, "sdd_new", map[string]any{"feature": "search", "apply": false}))
	for _, op := range again["ops"].([]any) {
		if op.(map[string]any)["action"] != "skip" {
			t.Errorf("existing files must be skipped: %v", op)
		}
	}
}

// sdd-guidance AC-6: no capability touches baseline, lockfile, Git or ADR decisions; paths stay in the repository.
func TestNoForbiddenCapabilities(t *testing.T) {
	root := project(t)
	s := newServer(root, true)
	res := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, `{"jsonrpc":"2.0","id":2,"method":"resources/list"}`, `{"jsonrpc":"2.0","id":3,"method":"prompts/list"}`)
	forbidden := regexp.MustCompile(`(?i)baseline|accept|lock|adopt|commit|branch|push|merge`)
	for _, t2 := range res[0]["result"].(map[string]any)["tools"].([]any) {
		name := t2.(map[string]any)["name"].(string)
		if forbidden.MatchString(name) {
			t.Errorf("forbidden tool %s", name)
		}
	}
	if n := len(res[0]["result"].(map[string]any)["tools"].([]any)); n != 11 {
		t.Errorf("tool set changed (%d tools); review it against ADR 0002", n)
	}
	for _, r := range res[1]["result"].(map[string]any)["resources"].([]any) {
		uri := r.(map[string]any)["uri"].(string)
		if strings.Contains(uri, ".sdd/") {
			t.Errorf("tool state exposed as resource: %s", uri)
		}
	}
	for _, uri := range []string{"sdd://artifact/../etc/passwd", "sdd://artifact/api/export_test.go", "sdd://artifact/.sdd/baseline.json", "file:///etc/passwd"} {
		r := exchange(t, s, `{"jsonrpc":"2.0","id":9,"method":"resources/read","params":{"uri":"`+uri+`"}}`)
		if r[0]["error"] == nil {
			t.Errorf("%s readable", uri)
		}
	}
	ok := exchange(t, s, `{"jsonrpc":"2.0","id":9,"method":"resources/read","params":{"uri":"sdd://artifact/docs/prd/billing.md"}}`)
	if ok[0]["error"] != nil {
		t.Errorf("artifact not readable: %v", ok)
	}
	if r := call(t, s, "sdd_new", map[string]any{"types": []string{"adr"}, "title": "x", "apply": false}); r["isError"] == true ||
		!strings.Contains(r["content"].([]any)[0].(map[string]any)["text"].(string), "status: proposed") {
		t.Errorf("ADRs are created as proposed only: %v", r)
	}
}

// sdd-guidance AC-7: every guided step names its rules and requires validating drafts.
func TestPrompts(t *testing.T) {
	s := newServer(project(t), false)
	res := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"prompts/list"}`)
	names := []string{}
	for _, p := range res[0]["result"].(map[string]any)["prompts"].([]any) {
		names = append(names, p.(map[string]any)["name"].(string))
	}
	if strings.Join(names, ",") != "sdd-idea,sdd-techspec,sdd-adr,sdd-plan,sdd-conflicts,sdd-review" {
		t.Fatalf("prompts %v", names)
	}
	args := map[string]string{"sdd-idea": `{}`, "sdd-techspec": `{"feature":"billing"}`, "sdd-adr": `{"title":"x"}`, "sdd-plan": `{"feature":"billing"}`, "sdd-conflicts": `{"feature":"billing"}`, "sdd-review": `{"feature":"billing"}`}
	wantRule := map[string]string{"sdd-idea": "SDD-023", "sdd-techspec": "SDD-031", "sdd-adr": "SDD-011", "sdd-plan": "SDD-041", "sdd-conflicts": "SDD-010", "sdd-review": "SDD-022"}
	for _, name := range names {
		r := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"`+name+`","arguments":`+args[name]+`}}`)
		text := r[0]["result"].(map[string]any)["messages"].([]any)[0].(map[string]any)["content"].(map[string]any)["text"].(string)
		for _, want := range []string{"sdd_validate_draft", "Frage, statt zu erfinden", wantRule[name], "dem Menschen vorbehalten"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}
	if r := exchange(t, s, `{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"sdd-plan","arguments":{}}}`); r[0]["error"] == nil {
		t.Errorf("missing required argument accepted")
	}
}

func readTool(t *testing.T, s *Server, path string) map[string]any {
	t.Helper()
	return call(t, s, "sdd_read", map[string]any{"path": path})
}

// sdd-conflicts AC-1: without a path the readable artifacts are listed, with a path the text comes with line numbers.
func TestReadTool(t *testing.T) {
	root := project(t)
	s := newServer(root, false)
	list := structured(t, call(t, s, "sdd_read", map[string]any{}))["artifacts"].([]any)
	got := []string{}
	for _, p := range list {
		got = append(got, p.(string))
	}
	for _, want := range []string{"constitution.md", "docs/adr/0001-api.md", "docs/prd/billing.md", "docs/design/billing.md", "docs/plans/billing.md"} {
		if !strings.Contains(strings.Join(got, ","), want) {
			t.Errorf("%s missing in %v", want, got)
		}
	}
	if strings.Contains(strings.Join(got, ","), "api/export_test.go") {
		t.Errorf("code listed as artifact: %v", got)
	}
	r := structured(t, readTool(t, s, "docs/design/billing.md"))
	text := r["text"].(string)
	if r["path"] != "docs/design/billing.md" || r["truncated"] != false || r["lines"].(float64) != 6 ||
		!strings.HasPrefix(text, "   1| ---\n") || !strings.Contains(text, "   6| | AC-1 | Export |\n") {
		t.Errorf("read %v", r)
	}
	if r := structured(t, readTool(t, s, "./docs/prd/../prd/billing.md")); r["path"] != "docs/prd/billing.md" {
		t.Errorf("clean path: %v", r["path"])
	}
}

// sdd-conflicts AC-2: anything that is not a readable artifact is refused; long texts are cut with a note.
func TestReadToolRefuses(t *testing.T) {
	root := project(t)
	write(t, root, "docs/prd/notes.txt", "geheim")
	write(t, root, ".env", "KEY=geheim")
	write(t, root, "notes/private.md", "geheim")
	s := newServer(root, false)
	for _, p := range []string{"notes/private.md", "README.md", "api/export_test.go", "../etc/passwd", "/etc/passwd", "docs/prd/notes.txt", ".env", ".sdd/baseline.json", "sdd.config.yaml", "docs/prd/missing.md"} {
		if r := readTool(t, s, p); r["isError"] != true {
			t.Errorf("%s readable: %v", p, r)
		}
	}
	// A constitution configured with another extension is listed but not readable: only .md and .sdd are.
	cfgPath := filepath.Join(root, "sdd.config.yaml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "sdd.config.yaml", regexp.MustCompile(`constitution: .*`).ReplaceAllString(string(cfg), `constitution: ".env"`))
	if r := readTool(t, s, ".env"); r["isError"] != true {
		t.Errorf("constitution with another extension readable: %v", r)
	}
	write(t, root, "sdd.config.yaml", string(cfg))

	long := "---\nfeature: big\nstatus: active\n---\n" + strings.Repeat(strings.Repeat("x", 99)+"\n", 4000)
	write(t, root, "docs/prd/big.md", long)
	r := structured(t, readTool(t, s, "docs/prd/big.md"))
	text := r["text"].(string)
	if r["truncated"] != true || len(text) > maxReadBytes+200 || !strings.Contains(text, "gekürzt nach Zeile") || r["lines"].(float64) != 4004 {
		t.Errorf("truncation: truncated=%v len=%d lines=%v", r["truncated"], len(text), r["lines"])
	}
}

// sdd-conflicts AC-3: the guided check reads with sdd_read, reports only quoted contradictions with lines on both sides and changes nothing.
func TestConflictsPrompt(t *testing.T) {
	s := newServer(project(t), false)
	r := exchange(t, s, `{"jsonrpc":"2.0","id":1,"method":"prompts/get","params":{"name":"sdd-conflicts","arguments":{"feature":"billing"}}}`)
	text := r[0]["result"].(map[string]any)["messages"].([]any)[0].(map[string]any)["content"].(map[string]any)["text"].(string)
	for _, want := range []string{"sdd_read", "feature: billing", "nur Pfade aus dieser Liste", "wörtlich zitieren", "Datei:Zeile", "Ohne wörtliches Zitat keine Meldung", "Fehlt etwas in der Tech Spec, gehört es nicht in die Tabelle", "einzeln durch", "nicht die SDD-Regeln des Katalogs", "kein Widerspruch gefunden",
		"Widerspruch, Spannung oder unklar", "Tech Spec anpassen", "ADR für eine Verfassungsänderung", "kein Befund", "Ändere nichts", "SDD-010"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt misses %q", want)
		}
	}
	if r := exchange(t, s, `{"jsonrpc":"2.0","id":2,"method":"prompts/get","params":{"name":"sdd-conflicts","arguments":{}}}`); r[0]["error"] == nil {
		t.Error("missing feature accepted")
	}
}

// sdd-conflicts AC-4: SDD-060 is no catalog rule; concept and handbook describe it as an assisted check in sdd-desk.
func TestSDD060NotInCatalog(t *testing.T) {
	cat, err := os.ReadFile("../../catalog/catalog.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cat), "SDD-060") {
		t.Error("SDD-060 must not be a catalog rule: an LLM result is no finding")
	}
	for _, f := range []string{"../../docs/sdd_konzept.md", "../../handbuch/04-agenten.md"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "SDD-060") || !strings.Contains(string(data), "sdd-conflicts") || !strings.Contains(string(data), "kein Befund") {
			t.Errorf("%s must describe SDD-060 as assisted check (sdd-conflicts, kein Befund)", f)
		}
	}
}

// sdd-guidance AC-8: sdd (core, CLI, ui, mcp) contains no code for outbound connections (ADR 0002).
func TestNoOutboundNetworkCode(t *testing.T) {
	outbound := regexp.MustCompile(`http\.(Get|Post|PostForm|Head|NewRequest|NewRequestWithContext|DefaultClient|DefaultTransport)\b|http\.Client\b|net\.Dial|\.DialContext\(|tls\.Dial|net/rpc|net/smtp`)
	for _, dir := range []string{"../sddcore", "../ui", ".", "../../cmd/sdd"} {
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if m := outbound.FindString(string(data)); m != "" {
				t.Errorf("%s contains outbound network code: %s", p, m)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
