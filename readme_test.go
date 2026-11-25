package sddstudio_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// readme returns the README of this repository.
func readme(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Issue #37: the README names the version it documents, and that version is the one the program reports. Without
// this check the headline quietly drifts from report.ToolVersion — the README would advertise a version that is not
// the one in the binary.
func TestReadmeNamesTheToolVersion(t *testing.T) {
	if want := "**Version `" + report.ToolVersion + "`**"; !strings.Contains(readme(t), want) {
		t.Errorf("README must name %s", want)
	}
}

// Issue #37: the "Stand" section was dissolved — it re-told the version history that CHANGELOG.md owns. A heading of
// that name, or an entry for it in the table of contents, means the duplication is back.
func TestReadmeHasNoStandSection(t *testing.T) {
	text := readme(t)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Stand" || trimmed == "### Stand" {
			t.Errorf("README must not carry a Stand section again: %q", trimmed)
		}
	}
	if strings.Contains(text, "[Stand](#stand)") {
		t.Error("the table of contents must not link a Stand section")
	}
	// The parts that lived in "Stand" have to stay reachable under their own headings.
	for _, want := range []string{"## Regeln", "## Wo weiter", "[Regeln](#regeln)", "[Wo weiter](#wo-weiter)"} {
		if !strings.Contains(text, want) {
			t.Errorf("README must keep %q", want)
		}
	}
}
