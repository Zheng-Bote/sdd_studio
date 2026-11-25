package desk

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ErrCLINotOfferable is returned when a session is requested for a CLI that must not be offered.
var ErrCLINotOfferable = errors.New("diese CLI wird nicht angeboten")

// CLIRun is one running CLI session. Its lifetime bounds its cards: when it ends, every open call is withdrawn
// (ADR 0010, consequence "abort and timeout across process boundaries"; agenten-cli-proxy AC-5).
type CLIRun struct {
	ID      string // session id
	CLI     string
	proxy   *CLIProxy
	start   CLIStart
	cmd     *exec.Cmd
	done    chan struct{}
	once    sync.Once
	waitErr error
}

// CLIProxy bundles what a run needs of the desktop.
type CLIProxy struct {
	Approvals *Approvals
	Sessions  *CLISessions
	URL       string
}

// CLIManager starts and tracks CLI sessions.
type CLIManager struct {
	Proxy   *CLIProxy
	WorkDir string
	// Env is the base environment of the child; nil means os.Environ().
	Env []string
	// LookPath and StartSpec let tests replace binary discovery and the start line.
	LookPath  func(string) (string, error)
	StartSpec func(CLISpec, SessionConfig) (CLIStart, error)

	mu   sync.Mutex
	runs map[string]*CLIRun
}

// NewCLIManager returns a manager over one proxy.
func NewCLIManager(proxy *CLIProxy, workDir string) *CLIManager {
	return &CLIManager{Proxy: proxy, WorkDir: workDir, runs: map[string]*CLIRun{}}
}

func (m *CLIManager) lookPath() func(string) (string, error) {
	if m.LookPath != nil {
		return m.LookPath
	}
	return lookPath
}

func (m *CLIManager) startSpec() func(CLISpec, SessionConfig) (CLIStart, error) {
	if m.StartSpec != nil {
		return m.StartSpec
	}
	return func(s CLISpec, sc SessionConfig) (CLIStart, error) { return s.Start(sc) }
}

// Start opens a session for one CLI and runs it. Without the cloud notice acknowledged it refuses (AC-6); for a CLI
// that is not offerable it refuses with the documented reason (AC-1).
//
// agenten-cli-proxy AC-1, AC-6, AC-7
func (m *CLIManager) Start(cliID, prompt string, disclosureAck bool) (*CLIRun, error) {
	spec, ok := CLISpecByID(cliID)
	if !ok {
		return nil, fmt.Errorf("%w: unbekannte CLI %q", ErrCLINotOfferable, cliID)
	}
	// Installation is checked through the same seam tests replace.
	_, lpErr := m.lookPath()(spec.Binary)
	if reason := spec.BlockReason(lpErr == nil); reason != "" {
		return nil, fmt.Errorf("%w: %s", ErrCLINotOfferable, reason)
	}
	if spec.Cloud && !disclosureAck {
		return nil, ErrDisclosure
	}

	session, token, err := m.Proxy.Sessions.Open(spec.ID)
	if err != nil {
		return nil, err
	}
	sc := SessionConfig{
		ProxyURL: m.Proxy.URL,
		Token:    token,
		EnvToken: DefaultEnvToken,
		WorkDir:  m.WorkDir,
		Extra:    map[string]string{"prompt": prompt},
	}
	start, err := m.startSpec()(spec, sc)
	if err != nil {
		m.Proxy.Sessions.Close(session)
		return nil, err
	}
	if bad := ForbiddenFlag(start.Argv); bad != "" {
		m.Proxy.Sessions.Close(session)
		return nil, fmt.Errorf("Startzeile enthält ein verbotenes Flag %q; die CLI wird nicht gestartet", bad)
	}

	argv := start.Argv
	if len(argv) == 0 {
		m.Proxy.Sessions.Close(session)
		return nil, errors.New("leere Startzeile")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = m.WorkDir
	env := m.Env
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = append(append([]string{}, env...), start.Env...)
	cmd.Stderr = os.Stderr
	if start.Before != nil {
		if err := start.Before(); err != nil {
			m.Proxy.Sessions.Close(session)
			return nil, err
		}
	}
	if err := cmd.Start(); err != nil {
		if start.Cleanup != nil {
			start.Cleanup()
		}
		m.Proxy.Sessions.Close(session)
		return nil, fmt.Errorf("%s nicht startbar: %w", spec.Binary, err)
	}

	run := &CLIRun{ID: session, CLI: spec.ID, proxy: m.Proxy, start: start, cmd: cmd, done: make(chan struct{})}
	m.mu.Lock()
	if m.runs == nil {
		m.runs = map[string]*CLIRun{}
	}
	m.runs[session] = run
	m.mu.Unlock()
	go func() {
		run.waitErr = cmd.Wait()
		run.end("CLI beendet")
	}()
	return run, nil
}

// end withdraws the session's cards, closes its token and runs the cleanup exactly once.
func (r *CLIRun) end(reason string) {
	r.once.Do(func() {
		r.proxy.Approvals.WithdrawSession(r.ID, reason)
		r.proxy.Sessions.Close(r.ID)
		if r.start.Cleanup != nil {
			r.start.Cleanup()
		}
		close(r.done)
	})
}

// Done is closed when the CLI process has ended.
func (r *CLIRun) Done() <-chan struct{} { return r.done }

// Close ends the session: it stops the process, withdraws open cards and closes the token.
func (r *CLIRun) Close() error {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
	select {
	case <-r.done:
	case <-time.After(3 * time.Second):
	}
	r.end("Sitzung beendet")
	return nil
}

// Runs lists the running sessions.
func (m *CLIManager) Runs() []*CLIRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*CLIRun, 0, len(m.runs))
	for _, r := range m.runs {
		out = append(out, r)
	}
	return out
}

// CloseAll ends every running session; the desktop calls it on shutdown so no child outlives it (AC-5).
//
// agenten-cli-proxy AC-5
func (m *CLIManager) CloseAll() {
	m.mu.Lock()
	runs := make([]*CLIRun, 0, len(m.runs))
	for _, r := range m.runs {
		runs = append(runs, r)
	}
	m.runs = map[string]*CLIRun{}
	m.mu.Unlock()
	for _, r := range runs {
		_ = r.Close()
	}
}
