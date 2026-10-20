package desk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Message is one entry of the provider-neutral conversation.
type Message struct {
	Role       string     `json:"role"` // user, assistant, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
}

// ToolCall is a model's request to run a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolDef describes a tool for the model (taken from `sdd mcp` tools/list).
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"input_schema"`
}

// Chat completes one model step.
type Chat interface {
	Complete(ctx context.Context, model, system string, msgs []Message, tools []ToolDef) (Message, error)
}

// NewChat returns the client for a provider's protocol.
func NewChat(p Provider, client *http.Client) Chat {
	switch p.Protocol {
	case ProtocolClaude:
		return &claudeChat{p: p, c: client}
	case ProtocolOpenAI:
		return &openAIChat{p: p, c: client}
	default:
		return &ollamaChat{p: p, c: client}
	}
}

// post sends a JSON body and returns the response if its status is 2xx; otherwise the error carries the body's start.
func post(ctx context.Context, c *http.Client, url string, headers map[string]string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 500 {
			msg = msg[:500] + "…"
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return resp, nil
}

func postJSON(ctx context.Context, c *http.Client, url string, headers map[string]string, body, out any) error {
	resp, err := post(ctx, c, url, headers, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unerwartete Antwort: %w", err)
	}
	return nil
}

func schemaOf(t ToolDef) json.RawMessage {
	if len(t.Schema) == 0 {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return t.Schema
}

func argsObject(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

// ---------- Ollama (local and cloud) ----------

type ollamaChat struct {
	p Provider
	c *http.Client
}

type ollamaFn struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ollamaCall struct {
	Function ollamaFn `json:"function"`
}

type ollamaMsg struct {
	Role      string       `json:"role"`
	Content   string       `json:"content"`
	ToolCalls []ollamaCall `json:"tool_calls,omitempty"`
	ToolName  string       `json:"tool_name,omitempty"`
	Thinking  string       `json:"thinking,omitempty"` // only in answers; never sent back
}

// request returns headers and body of a /api/chat call.
func (o *ollamaChat) request(model, system string, msgs []Message, tools []ToolDef, stream bool) (map[string]string, map[string]any) {
	out := []ollamaMsg{{Role: "system", Content: system}}
	for _, m := range msgs {
		om := ollamaMsg{Role: m.Role, Content: m.Content, ToolName: m.ToolName}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, ollamaCall{ollamaFn{tc.Name, argsObject(tc.Arguments)}})
		}
		out = append(out, om)
	}
	var defs []map[string]any
	for _, t := range tools {
		defs = append(defs, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": schemaOf(t)}})
	}
	headers := map[string]string{}
	if o.p.APIKey != "" {
		headers["Authorization"] = "Bearer " + o.p.APIKey
	}
	return headers, map[string]any{"model": model, "messages": out, "tools": defs, "stream": stream}
}

func (o *ollamaChat) Complete(ctx context.Context, model, system string, msgs []Message, tools []ToolDef) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, false)
	var resp struct {
		Message ollamaMsg `json:"message"`
	}
	if err := postJSON(ctx, o.c, o.p.Endpoint+"/api/chat", headers, body, &resp); err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	res := Message{Role: "assistant", Content: resp.Message.Content}
	for _, tc := range resp.Message.ToolCalls {
		res.ToolCalls = append(res.ToolCalls, ToolCall{Name: tc.Function.Name, Arguments: argsObject(tc.Function.Arguments)})
	}
	return res, nil
}

// ListLocalModels asks a local Ollama instance for its models (GET /api/tags).
func ListLocalModels(ctx context.Context, c *http.Client, endpoint string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&tags); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range tags.Models {
		out = append(out, m.Name)
	}
	return out, nil
}

// ---------- OpenAI-compatible (OpenRouter, NVIDIA NIM, Gemini) ----------

type openAIChat struct {
	p Provider
	c *http.Client
}

type openAIFn struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAICall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function openAIFn `json:"function"`
}

type openAIMsg struct {
	Role       string       `json:"role"`
	Content    *string      `json:"content"`
	ToolCalls  []openAICall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

// request returns headers and body of a /chat/completions call.
func (o *openAIChat) request(model, system string, msgs []Message, tools []ToolDef, stream bool) (map[string]string, map[string]any) {
	text := func(s string) *string { return &s }
	out := []openAIMsg{{Role: "system", Content: text(system)}}
	for _, m := range msgs {
		om := openAIMsg{Role: m.Role, Content: text(m.Content), ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openAICall{ID: tc.ID, Type: "function", Function: openAIFn{tc.Name, string(argsObject(tc.Arguments))}})
		}
		out = append(out, om)
	}
	var defs []map[string]any
	for _, t := range tools {
		defs = append(defs, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": schemaOf(t)}})
	}
	body := map[string]any{"model": model, "messages": out, "tools": defs}
	if stream {
		body["stream"] = true
	}
	return map[string]string{"Authorization": "Bearer " + o.p.APIKey}, body
}

func (o *openAIChat) Complete(ctx context.Context, model, system string, msgs []Message, tools []ToolDef) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, false)
	var resp struct {
		Choices []struct {
			Message openAIMsg `json:"message"`
		} `json:"choices"`
	}
	if err := postJSON(ctx, o.c, o.p.Endpoint+"/chat/completions", headers, body, &resp); err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	if len(resp.Choices) == 0 {
		return Message{}, fmt.Errorf("%s: Antwort ohne choices", o.p.Name)
	}
	m := resp.Choices[0].Message
	res := Message{Role: "assistant"}
	if m.Content != nil {
		res.Content = *m.Content
	}
	for _, tc := range m.ToolCalls {
		res.ToolCalls = append(res.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: argsObject(json.RawMessage(tc.Function.Arguments))})
	}
	return res, nil
}

// ---------- Claude (Anthropic Messages API) ----------

type claudeChat struct {
	p Provider
	c *http.Client
}

// request returns headers and body of a /v1/messages call.
func (o *claudeChat) request(model, system string, msgs []Message, tools []ToolDef, stream bool) (map[string]string, map[string]any) {
	type block map[string]any
	type msg struct {
		Role    string  `json:"role"`
		Content []block `json:"content"`
	}
	var out []msg
	appendBlocks := func(role string, blocks ...block) {
		if n := len(out); n > 0 && out[n-1].Role == role {
			out[n-1].Content = append(out[n-1].Content, blocks...) // consecutive tool results form one user turn
			return
		}
		out = append(out, msg{Role: role, Content: blocks})
	}
	for _, m := range msgs {
		switch m.Role {
		case "tool":
			appendBlocks("user", block{"type": "tool_result", "tool_use_id": m.ToolCallID, "content": m.Content})
		case "assistant":
			var blocks []block
			if strings.TrimSpace(m.Content) != "" {
				blocks = append(blocks, block{"type": "text", "text": m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, block{"type": "tool_use", "id": tc.ID, "name": tc.Name, "input": argsObject(tc.Arguments)})
			}
			appendBlocks("assistant", blocks...)
		default:
			appendBlocks("user", block{"type": "text", "text": m.Content})
		}
	}
	var defs []map[string]any
	for _, t := range tools {
		defs = append(defs, map[string]any{"name": t.Name, "description": t.Description, "input_schema": schemaOf(t)})
	}
	headers := map[string]string{"x-api-key": o.p.APIKey, "anthropic-version": "2023-06-01"}
	body := map[string]any{"model": model, "max_tokens": 8192, "system": system, "messages": out, "tools": defs}
	if stream {
		body["stream"] = true
	}
	return headers, body
}

func (o *claudeChat) Complete(ctx context.Context, model, system string, msgs []Message, tools []ToolDef) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, false)
	var resp struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := postJSON(ctx, o.c, o.p.Endpoint+"/v1/messages", headers, body, &resp); err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	res := Message{Role: "assistant"}
	var texts []string
	for _, b := range resp.Content {
		switch b.Type {
		case "text":
			texts = append(texts, b.Text)
		case "tool_use":
			res.ToolCalls = append(res.ToolCalls, ToolCall{ID: b.ID, Name: b.Name, Arguments: argsObject(b.Input)})
		}
	}
	res.Content = strings.Join(texts, "\n\n")
	return res, nil
}
