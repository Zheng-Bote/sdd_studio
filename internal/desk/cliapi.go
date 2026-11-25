package desk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// cliView is one agent CLI as the desktop shows it. Offerable carries the decision; blocked_because names the reason
// in plain words, so a CLI that is missing is missing for a stated reason (agenten-cli-proxy AC-1).
type cliView struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Cloud          bool        `json:"cloud"`
	Installed      bool        `json:"installed"`
	Offerable      bool        `json:"offerable"`
	BlockedBecause string      `json:"blocked_because,omitempty"`
	Evidence       CLIEvidence `json:"evidence"`
}

// handleCLIList reports which agent CLIs exist and which may be offered.
//
// agenten-cli-proxy AC-1, AC-8
func (s *Server) handleCLIList(w http.ResponseWriter, r *http.Request) {
	views := make([]cliView, 0, len(CLISpecs))
	for _, spec := range CLISpecs {
		installed := spec.Installed()
		views = append(views, cliView{
			ID: spec.ID, Name: spec.Name, Cloud: spec.Cloud,
			Installed:      installed,
			Offerable:      spec.Offerable(installed),
			BlockedBecause: spec.BlockReason(installed),
			Evidence:       EvidenceForCLI(spec.ID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"cloud_notice": CloudNotice, "clis": views})
}

// handleCLICards lists the open calls of all CLI sessions.
//
// agenten-cli-proxy AC-3, AC-10
func (s *Server) handleCLICards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"cloud_notice": CloudNotice,
		"cards":        s.Proxy.Approvals.Open(),
	})
}

type cliDecision struct {
	CallID   string `json:"call_id"`
	Decision string `json:"decision"`
}

// handleCLIDecide answers one card by its call id. A session allowance on a critical call is refused with 400, not
// narrowed to "once": the client must not be able to blur the difference (AC-2).
//
// agenten-cli-proxy AC-2, AC-3, AC-4, AC-10
func (s *Server) handleCLIDecide(w http.ResponseWriter, r *http.Request) {
	var req cliDecision
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültige Anfrage: %w", err))
		return
	}
	var v Verdict
	switch req.Decision {
	case "allow":
		v = Verdict{Allow: true, Scope: ScopeCall}
	case "deny":
		v = Verdict{Allow: false, Scope: ScopeCall}
	case "allow_tool_session":
		v = Verdict{Allow: true, Scope: ScopeToolSession}
	case "allow_write_session":
		v = Verdict{Allow: true, Scope: ScopeWriteSession}
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unbekannte Entscheidung %q", req.Decision))
		return
	}
	if err := s.Proxy.Approvals.Decide(req.CallID, v); err != nil {
		switch {
		case errors.Is(err, ErrScopeNotAllowed):
			writeError(w, http.StatusBadRequest, err)
		case errors.Is(err, ErrUnknownCall):
			writeError(w, http.StatusNotFound, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"call_id": req.CallID, "decision": req.Decision})
}

type cliStartRequest struct {
	CLI           string `json:"cli"`
	Prompt        string `json:"prompt"`
	DisclosureAck bool   `json:"disclosure_ack"`
}

// handleCLIStart starts one CLI session. Without the cloud notice acknowledged it refuses (AC-6).
//
// agenten-cli-proxy AC-1, AC-6
func (s *Server) handleCLIStart(w http.ResponseWriter, r *http.Request) {
	var req cliStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültige Anfrage: %w", err))
		return
	}
	run, err := s.CLIs.Start(req.CLI, req.Prompt, req.DisclosureAck)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrDisclosure) {
			status = http.StatusForbidden
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": run.ID, "cli": run.CLI})
}
