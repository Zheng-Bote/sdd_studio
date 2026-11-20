package config_test

import (
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

// bmad-import AC-1: pipeline accepts the empty value, sdd and bmad; every other value is refused with the allowed
// values named, and artifacts.initiative defaults to docs/initiative.md.
func TestPipelineValues(t *testing.T) {
	for _, value := range []string{"", "sdd", "bmad"} {
		cfg := &config.Config{Version: 1, Profile: "standard", Pipeline: value}
		if err := cfg.Validate(); err != nil {
			t.Errorf("pipeline %q refused: %v", value, err)
		}
	}
	cfg := &config.Config{Version: 1, Profile: "standard", Pipeline: "spec-kit"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "sdd") || !strings.Contains(err.Error(), "bmad") {
		t.Fatalf("unknown pipeline: %v", err)
	}
}

// bmad-import AC-1: the initiative path falls back to the default and can be configured.
func TestInitiativePathDefault(t *testing.T) {
	if got := (config.ArtifactsConfig{}).InitiativePath(); got != config.DefaultInitiative {
		t.Errorf("default = %q, want %q", got, config.DefaultInitiative)
	}
	if got := (config.ArtifactsConfig{Initiative: "_bmad-output/x/initiative-x.md"}).InitiativePath(); got != "_bmad-output/x/initiative-x.md" {
		t.Errorf("configured = %q", got)
	}
}
