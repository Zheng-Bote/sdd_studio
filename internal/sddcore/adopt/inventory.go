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
	// The macro level of the BMAD pipeline (ADR 0009): sources of the take-over, not feature artifacts.
	KindInitiative Kind = "Initiative"
	KindEpic       Kind = "Epic"
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
	CallsGate          bool        // a CI workflow runs `sdd gate`
	BMAD               *BMADLayout // set when a BMAD v6 tree was found (ADR 0009)
}

// BMADLayout is what an existing BMAD v6 tree offers the take-over (ADR 0009). Its files are sources, not artifacts:
// `_bmad-output/` stays BMAD's ledger and is never rewritten.
type BMADLayout struct {
	Initiatives []string // every `initiative-*.md`, sorted
	Epics       []string // every `epic-*.md` below the tickets folders
}

// Initiative is the single initiative of the tree, or "" if there is none or more than one.
func (b *BMADLayout) Initiative() string {
	if b == nil || len(b.Initiatives) != 1 {
		return ""
	}
	return b.Initiatives[0]
}

// excludeGlobs keeps the BMAD tool tree and its memlogs out of the inventory; nil without BMAD.
func (b *BMADLayout) excludeGlobs() []string {
	if b == nil {
		return nil
	}
	return append(append([]string{}, bmadToolGlobs...), bmadIgnoreGlobs...)
}

// detectBMAD reports a BMAD v6 tree, recognised by `_bmad/` or `_bmad-output/`; nil otherwise. It reads only names.
func detectBMAD(root string) *BMADLayout {
	found := false
	for _, dir := range []string{"_bmad", "_bmad-output"} {
		if info, err := os.Stat(filepath.Join(root, dir)); err == nil && info.IsDir() {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	b := &BMADLayout{}
	if list, err := scan.WalkFiles(root, []string{bmadInitiative}, nil); err == nil {
		b.Initiatives = list
	}
	if list, err := scan.WalkFiles(root, []string{bmadEpics}, nil); err == nil {
		b.Epics = list
	}
	return b
}

var (
	adrFileRegex  = regexp.MustCompile(`^[0-9]{4}-.*\.md$`)
	gateCallRegex = regexp.MustCompile(`sdd(?:\.exe)?["']?\s+gate\b|sdd_studio@`)

	// BMAD v6 (ADR 0009): the tool tree `_bmad/`, the ledger `_bmad-output/` with one folder per initiative, and the
	// tickets below it. v4 (`.bmad-core/`) is not supported.
	//
	// toolGlobs are the agent tool trees a BMAD install brings along (skill mirrors for Claude, Copilot, Codex and
	// others). They are neither artifacts nor documentation of the project, so they stay out of the inventory.
	bmadToolGlobs   = []string{"_bmad/**", ".agent/**", ".agents/**", ".claude/**", ".codewhale/**", ".cursor/**"}
	bmadIgnoreGlobs = []string{"**/.memlog.md"}
	bmadInitiative  = "_bmad-output/**/initiative-*.md"
	bmadEpics       = "_bmad-output/**/tickets/**/epic-*.md"
)

// adrCandidates and constitutionCandidates are probed in this order when no configuration exists.
var (
	adrCandidates          = []string{"docs/adr", "docs/architecture/adr", "doc/adr", "adr"}
	constitutionCandidates = []string{"constitution.md", "CONSTITUTION.md", "docs/constitution.md"}
	workflowGlobs          = []string{".github/workflows/*.yml", ".github/workflows/*.yaml", ".gitlab-ci.yml"}
)

// Take builds the inventory. Without sdd.config.yaml a configuration is proposed from the detected layout.
func Take(root, profile string) (*Inventory, error) {
	inv := &Inventory{BMAD: detectBMAD(root)}
	cfg, err := config.Load(root)
	switch {
	case err == nil:
		inv.Config, inv.ConfigExists, inv.Layout = cfg, true, "vorhandene sdd.config.yaml"
	case errors.Is(err, config.ErrConfigNotFound):
		if inv.Config, inv.Layout, err = propose(root, profile, inv.BMAD); err != nil {
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
	exclude = append(exclude, inv.BMAD.excludeGlobs()...)
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
func propose(root, profile string, bmad *BMADLayout) (*config.Config, string, error) {
	cfg := scaffold.DefaultConfig(profile)
	if err := cfg.Validate(); err != nil {
		return nil, "", err
	}
	layout := "Standardpfade von sdd init"
	if bmad != nil {
		cfg.Pipeline = config.PipelineBMAD
		cfg.Exclude = append(cfg.Exclude, bmad.excludeGlobs()...)
		// The ledger stays the source; the artifact the project works with is the configured initiative, so the
		// take-over never has to write into BMAD's ledger.
		switch in := bmad.Initiative(); {
		case in != "":
			layout = fmt.Sprintf("BMAD v6 (%d Epics, Quelle %s → %s)", len(bmad.Epics), in, cfg.Artifacts.InitiativePath())
		case len(bmad.Initiatives) > 1:
			layout = fmt.Sprintf("BMAD v6 (%d Epics, %d Initiativen — die richtige von Hand übernehmen)", len(bmad.Epics), len(bmad.Initiatives))
		default:
			layout = fmt.Sprintf("BMAD v6 (%d Epics, keine Initiative gefunden)", len(bmad.Epics))
		}
	}

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
	if f, ok := classifyMacro(p, cfg); ok {
		return f
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
		return classifyArchived(p, a.Archive, map[Kind]string{KindPRD: a.PRD, KindDesign: a.Design, KindPlan: a.Plans})
	}
	return File{Path: p, Kind: KindDoc, Reason: "kein Muster"}
}

// classifyMacro recognises the macro level before the ADR and feature paths (ADR 0009): the configured initiative and
// the files of a BMAD v6 ledger are sources of the take-over, not feature artifacts. Everything else in the ledger —
// forge reports, memlogs — stays documentation.
func classifyMacro(p string, cfg *config.Config) (File, bool) {
	if c, err := scan.CleanRel(cfg.Artifacts.InitiativePath()); err == nil && c == p {
		return File{Path: p, Kind: KindInitiative, Reason: "artifacts.initiative"}, true
	}
	if !scan.UnderDir(p, "_bmad-output") {
		return File{}, false
	}
	base := path.Base(p)
	switch {
	case strings.HasPrefix(base, "initiative-") && strings.HasSuffix(base, ".md"):
		return File{Path: p, Kind: KindInitiative, Reason: "BMAD-Ledger (initiative-*.md)"}, true
	case strings.HasPrefix(base, "epic-") && strings.HasSuffix(base, ".md"):
		return File{Path: p, Kind: KindEpic, Reason: "BMAD-Ledger (epic-*.md)"}, true
	}
	return File{Path: p, Kind: KindDoc, Reason: "BMAD-Bericht, kein Artefakt"}, true
}

// classifyArchived derives the type of an archived feature artifact: from the archive subfolder named like the
// active folder (archive/prd, archive/design, archive/plans — sdd_process_3 section 11), else from the Spec Kit file
// name (plan.md, tasks.md), else PRD.
func classifyArchived(p, archive string, active map[Kind]string) File {
	reason := "artifacts.archive (" + archive + ")"
	if !scan.HasWildcard(archive) {
		rel := strings.TrimPrefix(p, strings.TrimSuffix(archive, "/")+"/")
		if sub, _, ok := strings.Cut(rel, "/"); ok {
			for _, k := range []Kind{KindPRD, KindDesign, KindPlan} {
				if spec := active[k]; spec != "" && !scan.HasWildcard(spec) && path.Base(strings.TrimSuffix(spec, "/")) == sub {
					return File{Path: p, Kind: k, Archived: true, Reason: reason + ", Typ nach Unterordner " + sub}
				}
			}
		}
	}
	kind := KindPRD
	switch path.Base(p) {
	case "plan.md":
		kind = KindDesign
	case "tasks.md":
		kind = KindPlan
	}
	return File{Path: p, Kind: kind, Archived: true, Reason: reason + ", Typ nach Dateiname"}
}
