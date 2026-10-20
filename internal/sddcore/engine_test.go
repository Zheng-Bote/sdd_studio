package sddcore_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
)

// sdd-core-v1 AC-3: the sdd repository passes its own lint.
func TestSelfDogfoodingClean(t *testing.T) {
	// Run sdd check against the repository root itself.
	repoRoot := filepath.Clean("../..")
	rep, err := sddcore.RunCheck(sddcore.RunOptions{
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatalf("RunCheck failed on repo root: %v", err)
	}
	if rep.ExitCode() != 0 || len(rep.Findings) != 0 {
		t.Fatalf("expected 0 findings on repo root, got %+v", rep.Findings)
	}
}

// sdd-core-v1 AC-1: a missing sdd.config.yaml is an execution error, never silent defaults.
func TestMissingConfigReturnsErrConfigNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := sddcore.RunCheck(sddcore.RunOptions{
		RepoRoot: tmp,
	})
	if !errors.Is(err, config.ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

// sdd-core-v1 AC-2: SDD-070..073 report broken catalog rules.
func TestSelfChecksDetectViolations(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Profile: "standard",
	}
	cat := &catalog.Catalog{
		Version: "1.0.0",
		Rules: []catalog.Rule{
			{
				ID:          "SDD-070",
				Severity:    catalog.SeveritySpec{Default: "error"},
				Profile:     "minimal",
				Check:       "catalog_handbook_section",
				GateDefault: []string{"pre-merge"},
				Baseline:    "forbidden",
				Contract:    "handbuch/03-mechanismen.md#sdd-070-vertrag",
				Handbook:    "handbuch/03-mechanismen.md#sdd-070",
			},
			{
				ID:          "SDD-071",
				Severity:    catalog.SeveritySpec{Default: "error"},
				Profile:     "minimal",
				Check:       "catalog_check_registered",
				GateDefault: []string{"pre-merge"},
				Baseline:    "forbidden",
				Contract:    "handbuch/03-mechanismen.md#sdd-071-vertrag",
				Handbook:    "handbuch/03-mechanismen.md#sdd-071",
			},
			{
				ID:          "SDD-072",
				Severity:    catalog.SeveritySpec{Default: "error"},
				Profile:     "minimal",
				Check:       "catalog_gate_coverage",
				GateDefault: []string{"pre-merge"},
				Baseline:    "forbidden",
				Contract:    "handbuch/03-mechanismen.md#sdd-072-vertrag",
				Handbook:    "handbuch/03-mechanismen.md#sdd-072",
			},
			{
				ID:          "SDD-073",
				Severity:    catalog.SeveritySpec{Default: "error"},
				Profile:     "minimal",
				Check:       "catalog_error_contract",
				GateDefault: []string{"pre-merge"},
				Baseline:    "forbidden",
				Contract:    "handbuch/03-mechanismen.md#sdd-073-vertrag",
				Handbook:    "handbuch/03-mechanismen.md#sdd-073",
			},
			{
				// Deliberately broken rule triggering SDD-070, SDD-071, SDD-072, SDD-073
				ID:          "SDD-999",
				Severity:    catalog.SeveritySpec{Default: "error"},
				Profile:     "standard",
				Check:       "non_existent_check",
				GateDefault: nil, // not in any gate -> SDD-072
				Baseline:    "allowed",
				Contract:    "", // missing contract -> SDD-073
				Handbook:    "handbuch/03-mechanismen.md#sdd-999",
			},
		},
	}

	reg := rules.NewRegistry()
	ctx := &rules.Context{
		RepoRoot:    filepath.Clean("../.."),
		Config:      cfg,
		Catalog:     cat,
		CatalogPath: "catalog/catalog.yaml",
	}

	findings, err := rules.Evaluate(ctx, reg)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	triggered := make(map[string]bool)
	for _, f := range findings {
		triggered[f.RuleID] = true
	}
	for _, expected := range []string{"SDD-070", "SDD-071", "SDD-072", "SDD-073"} {
		if !triggered[expected] {
			t.Errorf("expected rule %s to trigger on broken rule SDD-999, got findings: %+v", expected, findings)
		}
	}
}

// sdd-core-v1 AC-4: gate-bound baseline with deterministic ordinal; baseline: forbidden is never suppressed.
func TestBaselineRespectsForbiddenAndOrdinal(t *testing.T) {
	tmp := t.TempDir()
	cat := &catalog.Catalog{
		Version: "1.0.0",
		Rules: []catalog.Rule{
			{ID: "SDD-021", Baseline: "allowed"},
			{ID: "SDD-070", Baseline: "forbidden"},
		},
	}

	rawFindings := []report.Finding{
		{
			RuleID:      "SDD-021",
			Severity:    "error",
			Path:        "docs/prd/a.md",
			Line:        20,
			ByteOffset:  200,
			Message:     "AC fehlt",
			ContentHash: rules.HashContent("AC fehlt"),
		},
		{
			RuleID:      "SDD-021",
			Severity:    "error",
			Path:        "docs/prd/a.md",
			Line:        10,
			ByteOffset:  100,
			Message:     "AC fehlt",
			ContentHash: rules.HashContent("AC fehlt"),
		},
		{
			RuleID:      "SDD-070",
			Severity:    "error",
			Path:        "catalog/catalog.yaml",
			Line:        1,
			ByteOffset:  10,
			Message:     "Handbuch fehlt",
			ContentHash: rules.HashContent("Handbuch fehlt"),
		},
	}

	entries := baseline.BuildEntriesFromFindings(cat, "pre-merge", rawFindings)
	if len(entries) != 2 {
		t.Fatalf("expected 2 baselineable entries (SDD-070 is forbidden), got %d", len(entries))
	}
	if entries[0].Ordinal != 0 || entries[0].Line != 10 || entries[1].Ordinal != 1 || entries[1].Line != 20 {
		t.Fatalf("expected deterministic ordinals sorted by ByteOffset, got %+v", entries)
	}

	bf := &baseline.File{Version: 1, Entries: entries}
	if err := baseline.Save(tmp, bf); err != nil {
		t.Fatalf("Save baseline failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".sdd", "baseline.json")); err != nil {
		t.Fatalf("expected .sdd/baseline.json to exist: %v", err)
	}

	applied, suppressed := baseline.Apply(bf, cat, "pre-merge", rawFindings)
	if suppressed != 2 {
		t.Fatalf("expected 2 suppressed findings, got %d", suppressed)
	}
	rep := report.Build("1.0.0", "standard", "pre-merge", "", true, suppressed, applied)
	if rep.Summary.Error != 1 {
		t.Fatalf("expected 1 remaining error (from forbidden SDD-070), got %d", rep.Summary.Error)
	}
}
