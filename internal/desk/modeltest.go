package desk

import (
	"context"
	"time"
)

// Fixed texts of a model test: nothing from the conversation or the project is sent.
const (
	modelTestSystem = "Verbindungstest von sdd-desk. Antworte in einem kurzen Satz und rufe kein Werkzeug auf."
	modelTestPrompt = "Hallo! Bitte antworte kurz mit „Hello World“."
)

// modelTestTimeout bounds each attempt of a model test (a variable so tests can shorten it). Local models may need
// to load first.
var modelTestTimeout = 120 * time.Second

// ModelTestRequest selects the model to test (a subset of TurnRequest).
type ModelTestRequest struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	DisclosureAck bool   `json:"disclosure_ack"`
}

// ModelTestResult tells whether the model answers and whether it accepts the sdd tools. Tools is nil if no attempt
// succeeded. Errors are the provider's messages as returned by post: status and body start, never headers.
type ModelTestResult struct {
	OK         bool   `json:"ok"`
	Timeout    bool   `json:"timeout,omitempty"`
	Tools      *bool  `json:"tools,omitempty"`
	Reply      string `json:"reply,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorPlain string `json:"error_plain,omitempty"`
	MS         int64  `json:"ms,omitempty"`
}

// TestModel sends the fixed test message with the sdd tool definitions; if the provider refuses, it tries again
// without tools. A timeout or cancel is no refusal of the tools and ends the test. Tool calls in the answer are never
// run.
func (a *Agent) TestModel(ctx context.Context, req ModelTestRequest) (*ModelTestResult, error) {
	provider, err := a.Check(TurnRequest{Provider: req.Provider, Model: req.Model, DisclosureAck: req.DisclosureAck})
	if err != nil {
		return nil, err
	}
	chat := a.NewChat(*provider)
	msgs := []Message{{Role: "user", Content: modelTestPrompt}}
	res := &ModelTestResult{}
	attempt := func(tools []ToolDef) (Message, int64, error) {
		actx, cancel := context.WithTimeout(ctx, modelTestTimeout)
		defer cancel()
		start := time.Now()
		reply, err := chat.Complete(actx, req.Model, modelTestSystem, msgs, tools)
		if err != nil && actx.Err() != nil {
			res.Timeout = ctx.Err() == nil
		}
		return reply, time.Since(start).Milliseconds(), err
	}

	reply, ms, err := attempt(a.Tools)
	withTools := err == nil
	if err != nil {
		res.Error = err.Error()
		if ctx.Err() != nil || res.Timeout {
			return res, nil
		}
		if reply, ms, err = attempt(nil); err != nil {
			res.ErrorPlain = err.Error()
			return res, nil
		}
	}
	res.OK, res.Tools, res.MS, res.Reply = true, &withTools, ms, replyText(reply)
	return res, nil
}

// replyText is the start of the answer, or the name of a tool the model called anyway.
func replyText(m Message) string {
	text := m.Content
	if text == "" && len(m.ToolCalls) > 0 {
		text = "(ruft Werkzeug " + m.ToolCalls[0].Name + " auf)"
	}
	if r := []rune(text); len(r) > 200 {
		text = string(r[:200]) + "…"
	}
	return text
}
