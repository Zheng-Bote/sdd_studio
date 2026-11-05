package rules

import (
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

const pinnedCatalog = `version: "test"
rules:
  - id: SDD-900
    title: angeheftet
    severity: error
    profile: minimal
    pinned: true
    check: x
    gate_default: [pre-merge]
    baseline: forbidden
  - id: SDD-901
    title: gewöhnlich
    severity: error
    profile: minimal
    check: x
    gate_default: [pre-merge, pre-release]
    baseline: allowed
`

// config-weakening AC-1: a pinned rule stays in every gate of its gate_default even when an override leaves it out;
// other rules follow the override as before.
func TestPinnedSurvivesOverride(t *testing.T) {
	cat, err := catalog.Parse([]byte(pinnedCatalog))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Profile: "standard"}
	cfg.Gates.PreMerge.Override = []string{"SDD-901"}
	cfg.Gates.PreRelease.Override = []string{"SDD-901"}

	merge := EffectiveRulesForGate(cfg, cat, "pre-merge")
	if !merge["SDD-900"] || !merge["SDD-901"] || len(merge) != 2 {
		t.Errorf("pre-merge with override: %v", merge)
	}
	if release := EffectiveRulesForGate(cfg, cat, "pre-release"); release["SDD-900"] {
		t.Errorf("pinned rule added outside its gate_default: %v", release)
	}
	cfg.Gates.PreMerge.Override = []string{"SDD-002a"}
	if merge := EffectiveRulesForGate(cfg, cat, "pre-merge"); merge["SDD-901"] || !merge["SDD-900"] {
		t.Errorf("unpinned rule must follow the override, pinned must stay: %v", merge)
	}
}

// config-weakening AC-2: pinned requires profile minimal and baseline forbidden.
func TestPinnedNeedsMinimalAndForbidden(t *testing.T) {
	for name, cat := range map[string]string{
		"standard profile": strings.Replace(pinnedCatalog, "    profile: minimal\n    pinned: true", "    profile: standard\n    pinned: true", 1),
		"baseline allowed": strings.Replace(pinnedCatalog, "    gate_default: [pre-merge]\n    baseline: forbidden", "    gate_default: [pre-merge]\n    baseline: allowed", 1),
	} {
		if cat == pinnedCatalog {
			t.Fatalf("%s: test setup did not change the catalog", name)
		}
		if _, err := catalog.Parse([]byte(cat)); err == nil || !strings.Contains(err.Error(), "SDD-900 ist pinned") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
