// Package mcp implements `sdd mcp`: a Model Context Protocol server over stdio (JSON-RPC 2.0, one message per line).
// It is a narrow, dependency-free implementation of the methods sdd needs (ADR 0002): it exposes the library as tools,
// resources and prompts, contains no LLM code and opens no network connection.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// SupportedVersions are the protocol revisions this server speaks, newest first.
var SupportedVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// maxLine bounds one incoming message.
const maxLine = 16 << 20

// Server serves one repository.
type Server struct {
	Root       string
	AllowWrite bool
	Version    string
	Now        func() time.Time
	Log        io.Writer

	mu  sync.Mutex
	out io.Writer
}

// New creates a server for root.
func New(root string, allowWrite bool, version string, log io.Writer) *Server {
	return &Server{Root: root, AllowWrite: allowWrite, Version: version, Now: time.Now, Log: log}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve reads messages from r until EOF and writes responses to w. Requests are answered in order.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	s.out = w
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), maxLine)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handle(line)
	}
	return sc.Err()
}

func (s *Server) handle(line []byte) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.send(rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParseError, "ungültiges JSON: " + err.Error()}})
		return
	}
	isRequest := len(req.ID) > 0 && string(req.ID) != "null"
	if req.Method == "" {
		if !isRequest {
			return // a response from the client (we send no requests) or garbage without id
		}
		s.send(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{codeInvalidRequest, "method fehlt"}})
		return
	}
	if !isRequest {
		return // notifications (initialized, cancelled, …) need no answer
	}
	if req.JSONRPC != "2.0" {
		s.send(rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{codeInvalidRequest, "jsonrpc muss \"2.0\" sein"}})
		return
	}
	result, rerr := s.dispatch(req.Method, req.Params)
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rerr != nil {
		resp.Error = rerr
	} else {
		resp.Result = result
	}
	s.send(resp)
}

func (s *Server) send(resp rpcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		data, _ = json.Marshal(rpcResponse{JSONRPC: "2.0", ID: resp.ID, Error: &rpcError{codeInternalError, err.Error()}})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.out.Write(append(data, '\n'))
}

func (s *Server) logf(format string, args ...any) {
	if s.Log != nil {
		fmt.Fprintf(s.Log, "sdd mcp: "+format+"\n", args...)
	}
}

func params[T any](raw json.RawMessage) (T, *rpcError) {
	var v T
	if len(raw) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, &rpcError{codeInvalidParams, "ungültige Parameter: " + err.Error()}
	}
	return v, nil
}

func (s *Server) dispatch(method string, raw json.RawMessage) (any, *rpcError) {
	switch method {
	case "initialize":
		p, err := params[struct {
			ProtocolVersion string `json:"protocolVersion"`
		}](raw)
		if err != nil {
			return nil, err
		}
		version := SupportedVersions[0]
		for _, v := range SupportedVersions {
			if v == p.ProtocolVersion {
				version = v
			}
		}
		s.logf("initialisiert (Protokoll %s, Schreiben %v)", version, s.AllowWrite)
		return map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}, "resources": map[string]any{}, "prompts": map[string]any{}},
			"serverInfo":      map[string]any{"name": "sdd", "title": "SDD Studio", "version": s.Version},
			"instructions":    instructions,
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolList()}, nil
	case "tools/call":
		p, err := params[struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}](raw)
		if err != nil {
			return nil, err
		}
		return s.callTool(p.Name, p.Arguments)
	case "resources/list":
		return s.listResources()
	case "resources/read":
		p, err := params[struct {
			URI string `json:"uri"`
		}](raw)
		if err != nil {
			return nil, err
		}
		return s.readResource(p.URI)
	case "prompts/list":
		return map[string]any{"prompts": promptList()}, nil
	case "prompts/get":
		p, err := params[struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}](raw)
		if err != nil {
			return nil, err
		}
		return s.getPrompt(p.Name, p.Arguments)
	default:
		return nil, &rpcError{codeMethodNotFound, "unbekannte Methode " + method}
	}
}

const instructions = `sdd prüft Spec-Driven Development deterministisch. Ablauf: vor einer Änderung sdd_check_intent und sdd_context; ` +
	`für den Prozessstand sdd_next; Entwürfe vor dem Schreiben mit sdd_validate_draft prüfen; danach sdd_lint bzw. sdd_gate. ` +
	`Baseline, ADR-Lockfile, adopt --apply und das Entscheiden von ADRs sind dem Menschen vorbehalten.`
