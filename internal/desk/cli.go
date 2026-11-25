package desk

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloudNotice is the text the desktop shows before the first turn of an agent CLI. Both offered CLIs are cloud
// clients: the process runs here, the model does not (ADR 0010, decision 6).
const CloudNotice = "Inhalte verlassen das Gerät: Die Agenten-CLI arbeitet mit einem Cloud-Modell."

// DefaultEnvToken is the variable name through which a CLI reads its bearer token.
const DefaultEnvToken = "SDD_PROXY_TOKEN"

// forbiddenFlags must never appear in a start line. They would give the CLI back the write access the desktop is
// required to prevent, or hand it the critical tools (ADR 0010, conditions 5b and 2; agenten-cli-proxy AC-7).
var forbiddenFlags = []string{
	"--dangerously-skip-permissions",
	"--dangerously-bypass-approvals-and-sandbox",
	"--allow-sandbox-elevation",
	"danger-full-access",
	"--allow-critical",
	"--allow-write",
}

// ForbiddenFlag returns the first flag of argv that must not be passed to a CLI, or "".
func ForbiddenFlag(argv []string) string {
	for _, a := range argv {
		for _, f := range forbiddenFlags {
			if a == f || a == "--sandbox="+f {
				return f
			}
		}
	}
	return ""
}

// SessionConfig is what one CLI run needs to reach the desktop's MCP endpoint.
type SessionConfig struct {
	ProxyURL string            // e.g. http://127.0.0.1:54777/mcp
	Token    string            // per-session bearer token
	EnvToken string            // name of the variable that carries the token in the CLI's environment
	WorkDir  string            // project directory
	Extra    map[string]string // CLI-specific values, e.g. "prompt"
}

// CLIStart is one prepared run: argv, extra environment and what has to be undone afterwards.
type CLIStart struct {
	Argv    []string
	Env     []string
	Before  func() error // registers the server in the CLI's own configuration where the CLI needs that
	Cleanup func()       // removes temp files and configuration entries again
}

// CLISpec is one agent CLI the desktop knows.
type CLISpec struct {
	ID     string
	Name   string
	Cloud  bool
	Binary string
	// Start builds a run. It prepares argv, env and files; it runs nothing.
	Start func(SessionConfig) (CLIStart, error)
}

// CLISpecs is the set of CLIs the desktop knows. Knowing is not offering: Offerable decides that from evidence and
// installation.
var CLISpecs = []CLISpec{
	{ID: "claude", Name: "Claude Code", Cloud: true, Binary: "claude", Start: claudeStart},
	{ID: "codewhale", Name: "codewhale", Cloud: true, Binary: "codewhale", Start: codewhaleStart},
	{ID: "agy", Name: "Google Antigravity", Cloud: true, Binary: "agy"},
}

// lookPath is exec.LookPath, replaceable in tests.
var lookPath = exec.LookPath

// Installed reports whether the CLI's binary is on the PATH.
func (s CLISpec) Installed() bool {
	if s.Binary == "" {
		return false
	}
	_, err := lookPath(s.Binary)
	return err == nil
}

// BlockReason names in plain words why a CLI is not offered, or "" when it is. The reasons are ordered by what the
// user can act on: installation first, then proof.
//
// agenten-cli-proxy AC-1
func (s CLISpec) BlockReason(installed bool) string {
	if !installed {
		return fmt.Sprintf("nicht installiert (%s nicht im PATH)", s.Binary)
	}
	// The proof comes before the start line: for agy the real reason is the missing write lock, not a missing
	// table entry. A CLI with proof but no start line is a defect in this table and says so.
	ev := EvidenceForCLI(s.ID)
	if ev.Status != EvidenceProven {
		return "die Bedingungen aus ADR 0010 sind nicht belegt"
	}
	if !ev.ProxyProven {
		return "der Transportbeleg gegen den Proxy steht aus"
	}
	if s.Start == nil {
		return "kein Startvertrag hinterlegt"
	}
	return ""
}

// Offerable reports whether the desktop may offer this CLI.
//
// agenten-cli-proxy AC-1
func (s CLISpec) Offerable(installed bool) bool {
	return s.BlockReason(installed) == ""
}

// claudeStart builds the Claude Code run. --allowedTools is an allowlist: every tool not named is refused without
// asking, which is the documented write lock (docs/notes/headless-vertraege-agenten-clis.md). It also carries the
// only way for the CLI to reach the proxy, so it stays narrow on purpose.
func claudeStart(sc SessionConfig) (CLIStart, error) {
	dir, err := os.MkdirTemp("", "sdd-claude-")
	if err != nil {
		return CLIStart{}, fmt.Errorf("MCP-Konfiguration nicht anlegbar: %w", err)
	}
	cfg := map[string]any{"mcpServers": map[string]any{"sdd": map[string]any{
		"type":    "http",
		"url":     sc.ProxyURL,
		"headers": map[string]string{"Authorization": "Bearer " + sc.Token},
	}}}
	data, err := json.Marshal(cfg)
	if err != nil {
		_ = os.RemoveAll(dir)
		return CLIStart{}, err
	}
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return CLIStart{}, fmt.Errorf("MCP-Konfiguration nicht schreibbar: %w", err)
	}
	// The prompt goes directly after -p and nothing follows the variadic flags: `--mcp-config` and
	// `--allowedTools` both swallow the following words as values, so a prompt placed after them would be read
	// as another config file or another allowed tool (found in the transport proof, 2026-11-25).
	prompt := strings.TrimSpace(sc.Extra["prompt"])
	if prompt == "" {
		_ = os.RemoveAll(dir)
		return CLIStart{}, errors.New("ohne Auftrag startet keine CLI-Sitzung")
	}
	argv := []string{
		"claude", "-p", prompt,
		"--output-format", "stream-json",
		"--mcp-config", path,
		"--allowedTools", "mcp__sdd",
	}
	return CLIStart{Argv: argv, Cleanup: func() { _ = os.RemoveAll(dir) }}, nil
}

// codewhaleStart builds the codewhale run. --allowed-tools is deliberately absent: as an allowlist it also removes
// the discovery path (tool_search), so the permitted MCP tool cannot even be loaded. The write lock rests on
// --sandbox read-only, which the note proves at the operating-system level.
func codewhaleStart(sc SessionConfig) (CLIStart, error) {
	name := "sdd"
	envToken := sc.EnvToken
	if envToken == "" {
		envToken = DefaultEnvToken
	}
	argv := []string{
		"codewhale", "exec", "--auto",
		"--sandbox", "read-only",
		"--output-format", "stream-json",
		"--parent-death-watch",
	}
	if p := sc.Extra["prompt"]; p != "" {
		argv = append(argv, p)
	}
	url := sc.ProxyURL
	return CLIStart{
		Argv: argv,
		Env:  []string{envToken + "=" + sc.Token},
		// codewhale knows servers only through its own configuration; the entry is added for the run and removed
		// afterwards. This touches the user's CLI configuration — see "Offene Punkte" in the tech spec.
		Before: func() error {
			return runQuiet("codewhale", "mcp", "add", name, "--url", url, "--bearer-token-env-var", envToken)
		},
		Cleanup: func() { _ = runQuiet("codewhale", "mcp", "remove", name) },
	}, nil
}

// runQuiet runs a short configuration command and returns its error with the captured output.
func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %w (%s)", name, args, err, string(out))
	}
	return nil
}

// CLISpecByID returns the spec with this ID.
func CLISpecByID(id string) (CLISpec, bool) {
	for _, s := range CLISpecs {
		if s.ID == id {
			return s, true
		}
	}
	return CLISpec{}, false
}
