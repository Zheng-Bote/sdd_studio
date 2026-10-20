// Package adopt takes over an existing repository: inventory, classification, maturity, plan and application on a branch.
package adopt

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Kind is one of the six artifact types (concept section 4.3).
type Kind string

const (
	KindConstitution Kind = "Verfassung"
	KindADR          Kind = "ADR"
	KindPRD          Kind = "PRD"
	KindDesign       Kind = "Tech Spec"
	KindPlan         Kind = "Plan"
	KindDoc          Kind = "Doku/Exploration"
)

// IsFeature reports whether the kind is a feature artifact (PRD, Tech Spec, Plan).
func (k Kind) IsFeature() bool {
	return k == KindPRD || k == KindDesign || k == KindPlan
}

// File is one classified Markdown file.
type File struct {
	Path           string
	Kind           Kind
	Reason         string
	Archived       bool
	Frontmatter    scan.Frontmatter
	HasFrontmatter bool
}

// Inventory is everything adopt knows about the repository before changing it.
type Inventory struct {
	Config             *config.Config // loaded or proposed
	ConfigExists       bool
	Layout             string // how the artifact paths were determined
	ConstitutionExists bool
	BaselineExists     bool
	Files              []File
	Workflows          []string
	CallsGate          bool // a CI workflow runs `sdd gate`
}

var (
	adrFileRegex  = regexp.MustCompile(`^[0-9]{4}-.*\.md$`)
	gateCallRegex = regexp.MustCompile(`sdd(?:\.exe)?["']?\s+gate\b|sdd_studio@`)
)

// adrCandidates and constitutionCandidates are probed in this order when no configuration exists.
var (
	adrCandidates          = []string{"docs/adr", "docs/architecture/adr", "doc/adr", "adr"}
	constitutionCandidates = []string{"constitution.md", "CONSTITUTION.md", "docs/constitution.md"}
	workflowGlobs          = []string{".github/workflows/*.yml", ".github/workflows/*.yaml", ".gitlab-ci.yml"}
)

// Take builds the inventory. Without sdd.config.yaml a configuration is proposed from the detected layout.
func Take(root, profile string) (*Inventory, error) {
	inv := &Inventory{}
	cfg, err := config.Load(root)
	switch {
	case err == nil:
		inv.Config, inv.ConfigExists, inv.Layout = cfg, true, "vorhandene sdd.config.yaml"
	case errors.Is(err, config.ErrConfigNotFound):
		if inv.Config, inv.Layout, err = propose(root, profile); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	if c := strings.TrimSpace(inv.Config.Artifacts.Constitution); c != "" {
		if clean, err := scan.CleanRel(c); err == nil {
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(clean))); err == nil && info.Mode().IsRegular() {
				inv.ConstitutionExists = true
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(baseline.DefaultBaselineRelPath))); err == nil {
		inv.BaselineExists = true
	}

	exclude := append(append([]string{}, inv.Config.Exclude...), ".sdd/**")
	mdFiles, err := scan.WalkFiles(root, []string{"**/*.md"}, exclude)
	if err != nil {
		return nil, fmt.Errorf("Inventar nicht lesbar: %w", err)
	}
	for _, p := range mdFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return nil, fmt.Errorf("%s nicht lesbar: %w", p, err)
		}
		f := Classify(p, inv.Config)
		f.Frontmatter, f.HasFrontmatter = scan.ParseFrontmatter(data)
		inv.Files = append(inv.Files, f)
	}

	if inv.Workflows, err = scan.WalkFiles(root, workflowGlobs, nil); err != nil {
		return nil, fmt.Errorf("CI-Konfiguration nicht lesbar: %w", err)
	}
	action, _ := os.ReadFile(filepath.Join(root, "action.yml"))
	localActionRunsGate := strings.Contains(string(action), "sdd gate")
	for _, w := range inv.Workflows {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(w)))
		if err != nil {
			return nil, fmt.Errorf("%s nicht lesbar: %w", w, err)
		}
		text := string(data)
		if gateCallRegex.MatchString(text) || (localActionRunsGate && strings.Contains(text, "uses: ./")) {
			inv.CallsGate = true
		}
	}
	return inv, nil
}

// propose derives a configuration from the repository layout (Tech Spec sdd-adopt, "Layout-Erkennung").
func propose(root, profile string) (*config.Config, string, error) {
	cfg := scaffold.DefaultConfig(profile)
	if err := cfg.Validate(); err != nil {
		return nil, "", err
	}
	layout := "Standardpfade von sdd init"

	specs, err := scan.ListArtifacts(root, "specs/features/*/spec.md", cfg.Exclude)
	if err != nil {
		return nil, "", err
	}
	if len(specs) > 0 {
		cfg.Artifacts.PRD = "specs/features/*/spec.md"
		cfg.Artifacts.Design = "specs/features/*/plan.md"
		cfg.Artifacts.Plans = "specs/features/*/tasks.md"
		cfg.Artifacts.Archive = "specs/archive"
		layout = fmt.Sprintf("Spec Kit (%d Feature-Ordner unter specs/features)", len(specs))
	}

	for _, dir := range adrCandidates {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			continue
		}
		found := false
		for _, e := range entries {
			if !e.IsDir() && adrFileRegex.MatchString(e.Name()) {
				found = true
				break
			}
		}
		if found {
			cfg.Artifacts.ADR = dir
			break
		}
	}

	for _, c := range constitutionCandidates {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil && info.Mode().IsRegular() {
			cfg.Artifacts.Constitution = c
			return cfg, layout, nil
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "*.sdd")); len(matches) == 1 {
		cfg.Artifacts.Constitution = filepath.Base(matches[0])
	}
	return cfg, layout, nil
}

// Classify assigns a Markdown file to exactly one artifact type; the first matching rule wins.
func Classify(p string, cfg *config.Config) File {
	a := cfg.Artifacts
	if c, err := scan.CleanRel(a.Constitution); err == nil && c == p {
		return File{Path: p, Kind: KindConstitution, Reason: "artifacts.constitution"}
	}
	if strings.TrimSpace(a.ADR) != "" && scan.UnderDir(p, a.ADR) && adrFileRegex.MatchString(path.Base(p)) {
		return File{Path: p, Kind: KindADR, Reason: "NNNN-*.md unter " + a.ADR}
	}
	for _, m := range []struct {
		spec, key string
		kind      Kind
	}{
		{a.PRD, "artifacts.prd", KindPRD},
		{a.Design, "artifacts.design", KindDesign},
		{a.Plans, "artifacts.plans", KindPlan},
	} {
		if strings.TrimSpace(m.spec) != "" && scan.InSpec(p, m.spec) {
			return File{Path: p, Kind: m.kind, Reason: m.key + " (" + m.spec + ")"}
		}
	}
	if strings.TrimSpace(a.Archive) != "" && scan.InSpec(p, a.Archive) {
		kind := KindPRD
		switch path.Base(p) {
		case "plan.md":
			kind = KindDesign
		case "tasks.md":
			kind = KindPlan
		}
		return File{Path: p, Kind: kind, Archived: true, Reason: "artifacts.archive (" + a.Archive + "), Typ nach Dateiname"}
	}
	return File{Path: p, Kind: KindDoc, Reason: "kein Muster"}
}
