package scan_test

import (
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// bmad-import AC-2: type, parent, covers, refined and unplanned are read from the frontmatter without disturbing the
// existing fields; an artifact without them keeps the plain feature behaviour.
func TestParseFrontmatterBMADFields(t *testing.T) {
	data := []byte(`---
feature: epic-import
status: active
type: epic
parent: initiative-gallery
covers: ["R1", "R2"]
refined: false
---

# Epic
`)
	fm, ok := scan.ParseFrontmatter(data)
	if !ok {
		t.Fatal("frontmatter not parsed")
	}
	if fm.Feature != "epic-import" || fm.Status != "active" {
		t.Errorf("existing fields changed: %+v", fm)
	}
	if !fm.IsEpic() || fm.IsInitiative() {
		t.Errorf("type not read: %q", fm.Type)
	}
	if fm.Parent != "initiative-gallery" {
		t.Errorf("parent = %q", fm.Parent)
	}
	if len(fm.Covers) != 2 || fm.Covers[0] != "R1" || fm.Covers[1] != "R2" {
		t.Errorf("covers = %v", fm.Covers)
	}
	if fm.Refined == nil || *fm.Refined {
		t.Fatalf("refined = %v, want false", fm.Refined)
	}
	if !fm.EpicUnrefined() {
		t.Error("EpicUnrefined() = false, want true")
	}
}

// bmad-import AC-2: an initiative lists the requirements it deliberately leaves unplanned.
func TestParseFrontmatterUnplanned(t *testing.T) {
	fm, ok := scan.ParseFrontmatter([]byte("---\ntype: initiative\nunplanned: [\"R7\", \"R8\"]\n---\n# Initiative\n"))
	if !ok || !fm.IsInitiative() {
		t.Fatalf("initiative not recognised: %+v", fm)
	}
	if len(fm.Unplanned) != 2 || fm.Unplanned[1] != "R8" {
		t.Errorf("unplanned = %v", fm.Unplanned)
	}
}

// bmad-import AC-2: a plain feature PRD is neither epic nor initiative, and an absent refined is not "false".
func TestParseFrontmatterPlainFeature(t *testing.T) {
	fm, ok := scan.ParseFrontmatter([]byte("---\nfeature: billing\nstatus: draft\n---\n# PRD\n"))
	if !ok {
		t.Fatal("frontmatter not parsed")
	}
	if fm.IsEpic() || fm.IsInitiative() || fm.EpicUnrefined() {
		t.Errorf("plain PRD got macro fields: %+v", fm)
	}
}
