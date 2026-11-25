package desk

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CLISessions hands out one bearer token per CLI run. A session knows which CLI it belongs to, so every card can
// name it (agenten-cli-proxy AC-3, AC-10).
type CLISessions struct {
	mu      sync.Mutex
	byToken map[string]sessionEntry
}

type sessionEntry struct {
	session string
	cli     string
}

// NewCLISessions returns an empty registry.
func NewCLISessions() *CLISessions {
	return &CLISessions{byToken: map[string]sessionEntry{}}
}

// Open starts a session for one CLI and returns its id and token.
func (s *CLISessions) Open(cli string) (session, token string, err error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	b2 := make([]byte, 16)
	if _, err := rand.Read(b2); err != nil {
		return "", "", err
	}
	session = "s-" + hex.EncodeToString(b)
	token = hex.EncodeToString(b2)
	s.mu.Lock()
	s.byToken[token] = sessionEntry{session: session, cli: cli}
	s.mu.Unlock()
	return session, token, nil
}

// Lookup maps a bearer token to its session and CLI. The comparison is constant time.
func (s *CLISessions) Lookup(token string) (session, cli string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, e := range s.byToken {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			return e.session, e.cli, true
		}
	}
	return "", "", false
}

// Close ends a session: its token stops working.
func (s *CLISessions) Close(session string) {
	s.mu.Lock()
	for t, e := range s.byToken {
		if e.session == session {
			delete(s.byToken, t)
		}
	}
	s.mu.Unlock()
}

// Proxy serves the MCP endpoint an agent CLI talks to. It carries no tool catalog of its own: tools/list and
// tools/call go to the running `sdd mcp` through the same MCPClient the desktop's own loop uses (ADR 0010,
// decision 3 and consequence "the catalog stays one source").
type Proxy struct {
	Approvals *Approvals
	Sessions  *CLISessions
	Tools     func() ([]ToolDef, error)
	// Preview produces the plan of a write or critical call without applying it; summary is the one-line effect.
	Preview  func(name string, args json.RawMessage) (json.RawMessage, string, error)
	CallTool func(name string, args json.RawMessage) (*ToolResult, error)
	Now      func() time.Time
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// ServeHTTP answers one JSON-RPC request of one CLI session. It never runs a tool without a decision.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "nur POST", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || p.Sessions == nil {
		http.Error(w, "Token fehlt", http.StatusUnauthorized)
		return
	}
	session, cli, ok := p.Sessions.Lookup(token)
	if !ok {
		http.Error(w, "Token unbekannt", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, "Anfrage nicht lesbar", http.StatusBadRequest)
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(w, nil, -32700, "Parse error")
		return
	}

	switch req.Method {
	case "initialize":
		writeRPCResult(w, req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "sdd-desk", "version": "0.11.1"},
		})
	case "notifications/initialized", "notifications/cancelled":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		p.handleList(w, req)
	case "tools/call":
		p.handleCall(w, r, req, session, cli)
	default:
		writeRPCError(w, req.ID, -32601, "Methode unbekannt: "+req.Method)
	}
}

func (p *Proxy) handleList(w http.ResponseWriter, req rpcRequest) {
	defs, err := p.Tools()
	if err != nil {
		writeRPCError(w, req.ID, -32603, err.Error())
		return
	}
	tools := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, map[string]any{
			"name": d.Name, "description": d.Description, "inputSchema": d.Schema,
		})
	}
	writeRPCResult(w, req.ID, map[string]any{"tools": tools})
}

// handleCall parks the call, shows the card and runs only what was allowed. A call that nobody decides is refused;
// it never runs (agenten-cli-proxy AC-2, AC-4).
func (p *Proxy) handleCall(w http.ResponseWriter, r *http.Request, req rpcRequest, session, cli string) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPCError(w, req.ID, -32602, "Parameter unlesbar")
		return
	}
	defs, err := p.Tools()
	if err != nil {
		writeRPCError(w, req.ID, -32603, err.Error())
		return
	}
	known := false
	for _, d := range defs {
		if d.Name == params.Name {
			known = true
			break
		}
	}
	if !known {
		writeRPCToolError(w, req.ID, "Abgelehnt: "+params.Name+" ist kein Werkzeug von sdd mcp.")
		return
	}

	args := argsObject(params.Arguments)
	t := tier(ToolCall{Name: params.Name, Arguments: args})
	call, granted, err := p.Approvals.Register(session, cli, params.Name, args, t)
	if err != nil {
		writeRPCError(w, req.ID, -32603, err.Error())
		return
	}
	verdict := Verdict{}
	if granted != nil {
		verdict = *granted
	} else {
		if t != tierRead && p.Preview != nil {
			preview, summary, err := p.Preview(params.Name, args)
			if err == nil {
				p.Approvals.SetSummary(call.ID, summary)
				_ = preview
			}
		}
		verdict = p.Approvals.Await(r.Context(), call)
	}
	if !verdict.Allow {
		writeRPCToolError(w, req.ID, refusalText(verdict.Reason))
		return
	}
	// A critical call only runs when exactly this call was allowed. The register refuses a session scope for it;
	// this guard keeps the rule true even if a caller builds a verdict by hand.
	if t == tierCritical && verdict.Scope != ScopeCall {
		writeRPCToolError(w, req.ID, refusalText("kritische Aktionen nur einzeln"))
		return
	}
	p.Approvals.Remove(call.ID)
	res, err := p.CallTool(params.Name, args)
	if err != nil {
		writeRPCToolError(w, req.ID, "Fehler beim Ausführen: "+err.Error())
		return
	}
	if res.IsError {
		writeRPCToolError(w, req.ID, "Werkzeugfehler: "+res.Text)
		return
	}
	out := map[string]any{
		"content": []map[string]string{{"type": "text", "text": res.Text}},
		"isError": false,
	}
	if len(res.Structured) > 0 {
		out["structuredContent"] = res.Structured
	}
	writeRPCResult(w, req.ID, out)
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": rawID(id), "result": result})
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0", "id": rawID(id),
		"error": map[string]any{"code": code, "message": msg},
	})
}

// writeRPCToolError reports a refused or failed tool call as a tool result, not as a protocol error: the CLI must
// see that the call did not happen, without treating the server as broken.
func writeRPCToolError(w http.ResponseWriter, id json.RawMessage, text string) {
	writeRPCResult(w, id, map[string]any{
		"content": []map[string]string{{"type": "text", "text": text}},
		"isError": true,
	})
}

// rawID keeps a null id when the request had none.
func rawID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

// ErrNoProxySession is returned when a decision names a session that is not open.
var ErrNoProxySession = errors.New("Sitzung nicht offen")
