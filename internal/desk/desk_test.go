package desk

import (
	"context"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/mcp"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

func writeFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
}

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// sdd-desk AC-1: llm.env next to the program, process environment first, keyless providers off, open permissions refused.
func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config", "llm.env")
	writeFile(t, path, "# Kommentar\nexport CLAUDE_API_KEY=\"sk-file\"\nCLAUDE_LLM_01=claude-a\nOPENROUTER_API_KEY=none\nOPENROUTER_LLM_01=x\nGEMINI_API_KEY=\nNVIDIA_API_KEY=nv\nNVIDIA_LLM_01=n1\nNVIDIA_LLM_02=n2\nOLLAMA_LOCAL_ENDPOINT=http://127.0.0.1:1/\n", 0o600)

	cfg, err := LoadConfig(path, true, "", env(map[string]string{"CLAUDE_API_KEY": "sk-env", "CLAUDE_LLM_01": "claude-env"}))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Provider{}
	for _, p := range cfg.Providers {
		byID[p.ID] = p
	}
	if len(byID) != 3 || byID["openrouter"].ID != "" || byID["gemini"].ID != "" {
		t.Fatalf("providers %+v", cfg.Providers)
	}
	if c := byID["claude"]; c.APIKey != "sk-env" || len(c.Models) != 1 || c.Models[0] != "claude-env" || !c.Cloud {
		t.Errorf("process environment must win: %+v", c)
	}
	if n := byID["nvidia"]; strings.Join(n.Models, ",") != "n1,n2" || n.Endpoint != "https://integrate.api.nvidia.com/v1" {
		t.Errorf("nvidia %+v", n)
	}
	if l := byID[LocalOllamaID]; l.Cloud || l.Endpoint != "http://127.0.0.1:1" || l.APIKey != "" {
		t.Errorf("local ollama %+v", l)
	}
	data, _ := json.Marshal(cfg)
	if strings.Contains(string(data), "sk-env") || strings.Contains(string(data), "sk-file") {
		t.Errorf("API keys must never be serialized: %s", data)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path, true, "", env(nil)); !errors.Is(err, ErrInsecureConfig) {
		t.Errorf("readable for others: want ErrInsecureConfig, got %v", err)
	}
	if _, err := LoadConfig(filepath.Join(dir, "missing.env"), true, "", env(nil)); err == nil {
		t.Errorf("missing explicit file must fail")
	}
	cfg, err = LoadConfig(filepath.Join(dir, "missing.env"), false, "", env(nil))
	if err != nil || len(cfg.Providers) != 1 || cfg.Providers[0].ID != LocalOllamaID || len(cfg.Warnings) == 0 {
		t.Errorf("missing default file: only local ollama with a warning, got %+v %v", cfg, err)
	}
}

// fakeServer records the last request and answers with a fixed body.
func fakeServer(t *testing.T, answer string) (*httptest.Server, *http.Request, *[]byte) {
	t.Helper()
	var last http.Request
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = *r
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(srv.Close)
	return srv, &last, &body
}

var testMsgs = []Message{
	{Role: "user", Content: "Hallo"},
	{Role: "assistant", Content: "Ich prüfe.", ToolCalls: []ToolCall{{ID: "c1", Name: "sdd_next", Arguments: json.RawMessage(`{"feature":"x"}`)}}},
	{Role: "tool", ToolCallID: "c1", ToolName: "sdd_next", Content: "ok"},
}
var testTools = []ToolDef{{Name: "sdd_next", Description: "next", Schema: json.RawMessage(`{"type":"object"}`)}}

// sdd-desk AC-2: each protocol is spoken correctly, including tool calls and tool results.
func TestProviders(t *testing.T) {
	ctx := context.Background()

	t.Run("ollama cloud", func(t *testing.T) {
		srv, req, body := fakeServer(t, `{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"sdd_lint","arguments":{"gate":"pre-merge"}}}]}}`)
		got, err := NewChat(Provider{Name: "o", Protocol: ProtocolOllama, Endpoint: srv.URL, APIKey: "k"}, srv.Client()).Complete(ctx, "m", "SYS", testMsgs, testTools)
		if err != nil {
			t.Fatal(err)
		}
		if req.URL.Path != "/api/chat" || req.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("request %s %v", req.URL.Path, req.Header)
		}
		var b map[string]any
		_ = json.Unmarshal(*body, &b)
		msgs := b["messages"].([]any)
		if msgs[0].(map[string]any)["role"] != "system" || len(msgs) != 4 || b["stream"] != false || len(b["tools"].([]any)) != 1 {
			t.Errorf("body %s", *body)
		}
		if len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "sdd_lint" || string(got.ToolCalls[0].Arguments) != `{"gate":"pre-merge"}` {
			t.Errorf("parsed %+v", got)
		}
	})

	t.Run("openai compatible", func(t *testing.T) {
		srv, req, body := fakeServer(t, `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"x1","type":"function","function":{"name":"sdd_next","arguments":"{\"feature\":\"a\"}"}}]}}]}`)
		got, err := NewChat(Provider{Name: "or", Protocol: ProtocolOpenAI, Endpoint: srv.URL, APIKey: "k"}, srv.Client()).Complete(ctx, "m", "SYS", testMsgs, testTools)
		if err != nil {
			t.Fatal(err)
		}
		if req.URL.Path != "/chat/completions" || req.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("request %s", req.URL.Path)
		}
		s := string(*body)
		if !strings.Contains(s, `"tool_call_id":"c1"`) || !strings.Contains(s, `"arguments":"{\"feature\":\"x\"}"`) {
			t.Errorf("body %s", s)
		}
		if got.ToolCalls[0].ID != "x1" || string(got.ToolCalls[0].Arguments) != `{"feature":"a"}` {
			t.Errorf("parsed %+v", got)
		}
	})

	t.Run("claude", func(t *testing.T) {
		srv, req, body := fakeServer(t, `{"content":[{"type":"text","text":"Gern."},{"type":"tool_use","id":"tu1","name":"sdd_next","input":{}}]}`)
		got, err := NewChat(Provider{Name: "c", Protocol: ProtocolClaude, Endpoint: srv.URL, APIKey: "k"}, srv.Client()).Complete(ctx, "m", "SYS", testMsgs, testTools)
		if err != nil {
			t.Fatal(err)
		}
		if req.URL.Path != "/v1/messages" || req.Header.Get("x-api-key") != "k" || req.Header.Get("anthropic-version") == "" {
			t.Errorf("request %s %v", req.URL.Path, req.Header)
		}
		var b map[string]any
		_ = json.Unmarshal(*body, &b)
		msgs := b["messages"].([]any)
		last := msgs[len(msgs)-1].(map[string]any)
		if b["system"] != "SYS" || len(msgs) != 3 || last["role"] != "user" || !strings.Contains(string(*body), `"tool_result"`) || !strings.Contains(string(*body), `"input_schema"`) {
			t.Errorf("body %s", *body)
		}
		if got.Content != "Gern." || got.ToolCalls[0].ID != "tu1" {
			t.Errorf("parsed %+v", got)
		}
	})

	t.Run("http error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "kaputt", 500) }))
		defer srv.Close()
		if _, err := NewChat(Provider{Name: "x", Protocol: ProtocolOpenAI, Endpoint: srv.URL}, srv.Client()).Complete(ctx, "m", "", nil, nil); err == nil || !strings.Contains(err.Error(), "500") {
			t.Errorf("want HTTP error, got %v", err)
		}
	})
}

// sdd-desk-config-hint AC-1: without a file at the default location the warning names the hint, otherwise the default location; the hint is never read.
func TestLoadConfigHint(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "bin", "config", "llm.env")
	hint := filepath.Join(dir, "app", "llm.env")
	writeFile(t, hint, "CLAUDE_API_KEY=sk-hint\nCLAUDE_LLM_01=c\n", 0o644) // must not be read (and its permissions not checked)

	cfg, err := LoadConfig(missing, false, hint, env(nil))
	if err != nil || len(cfg.Providers) != 1 || len(cfg.Warnings) != 1 {
		t.Fatalf("cfg %+v, err %v", cfg, err)
	}
	if w := cfg.Warnings[0]; !strings.Contains(w, hint) || strings.Contains(w, missing) || !strings.Contains(w, "0600") {
		t.Errorf("warning must name the hint: %q", w)
	}
	cfg, _ = LoadConfig(missing, false, "", env(nil))
	if w := cfg.Warnings[0]; !strings.Contains(w, missing) {
		t.Errorf("without hint the default location: %q", w)
	}
	if _, err := LoadConfig(missing, true, hint, env(nil)); err == nil {
		t.Error("an explicit missing file stays an error, whatever the hint")
	}
}

// ---------- agent with a scripted model and a real in-process sdd mcp ----------

type scripted struct {
	replies []Message
	calls   int
	seen    [][]Message
}

func (s *scripted) Complete(_ context.Context, _, _ string, msgs []Message, _ []ToolDef) (Message, error) {
	s.seen = append(s.seen, append([]Message{}, msgs...))
	if s.calls >= len(s.replies) {
		return Message{Role: "assistant", Content: "fertig"}, nil
	}
	r := s.replies[s.calls]
	s.calls++
	return r, nil
}

func sddProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	return root
}

// inProcessMCP connects an MCPClient to a real mcp.Server (the same code `sdd mcp` runs) through pipes.
func inProcessMCP(t *testing.T, root string) *MCPClient {
	t.Helper()
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()
	srv := mcp.New(root, true, "test", nil)
	go func() { _ = srv.Serve(serverR, serverW); _ = serverW.Close() }()
	c := NewMCPClient(clientR, clientW)
	t.Cleanup(func() { _ = clientW.Close() })
	if err := c.Initialize(); err != nil {
		t.Fatal(err)
	}
	return c
}

func newAgent(t *testing.T, root string, chat *scripted) *Agent {
	t.Helper()
	c := inProcessMCP(t, root)
	tools, err := c.Tools()
	if err != nil {
		t.Fatal(err)
	}
	return &Agent{Tools: tools, Runner: c, System: "S",
		Providers: []Provider{{ID: LocalOllamaID, Protocol: ProtocolOllama}, {ID: "claude", Protocol: ProtocolClaude, Cloud: true}},
		NewChat:   func(Provider) Chat { return chat }}
}

func call(id, name, args string) Message {
	return Message{Role: "assistant", ToolCalls: []ToolCall{{ID: id, Name: name, Arguments: json.RawMessage(args)}}}
}

func lastTool(msgs []Message) Message {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "tool" {
			return msgs[i]
		}
	}
	return Message{}
}

// sdd-desk AC-3: only tools of sdd mcp exist for the model; other names are refused without running anything.
func TestAgentUsesOnlySddTools(t *testing.T) {
	root := sddProject(t)
	chat := &scripted{replies: []Message{call("c1", "shell_exec", `{"cmd":"rm -rf /"}`)}}
	a := newAgent(t, root, chat)
	names := []string{}
	for _, tl := range a.Tools {
		names = append(names, tl.Name)
	}
	if len(names) != 11 || names[0] != "sdd_next" {
		t.Fatalf("tools must be exactly those of sdd mcp: %v", names)
	}
	res, err := a.Turn(context.Background(), TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: []Message{{Role: "user", Content: "x"}}, AllowedTools: []string{"shell_exec"}})
	if err != nil {
		t.Fatal(err)
	}
	if tm := lastTool(res.Messages); !strings.Contains(tm.Content, "kein Werkzeug von sdd mcp") || len(res.Pending) != 0 {
		t.Fatalf("unknown tool: %+v", res)
	}
}

// sdd-desk AC-4: every call needs permission (once, session, deny); writes always need a one-time approval with preview.
func TestApprovalFlow(t *testing.T) {
	root := sddProject(t)
	ctx := context.Background()
	base := []Message{{Role: "user", Content: "Was ist der nächste Schritt?"}}

	chat := &scripted{replies: []Message{call("c1", "sdd_next", `{}`)}}
	a := newAgent(t, root, chat)
	res, err := a.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: base})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pending) != 1 || res.Pending[0].Tool != "sdd_next" || res.Pending[0].Writes || chat.calls != 1 {
		t.Fatalf("first call must wait for permission: %+v", res)
	}

	res, err = a.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: res.Messages, ApprovedCalls: []string{"c1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pending) != 0 || !strings.Contains(lastTool(res.Messages).Content, "sdd new --feature") || res.Messages[len(res.Messages)-1].Content != "fertig" {
		t.Fatalf("approved once: %+v", res.Messages)
	}

	chat2 := &scripted{replies: []Message{call("d1", "sdd_lint", `{}`)}}
	a2 := newAgent(t, root, chat2)
	res, err = a2.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: base, AllowedTools: []string{"sdd_lint"}})
	if err != nil || len(res.Pending) != 0 || !strings.Contains(lastTool(res.Messages).Content, "schema_version") {
		t.Fatalf("session-allowed read tool must run directly: %+v %v", res, err)
	}

	chat3 := &scripted{replies: []Message{call("e1", "sdd_report", `{}`)}}
	a3 := newAgent(t, root, chat3)
	res, _ = a3.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: base})
	res, err = a3.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: res.Messages, DeniedCalls: []string{"e1"}})
	if err != nil || !strings.Contains(lastTool(res.Messages).Content, "abgelehnt") {
		t.Fatalf("denied: %+v %v", res, err)
	}

	chat4 := &scripted{replies: []Message{call("w1", "sdd_new", `{"feature":"billing","apply":true}`)}}
	a4 := newAgent(t, root, chat4)
	res, err = a4.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: base, AllowedTools: []string{"sdd_new"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pending) != 1 || !res.Pending[0].Writes || !strings.Contains(string(res.Pending[0].Preview), "feature: billing") {
		t.Fatalf("write must wait with preview even when the tool is allowed for the session: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err == nil {
		t.Fatalf("preview wrote a file")
	}
	if _, err = a4.Turn(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: res.Messages, ApprovedCalls: []string{"w1"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err != nil {
		t.Fatalf("approved write did not create the file: %v", err)
	}
}

// sdd-desk AC-5: a cloud provider is refused until the disclosure is acknowledged; local Ollama needs none.
func TestCloudNeedsDisclosure(t *testing.T) {
	root := sddProject(t)
	chat := &scripted{}
	a := newAgent(t, root, chat)
	msgs := []Message{{Role: "user", Content: "x"}}
	if _, err := a.Turn(context.Background(), TurnRequest{Provider: "claude", Model: "m", Messages: msgs}); !errors.Is(err, ErrDisclosure) || chat.calls != 0 && len(chat.seen) != 0 {
		t.Fatalf("want ErrDisclosure without a model call, got %v (calls %d)", err, len(chat.seen))
	}
	if _, err := a.Turn(context.Background(), TurnRequest{Provider: "claude", Model: "m", Messages: msgs, DisclosureAck: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Turn(context.Background(), TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: msgs}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Turn(context.Background(), TurnRequest{Provider: "unbekannt", Model: "m", Messages: msgs, DisclosureAck: true}); !errors.Is(err, ErrUnknownProvider) {
		t.Errorf("unknown provider: %v", err)
	}
}

func testServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	root := sddProject(t)
	c := inProcessMCP(t, root)
	tools, _ := c.Tools()
	cfg := &Config{Path: "x", Providers: []Provider{{ID: "claude", Name: "Claude", Cloud: true, APIKey: "geheim", Models: []string{"m"}}}}
	s := &Server{Addr: "127.0.0.1:48001", Token: "tok", Project: "demo", MCP: c, HTTP: http.DefaultClient, Config: cfg,
		Agent: &Agent{Tools: tools, Runner: c, Providers: cfg.Providers}}
	return s, s.Handler()
}

func request(t *testing.T, h http.Handler, method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://127.0.0.1:48001"+path, strings.NewReader(body))
	req.Host = "127.0.0.1:48001"
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var auth = map[string]string{"X-SDD-Token": "tok", "Content-Type": "application/json"}

// sdd-desk AC-6: guided prompts of sdd mcp and the process state are available to the UI.
func TestPromptsAndNext(t *testing.T) {
	_, h := testServer(t)
	rec := request(t, h, http.MethodGet, "/api/prompts", "", auth)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "sdd-idea") {
		t.Fatalf("prompts %d %s", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, "/api/prompts/get", `{"name":"sdd-idea","arguments":{"idea":"CSV-Export"}}`, auth)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "CSV-Export") {
		t.Fatalf("prompt get %d %s", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet, "/api/next", "", auth)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"start"`) {
		t.Fatalf("next %d %s", rec.Code, rec.Body.String())
	}
}

// sdd-desk AC-7: Host check, token for every API call, CSP, and no key in any response.
func TestServerSecurity(t *testing.T) {
	_, h := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "http://evil.example/api/meta", nil)
	req.Host = "evil.example:48001"
	req.Header.Set("X-SDD-Token", "tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("foreign host: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodGet, "/api/meta", "", nil); rec.Code != http.StatusForbidden {
		t.Errorf("no token: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, "/api/turn", `{}`, map[string]string{"X-SDD-Token": "tok", "Content-Type": "text/plain"}); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("form post: %d", rec.Code)
	}
	rec = request(t, h, http.MethodGet, "/api/meta", "", auth)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "geheim") || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("meta %d %v %s", rec.Code, rec.Header(), rec.Body.String())
	}
	if rec := request(t, h, http.MethodPost, "/api/turn", `{"provider":"claude","model":"m","messages":[]}`, auth); rec.Code != http.StatusPreconditionRequired {
		t.Errorf("cloud turn without disclosure: %d", rec.Code)
	}
}

// sdd-desk AC-8: sdd (core, CLI, ui, mcp) never imports the desktop.
func TestSddDoesNotImportDesk(t *testing.T) {
	for _, dir := range []string{"../sddcore", "../ui", "../mcp", "../../cmd/sdd"} {
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
				return err
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				if strings.Contains(imp.Path.Value, "/internal/desk") {
					t.Errorf("%s imports %s", p, imp.Path.Value)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
