package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

func applyAgents(t *testing.T, root string) []string {
	t.Helper()
	plan, details, err := scaffold.AgentFilesPlan(root, scaffold.DefaultConfig("standard"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	return details
}

// agent-files AC-1: AGENTS.md names constitution, precedence with the configured paths, process, test convention,
// human decisions and sdd mcp.
// agent-files AC-2: CLAUDE.md and GEMINI.md are created pointing to AGENTS.md.
func TestAgentFilesCreate(t *testing.T) {
	root := t.TempDir()
	applyAgents(t, root)
	agents := readFile(t, root, "AGENTS.md")
	for _, want := range []string{scaffold.AgentsMarker, "[`constitution.md`](constitution.md) ist das ranghöchste Artefakt",
		"Verfassung → ADR (`docs/adr`) → Tech Spec (`docs/design`) → PRD (`docs/prd`) → Plan (`docs/plans`)",
		"Tests je Kriterium → Umsetzung", "// <slug> AC-<n>: …", "Der Mensch entscheidet", `"args": ["mcp"]`} {
		if !strings.Contains(agents, want) {
			t.Errorf("AGENTS.md misses %q", want)
		}
	}
	for _, rel := range []string{"CLAUDE.md", "GEMINI.md"} {
		if got := readFile(t, root, rel); got != scaffold.AgentsMarker+"\nLies zuerst @AGENTS.md — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).\n" {
			t.Errorf("%s: %q", rel, got)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".gemini")); err == nil {
		t.Errorf(".gemini created")
	}
}

// agent-files AC-2: a missing .github/copilot-instructions.md is created with a plain Markdown link to ../AGENTS.md.
// agent-files AC-3: an existing one without the marker gets that link in front, the rest unchanged.
func TestAgentFilesCopilot(t *testing.T) {
	const pointer = scaffold.AgentsMarker + "\nLies zuerst [AGENTS.md](../AGENTS.md) — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).\n"
	root := t.TempDir()
	applyAgents(t, root)
	if got := readFile(t, root, scaffold.CopilotFile); got != pointer {
		t.Errorf("created: %q", got)
	}
	root = t.TempDir()
	mustWrite(t, filepath.Join(root, scaffold.CopilotFile), "Prefer table-driven tests.\n")
	applyAgents(t, root)
	if got := readFile(t, root, scaffold.CopilotFile); got != pointer+"\nPrefer table-driven tests.\n" {
		t.Errorf("prepended: %q", got)
	}
	if plan, _, err := scaffold.AgentFilesPlan(root, scaffold.DefaultConfig("standard")); err != nil || len(plan.Ops) != 0 {
		t.Errorf("second run plans %+v, %v", plan.Ops, err)
	}
}

// agent-files AC-2: with .gemini/GEMINI.md (or gemini/GEMINI.md) present, no root GEMINI.md is created; the existing
// file gets the pointer with the relative path.
func TestAgentFilesGeminiVariants(t *testing.T) {
	for _, existing := range []string{".gemini/GEMINI.md", "gemini/GEMINI.md"} {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, existing), "Use Go only.\n")
		applyAgents(t, root)
		if _, err := os.Stat(filepath.Join(root, "GEMINI.md")); err == nil {
			t.Errorf("%s: root GEMINI.md created although %s exists", existing, existing)
		}
		if got := readFile(t, root, existing); got != scaffold.AgentsMarker+"\nLies zuerst @../AGENTS.md — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).\n\nUse Go only.\n" {
			t.Errorf("%s: %q", existing, got)
		}
	}
}

// agent-files AC-3: existing files get the block in front, the rest unchanged; marked files and files with
// frontmatter stay untouched; a second run changes nothing.
func TestAgentFilesPrepend(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "AGENTS.md"), "# Eigene Regeln\n")
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "Bitte knapp antworten.\n")
	mustWrite(t, filepath.Join(root, "GEMINI.md"), "---\ntitle: x\n---\nText\n")
	mustWrite(t, filepath.Join(root, ".gemini/GEMINI.md"), scaffold.AgentsMarker+"\nschon verknüpft\n")
	details := applyAgents(t, root)

	agents := readFile(t, root, "AGENTS.md")
	if !strings.HasPrefix(agents, scaffold.AgentsMarker+"\n> **sdd:** [`constitution.md`](constitution.md) ist das ranghöchste Artefakt") || !strings.HasSuffix(agents, "\n\n# Eigene Regeln\n") {
		t.Errorf("AGENTS.md: %q", agents)
	}
	if got := readFile(t, root, "CLAUDE.md"); got != scaffold.AgentsMarker+"\nLies zuerst @AGENTS.md — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).\n\nBitte knapp antworten.\n" {
		t.Errorf("CLAUDE.md: %q", got)
	}
	if got := readFile(t, root, "GEMINI.md"); got != "---\ntitle: x\n---\nText\n" {
		t.Errorf("GEMINI.md with frontmatter changed: %q", got)
	}
	if !strings.Contains(strings.Join(details, "\n"), "GEMINI.md: Frontmatter vorhanden") {
		t.Errorf("frontmatter not reported: %v", details)
	}
	if got := readFile(t, root, ".gemini/GEMINI.md"); got != scaffold.AgentsMarker+"\nschon verknüpft\n" {
		t.Errorf("marked file changed: %q", got)
	}
	plan, _, err := scaffold.AgentFilesPlan(root, scaffold.DefaultConfig("standard"))
	if err != nil || len(plan.Ops) != 0 {
		t.Errorf("second run plans %+v, %v", plan.Ops, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
