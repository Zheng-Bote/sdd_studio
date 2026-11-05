package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sdd-archive AC-4: sdd_archive refuses untested criteria, previews every move with its diff and moves only with
// --allow-write.
func TestArchiveTool(t *testing.T) {
	root := project(t) // billing: AC-1 tested, AC-2 not
	if r := call(t, newServer(root, true), "sdd_archive", map[string]any{"feature": "billing", "apply": true}); r["isError"] != true || !strings.Contains(errText(r), "AC-2 ohne Testverweis") {
		t.Fatalf("untested AC-2 must block: %v", r)
	}
	write(t, root, "api/export_test.go", "// billing AC-1: export\n// billing AC-2: import\n")

	prev := structured(t, call(t, newServer(root, false), "sdd_archive", map[string]any{"feature": "billing", "release": "v1.0.0"}))
	ops := prev["ops"].([]any)
	if len(ops) != 3 {
		t.Fatalf("preview: %v", prev)
	}
	first := ops[0].(map[string]any)
	if first["action"] != "move" || first["from"] != "docs/prd/billing.md" || first["path"] != "docs/archive/prd/billing.md" ||
		!strings.Contains(first["diff"].(string), "+status: archived") || !strings.Contains(first["diff"].(string), "+release: v1.0.0") {
		t.Errorf("preview op: %v", first)
	}
	if r := call(t, newServer(root, false), "sdd_archive", map[string]any{"feature": "billing", "apply": true}); r["isError"] != true {
		t.Errorf("apply without --allow-write: %v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err != nil {
		t.Fatalf("moved without --allow-write")
	}

	res := structured(t, call(t, newServer(root, true), "sdd_archive", map[string]any{"feature": "billing", "apply": true}))
	if r := res["results"].([]any)[0].(map[string]any); res["applied"] != true || r["status"] != "moved" || r["from"] != "docs/prd/billing.md" {
		t.Errorf("result: %v", res)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err == nil {
		t.Errorf("source still exists")
	}
	if !strings.Contains(readFileT(t, root, "docs/archive/prd/billing.md"), "status: archived") {
		t.Errorf("archived PRD lacks status: archived")
	}
}

// archive-not-draft AC-2: sdd_archive refuses a feature with a draft PRD, as preview and with apply.
func TestArchiveToolRefusesDraft(t *testing.T) {
	root := project(t)
	write(t, root, "api/export_test.go", "// billing AC-1: export\n// billing AC-2: import\n")
	prd := readFileT(t, root, "docs/prd/billing.md")
	write(t, root, "docs/prd/billing.md", strings.Replace(prd, "status: active", "status: draft", 1))
	if readFileT(t, root, "docs/prd/billing.md") == prd {
		t.Fatal("test setup: billing PRD has no status: active")
	}
	for _, apply := range []bool{false, true} {
		r := call(t, newServer(root, true), "sdd_archive", map[string]any{"feature": "billing", "apply": apply})
		if r["isError"] != true || !strings.Contains(errText(r), "status: draft") {
			t.Errorf("apply %v: %v", apply, r)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err != nil {
		t.Fatal("draft PRD moved")
	}
}
