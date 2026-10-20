package desk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// maxModelSteps bounds the model calls of one turn.
const maxModelSteps = 8

// ToolRunner executes sdd tools (the MCP client in production).
type ToolRunner interface {
	CallTool(name string, args json.RawMessage) (*ToolResult, error)
}

// Agent runs conversation turns: the model may request sdd tools, every call needs the user's permission.
type Agent struct {
	Tools     []ToolDef
	Runner    ToolRunner
	Providers []Provider
	NewChat   func(Provider) Chat
	System    string
}

// TurnRequest is one turn from the UI. The UI keeps the conversation and the session's permissions.
type TurnRequest struct {
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	Messages      []Message `json:"messages"`
	AllowedTools  []string  `json:"allowed_tools"`  // allowed for the session (read-only calls)
	ApprovedCalls []string  `json:"approved_calls"` // call ids allowed once
	DeniedCalls   []string  `json:"denied_calls"`   // call ids refused
	DisclosureAck bool      `json:"disclosure_ack"` // the user confirmed that content leaves the device
}

// Pending is a tool call waiting for the user's decision.
type Pending struct {
	CallID    string          `json:"call_id"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
	Writes    bool            `json:"writes"`
	Preview   json.RawMessage `json:"preview,omitempty"` // for writes: the plan that would be applied
}

// TurnResponse returns the extended conversation and the calls that need a decision.
type TurnResponse struct {
	Messages []Message `json:"messages"`
	Pending  []Pending `json:"pending"`
}

// Event is one step of a streamed turn: a text piece (delta), a piece of the model's reasoning (thinking), a finished
// message, the final state (done) or an error.
type Event struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Message  *Message  `json:"message,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Pending  []Pending `json:"pending,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// Errors of a turn that the UI must handle.
var (
	ErrDisclosure      = errors.New("Cloud-Anbieter: Hinweis „Inhalte verlassen das Gerät“ nicht bestätigt")
	ErrUnknownProvider = errors.New("unbekannter oder nicht konfigurierter Anbieter")
)

// writes reports whether a call changes files: sdd_new with apply.
func writes(tc ToolCall) bool {
	if tc.Name != "sdd_new" {
		return false
	}
	var a struct {
		Apply bool `json:"apply"`
	}
	_ = json.Unmarshal(tc.Arguments, &a)
	return a.Apply
}

func (a *Agent) toolKnown(name string) bool {
	for _, t := range a.Tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

// openCalls returns the tool calls of the last assistant message that have no tool result yet.
func openCalls(msgs []Message) []ToolCall {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != "assistant" {
			continue
		}
		answered := map[string]bool{}
		for _, m := range msgs[i+1:] {
			if m.Role == "tool" {
				answered[m.ToolCallID] = true
			}
		}
		var open []ToolCall
		for _, tc := range msgs[i].ToolCalls {
			if !answered[tc.ID] {
				open = append(open, tc)
			}
		}
		return open
	}
	return nil
}

// Check validates provider, disclosure and model before anything is sent or streamed.
func (a *Agent) Check(req TurnRequest) (*Provider, error) {
	var provider *Provider
	for i := range a.Providers {
		if a.Providers[i].ID == req.Provider {
			provider = &a.Providers[i]
		}
	}
	if provider == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, req.Provider)
	}
	if provider.Cloud && !req.DisclosureAck {
		return nil, ErrDisclosure
	}
	if req.Model == "" {
		return nil, errors.New("kein Modell gewählt")
	}
	return provider, nil
}

// Turn runs open tool calls the user decided on and asks the model until it answers or a call needs a decision.
func (a *Agent) Turn(ctx context.Context, req TurnRequest) (*TurnResponse, error) {
	return a.run(ctx, req, nil)
}

// TurnStream works like Turn and reports text pieces and every finished message to emit as they happen.
// The model streams if its chat implements Streamer. Permissions are decided exactly as in Turn.
func (a *Agent) TurnStream(ctx context.Context, req TurnRequest, emit func(Event)) (*TurnResponse, error) {
	return a.run(ctx, req, emit)
}

func (a *Agent) run(ctx context.Context, req TurnRequest, emit func(Event)) (*TurnResponse, error) {
	provider, err := a.Check(req)
	if err != nil {
		return nil, err
	}
	chat := a.NewChat(*provider)
	msgs := append([]Message{}, req.Messages...)
	add := func(m Message) {
		msgs = append(msgs, m)
		if emit != nil {
			emit(Event{Type: "message", Message: &m})
		}
	}

	for step := 0; ; step++ {
		var pending []Pending
		for _, tc := range openCalls(msgs) {
			result, wait, err := a.decide(tc, req)
			if err != nil {
				return nil, err
			}
			if wait != nil {
				pending = append(pending, *wait)
				continue
			}
			add(Message{Role: "tool", ToolCallID: tc.ID, ToolName: tc.Name, Content: result})
		}
		if len(pending) > 0 {
			return &TurnResponse{Messages: msgs, Pending: pending}, nil
		}
		if step >= maxModelSteps {
			add(Message{Role: "assistant", Content: fmt.Sprintf("(Abgebrochen nach %d Modellaufrufen in einem Zug.)", maxModelSteps)})
			return &TurnResponse{Messages: msgs, Pending: []Pending{}}, nil
		}
		var reply Message
		if s, ok := chat.(Streamer); ok && emit != nil {
			reply, err = s.Stream(ctx, req.Model, a.System, msgs, a.Tools, Deltas{
				Text:     func(text string) { emit(Event{Type: "delta", Text: text}) },
				Thinking: func(text string) { emit(Event{Type: "thinking", Text: text}) },
			})
		} else {
			reply, err = chat.Complete(ctx, req.Model, a.System, msgs, a.Tools)
		}
		if err != nil {
			return nil, err
		}
		for i := range reply.ToolCalls {
			if reply.ToolCalls[i].ID == "" {
				reply.ToolCalls[i].ID = fmt.Sprintf("call-%d-%d", len(msgs), i)
			}
		}
		add(reply)
		if len(reply.ToolCalls) == 0 {
			return &TurnResponse{Messages: msgs, Pending: []Pending{}}, nil
		}
	}
}

// decide runs, refuses or parks one tool call. Unknown tools are never run; writes need a one-time approval even if
// the tool is allowed for the session, and are parked with a preview.
func (a *Agent) decide(tc ToolCall, req TurnRequest) (string, *Pending, error) {
	if !a.toolKnown(tc.Name) {
		return "Abgelehnt: " + tc.Name + " ist kein Werkzeug von sdd mcp.", nil, nil
	}
	if slices.Contains(req.DeniedCalls, tc.ID) {
		return "Der Nutzer hat diesen Aufruf abgelehnt.", nil, nil
	}
	w := writes(tc)
	approved := slices.Contains(req.ApprovedCalls, tc.ID) || (!w && slices.Contains(req.AllowedTools, tc.Name))
	if !approved {
		p := &Pending{CallID: tc.ID, Tool: tc.Name, Arguments: argsObject(tc.Arguments), Writes: w}
		if w {
			preview, err := a.preview(tc)
			if err != nil {
				return "", nil, err
			}
			p.Preview = preview
		}
		return "", p, nil
	}
	res, err := a.Runner.CallTool(tc.Name, tc.Arguments)
	if err != nil {
		return "Fehler beim Ausführen: " + err.Error(), nil, nil
	}
	if res.IsError {
		return "Werkzeugfehler: " + res.Text, nil, nil
	}
	return res.Text, nil, nil
}

// preview runs a write call with apply=false so the user sees files and content before approving.
func (a *Agent) preview(tc ToolCall) (json.RawMessage, error) {
	var args map[string]any
	if err := json.Unmarshal(argsObject(tc.Arguments), &args); err != nil {
		return nil, err
	}
	args["apply"] = false
	data, _ := json.Marshal(args)
	res, err := a.Runner.CallTool(tc.Name, data)
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return json.Marshal(map[string]string{"error": res.Text})
	}
	return res.Structured, nil
}
