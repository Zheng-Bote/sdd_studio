package scaffold

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

// WorkflowPath is where `sdd init` places the CI workflow.
const WorkflowPath = ".github/workflows/sdd.yml"

// InitOptions configures `sdd init`.
type InitOptions struct {
	Profile string // minimal, standard or regulated
	Project string // project name used in the constitution heading
	CI      bool   // write the GitHub Actions workflow
}

// DefaultArtifacts are the artifact paths `sdd init` writes and `sdd new` falls back to without a configuration.
func DefaultArtifacts() config.ArtifactsConfig {
	return config.ArtifactsConfig{
		Constitution: "constitution.md",
		ADR:          "docs/adr",
		PRD:          "docs/prd",
		Design:       "docs/design",
		Plans:        "docs/plans",
		Archive:      "docs/archive",
		Docs:         []string{"docs/notes"},
	}
}

// DefaultConfig is the configuration `sdd init` writes for the given profile.
func DefaultConfig(profile string) *config.Config {
	return &config.Config{
		Version:   1,
		Profile:   profile,
		Language:  "de",
		Artifacts: DefaultArtifacts(),
		Thresholds: config.ThresholdsConfig{
			FeatureArtifactsRequiredWhen: config.FeatureThresholdTrigger{MinLayersTouched: 2},
		},
		Exclude: []string{"vendor/**", "node_modules/**", ".git/**"},
	}
}

// InitPlan builds the plan for a new project: configuration, constitution, artifact directories, CI workflow and an empty baseline.
func InitPlan(opts InitOptions) (Plan, error) {
	cfg := DefaultConfig(opts.Profile)
	if err := cfg.Validate(); err != nil {
		return Plan{}, err
	}
	data := struct {
		Profile   string
		Project   string
		Artifacts config.ArtifactsConfig
	}{opts.Profile, opts.Project, cfg.Artifacts}

	var plan Plan
	for _, f := range []struct{ path, tmpl string }{
		{config.DefaultConfigFileName, "config.yaml.tmpl"},
		{cfg.Artifacts.Constitution, "constitution.md.tmpl"},
	} {
		content, err := render(f.tmpl, data)
		if err != nil {
			return Plan{}, err
		}
		plan.Ops = append(plan.Ops, Op{Path: f.path, Content: content})
	}

	dirs := []string{cfg.Artifacts.ADR, cfg.Artifacts.PRD, cfg.Artifacts.Design, cfg.Artifacts.Plans, cfg.Artifacts.Archive}
	dirs = append(dirs, cfg.Artifacts.Docs...)
	for _, d := range dirs {
		plan.Ops = append(plan.Ops, Op{Path: path.Join(d, ".gitkeep"), KeepDir: true})
	}

	if opts.CI {
		content, err := render("workflow.yml.tmpl", data)
		if err != nil {
			return Plan{}, err
		}
		plan.Ops = append(plan.Ops, Op{Path: WorkflowPath, Content: content})
	}

	bl, err := json.MarshalIndent(baseline.File{Version: 1, Entries: []baseline.Entry{}}, "", "  ")
	if err != nil {
		return Plan{}, fmt.Errorf("Baseline nicht serialisierbar: %w", err)
	}
	plan.Ops = append(plan.Ops, Op{Path: baseline.DefaultBaselineRelPath, Content: append(bl, '\n')})
	return plan, nil
}
