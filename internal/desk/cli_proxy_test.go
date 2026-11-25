package desk

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRunner records tool calls instead of reaching `sdd mcp`.
type fakeRunner struct {
	mu    sync.Mutex
	calls []string
	args  []json.RawMessage
}

func (f *fakeRunner) CallTool(name string, args json.RawMessage) (*ToolResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
	f.args = append(f.args, args)
	return &ToolResult{Text: "ok:" + name, Structured: json.RawMessage(`{"done":true}`)}, nil
}

func (f *fakeRunner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeRunner) lastArgs() json.RawMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.args) == 0 {
		return nil
	}
	return f.args[len(f.args)-1]
}

func proxyTestTools() []ToolDef {
	return []ToolDef{
		{Name: "sdd_next", Description: "Stand", Schema: json.RawMessage(`{}`)},
		{Name: "sdd_write", Description: "schreiben", Schema: json.RawMessage(`{}`)},
		{Name: "sdd_baseline", Description: "Baseline", Schema: json.RawMessage(`{}`)},
	}
}

func newTestProxy(t *testing.T, runner ToolRunner) (*Proxy, string) {
	t.Helper()
	sessions := NewCLISessions()
	p := &Proxy{
		Approvals: NewApprovals(2 * time.Second),
		Sessions:  sessions,
		Tools:     func() ([]ToolDef, error) { return proxyTestTools(), nil },
		CallTool:  runner.CallTool,
	}
	_, token, err := sessions.Open("codewhale")
	if err != nil {
		t.Fatal(err)
	}
	return p, token
}

func doRPC(p *Proxy, token, method string, params any, id int) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, r)
	return w
}

func waitCards(t *testing.T, p *Proxy, n int) []CallView {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cards := p.Approvals.Open(); len(cards) >= n {
			return cards
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("erwartete %d offene Karten, hatte %d", n, p.Approvals.CountOpen())
	return nil
}

func testEvidence(t *testing.T, entries ...CLIEvidence) {
	t.Helper()
	saved := CLIEvidenceTable
	CLIEvidenceTable = entries
	t.Cleanup(func() { CLIEvidenceTable = saved })
}

func testSpecs(t *testing.T, specs ...CLISpec) {
	t.Helper()
	saved := CLISpecs
	CLISpecs = specs
	t.Cleanup(func() { CLISpecs = saved })
}

// agenten-cli-proxy AC-1: only a CLI that is installed, whose ADR conditions are proven and whose run against the
// proxy is proven is offered; every refusal names its reason. agy is never offered.
func TestCLIOfferableOnlyWithProof(t *testing.T) {
	testSpecs(t,
		CLISpec{ID: "claude", Binary: "claude", Start: claudeStart},
		CLISpec{ID: "agy", Binary: "agy"},
	)

	testEvidence(t,
		CLIEvidence{CLI: "claude", Status: EvidenceProven, ProxyProven: true, Source: "test"},
		CLIEvidence{CLI: "agy", Status: EvidenceUnproven, Source: "test"},
	)

	claude, _ := CLISpecByID("claude")
	if !claude.Offerable(true) {
		t.Errorf("claude with full proof must be offerable: %s", claude.BlockReason(true))
	}
	if r := claude.BlockReason(false); !strings.Contains(r, "nicht installiert") {
		t.Errorf("installed=false must name installation, got %q", r)
	}

	testEvidence(t, CLIEvidence{CLI: "claude", Status: EvidenceProven, ProxyProven: false, Source: "test"})
	if r := claude.BlockReason(true); !strings.Contains(r, "Transportbeleg") {
		t.Errorf("missing transport proof must name it, got %q", r)
	}

	agy, _ := CLISpecByID("agy")
	if agy.Offerable(true) {
		t.Error("agy must never be offerable without a proven write lock")
	}
	if r := agy.BlockReason(true); !strings.Contains(r, "nicht belegt") {
		t.Errorf("agy must name the missing proof, got %q", r)
	}
}

// agenten-cli-proxy AC-2: every tools/call waits for a decision; a critical call is never covered by a session
// allowance, and the refusal is explicit instead of silently narrowed.
func TestProxyCallNeedsDecision(t *testing.T) {
	runner := &fakeRunner{}
	p, token := newTestProxy(t, runner)

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- doRPC(p, token, "tools/call", map[string]any{
			"name": "sdd_write", "arguments": map[string]any{"apply": true, "path": "docs/x.md"},
		}, 1)
	}()
	cards := waitCards(t, p, 1)
	if runner.count() != 0 {
		t.Fatal("tool ran before the decision")
	}
	if cards[0].Tier != tierWrite || cards[0].CLI != "codewhale" || cards[0].Session == "" {
		t.Errorf("card must name tier, CLI and session: %+v", cards[0])
	}
	if err := p.Approvals.Decide(cards[0].CallID, Verdict{Allow: true, Scope: ScopeCall}); err != nil {
		t.Fatal(err)
	}
	w := <-done
	if runner.count() != 1 {
		t.Fatalf("tool must run once after approval, ran %d times", runner.count())
	}
	if !strings.Contains(w.Body.String(), "ok:sdd_write") {
		t.Errorf("response must carry the tool result, got %s", w.Body.String())
	}

	// Critical: a session allowance is refused, the call stays open and does not run.
	doneCrit := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		doneCrit <- doRPC(p, token, "tools/call", map[string]any{
			"name": "sdd_baseline", "arguments": map[string]any{"apply": true},
		}, 2)
	}()
	crit := waitCards(t, p, 1)
	if crit[0].Tier != tierCritical {
		t.Fatalf("sdd_baseline with apply must be critical, got %q", crit[0].Tier)
	}
	if err := p.Approvals.Decide(crit[0].CallID, Verdict{Allow: true, Scope: ScopeWriteSession}); !errors.Is(err, ErrScopeNotAllowed) {
		t.Errorf("session allowance on a critical call must be refused, got %v", err)
	}
	if runner.count() != 1 {
		t.Fatal("critical call ran despite the refused scope")
	}
	if err := p.Approvals.Decide(crit[0].CallID, Verdict{Allow: true, Scope: ScopeCall}); err != nil {
		t.Fatal(err)
	}
	<-doneCrit
	if runner.count() != 2 {
		t.Fatalf("approved critical call must run, count=%d", runner.count())
	}
}

// agenten-cli-proxy AC-3: the decision is bound to the call id, not to the order. Deciding the second card first
// runs the second call.
func TestProxyCorrelatesByID(t *testing.T) {
	runner := &fakeRunner{}
	p, token := newTestProxy(t, runner)

	paths := []string{"a.md", "b.md"}
	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func(id int, path string) {
			defer wg.Done()
			doRPC(p, token, "tools/call", map[string]any{
				"name": "sdd_write", "arguments": map[string]any{"apply": true, "path": path},
			}, id)
		}(i, path)
	}
	cards := waitCards(t, p, 2)
	byID := map[string]string{}
	for _, c := range cards {
		var args map[string]any
		_ = json.Unmarshal(c.Arguments, &args)
		byID[c.CallID], _ = args["path"].(string)
	}

	second := cards[1]
	wantPath := byID[second.CallID]
	if err := p.Approvals.Decide(second.CallID, Verdict{Allow: true, Scope: ScopeCall}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for runner.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if runner.count() != 1 {
		t.Fatalf("approved card must run exactly one call, ran %d", runner.count())
	}
	if got := string(runner.lastArgs()); !strings.Contains(got, wantPath) {
		t.Errorf("approved card %s must run its own call (%s), ran %s", second.CallID, wantPath, got)
	}

	_ = p.Approvals.Decide(cards[0].CallID, Verdict{Allow: false, Scope: ScopeCall})
	wg.Wait()
	if runner.count() != 1 {
		t.Errorf("the refused call must not run, count=%d", runner.count())
	}
}

// agenten-cli-proxy AC-4: an undecided call expires, refuses and runs nothing; a late decision finds no call.
func TestProxyCallExpires(t *testing.T) {
	runner := &fakeRunner{}
	sessions := NewCLISessions()
	p := &Proxy{
		Approvals: NewApprovals(60 * time.Millisecond),
		Sessions:  sessions,
		Tools:     func() ([]ToolDef, error) { return proxyTestTools(), nil },
		CallTool:  runner.CallTool,
	}
	_, token, err := sessions.Open("claude")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- doRPC(p, token, "tools/call", map[string]any{"name": "sdd_next", "arguments": map[string]any{}}, 7)
	}()
	cards := waitCards(t, p, 1)
	callID := cards[0].CallID
	w := <-done
	if runner.count() != 0 {
		t.Fatal("expired call ran")
	}
	if !strings.Contains(w.Body.String(), `"isError":true`) {
		t.Errorf("expired call must end as a tool error, got %s", w.Body.String())
	}
	if err := p.Approvals.Decide(callID, Verdict{Allow: true, Scope: ScopeCall}); !errors.Is(err, ErrUnknownCall) {
		t.Errorf("late decision must find nothing, got %v", err)
	}
	if runner.count() != 0 {
		t.Fatal("late decision executed a call")
	}
}

// agenten-cli-proxy AC-5: ending a session withdraws its open cards; no call runs afterwards. Closing the manager
// ends the child processes.
func TestSessionEndWithdrawsCards(t *testing.T) {
	runner := &fakeRunner{}
	p, token := newTestProxy(t, runner)

	done := make(chan struct{})
	go func() {
		doRPC(p, token, "tools/call", map[string]any{"name": "sdd_write", "arguments": map[string]any{"apply": true}}, 1)
		close(done)
	}()
	cards := waitCards(t, p, 1)
	p.Approvals.WithdrawSession(cards[0].Session, "CLI beendet")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("withdrawn call kept the CLI waiting")
	}
	if runner.count() != 0 {
		t.Fatal("withdrawn call ran")
	}
	if n := p.Approvals.CountOpen(); n != 0 {
		t.Errorf("cards must be gone, %d open", n)
	}
}

// agenten-cli-proxy AC-6: without acknowledging the cloud notice no session starts.
func TestCLICloudNoticeRequired(t *testing.T) {
	testSpecs(t, CLISpec{ID: "fake", Binary: "sh", Cloud: true, Start: func(SessionConfig) (CLIStart, error) {
		return CLIStart{Argv: []string{"sh", "-c", "sleep 5"}}, nil
	}})
	testEvidence(t, CLIEvidence{CLI: "fake", Status: EvidenceProven, ProxyProven: true, Source: "test"})

	sessions := NewCLISessions()
	m := NewCLIManager(&CLIProxy{Approvals: NewApprovals(time.Second), Sessions: sessions}, t.TempDir())
	m.LookPath = func(string) (string, error) { return "/bin/sh", nil }

	if _, err := m.Start("fake", "hallo", false); !errors.Is(err, ErrDisclosure) {
		t.Fatalf("start without disclosure must fail with ErrDisclosure, got %v", err)
	}
	if len(sessions.byToken) != 0 {
		t.Error("a refused start must not open a session")
	}
	run, err := m.Start("fake", "hallo", true)
	if err != nil {
		t.Fatalf("start with acknowledgement failed: %v", err)
	}
	m.CloseAll()
	select {
	case <-run.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("CLI child outlived CloseAll")
	}
}

// agenten-cli-proxy AC-7: the start lines carry the proven write lock and never a bypass flag; codewhale gets no
// --allowed-tools because that allowlist also removes the discovery path.
func TestCLIStartFlags(t *testing.T) {
	claude, err := claudeStart(SessionConfig{ProxyURL: "http://127.0.0.1:1/mcp", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	defer claude.Cleanup()
	if !contains(claude.Argv, "--allowedTools") || !contains(claude.Argv, "--mcp-config") {
		t.Errorf("claude must restrict tools and load the proxy config: %v", claude.Argv)
	}

	cw, err := codewhaleStart(SessionConfig{ProxyURL: "http://127.0.0.1:1/mcp", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(cw.Argv, "read-only") || !contains(cw.Argv, "--sandbox") {
		t.Errorf("codewhale must keep the OS write lock: %v", cw.Argv)
	}
	if !contains(cw.Argv, "--parent-death-watch") {
		t.Errorf("codewhale must end with its parent: %v", cw.Argv)
	}
	if contains(cw.Argv, "--allowed-tools") {
		t.Errorf("codewhale must not use --allowed-tools (it removes tool_search): %v", cw.Argv)
	}
	for name, start := range map[string]CLIStart{"claude": claude, "codewhale": cw} {
		if bad := ForbiddenFlag(start.Argv); bad != "" {
			t.Errorf("%s start line carries %q", name, bad)
		}
	}
}

// agenten-cli-proxy AC-8: a CLI without a valid entry is "unproven"; an entry without a source is no entry.
func TestCLIEvidenceRequiresSource(t *testing.T) {
	testEvidence(t, CLIEvidence{CLI: "without-source", Status: EvidenceProven})
	if e := EvidenceForCLI("without-source"); e.Status != EvidenceUnproven {
		t.Errorf("entry without source must not be used, got %+v", e)
	}
	if e := EvidenceForCLI("unknown"); e.Status != EvidenceUnproven {
		t.Errorf("unknown CLI must be unproven, got %+v", e)
	}

	testEvidence(t, CLIEvidence{CLI: "proven", Status: EvidenceProven, ProxyProven: true, Source: "test"})
	if e := EvidenceForCLI("proven"); e.Status != EvidenceProven || e.Source == "" {
		t.Errorf("valid entry must be returned with its source, got %+v", e)
	}
}

// agenten-cli-proxy AC-9: the README limits its promise to what holds for a hosted CLI and names the CLI mode.
func TestReadmeLimitsThePromise(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"kein Dateizugriff an `sdd` vorbei",
		"Agenten-CLIs als Antrieb",
		"eigene** Werkzeuge",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README must contain %q", want)
		}
	}
	if strings.Contains(text, "- **Nur die Werkzeuge von `sdd mcp`**, keine eigenen, kein Dateizugriff daran vorbei.") {
		t.Error("README still carries the unrestricted promise")
	}
}

// agenten-cli-proxy AC-10: two sessions stay separate; a session allowance of one does not reach the other.
func TestProxyTwoSessionsStaySeparate(t *testing.T) {
	runner := &fakeRunner{}
	sessions := NewCLISessions()
	p := &Proxy{
		Approvals: NewApprovals(2 * time.Second),
		Sessions:  sessions,
		Tools:     func() ([]ToolDef, error) { return proxyTestTools(), nil },
		CallTool:  runner.CallTool,
	}
	_, tokA, _ := sessions.Open("claude")
	_, tokB, _ := sessions.Open("codewhale")

	doneA := make(chan struct{})
	go func() {
		doRPC(p, tokA, "tools/call", map[string]any{"name": "sdd_next", "arguments": map[string]any{}}, 1)
		close(doneA)
	}()
	doneB := make(chan struct{})
	go func() {
		doRPC(p, tokB, "tools/call", map[string]any{"name": "sdd_next", "arguments": map[string]any{}}, 2)
		close(doneB)
	}()
	cards := waitCards(t, p, 2)
	if cards[0].Session == cards[1].Session {
		t.Fatal("two sessions must have distinct ids")
	}
	// Allow the tool for session A only.
	var cardA CallView
	for _, c := range cards {
		if c.CLI == "claude" {
			cardA = c
		}
	}
	if err := p.Approvals.Decide(cardA.CallID, Verdict{Allow: true, Scope: ScopeToolSession}); err != nil {
		t.Fatal(err)
	}
	<-doneA
	if n := p.Approvals.CountOpen(); n != 1 {
		t.Errorf("the other session's card must stay open, %d open", n)
	}
	var cardB CallView
	for _, c := range p.Approvals.Open() {
		cardB = c
	}
	if cardB.Session == cardA.Session {
		t.Error("remaining card belongs to the wrong session")
	}
	if err := p.Approvals.Decide(cardB.CallID, Verdict{Allow: true, Scope: ScopeCall}); err != nil {
		t.Fatal(err)
	}
	<-doneB
	if runner.count() != 2 {
		t.Errorf("both approved calls must run, count=%d", runner.count())
	}
}

// The proxy refuses a request without a valid session token.
func TestProxyRequiresSessionToken(t *testing.T) {
	p, _ := newTestProxy(t, &fakeRunner{})
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	r := httptest.NewRequest(http.MethodPost, "/mcp", body)
	r.Header.Set("Authorization", "Bearer falsch")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unknown token must be 401, got %d", w.Code)
	}
}

// The proxy answers tools/list from sdd mcp and never invents a catalog.
func TestProxyToolsListComesFromSDD(t *testing.T) {
	p, token := newTestProxy(t, &fakeRunner{})
	w := doRPC(p, token, "tools/list", map[string]any{}, 3)
	if w.Code != http.StatusOK {
		t.Fatalf("tools/list failed: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sdd_write") {
		t.Errorf("tools/list must carry the sdd tools, got %s", w.Body.String())
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// A session allowance covers later calls of the same tool without another card.
func TestSessionAllowanceCoversLaterCalls(t *testing.T) {
	runner := &fakeRunner{}
	p, token := newTestProxy(t, runner)

	done := make(chan struct{})
	go func() {
		doRPC(p, token, "tools/call", map[string]any{"name": "sdd_write", "arguments": map[string]any{"apply": true}}, 1)
		close(done)
	}()
	cards := waitCards(t, p, 1)
	if err := p.Approvals.Decide(cards[0].CallID, Verdict{Allow: true, Scope: ScopeWriteSession}); err != nil {
		t.Fatal(err)
	}
	<-done

	// Second call of the same tool runs without a card.
	w := doRPC(p, token, "tools/call", map[string]any{"name": "sdd_write", "arguments": map[string]any{"apply": true}}, 2)
	if runner.count() != 2 {
		t.Fatalf("session allowance must cover the later call, count=%d", runner.count())
	}
	if !strings.Contains(w.Body.String(), "ok:sdd_write") {
		t.Errorf("unexpected response %s", w.Body.String())
	}

	// A critical call is never covered by that allowance.
	doneCrit := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		doneCrit <- doRPC(p, token, "tools/call", map[string]any{"name": "sdd_baseline", "arguments": map[string]any{"apply": true}}, 3)
	}()
	crit := waitCards(t, p, 1)
	_ = p.Approvals.Decide(crit[0].CallID, Verdict{Allow: false, Scope: ScopeCall})
	<-doneCrit
}
