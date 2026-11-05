package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

// archive-not-draft AC-1: sdd archive refuses a feature with a draft PRD with exit code 2, also with --dry-run, and
// moves nothing.
func TestArchiveRefusesDraftCLI(t *testing.T) {
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	prd := filepath.Join(root, "docs/prd/billing.md")
	if err := os.WriteFile(prd, []byte("---\nfeature: billing\nstatus: draft\narchive-after:\ntouches: []\n---\n\n# PRD\n\n## Nicht-Ziele\n\n- nichts\n\n## Akzeptanzkriterien\n\n- **AC-1** Export\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "billing_test.go"), []byte("// billing AC-1: Export\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{{"--dry-run"}, nil} {
		var out, errOut bytes.Buffer
		args := append([]string{"archive", "--feature", "billing"}, extra...)
		if code := run(args, root, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "status: draft") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut.String())
		}
		if _, err := os.Stat(prd); err != nil {
			t.Fatalf("%v: draft PRD moved", args)
		}
	}
}
