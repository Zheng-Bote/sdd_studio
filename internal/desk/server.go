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

type providerView struct {
	Provider
	Available bool   `json:"available"`
	Note      string `json:"note,omitempty"`
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	var providers []providerView
	for _, p := range s.Config.Providers {
		v := providerView{Provider: p, Available: true}
		if p.ID == LocalOllamaID {
			ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
			models, err := ListLocalModels(ctx, s.HTTP, p.Endpoint)
			cancel()
			if err != nil {
				v.Available, v.Note = len(p.Models) > 0, "lokales Ollama nicht erreichbar: "+err.Error()
			}
			for _, m := range models {
				if !containsString(v.Models, m) {
					v.Models = append(v.Models, m)
				}
			}
		}
		providers = append(providers, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": s.Project, "config_path": s.Config.Path, "warnings": s.Config.Warnings, "providers": providers,
	})
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
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
	ctx, cancel := context.WithTimeout(r.Context(), turnTimeout)
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

// turnTimeout bounds one conversation turn.
const turnTimeout = 10 * time.Minute

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
	ctx, cancel := context.WithTimeout(r.Context(), turnTimeout)
	defer cancel()
	res, err := s.Agent.TurnStream(ctx, req, emit)
	if err != nil {
		emit(Event{Type: "error", Error: err.Error()})
		return
	}
	emit(Event{Type: "done", Messages: res.Messages, Pending: res.Pending})
}
