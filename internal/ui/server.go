// Package ui serves `sdd ui`: a local web view on the same library the CLI uses (concept sections 8.6 and 10).
package ui

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

//go:embed web/*
var webFS embed.FS

// maxBody limits JSON request bodies of write endpoints.
const maxBody = 64 << 10

// Listen binds to 127.0.0.1 only — never "localhost", which may resolve to ::1 or a wildcard address.
// Port 0 picks a free port.
func Listen(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

// NewToken returns a random session token for write requests.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Server is the UI handler for one repository, one listener address and one session token.
type Server struct {
	Root    string
	Addr    string // host:port the listener is bound to; the only accepted Host header
	Token   string
	Project string
	// Now and Exec are injectable for tests.
	Now  func() time.Time
	Exec func(name string, args ...string) error

	mux *http.ServeMux
}

// New wires the routes.
func New(root, addr, token, project string) *Server {
	s := &Server{Root: root, Addr: addr, Token: token, Project: project, Now: time.Now, Exec: startDetached}
	static, _ := fs.Sub(webFS, "web")
	m := http.NewServeMux()
	m.Handle("GET /", http.FileServerFS(static))
	m.HandleFunc("GET /handbook", s.handleHandbook)
	m.HandleFunc("GET /api/meta", s.handleMeta)
	m.HandleFunc("GET /api/overview", s.handleOverview)
	m.HandleFunc("GET /api/findings", s.handleFindings)
	m.HandleFunc("GET /api/artifacts", s.handleArtifacts)
	m.HandleFunc("GET /api/adopt", s.handleAdoptPreview)
	m.HandleFunc("POST /api/new", s.requireWrite(s.handleNew))
	m.HandleFunc("POST /api/adopt/apply", s.requireWrite(s.handleAdoptApply))
	m.HandleFunc("POST /api/open", s.requireWrite(s.handleOpen))
	s.mux = m
	return s
}

// ServeHTTP enforces the Host check and the security headers for every request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host != s.Addr {
		http.Error(w, "unerwarteter Host", http.StatusForbidden)
		return
	}
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	s.mux.ServeHTTP(w, r)
}

// requireWrite admits a write request only with the session token, the own Origin and a JSON body.
func (s *Server) requireWrite(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-SDD-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.Token)) != 1 {
			writeError(w, http.StatusForbidden, errors.New("Sitzungstoken fehlt oder ist ungültig"))
			return
		}
		if r.Header.Get("Origin") != "http://"+s.Addr {
			writeError(w, http.StatusForbidden, errors.New("unerwarteter Origin"))
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type application/json erforderlich"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("ungültige Anfrage: %w", err)
	}
	return nil
}
