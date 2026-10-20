package rules

import (
	"testing"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

// sdd-core-v1 AC-10: completed feature artifacts outside the archive are reported; pre-release escalates to error.
func TestSDD050(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/active.md", "---\nfeature: active\nstatus: active\narchive-after: 2026-10-20\n---\n")
	writeFile(t, root, "docs/prd/done.md", "---\nfeature: done\nstatus: implemented\n---\n")
	writeFile(t, root, "docs/design/expired.md", "---\nfeature: expired\nstatus: active\narchive-after: 2026-10-19\n---\n")
	writeFile(t, root, "docs/plans/badDate.md", "---\nfeature: bad-date\nstatus: draft\narchive-after: v1.2.0\n---\n")
	writeFile(t, root, "docs/plans/note.md", "---\nstatus: implemented\n---\n") // no feature: not a feature artifact
	writeFile(t, root, "docs/archive/ok.md", "---\nfeature: ok\nstatus: archived\n---\n")
	writeFile(t, root, "docs/archive/moved.md", "---\nfeature: moved\nstatus: implemented\n---\n")

	cfg := featureConfig()
	cfg.Artifacts.Archive = "docs/archive"
	ctx := &Context{RepoRoot: root, Config: cfg, Now: func() time.Time { return time.Date(2026, 10, 20, 15, 0, 0, 0, time.Local) }}
	findings, err := checkSDD050(ctx, catalog.Rule{ID: "SDD-050"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range findings {
		got[f.Path] = true
	}
	want := []string{"docs/prd/done.md", "docs/design/expired.md", "docs/plans/badDate.md", "docs/archive/moved.md"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %+v", len(want), findings)
	}
	for _, p := range want {
		if !got[p] {
			t.Errorf("expected finding for %s, got %+v", p, findings)
		}
	}

	// Severity: warn by default, error in pre-release (catalog by_gate).
	rule := catalog.Rule{Severity: catalog.SeveritySpec{Default: "warn", ByGate: map[string]string{"pre-release": "error"}}}
	if rule.EffectiveSeverity("") != "warn" || rule.EffectiveSeverity("pre-merge") != "warn" || rule.EffectiveSeverity("pre-release") != "error" {
		t.Fatalf("unexpected severity mapping")
	}
}
