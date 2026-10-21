package desk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// probe is a model that records each attempt and can refuse requests with tools.
type probe struct {
	refuseTools, refuseAll bool
	reply                  Message
	systems                []string
	seen                   [][]Message
	tools                  [][]ToolDef
}

func (p *probe) Complete(_ context.Context, _, system string, msgs []Message, tools []ToolDef) (Message, error) {
	p.systems = append(p.systems, system)
	p.seen = append(p.seen, msgs)
	p.tools = append(p.tools, tools)
	switch {
	case p.refuseAll:
		return Message{}, errors.New("HTTP 404: No endpoints found for x/y.")
	case p.refuseTools && len(tools) > 0:
		return Message{}, errors.New("HTTP 404: No endpoints found that support tool use.")
	}
	return p.reply, nil
}

func probeAgent(p *probe) *Agent {
	return &Agent{Tools: testTools, System: "Systemtext mit Projektbezug",
		Providers: []Provider{{ID: LocalOllamaID, Protocol: ProtocolOllama}, {ID: "claude", Protocol: ProtocolClaude, Cloud: true}},
		NewChat:   func(Provider) Chat { return p }}
}

// sdd-desk-model-test AC-1: only the fixed test message and the sdd tool definitions are sent; the result has the reply and the duration.
func TestModelTestSendsOnlyFixedText(t *testing.T) {
	p := &probe{reply: Message{Role: "assistant", Content: "Hello World" + strings.Repeat("!", 300)}}
	res, err := probeAgent(p).TestModel(context.Background(), ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.seen) != 1 || len(p.seen[0]) != 1 || p.seen[0][0].Role != "user" || p.seen[0][0].Content != modelTestPrompt {
		t.Fatalf("messages sent: %+v", p.seen)
	}
	if p.systems[0] != modelTestSystem || len(p.tools[0]) != len(testTools) {
		t.Errorf("system %q, tools %+v", p.systems[0], p.tools[0])
	}
	if !res.OK || res.Tools == nil || !*res.Tools || res.Error != "" || !strings.HasPrefix(res.Reply, "Hello World") || len([]rune(res.Reply)) != 201 {
		t.Errorf("result %+v", res)
	}

	p = &probe{reply: call("c1", "sdd_new", `{"apply":true}`)}
	res, _ = probeAgent(p).TestModel(context.Background(), ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if !res.OK || !strings.Contains(res.Reply, "sdd_new") {
		t.Errorf("tool call in the answer must be named, not run: %+v", res)
	}
}

// sdd-desk-model-test AC-2: a refusal with tools is retried without tools; both outcomes are reported separately.
func TestModelTestWithoutTools(t *testing.T) {
	p := &probe{refuseTools: true, reply: Message{Role: "assistant", Content: "Hello World"}}
	res, err := probeAgent(p).TestModel(context.Background(), ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.tools) != 2 || len(p.tools[1]) != 0 {
		t.Fatalf("want a second attempt without tools, got %d attempts", len(p.tools))
	}
	if !res.OK || res.Tools == nil || *res.Tools || !strings.Contains(res.Error, "tool use") || res.ErrorPlain != "" || res.Reply != "Hello World" {
		t.Errorf("result %+v", res)
	}

	p = &probe{refuseAll: true}
	res, _ = probeAgent(p).TestModel(context.Background(), ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if res.OK || res.Tools != nil || res.Error == "" || res.ErrorPlain == "" {
		t.Errorf("model not reachable: %+v", res)
	}
	data, _ := json.Marshal(res)
	if strings.Contains(string(data), `"tools"`) {
		t.Errorf("tools must be absent when no attempt succeeded: %s", data)
	}
}

// sdd-desk-model-test AC-3: disclosure, unknown provider and missing model are refused before anything is sent; the key never appears.
func TestModelTestEndpointRefuses(t *testing.T) {
	p := &probe{reply: Message{Role: "assistant", Content: "Hello World"}}
	s, h := testServer(t)
	s.Agent.NewChat = func(Provider) Chat { return p }

	for body, want := range map[string]int{
		`{"provider":"claude","model":"m"}`:                          http.StatusPreconditionRequired,
		`{"provider":"unbekannt","model":"m","disclosure_ack":true}`: http.StatusBadRequest,
		`{"provider":"claude","disclosure_ack":true}`:                http.StatusBadRequest,
		`kein json`: http.StatusBadRequest,
	} {
		if rec := request(t, h, http.MethodPost, "/api/model/test", body, auth); rec.Code != want {
			t.Errorf("%s: status %d, want %d", body, rec.Code, want)
		}
	}
	if len(p.seen) != 0 {
		t.Fatalf("refused tests must not reach the model: %d calls", len(p.seen))
	}
	if rec := request(t, h, http.MethodPost, "/api/model/test", `{"provider":"claude","model":"m","disclosure_ack":true}`, map[string]string{"Content-Type": "application/json"}); rec.Code != http.StatusForbidden {
		t.Errorf("no token: %d", rec.Code)
	}

	rec := request(t, h, http.MethodPost, "/api/model/test", `{"provider":"claude","model":"m","disclosure_ack":true}`, auth)
	var res ModelTestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); rec.Code != 200 || err != nil || !res.OK {
		t.Fatalf("test: %d %s", rec.Code, rec.Body)
	}

	// A real provider that echoes the request: the key is sent as a header and must not come back in the result.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	s.Agent.Providers[0].Endpoint = srv.URL
	s.Agent.NewChat = func(p Provider) Chat { return NewChat(p, srv.Client()) }
	rec = request(t, h, http.MethodPost, "/api/model/test", `{"provider":"claude","model":"m","disclosure_ack":true}`, auth)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "HTTP 404") || strings.Contains(rec.Body.String(), "geheim") {
		t.Errorf("provider error: %d %s", rec.Code, rec.Body)
	}
}

// slow is a model that answers only when its context ends.
type slow struct{ attempts int }

func (s *slow) Complete(ctx context.Context, _, _ string, _ []Message, _ []ToolDef) (Message, error) {
	s.attempts++
	<-ctx.Done()
	return Message{}, ctx.Err()
}

// sdd-desk-model-test AC-4: an attempt ends after the timeout as a timeout, not as a refusal of the tools; a canceled test also skips the second attempt.
func TestModelTestTimeout(t *testing.T) {
	old := modelTestTimeout
	modelTestTimeout = 50 * time.Millisecond
	defer func() { modelTestTimeout = old }()

	m := &slow{}
	a := probeAgent(nil)
	a.NewChat = func(Provider) Chat { return m }
	start := time.Now()
	res, err := a.TestModel(context.Background(), ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if err != nil || res.OK || !res.Timeout || res.Tools != nil || !strings.Contains(res.Error, "deadline") || res.ErrorPlain != "" || m.attempts != 1 {
		t.Fatalf("result %+v, attempts %d, err %v", res, m.attempts, err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %v", d)
	}

	m = &slow{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, _ = a.TestModel(ctx, ModelTestRequest{Provider: LocalOllamaID, Model: "m"})
	if res.OK || res.Timeout || m.attempts != 1 {
		t.Errorf("canceled: %+v, attempts %d", res, m.attempts)
	}
}
