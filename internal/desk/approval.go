package desk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Errors a decision can carry. The HTTP layer maps ErrScopeNotAllowed to 400: a client that asks for a session
// allowance on a critical call gets a refusal, not a silent downgrade to "once" (ADR 0010, agenten-cli-proxy AC-2).
var (
	ErrUnknownCall     = errors.New("unbekannter oder abgelaufener Aufruf")
	ErrScopeNotAllowed = errors.New("diese Freigabe gilt für diese Stufe nicht")
)

// Allowance scopes of a decision.
const (
	ScopeCall         = "call"          // exactly this call
	ScopeToolSession  = "tool_session"  // this tool for the rest of the session, reading and previews only
	ScopeWriteSession = "write_session" // this tool for the rest of the session, including writes
)

// Verdict is the user's decision on one call.
type Verdict struct {
	Allow  bool   `json:"allow"`
	Scope  string `json:"scope"`
	Reason string `json:"reason,omitempty"`
}

// Call is one tools/call waiting for a decision. Its ID is what correlates the answer with the call — never the
// position in a queue, because a CLI may run calls in parallel, repeat or drop them (ADR 0010, decision 4).
type Call struct {
	ID      string
	Session string
	CLI     string
	Tool    string
	Args    json.RawMessage
	Tier    string
	Summary string
	Expires time.Time

	decide chan Verdict
}

// CallView is a call as the desktop shows it on its card.
type CallView struct {
	CallID    string          `json:"call_id"`
	CLI       string          `json:"cli"`
	Session   string          `json:"session"`
	Tool      string          `json:"tool"`
	Tier      string          `json:"tier"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Summary   string          `json:"summary,omitempty"`
	Deadline  time.Time       `json:"deadline"`
}

// Approvals keeps the open calls of all running CLI sessions. One instance per desktop.
type Approvals struct {
	mu      sync.Mutex
	calls   map[string]*Call
	order   []string
	allow   map[string]map[string]string // session -> tool -> scope granted for the session
	timeout time.Duration
	now     func() time.Time
}

// NewApprovals returns a register whose calls expire after timeout.
func NewApprovals(timeout time.Duration) *Approvals {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &Approvals{
		calls:   map[string]*Call{},
		allow:   map[string]map[string]string{},
		timeout: timeout,
		now:     time.Now,
	}
}

func newCallID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "c-" + hex.EncodeToString(b), nil
}

// Register parks one call. It returns a granted verdict when the session already allowed this tool at this tier;
// then nothing is parked and the caller runs the call at once.
func (a *Approvals) Register(session, cli, tool string, args json.RawMessage, tier string) (*Call, *Verdict, error) {
	if session == "" {
		return nil, nil, errors.New("Aufruf ohne Sitzung")
	}
	id, err := newCallID()
	if err != nil {
		return nil, nil, err
	}
	c := &Call{
		ID: id, Session: session, CLI: cli, Tool: tool, Args: args, Tier: tier,
		Expires: a.now().Add(a.timeout), decide: make(chan Verdict, 1),
	}
	a.mu.Lock()
	if v := a.allowanceLocked(session, tool, tier); v != nil {
		a.mu.Unlock()
		return c, v, nil
	}
	a.calls[id] = c
	a.order = append(a.order, id)
	a.mu.Unlock()

	time.AfterFunc(a.timeout, func() { a.expire(id) })
	return c, nil, nil
}

// allowanceLocked reports an existing session allowance for this tool and tier, or nil. A write allowance covers
// reading, a read allowance never covers writing; a critical call is never covered.
func (a *Approvals) allowanceLocked(session, tool, tier string) *Verdict {
	scope, ok := a.allow[session][tool]
	if !ok {
		return nil
	}
	switch {
	case tier == tierCritical:
		return nil
	case tier == tierWrite && scope != ScopeWriteSession:
		return nil
	}
	return &Verdict{Allow: true, Scope: scope}
}

// Decide answers one call by its ID. It records a session allowance when the scope is wider than this call and
// refuses an impossible combination instead of narrowing it silently.
func (a *Approvals) Decide(id string, v Verdict) error {
	a.mu.Lock()
	c, ok := a.calls[id]
	if !ok {
		a.mu.Unlock()
		return ErrUnknownCall
	}
	if !c.Expires.After(a.now()) {
		delete(a.calls, id)
		a.mu.Unlock()
		return ErrUnknownCall
	}
	if v.Allow {
		switch v.Scope {
		case ScopeCall:
		case ScopeToolSession:
			if c.Tier != tierRead {
				a.mu.Unlock()
				return ErrScopeNotAllowed
			}
		case ScopeWriteSession:
			if c.Tier != tierWrite {
				a.mu.Unlock()
				return ErrScopeNotAllowed
			}
		default:
			a.mu.Unlock()
			return ErrScopeNotAllowed
		}
	} else {
		v.Scope = ScopeCall
	}
	delete(a.calls, id)
	if v.Allow && v.Scope != ScopeCall {
		if a.allow[c.Session] == nil {
			a.allow[c.Session] = map[string]string{}
		}
		a.allow[c.Session][c.Tool] = v.Scope
	}
	a.mu.Unlock()
	c.decide <- v
	return nil
}

// Await blocks until the call is decided, the session ends, the caller goes away or the deadline passes. A call
// that nobody decides ends as a refusal; it never runs.
func (a *Approvals) Await(ctx context.Context, c *Call) Verdict {
	timer := time.NewTimer(maxDuration(time.Until(c.Expires), 0))
	defer timer.Stop()
	select {
	case v := <-c.decide:
		return v
	case <-ctx.Done():
		a.Remove(c.ID)
		return Verdict{Reason: "Sitzung beendet"}
	case <-timer.C:
		a.Remove(c.ID)
		return Verdict{Reason: "Frist abgelaufen"}
	}
}

// expire withdraws a call whose deadline passed. A late decision then finds nothing.
func (a *Approvals) expire(id string) {
	a.mu.Lock()
	c, ok := a.calls[id]
	if ok {
		delete(a.calls, id)
	}
	a.mu.Unlock()
	if ok {
		select {
		case c.decide <- Verdict{Reason: "Frist abgelaufen"}:
		default:
		}
	}
}

// Remove drops a call without a verdict (it already ran or its session ended).
func (a *Approvals) Remove(id string) {
	a.mu.Lock()
	delete(a.calls, id)
	a.mu.Unlock()
}

// WithdrawSession drops every open call of one session and refuses them. The session's allowances end with it.
func (a *Approvals) WithdrawSession(session, reason string) {
	if reason == "" {
		reason = "Sitzung beendet"
	}
	a.mu.Lock()
	var dropped []*Call
	for id, c := range a.calls {
		if c.Session == session {
			delete(a.calls, id)
			dropped = append(dropped, c)
		}
	}
	delete(a.allow, session)
	a.mu.Unlock()
	for _, c := range dropped {
		select {
		case c.decide <- Verdict{Reason: reason}:
		default:
		}
	}
}

// Open lists the calls that still wait for a decision, oldest first.
func (a *Approvals) Open() []CallView {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]CallView, 0, len(a.calls))
	for _, id := range a.order {
		c, ok := a.calls[id]
		if !ok {
			continue
		}
		out = append(out, CallView{
			CallID: c.ID, CLI: c.CLI, Session: c.Session, Tool: c.Tool, Tier: c.Tier,
			Arguments: c.Args, Summary: c.Summary, Deadline: c.Expires,
		})
	}
	return out
}

// SetSummary attaches the one-line effect of a parked call, filled in after the preview ran.
func (a *Approvals) SetSummary(id, summary string) {
	a.mu.Lock()
	if c, ok := a.calls[id]; ok {
		c.Summary = summary
	}
	a.mu.Unlock()
}

// CountOpen is the number of calls waiting for a decision.
func (a *Approvals) CountOpen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.calls)
}

func maxDuration(d, min time.Duration) time.Duration {
	if d < min {
		return min
	}
	return d
}

// refusalText is the tool result of a call that was not approved (same wording as the desktop's own loop).
func refusalText(reason string) string {
	if reason == "" {
		return deniedText
	}
	return fmt.Sprintf("%s (%s)", deniedText, reason)
}
