package desk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Streamer is implemented by chats that can deliver an answer piece by piece. The returned message equals what
// Complete would return for the same answer.
type Streamer interface {
	Stream(ctx context.Context, model, system string, msgs []Message, tools []ToolDef, out Deltas) (Message, error)
}

// Deltas receives the pieces of a streamed answer as they arrive. Thinking is the model's reasoning: shown to the
// user while the model works, but never part of the conversation.
type Deltas struct {
	Text     func(string)
	Thinking func(string)
}

func (d Deltas) text(s string) {
	if s != "" && d.Text != nil {
		d.Text(s)
	}
}

func (d Deltas) thinking(s string) {
	if s != "" && d.Thinking != nil {
		d.Thinking(s)
	}
}

// maxStreamLine bounds one NDJSON or SSE line.
const maxStreamLine = 1 << 20

// eachLine calls fn for every line of r until fn returns done or r ends.
func eachLine(r io.Reader, fn func(line []byte) (done bool, err error)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), maxStreamLine)
	for sc.Scan() {
		done, err := fn(bytes.TrimRight(sc.Bytes(), "\r"))
		if err != nil || done {
			return err
		}
	}
	return sc.Err()
}

// sseData returns the payload of an SSE "data:" line; other lines (event:, comments, blank) yield nil.
func sseData(line []byte) []byte {
	data, ok := bytes.CutPrefix(line, []byte("data:"))
	if !ok {
		return nil
	}
	return bytes.TrimSpace(data)
}

// toolArgs checks streamed arguments: empty means no arguments, anything else must be complete JSON.
func toolArgs(name, raw string) (json.RawMessage, error) {
	if strings.TrimSpace(raw) == "" {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("unvollständige Argumente für %s", name)
	}
	return json.RawMessage(raw), nil
}

// streamError extracts the message of an error object ({"error":"…"} or {"error":{"message":"…"}}).
func streamError(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return errors.New(text)
	}
	var obj struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Message != "" {
		return errors.New(obj.Message)
	}
	return errors.New(string(raw))
}

// ---------- Ollama: NDJSON ----------

func (o *ollamaChat) Stream(ctx context.Context, model, system string, msgs []Message, tools []ToolDef, out Deltas) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, true)
	resp, err := post(ctx, o.c, o.p.Endpoint+"/api/chat", headers, body)
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	defer resp.Body.Close()
	res := Message{Role: "assistant"}
	var text strings.Builder
	err = eachLine(resp.Body, func(line []byte) (bool, error) {
		if len(bytes.TrimSpace(line)) == 0 {
			return false, nil
		}
		var chunk struct {
			Message ollamaMsg       `json:"message"`
			Done    bool            `json:"done"`
			Error   json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			return false, fmt.Errorf("unerwartete Antwort: %w", err)
		}
		if err := streamError(chunk.Error); err != nil {
			return false, err
		}
		out.thinking(chunk.Message.Thinking)
		text.WriteString(chunk.Message.Content)
		out.text(chunk.Message.Content)
		for _, tc := range chunk.Message.ToolCalls {
			res.ToolCalls = append(res.ToolCalls, ToolCall{Name: tc.Function.Name, Arguments: argsObject(tc.Function.Arguments)})
		}
		return chunk.Done, nil
	})
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	res.Content = text.String()
	return res, nil
}

// ---------- OpenAI-compatible: SSE with deltas ----------

func (o *openAIChat) Stream(ctx context.Context, model, system string, msgs []Message, tools []ToolDef, out Deltas) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, true)
	resp, err := post(ctx, o.c, o.p.Endpoint+"/chat/completions", headers, body)
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	defer resp.Body.Close()

	type acc struct {
		id, name string
		args     strings.Builder
	}
	var calls []*acc
	byIndex := map[int]*acc{}
	var text strings.Builder
	err = eachLine(resp.Body, func(line []byte) (bool, error) {
		data := sseData(line)
		if len(data) == 0 {
			return false, nil
		}
		if string(data) == "[DONE]" {
			return true, nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          *string `json:"content"`
					Reasoning        string  `json:"reasoning"`
					ReasoningContent string  `json:"reasoning_content"`
					ToolCalls        []struct {
						Index    int      `json:"index"`
						ID       string   `json:"id"`
						Function openAIFn `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			return false, fmt.Errorf("unerwartete Antwort: %w", err)
		}
		if err := streamError(chunk.Error); err != nil {
			return false, err
		}
		for _, ch := range chunk.Choices {
			out.thinking(ch.Delta.Reasoning)
			out.thinking(ch.Delta.ReasoningContent)
			if c := ch.Delta.Content; c != nil {
				text.WriteString(*c)
				out.text(*c)
			}
			for _, tc := range ch.Delta.ToolCalls {
				a := byIndex[tc.Index]
				// Some servers send complete calls without an index; a new id then starts a new call.
				if a == nil || (tc.ID != "" && a.id != "" && tc.ID != a.id) {
					a = &acc{}
					byIndex[tc.Index] = a
					calls = append(calls, a)
				}
				if tc.ID != "" {
					a.id = tc.ID
				}
				a.name += tc.Function.Name
				a.args.WriteString(tc.Function.Arguments)
			}
		}
		return false, nil
	})
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	res := Message{Role: "assistant", Content: text.String()}
	for _, a := range calls {
		args, err := toolArgs(a.name, a.args.String())
		if err != nil {
			return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
		}
		res.ToolCalls = append(res.ToolCalls, ToolCall{ID: a.id, Name: a.name, Arguments: args})
	}
	return res, nil
}

// ---------- Claude: SSE with content blocks ----------

func (o *claudeChat) Stream(ctx context.Context, model, system string, msgs []Message, tools []ToolDef, out Deltas) (Message, error) {
	headers, body := o.request(model, system, msgs, tools, true)
	resp, err := post(ctx, o.c, o.p.Endpoint+"/v1/messages", headers, body)
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	defer resp.Body.Close()

	type blk struct {
		typ, id, name string
		text, input   strings.Builder
	}
	blocks := map[int]*blk{}
	textBlocks := 0
	err = eachLine(resp.Body, func(line []byte) (bool, error) {
		data := sseData(line)
		if len(data) == 0 {
			return false, nil
		}
		var ev struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
				Text string `json:"text"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				Thinking    string `json:"thinking"`
			} `json:"delta"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(data, &ev); err != nil {
			return false, fmt.Errorf("unerwartete Antwort: %w", err)
		}
		switch ev.Type {
		case "error":
			if err := streamError(ev.Error); err != nil {
				return false, err
			}
			return false, errors.New("Fehler im Stream")
		case "content_block_start":
			b := &blk{typ: ev.ContentBlock.Type, id: ev.ContentBlock.ID, name: ev.ContentBlock.Name}
			blocks[ev.Index] = b
			if b.typ == "text" {
				if textBlocks > 0 {
					out.text("\n\n") // the same separator Complete puts between text blocks
				}
				textBlocks++
			}
			b.text.WriteString(ev.ContentBlock.Text)
			out.text(ev.ContentBlock.Text)
		case "content_block_delta":
			b := blocks[ev.Index]
			if b == nil {
				return false, fmt.Errorf("Delta für unbekannten Block %d", ev.Index)
			}
			switch ev.Delta.Type {
			case "text_delta":
				b.text.WriteString(ev.Delta.Text)
				out.text(ev.Delta.Text)
			case "thinking_delta":
				out.thinking(ev.Delta.Thinking)
			case "input_json_delta":
				b.input.WriteString(ev.Delta.PartialJSON)
			}
		case "message_stop":
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
	}
	order := make([]int, 0, len(blocks))
	for i := range blocks {
		order = append(order, i)
	}
	sort.Ints(order)
	res := Message{Role: "assistant"}
	var texts []string
	for _, i := range order {
		b := blocks[i]
		switch b.typ {
		case "text":
			texts = append(texts, b.text.String())
		case "tool_use":
			args, err := toolArgs(b.name, b.input.String())
			if err != nil {
				return Message{}, fmt.Errorf("%s: %w", o.p.Name, err)
			}
			res.ToolCalls = append(res.ToolCalls, ToolCall{ID: b.id, Name: b.name, Arguments: args})
		}
	}
	res.Content = strings.Join(texts, "\n\n")
	return res, nil
}
