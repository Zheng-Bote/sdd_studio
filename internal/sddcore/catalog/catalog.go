package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	sddstudio "github.com/Zheng-Bote/sdd_studio"
	"gopkg.in/yaml.v3"
)

const DefaultCatalogRelPath = "catalog/catalog.yaml"

var AllGates = []string{"pre-implementation", "pre-merge", "pre-release"}

type Catalog struct {
	Version string `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

type Rule struct {
	ID          string       `yaml:"id"`
	Title       string       `yaml:"title"`
	Severity    SeveritySpec `yaml:"severity"`
	Profile     string       `yaml:"profile"`
	Check       string       `yaml:"check"`
	GateDefault []string     `yaml:"gate_default"`
	Baseline    string       `yaml:"baseline"`
	Contract    string       `yaml:"contract"`
	Rationale   string       `yaml:"rationale"`
	Handbook    string       `yaml:"handbook"`
}

// SeveritySpec normalizes both short syntax (`severity: error`) and object syntax
// (`severity: {default: warn, by_gate: {pre-release: error}}`) into a single structure.
type SeveritySpec struct {
	Default string            `yaml:"default"`
	ByGate  map[string]string `yaml:"by_gate"`
}

func (s *SeveritySpec) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var scalar string
		if err := value.Decode(&scalar); err != nil {
			return err
		}
		s.Default = scalar
		s.ByGate = nil
		return nil
	case yaml.MappingNode:
		type rawSeverity struct {
			Default string            `yaml:"default"`
			ByGate  map[string]string `yaml:"by_gate"`
		}
		var raw rawSeverity
		if err := value.Decode(&raw); err != nil {
			return err
		}
		s.Default = raw.Default
		s.ByGate = raw.ByGate
		return nil
	default:
		return fmt.Errorf("ungültiges Format für severity in Zeile %d", value.Line)
	}
}

// EffectiveSeverity returns the rule's severity for the given gate (or "" for default lint).
// Returns "" if neither by_gate[gate] nor default is set.
func (r Rule) EffectiveSeverity(gate string) string {
	if gate != "" && r.Severity.ByGate != nil {
		if sev, ok := r.Severity.ByGate[gate]; ok && sev != "" {
			return sev
		}
	}
	return r.Severity.Default
}

// HasErrorSeverity returns true if the rule can produce an "error" finding in default lint or any gate.
func (r Rule) HasErrorSeverity() bool {
	if r.Severity.Default == "error" {
		return true
	}
	for _, sev := range r.Severity.ByGate {
		if sev == "error" {
			return true
		}
	}
	return false
}

// IsActiveInProfile checks whether the rule applies to the configured project profile.
func (r Rule) IsActiveInProfile(projectProfile string) bool {
	return profileRank(r.Profile) <= profileRank(projectProfile)
}

func profileRank(profile string) int {
	switch profile {
	case "minimal":
		return 1
	case "standard":
		return 2
	case "regulated":
		return 3
	default:
		return 99
	}
}

// Load loads the rule catalog from repoRoot/catalog/catalog.yaml if it exists on disk,
// otherwise falls back to the embedded catalog.yaml.
func Load(repoRoot string) (*Catalog, string, error) {
	diskPath := filepath.Join(repoRoot, filepath.FromSlash(DefaultCatalogRelPath))
	if data, err := os.ReadFile(diskPath); err == nil {
		cat, err := Parse(data)
		return cat, DefaultCatalogRelPath, err
	}

	data, err := fs.ReadFile(sddstudio.EmbeddedAssets, DefaultCatalogRelPath)
	if err != nil {
		return nil, DefaultCatalogRelPath, fmt.Errorf("eingebetteter Katalog nicht lesbar: %w", err)
	}
	cat, err := Parse(data)
	return cat, DefaultCatalogRelPath, err
}

func Parse(data []byte) (*Catalog, error) {
	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("Regelkatalog ungültig: %w", err)
	}
	return &cat, nil
}
