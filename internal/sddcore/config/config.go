package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFileName = "sdd.config.yaml"

var ErrConfigNotFound = errors.New("keine Projektkonfiguration gefunden; `sdd adopt` oder `sdd init` ausführen")

type Config struct {
	Version  int    `yaml:"version"`
	Profile  string `yaml:"profile"`
	Language string `yaml:"language"`
	BaseRef  string `yaml:"base_ref"`
	// Pipeline selects the process shape: "" or "sdd" keeps the plain feature process, "bmad" adds the macro level
	// (initiative and epics) and its rules SDD-080 to SDD-082 (ADR 0009).
	Pipeline   string           `yaml:"pipeline"`
	Artifacts  ArtifactsConfig  `yaml:"artifacts"`
	Tests      TestsConfig      `yaml:"tests"`
	Thresholds ThresholdsConfig `yaml:"thresholds"`
	Gates      GatesConfig      `yaml:"gates"`
	Exclude    []string         `yaml:"exclude"`
	// ConstitutionSections overrides the known section labels of a .sdd constitution (SDD-002b).
	ConstitutionSections []string `yaml:"constitution_sections"`
}

type ArtifactsConfig struct {
	Constitution string   `yaml:"constitution"`
	ADR          string   `yaml:"adr"`
	PRD          string   `yaml:"prd"`
	Design       string   `yaml:"design"`
	Plans        string   `yaml:"plans"`
	Archive      string   `yaml:"archive"`
	Docs         []string `yaml:"docs"`
	// Architecture is the arc42 architecture documentation (ADR 0006); empty means DefaultArchitecture.
	Architecture string `yaml:"architecture"`
	// Initiative is the macro artifact with the numbered requirements R<n> (ADR 0009); empty means DefaultInitiative.
	Initiative string `yaml:"initiative"`
}

// DefaultInitiative is where `sdd init --pipeline bmad` puts the initiative unless configured.
const DefaultInitiative = "docs/initiative.md"

// PipelineBMAD is the pipeline value that enables the macro level and its rules.
const PipelineBMAD = "bmad"

// InitiativePath is the configured initiative, or DefaultInitiative.
func (a ArtifactsConfig) InitiativePath() string {
	if p := strings.TrimSpace(a.Initiative); p != "" {
		return p
	}
	return DefaultInitiative
}

// DefaultArchitecture is where `sdd new --type arc42` puts the architecture documentation unless configured.
const DefaultArchitecture = "docs/arc42.md"

// ArchitecturePath is the configured architecture documentation, or DefaultArchitecture.
func (a ArtifactsConfig) ArchitecturePath() string {
	if p := strings.TrimSpace(a.Architecture); p != "" {
		return p
	}
	return DefaultArchitecture
}

type TestsConfig struct {
	Include []string `yaml:"include"`
}

type ThresholdsConfig struct {
	Layers                       []string                `yaml:"layers"`
	FeatureArtifactsRequiredWhen FeatureThresholdTrigger `yaml:"feature_artifacts_required_when"`
}

type FeatureThresholdTrigger struct {
	MinLayersTouched int      `yaml:"min_layers_touched"`
	TouchedPaths     []string `yaml:"touched_paths"`
}

type GateOverride struct {
	Override []string `yaml:"override"`
}

type GatesConfig struct {
	PreImplementation GateOverride `yaml:"pre_implementation"`
	PreMerge          GateOverride `yaml:"pre_merge"`
	PreRelease        GateOverride `yaml:"pre_release"`
}

// Load reads and validates sdd.config.yaml from repoRoot.
// If the file does not exist, ErrConfigNotFound is returned (resulting in Exit 2).
func Load(repoRoot string) (*Config, error) {
	cfgPath := filepath.Join(repoRoot, DefaultConfigFileName)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("Konfiguration %s nicht lesbar: %w", DefaultConfigFileName, err)
	}
	return Parse(data)
}

// Parse decodes and validates the content of sdd.config.yaml, e.g. a draft before it is written.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Konfiguration %s ungültig: %w", DefaultConfigFileName, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("ungültige Konfigurationsversion %d (erwartet: 1)", c.Version)
	}
	switch c.Profile {
	case "minimal", "standard", "regulated":
	default:
		return fmt.Errorf("ungültiges Profil %q (erlaubt: minimal, standard, regulated)", c.Profile)
	}
	switch strings.TrimSpace(c.Pipeline) {
	case "", "sdd", PipelineBMAD:
	default:
		return fmt.Errorf("ungültige Pipeline %q (erlaubt: sdd, bmad oder leer)", c.Pipeline)
	}
	for _, l := range c.ConstitutionSections {
		if strings.TrimSpace(l) == "" {
			return fmt.Errorf("constitution_sections enthält einen leeren Eintrag")
		}
	}
	return nil
}

// GateOverrideFor returns the configured override list for a gate ("pre-implementation", "pre-merge", "pre-release").
func (c *Config) GateOverrideFor(gate string) []string {
	switch gate {
	case "pre-implementation", "pre_implementation":
		return c.Gates.PreImplementation.Override
	case "pre-merge", "pre_merge":
		return c.Gates.PreMerge.Override
	case "pre-release", "pre_release":
		return c.Gates.PreRelease.Override
	default:
		return nil
	}
}
