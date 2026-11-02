package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

// AgentsMarker marks the sdd part of an agent file, so a second run changes nothing.
const AgentsMarker = "<!-- sdd:agents -->"

// AgentsFile is the agent file with the content; CLAUDE.md and GEMINI.md point to it.
const AgentsFile = "AGENTS.md"

// geminiFiles are the places Gemini instructions are kept; GEMINI.md is created only if none exists.
var geminiFiles = []string{"GEMINI.md", ".gemini/GEMINI.md", "gemini/GEMINI.md"}

// CopilotFile holds the instructions for GitHub Copilot. It points to AGENTS.md with a plain Markdown link, because it
// is unclear whether Copilot resolves @-imports like Claude Code and Gemini CLI.
const CopilotFile = ".github/copilot-instructions.md"

type agentsData struct {
	Constitution, ADR, PRD, Design, Plans string
}

// AgentFilesPlan makes AGENTS.md, CLAUDE.md, GEMINI.md and .github/copilot-instructions.md lead to the constitution and the sdd process: missing files
// are created, existing ones without AgentsMarker get a block in front (nothing else changes). It returns the plan and
// one line per change or skipped file.
func AgentFilesPlan(root string, cfg *config.Config) (Plan, []string, error) {
	a := cfg.Artifacts
	data := agentsData{Constitution: a.Constitution, ADR: a.ADR, PRD: a.PRD, Design: a.Design, Plans: a.Plans}
	var plan Plan
	var details []string

	add := func(rel string, create func() ([]byte, error), block func() ([]byte, error)) error {
		current, exists, err := readRegular(root, rel)
		switch {
		case err != nil:
			details = append(details, fmt.Sprintf("%s übersprungen: %v", rel, err))
		case !exists:
			if create == nil {
				return nil
			}
			content, err := create()
			if err != nil {
				return err
			}
			plan.Ops = append(plan.Ops, Op{Path: rel, Content: content})
			details = append(details, rel+" anlegen")
		case bytes.Contains(current, []byte(AgentsMarker)):
			// already leads to sdd
		case bytes.HasPrefix(bytes.TrimPrefix(current, []byte("\xEF\xBB\xBF")), []byte("---")):
			details = append(details, rel+": Frontmatter vorhanden — Verweis von Hand ergänzen")
		default:
			head, err := block()
			if err != nil {
				return err
			}
			plan.Ops = append(plan.Ops, Op{Path: rel, Content: head, Prepend: true, Expect: current})
			details = append(details, rel+" ← Verweis auf die Verfassung bzw. "+AgentsFile)
		}
		return nil
	}
	pointer := func(rel string) func() ([]byte, error) {
		return func() ([]byte, error) {
			target, _ := filepath.Rel(filepath.FromSlash(path.Dir(rel)), AgentsFile)
			ref := "@" + filepath.ToSlash(target)
			if rel == CopilotFile {
				ref = "[" + AgentsFile + "](" + filepath.ToSlash(target) + ")"
			}
			return []byte(AgentsMarker + "\nLies zuerst " + ref + " — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).\n\n"), nil
		}
	}
	created := func(rel string) func() ([]byte, error) {
		return func() ([]byte, error) {
			b, err := pointer(rel)()
			return bytes.TrimSuffix(b, []byte("\n")), err
		}
	}

	if err := add(AgentsFile, func() ([]byte, error) { return render("agents.md.tmpl", data) },
		func() ([]byte, error) { return render("agents-block.md.tmpl", data) }); err != nil {
		return Plan{}, nil, err
	}
	if err := add("CLAUDE.md", created("CLAUDE.md"), pointer("CLAUDE.md")); err != nil {
		return Plan{}, nil, err
	}
	anyGemini := false
	for _, g := range geminiFiles {
		if _, exists, err := readRegular(root, g); exists || err != nil {
			anyGemini = true
		}
	}
	for _, g := range geminiFiles {
		var create func() ([]byte, error)
		if g == "GEMINI.md" && !anyGemini {
			create = created(g)
		}
		if err := add(g, create, pointer(g)); err != nil {
			return Plan{}, nil, err
		}
	}
	if err := add(CopilotFile, created(CopilotFile), pointer(CopilotFile)); err != nil {
		return Plan{}, nil, err
	}
	return plan, details, plan.Validate()
}

// readRegular reads rel below root; exists is false if it is missing. A non-regular file (link, folder) is an error.
func readRegular(root, rel string) ([]byte, bool, error) {
	target := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errors.New("keine reguläre Datei")
	}
	data, err := os.ReadFile(target)
	return data, err == nil, err
}
