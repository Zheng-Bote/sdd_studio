package desk

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed all:webdist
var webFS embed.FS

// maxBody bounds a request body (conversations can be long).
const maxBody = 8 << 20

// Server is the local HTTP API of sdd-desk (127.0.0.1 only).
type Server struct {
	Addr    string // host:port bound; the only accepted Host header
	Token   string
	Project string
	Config  *Config
	Agent   *Agent
	MCP     interface {
		Call(method string, params any) (json.RawMessage, error)
	}
	HTTP *http.Client // for model discovery of local Ollama

	mux *http.ServeMux
}

// Handler wires the routes.
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	if dist, err := fs.Sub(webFS, "webdist"); err == nil {
		if _, err := fs.Stat(dist, "index.html"); err == nil {
			m.Handle("GET /", http.FileServerFS(dist))
		} else {
			m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				fmt.Fprintln(w, "Die Oberfläche ist nicht gebaut: in apps/desk `npm run build` ausführen und sdd-desk neu bauen.")
			})
		}
	}
	m.HandleFunc("GET /api/meta", s.handleMeta)
	m.HandleFunc("GET /api/tools", s.handleTools)
	m.HandleFunc("GET /api/prompts", s.handlePrompts)
	m.HandleFunc("POST /api/prompts/get", s.handlePromptGet)
	m.HandleFunc("GET /api/next", s.handleNext)
	m.HandleFunc("POST /api/turn", s.handleTurn)
	m.HandleFunc("POST /api/turn/stream", s.handleTurnStream)
	m.HandleFunc("POST /api/model/test", s.handleModelTest)
	s.mux = m
	return s
}

// ServeHTTP checks Host for every request and the session token for every API call.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host != s.Addr {
		http.Error(w, "unerwarteter Host", http.StatusForbidden)
		return
	}
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-SDD-Token")), []byte(s.Token)) != 1 {
			writeError(w, http.StatusForbidden, errors.New("Sitzungstoken fehlt oder ist ungültig"))
			return
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type application/json erforderlich"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	}
	s.mux.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// modelView is one selectable model. Tools is nil when its capability is unknown: a model that cannot call tools
// is a finding, a model that was not asked is not. Evidence holds the recorded result per judgment flow; a flow
// without a recorded result is absent, never "proven".
type modelView struct {
	Name     string                   `json:"name"`
	Tools    *bool                    `json:"tools,omitempty"`
	Evidence map[string]EvidenceEntry `json:"evidence,omitempty"`
}

// providerView is one provider as the desktop shows it.
type providerView struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Protocol  string      `json:"protocol"`
	Endpoint  string      `json:"endpoint"`
	Cloud     bool        `json:"cloud"`
	Available bool        `json:"available"`
	Models    []modelView `json:"models"`
	Note      string      `json:"note,omitempty"`
}

// mergeLocalModels returns the local provider's models: the configured ones first, then what Ollama reports, each
// with its tools capability when it is known. Configured names stay visible when Ollama does not answer.
//
// desk-model-availability AC-6
func mergeLocalModels(configured, reported []string, caps map[string]*bool) []modelView {
	out := make([]modelView, 0, len(configured)+len(reported))
	seen := map[string]bool{}
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, modelView{Name: name, Tools: caps[name]})
	}
	for _, name := range configured {
		add(name)
	}
	for _, name := range reported {
		add(name)
	}
	return out
}

// localCapabilities asks /api/show for every model under one overall deadline, in parallel. A model that does not
// answer in time stays unknown; the map is never nil.
func localCapabilities(ctx context.Context, c *http.Client, endpoint string, models []string) map[string]*bool {
	caps := make(map[string]*bool, len(models))
	type answer struct {
		model string
		tools *bool
	}
	ch := make(chan answer, len(models))
	for _, m := range models {
		go func(model string) {
			mctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			tools, err := ModelCapabilities(mctx, c, endpoint, model)
			if err != nil {
				tools = nil
			}
			ch <- answer{model, tools}
		}(m)
	}
	for range models {
		select {
		case a := <-ch:
			caps[a.model] = a.tools
		case <-ctx.Done():
			return caps
		}
	}
	return caps
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	var providers []providerView
	for _, p := range s.Config.Providers {
		v := providerView{ID: p.ID, Name: p.Name, Protocol: p.Protocol, Endpoint: p.Endpoint, Cloud: p.Cloud, Available: true}
		if p.ID == LocalOllamaID {
			ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
			reported, err := ListLocalModels(ctx, s.HTTP, p.Endpoint)
			cancel()
			if err != nil {
				v.Available, v.Note = len(p.Models) > 0, "lokales Ollama nicht erreichbar: "+err.Error()
			}
			cctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			caps := localCapabilities(cctx, s.HTTP, p.Endpoint, mergeNames(p.Models, reported))
			cancel()
			v.Models = mergeLocalModels(p.Models, reported, caps)
		} else {
			for _, m := range p.Models {
				v.Models = append(v.Models, modelView{Name: m})
			}
		}
		annotateEvidence(p.Protocol, v.Models)
		providers = append(providers, v)
	}
	meta := map[string]any{
		"project": s.Project, "config_path": s.Config.Path, "warnings": s.Config.Warnings, "providers": providers,
		"judgment_flows": JudgmentFlows,
	}
	if state := s.projectState(); state != "" {
		meta["project_state"] = state
	}
	writeJSON(w, http.StatusOK, meta)
}

// mergeNames is the union of two name lists without duplicates, configured first.
func mergeNames(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{a, b} {
		for _, name := range list {
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// projectState says whether the project is covered by sdd: "configured", "existing" (files, no configuration) or
// "empty". The desk itself asks sdd_stack (read-only, not on behalf of the model); "" if that fails.
func (s *Server) projectState() string {
	if s.Agent == nil || s.Agent.Runner == nil {
		return ""
	}
	res, err := s.Agent.Runner.CallTool("sdd_stack", json.RawMessage(`{}`))
	if err != nil || res.IsError {
		return ""
	}
	var info struct {
		Configured bool `json:"configured"`
		Files      int  `json:"files"`
	}
	if err := json.Unmarshal(res.Structured, &info); err != nil {
		return ""
	}
	switch {
	case info.Configured:
		return "configured"
	case info.Files > 0:
		return "existing"
	}
	return "empty"
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": s.Agent.Tools})
}

func (s *Server) handlePrompts(w http.ResponseWriter, r *http.Request) {
	raw, err := s.MCP.Call("prompts/list", map[string]any{})
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (s *Server) handlePromptGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	raw, err := s.MCP.Call("prompts/get", req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

// handleNext shows the process state (sdd_next) for the sidebar; reading it needs no model and no permission.
func (s *Server) handleNext(w http.ResponseWriter, r *http.Request) {
	args, _ := json.Marshal(map[string]string{"feature": r.URL.Query().Get("feature")})
	res, err := s.Agent.Runner.CallTool("sdd_next", args)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if res.IsError {
		writeError(w, http.StatusBadRequest, errors.New(res.Text))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(res.Structured)
}

func (s *Server) handleTurn(w http.ResponseWriter, r *http.Request) {
	var req TurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültige Anfrage: %w", err))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.turnTimeout(req.Provider))
	defer cancel()
	res, err := s.Agent.Turn(ctx, req)
	if err != nil {
		writeError(w, turnErrorStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleModelTest sends the fixed test message to the selected model. Request errors are answered as in
// handleTurnStream; a model that fails is a result, not an error.
func (s *Server) handleModelTest(w http.ResponseWriter, r *http.Request) {
	var req ModelTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültige Anfrage: %w", err))
		return
	}
	res, err := s.Agent.TestModel(r.Context(), req)
	if err != nil {
		status := turnErrorStatus(err)
		if status == http.StatusBadGateway {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// turnTimeout is the provider's bound for one conversation turn; unknown providers are refused by Agent.Check anyway.
func (s *Server) turnTimeout(providerID string) time.Duration {
	for _, p := range s.Config.Providers {
		if p.ID == providerID && p.TurnTimeout > 0 {
			return p.TurnTimeout
		}
	}
	return cloudTurnTimeout
}

func turnErrorStatus(err error) int {
	switch {
	case errors.Is(err, ErrDisclosure):
		return http.StatusPreconditionRequired
	case errors.Is(err, ErrUnknownProvider):
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

// handleTurnStream runs a turn and sends its events as NDJSON. Request errors (provider, disclosure, model) are
// answered with a status before the stream starts; later errors arrive as an error event. The turn runs in the
// request's context, so a client that aborts also aborts the model request.
func (s *Server) handleTurnStream(w http.ResponseWriter, r *http.Request) {
	var req TurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültige Anfrage: %w", err))
		return
	}
	if _, err := s.Agent.Check(req); err != nil {
		status := turnErrorStatus(err)
		if status == http.StatusBadGateway {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	enc := json.NewEncoder(w)
	emit := func(e Event) {
		_ = enc.Encode(e)
		_ = rc.Flush()
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.turnTimeout(req.Provider))
	defer cancel()
	res, err := s.Agent.TurnStream(ctx, req, emit)
	if err != nil {
		emit(Event{Type: "error", Error: err.Error()})
		return
	}
	emit(Event{Type: "done", Messages: res.Messages, Pending: res.Pending})
}
