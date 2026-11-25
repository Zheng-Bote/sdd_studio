package sddstudio_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #33: AGENTS.md names how work is done — issue first, then a feature branch, then a pull request that closes
// the issue. This test holds the sentences in place, in the template (what new projects get) and in this repository's
// own file (which AgentFilesPlan does not rewrite, because it already carries the marker).
func TestAgentsFilesNameTheWorkingMode(t *testing.T) {
	readDocs := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return string(data)
	}

	full := []string{
		"## Arbeitsweise",
		"nie direkt auf dem Standard-Branch",
		"Closes #<n>",
		"Feature-Branch",
		".github/ISSUE_TEMPLATE/",
		"Mergen entscheidet der Mensch",
	}
	for _, path := range []string{
		filepath.Join("templates", "agents.md.tmpl"),
		"AGENTS.md",
	} {
		text := readDocs(path)
		for _, want := range full {
			if !strings.Contains(text, want) {
				t.Errorf("%s must name %q", path, want)
			}
		}
	}

	// The short block is prepended to an agent file that already has content; one sentence has to carry it.
	block := readDocs(filepath.Join("templates", "agents-block.md.tmpl"))
	for _, want := range []string{"Arbeitsweise", "Feature-Branch", "Closes #<n>"} {
		if !strings.Contains(block, want) {
			t.Errorf("agents-block.md.tmpl must name %q", want)
		}
	}

	// The repository's own AGENTS.md is the rendered template; it must not drift from it.
	rendered := readDocs("AGENTS.md")
	if strings.Contains(rendered, "{{") {
		t.Error("AGENTS.md still contains template markers")
	}
}

// Issue #33: the issue template fits this project — acceptance criteria as AC-<n>, non-goals, a reference to the
// constitution. It replaces a template from a different project (EARS, PostgreSQL, SPDX per layer), whose terms must
// not come back.
func TestIssueTemplateFitsThisProject(t *testing.T) {
	dir := filepath.Join(".github", "ISSUE_TEMPLATE")
	data, err := os.ReadFile(filepath.Join(dir, "feature.md"))
	if err != nil {
		t.Fatalf("feature.md: %v", err)
	}
	text := string(data)

	front := []string{"name:", "about:", "labels: 'feature'"}
	for _, want := range front {
		if !strings.Contains(text, want) {
			t.Errorf("issue template front matter must name %q", want)
		}
	}

	want := []string{
		"Feature-Slug",
		"## Problem",
		"## Ziel",
		"## Nicht-Ziele",
		"## Akzeptanzkriterien",
		"AC-1",
		"constitution.md",
		"Closes #<n>",
	}
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("issue template must name %q", w)
		}
	}

	// Terms of the template this one replaces. If one of them returns, the wrong template came back.
	foreign := []string{"EARS", "PostgreSQL", "Envelope Encryption", "SPDX", "mitm-2", "spec-kit"}
	for _, f := range foreign {
		if strings.Contains(text, f) {
			t.Errorf("issue template carries %q from a different project", f)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "feature_spec.md")); err == nil {
		t.Error("the old feature_spec.md is still there")
	}
}
