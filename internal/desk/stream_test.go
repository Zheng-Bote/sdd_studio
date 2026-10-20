package desk

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// streamServer answers every request with the given lines, flushing after each, and records the request body.
func streamServer(t *testing.T, lines ...string) (*httptest.Server, *[]byte) {
	t.Helper()
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		for _, l := range lines {
			_, _ = io.WriteString(w, l+"\n")
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &body
}

var lastThinking string

func streamOf(t *testing.T, p Provider, srv *httptest.Server) (Message, []string, error) {
	t.Helper()
	p.Endpoint, p.APIKey = srv.URL, "k"
	var pieces []string
	lastThinking = ""
	msg, err := NewChat(p, srv.Client()).(Streamer).Stream(context.Background(), "m", "SYS", testMsgs, testTools, Deltas{
		Text:     func(s string) { pieces = append(pieces, s) },
		Thinking: func(s string) { lastThinking += s },
	})
	return msg, pieces, err
}

// sdd-desk-streaming AC-1: every protocol streams text pieces and yields the same message as a non-streamed answer.
// sdd-desk-streaming AC-6: reasoning arrives separately (thinking, reasoning, thinking_delta) and stays out of the message.
func TestStreamProviders(t *testing.T) {
	t.Run("ollama", func(t *testing.T) {
		srv, body := streamServer(t,
			`{"message":{"role":"assistant","content":"","thinking":"Der Nutzer "},"done":false}`,
			`{"message":{"role":"assistant","content":"","thinking":"grüßt."},"done":false}`,
			`{"message":{"role":"assistant","content":"Hal"},"done":false}`,
			`{"message":{"role":"assistant","content":"lo"},"done":false}`,
			`{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"sdd_lint","arguments":{"gate":"pre-merge"}}}]},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true}`,
		)
		got, pieces, err := streamOf(t, Provider{Name: "o", Protocol: ProtocolOllama}, srv)
		if err != nil {
			t.Fatal(err)
		}
		want := Message{Role: "assistant", Content: "Hallo", ToolCalls: []ToolCall{{Name: "sdd_lint", Arguments: json.RawMessage(`{"gate":"pre-merge"}`)}}}
		if !reflect.DeepEqual(got, want) || strings.Join(pieces, "") != "Hallo" || len(pieces) != 2 {
			t.Errorf("got %+v pieces %q", got, pieces)
		}
		if lastThinking != "Der Nutzer grüßt." {
			t.Errorf("thinking %q", lastThinking)
		}
		if !strings.Contains(string(*body), `"stream":true`) {
			t.Errorf("body %s", *body)
		}
	})

	t.Run("openai compatible", func(t *testing.T) {
		srv, body := streamServer(t,
			`data: {"choices":[{"delta":{"role":"assistant","content":"","reasoning":"Erst "}}]}`, ``,
			`data: {"choices":[{"delta":{"reasoning_content":"denken."}}]}`, ``,
			`data: {"choices":[{"delta":{"role":"assistant","content":"Gu"}}]}`, ``,
			`data: {"choices":[{"delta":{"content":"t."}}]}`, ``,
			`: keep-alive`,
			`data: {"choices":[{"delta":{"content":null,"tool_calls":[{"index":0,"id":"x1","type":"function","function":{"name":"sdd_next","arguments":"{\"fea"}}]}}]}`, ``,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ture\":\"a\"}"}}]}}]}`, ``,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"x2","type":"function","function":{"name":"sdd_lint","arguments":""}}]}}]}`, ``,
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`, ``,
			`data: [DONE]`,
		)
		got, pieces, err := streamOf(t, Provider{Name: "or", Protocol: ProtocolOpenAI}, srv)
		if err != nil {
			t.Fatal(err)
		}
		want := Message{Role: "assistant", Content: "Gut.", ToolCalls: []ToolCall{
			{ID: "x1", Name: "sdd_next", Arguments: json.RawMessage(`{"feature":"a"}`)},
			{ID: "x2", Name: "sdd_lint", Arguments: json.RawMessage(`{}`)},
		}}
		if !reflect.DeepEqual(got, want) || strings.Join(pieces, "") != "Gut." || lastThinking != "Erst denken." {
			t.Errorf("got %+v pieces %q thinking %q", got, pieces, lastThinking)
		}
		if !strings.Contains(string(*body), `"stream":true`) {
			t.Errorf("body %s", *body)
		}
	})

	t.Run("complete calls without index", func(t *testing.T) {
		srv, _ := streamServer(t,
			`data: {"choices":[{"delta":{"tool_calls":[{"id":"g1","function":{"name":"sdd_next","arguments":"{}"}},{"id":"g2","function":{"name":"sdd_lint","arguments":"{}"}}]}}]}`,
			`data: [DONE]`,
		)
		got, _, err := streamOf(t, Provider{Name: "g", Protocol: ProtocolOpenAI}, srv)
		if err != nil || len(got.ToolCalls) != 2 || got.ToolCalls[1].ID != "g2" {
			t.Errorf("got %+v %v", got, err)
		}
	})

	t.Run("claude", func(t *testing.T) {
		srv, body := streamServer(t,
			`event: message_start`, `data: {"type":"message_start","message":{"id":"m1","role":"assistant","content":[]}}`, ``,
			`event: content_block_start`, `data: {"type":"content_block_start","index":9,"content_block":{"type":"thinking","thinking":""}}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":9,"delta":{"type":"thinking_delta","thinking":"Plan."}}`, ``,
			`event: content_block_start`, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`, ``,
			`event: ping`, `data: {"type":"ping"}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Gern"}}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"."}}`, ``,
			`event: content_block_stop`, `data: {"type":"content_block_stop","index":0}`, ``,
			`event: content_block_start`, `data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tu1","name":"sdd_next","input":{}}}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"feat"}}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"ure\":\"b\"}"}}`, ``,
			`event: content_block_stop`, `data: {"type":"content_block_stop","index":1}`, ``,
			`event: content_block_start`, `data: {"type":"content_block_start","index":2,"content_block":{"type":"text","text":""}}`, ``,
			`event: content_block_delta`, `data: {"type":"content_block_delta","index":2,"delta":{"type":"text_delta","text":"Noch was"}}`, ``,
			`event: message_delta`, `data: {"type":"message_delta","delta":{"stop_reason":"tool_use"}}`, ``,
			`event: message_stop`, `data: {"type":"message_stop"}`, ``,
		)
		got, pieces, err := streamOf(t, Provider{Name: "c", Protocol: ProtocolClaude}, srv)
		if err != nil {
			t.Fatal(err)
		}
		want := Message{Role: "assistant", Content: "Gern.\n\nNoch was", ToolCalls: []ToolCall{{ID: "tu1", Name: "sdd_next", Arguments: json.RawMessage(`{"feature":"b"}`)}}}
		if !reflect.DeepEqual(got, want) || strings.Join(pieces, "") != want.Content || lastThinking != "Plan." {
			t.Errorf("got %+v pieces %q thinking %q", got, pieces, lastThinking)
		}
		if !strings.Contains(string(*body), `"stream":true`) {
			t.Errorf("body %s", *body)
		}
	})

	t.Run("errors in the stream", func(t *testing.T) {
		cases := []struct {
			p    Provider
			line string
			want string
		}{
			{Provider{Name: "o", Protocol: ProtocolOllama}, `{"error":"model not found"}`, "model not found"},
			{Provider{Name: "or", Protocol: ProtocolOpenAI}, `data: {"error":{"message":"rate limit","code":429}}`, "rate limit"},
			{Provider{Name: "c", Protocol: ProtocolClaude}, `data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`, "Overloaded"},
		}
		for _, c := range cases {
			srv, _ := streamServer(t, c.line)
			if _, _, err := streamOf(t, c.p, srv); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("%s: want %q, got %v", c.p.Name, c.want, err)
			}
		}
		srv, _ := streamServer(t,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t","name":"sdd_next"}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"a\""}}`,
			`data: {"type":"message_stop"}`,
		)
		if _, _, err := streamOf(t, Provider{Name: "c", Protocol: ProtocolClaude}, srv); err == nil || !strings.Contains(err.Error(), "unvollständige") {
			t.Errorf("truncated arguments: %v", err)
		}
		bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "kein Schlüssel", 401) }))
		defer bad.Close()
		if _, _, err := streamOf(t, Provider{Name: "or", Protocol: ProtocolOpenAI}, bad); err == nil || !strings.Contains(err.Error(), "401") {
			t.Errorf("HTTP error: %v", err)
		}
	})
}

// streaming wraps a scripted model and delivers each reply's text in pieces of up to four bytes.
type streaming struct{ *scripted }

func (s streaming) Stream(ctx context.Context, model, system string, msgs []Message, tools []ToolDef, out Deltas) (Message, error) {
	m, err := s.Complete(ctx, model, system, msgs, tools)
	out.thinking("…")
	for rest := m.Content; rest != ""; {
		n := min(4, len(rest))
		out.text(rest[:n])
		rest = rest[n:]
	}
	return m, err
}

func collect(events *[]Event) func(Event) {
	return func(e Event) { *events = append(*events, e) }
}

// sdd-desk-streaming AC-2: a streamed turn reports text pieces and every finished message, and ends like Turn.
func TestTurnStreamEvents(t *testing.T) {
	root := sddProject(t)
	reply := Message{Role: "assistant", Content: "Ich schaue nach.", ToolCalls: []ToolCall{{ID: "c1", Name: "sdd_next", Arguments: json.RawMessage(`{}`)}}}
	req := TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: []Message{{Role: "user", Content: "Was nun?"}}, AllowedTools: []string{"sdd_next"}}

	a := newAgent(t, root, &scripted{})
	a.NewChat = func(Provider) Chat { return streaming{&scripted{replies: []Message{reply}}} }
	var events []Event
	res, err := a.TurnStream(context.Background(), req, collect(&events))
	if err != nil {
		t.Fatal(err)
	}

	var kinds []string
	var text strings.Builder
	for _, e := range events {
		switch e.Type {
		case "delta":
			text.WriteString(e.Text)
			if len(kinds) == 0 || kinds[len(kinds)-1] != "delta" {
				kinds = append(kinds, "delta")
			}
		case "thinking":
			kinds = append(kinds, "thinking")
		case "message":
			kinds = append(kinds, "message:"+e.Message.Role)
		}
	}
	if strings.Join(kinds, " ") != "thinking delta message:assistant message:tool thinking delta message:assistant" {
		t.Errorf("event order %v", kinds)
	}
	if text.String() != "Ich schaue nach.fertig" {
		t.Errorf("streamed text %q", text.String())
	}

	b := newAgent(t, root, &scripted{replies: []Message{reply}})
	plain, err := b.Turn(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res, plain) {
		t.Errorf("streamed turn differs from Turn:\n%+v\n%+v", res, plain)
	}
}

// sdd-desk-streaming AC-3: streaming changes nothing about permissions; writes wait with preview, clouds need the disclosure.
func TestTurnStreamKeepsApprovals(t *testing.T) {
	root := sddProject(t)
	inner := &scripted{replies: []Message{call("w1", "sdd_new", `{"feature":"billing","apply":true}`)}}
	a := newAgent(t, root, inner)
	a.NewChat = func(Provider) Chat { return streaming{inner} }
	base := []Message{{Role: "user", Content: "Leg das PRD an."}}

	var events []Event
	res, err := a.TurnStream(context.Background(), TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: base, AllowedTools: []string{"sdd_new"}}, collect(&events))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pending) != 1 || !res.Pending[0].Writes || !strings.Contains(string(res.Pending[0].Preview), "feature: billing") {
		t.Fatalf("write must wait with preview: %+v", res)
	}
	for _, e := range events {
		if e.Type == "message" && e.Message.Role == "tool" {
			t.Errorf("tool ran without approval: %+v", e.Message)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err == nil {
		t.Fatal("write without approval")
	}

	events = nil
	_, err = a.TurnStream(context.Background(), TurnRequest{Provider: "claude", Model: "m", Messages: base}, collect(&events))
	if !errors.Is(err, ErrDisclosure) || len(events) != 0 || inner.calls != 1 {
		t.Errorf("cloud without disclosure: %v, %d events, %d model calls", err, len(events), inner.calls)
	}
}

// sdd-desk-streaming AC-2: the endpoint sends NDJSON events ending with done; checks fail with a status before streaming.
func TestTurnStreamEndpoint(t *testing.T) {
	s, h := testServer(t)
	s.Agent.NewChat = func(Provider) Chat {
		return streaming{&scripted{replies: []Message{{Role: "assistant", Content: "Hallo zusammen"}}}}
	}

	rec := request(t, h, http.MethodPost, "/api/turn/stream", `{"provider":"claude","model":"m","messages":[{"role":"user","content":"x"}]}`, auth)
	if rec.Code != http.StatusPreconditionRequired || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("without disclosure: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := request(t, h, http.MethodPost, "/api/turn/stream", `{}`, map[string]string{"Content-Type": "application/json"}); rec.Code != http.StatusForbidden {
		t.Errorf("no token: %d", rec.Code)
	}

	rec = request(t, h, http.MethodPost, "/api/turn/stream", `{"provider":"claude","model":"m","disclosure_ack":true,"messages":[{"role":"user","content":"x"}]}`, auth)
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/x-ndjson") {
		t.Fatalf("stream: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	var events []Event
	sc := bufio.NewScanner(rec.Body)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("line %q: %v", sc.Text(), err)
		}
		events = append(events, e)
	}
	last := events[len(events)-1]
	if events[0].Type != "thinking" || events[1].Type != "delta" || last.Type != "done" || len(last.Messages) != 2 || last.Messages[1].Content != "Hallo zusammen" {
		t.Errorf("events %+v", events)
	}
	if strings.Contains(rec.Body.String(), "geheim") {
		t.Error("key in stream")
	}
}

// sdd-desk-streaming AC-4: aborting the turn aborts the request to the provider.
func TestTurnStreamCancel(t *testing.T) {
	canceled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"message":{"role":"assistant","content":"Ein langer "},"done":false}`+"\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-time.After(3 * time.Second): // never block srv.Close if the abort does not arrive
		}
	}))
	defer srv.Close()

	a := newAgent(t, sddProject(t), &scripted{})
	a.Providers = []Provider{{ID: LocalOllamaID, Name: "lokal", Protocol: ProtocolOllama, Endpoint: srv.URL}}
	a.NewChat = func(p Provider) Chat { return NewChat(p, srv.Client()) }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := a.TurnStream(ctx, TurnRequest{Provider: LocalOllamaID, Model: "m", Messages: []Message{{Role: "user", Content: "x"}}}, func(e Event) {
			if e.Type == "delta" {
				cancel()
			}
		})
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want context.Canceled, got %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("turn did not end after cancel")
	}
	select {
	case <-canceled:
	case <-time.After(4 * time.Second):
		t.Fatal("provider request was not aborted")
	}
}
