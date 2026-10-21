package desk

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// MCPClient speaks JSON-RPC 2.0 over stdio with `sdd mcp` — the only way sdd-desk reaches sdd tools (ADR 0002).
type MCPClient struct {
	w            io.Writer
	mu           sync.Mutex
	nextID       int64
	pending      map[int64]chan rpcReply
	closed       chan struct{}
	err          error
	Instructions string
	cmd          *exec.Cmd
	Timeout      time.Duration
}

type rpcReply struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// StartMCP launches `<sdd> mcp --allow-write` in the project directory. Writing still needs the user's approval in
// sdd-desk; --allow-write only lets sdd_new apply an approved plan.
func StartMCP(sddPath, projectRoot string, stderr io.Writer) (*MCPClient, error) {
	cmd := exec.Command(sddPath, "mcp", "--allow-write")
	cmd.Dir = projectRoot
	cmd.Stderr = stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("sdd mcp nicht startbar (%s): %w", sddPath, err)
	}
	c := NewMCPClient(out, in)
	c.cmd = cmd
	if err := c.Initialize(); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

// NewMCPClient wraps an established stdio pair (used directly in tests).
func NewMCPClient(r io.Reader, w io.Writer) *MCPClient {
	c := &MCPClient{w: w, pending: map[int64]chan rpcReply{}, closed: make(chan struct{}), Timeout: 2 * time.Minute}
	go c.readLoop(r)
	return c
}

func (c *MCPClient) readLoop(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 32<<20)
	for sc.Scan() {
		var msg struct {
			ID json.RawMessage `json:"id"`
			rpcReply
		}
		if json.Unmarshal(sc.Bytes(), &msg) != nil {
			continue
		}
		var id int64
		if json.Unmarshal(msg.ID, &id) != nil {
			continue // notifications or foreign ids
		}
		c.mu.Lock()
		ch := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()
		if ch != nil {
			ch <- msg.rpcReply
		}
	}
	c.mu.Lock()
	c.err = errors.New("sdd mcp beendet")
	if sc.Err() != nil {
		c.err = sc.Err()
	}
	c.mu.Unlock()
	close(c.closed)
}

// Call sends a request and waits for its result.
func (c *MCPClient) Call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return nil, err
	}
	c.nextID++
	id := c.nextID
	ch := make(chan rpcReply, 1)
	c.pending[id] = ch
	data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err == nil {
		_, err = c.w.Write(append(data, '\n'))
	}
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		if r.Error != nil {
			return nil, fmt.Errorf("sdd mcp: %s (%d)", r.Error.Message, r.Error.Code)
		}
		return r.Result, nil
	case <-c.closed:
		return nil, c.err
	case <-time.After(c.Timeout):
		return nil, fmt.Errorf("sdd mcp: keine Antwort auf %s", method)
	}
}

// Initialize performs the MCP handshake and keeps the server's instructions for the system prompt.
func (c *MCPClient) Initialize() error {
	raw, err := c.Call("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "sdd-desk", "version": "0.2.0"},
	})
	if err != nil {
		return err
	}
	var res struct {
		Instructions string `json:"instructions"`
	}
	_ = json.Unmarshal(raw, &res)
	c.Instructions = res.Instructions
	data, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	c.mu.Lock()
	_, err = c.w.Write(append(data, '\n'))
	c.mu.Unlock()
	return err
}

// Tools lists the tools of sdd mcp as model tool definitions.
func (c *MCPClient) Tools() ([]ToolDef, error) {
	raw, err := c.Call("tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var res struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	out := make([]ToolDef, 0, len(res.Tools))
	for _, t := range res.Tools {
		out = append(out, ToolDef{Name: t.Name, Description: t.Description, Schema: t.InputSchema})
	}
	return out, nil
}

// ToolResult is the outcome of a tool call.
type ToolResult struct {
	Text       string          `json:"text"`
	Structured json.RawMessage `json:"structured,omitempty"`
	IsError    bool            `json:"is_error"`
}

// CallTool runs one tool of sdd mcp.
func (c *MCPClient) CallTool(name string, args json.RawMessage) (*ToolResult, error) {
	raw, err := c.Call("tools/call", map[string]any{"name": name, "arguments": argsObject(args)})
	if err != nil {
		return nil, err
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	out := &ToolResult{Structured: res.StructuredContent, IsError: res.IsError}
	for _, c := range res.Content {
		out.Text += c.Text
	}
	return out, nil
}

// Close ends the child process.
func (c *MCPClient) Close() error {
	if closer, ok := c.w.(io.Closer); ok {
		_ = closer.Close()
	}
	if c.cmd != nil {
		done := make(chan error, 1)
		go func() { done <- c.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = c.cmd.Process.Kill()
		}
	}
	return nil
}
