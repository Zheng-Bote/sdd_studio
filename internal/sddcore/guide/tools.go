package guide

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	sddstudio "github.com/Zheng-Bote/sdd_studio"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// ---------- Explain ----------

// Explanation is a rule with its handbook section and contract text.
type Explanation struct {
	Rule     catalog.Rule `json:"rule"`
	Section  string       `json:"section"`
	Contract string       `json:"contract,omitempty"`
}

var (
	mdHeadingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	mdAnchorRegex  = regexp.MustCompile(`\{#([A-Za-z0-9_-]+)\}\s*$`)
)

// Explain returns the catalog entry of a rule and the handbook text behind its handbook and contract anchors.
func Explain(root, ruleID string) (*Explanation, error) {
	cat, _, err := catalog.Load(root)
	if err != nil {
		return nil, err
	}
	for _, r := range cat.Rules {
		if !strings.EqualFold(r.ID, ruleID) {
			continue
		}
		e := &Explanation{Rule: r}
		e.Section = handbookSection(root, r.Handbook)
		if r.Contract != "" && r.Contract != r.Handbook {
			e.Contract = handbookSection(root, r.Contract)
		}
		return e, nil
	}
	return nil, fmt.Errorf("unbekannte Regel %q", ruleID)
}

// handbookSection extracts the Markdown from the heading with the given {#anchor} up to the next heading of the same
// or a higher level. ref is "<file>#<anchor>"; the working tree is read first, the embedded handbook second.
func handbookSection(root, ref string) string {
	file, anchor, ok := strings.Cut(ref, "#")
	if !ok {
		return ""
	}
	data, err := readRepoOrEmbedded(root, file)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	start, level := -1, 0
	for i, l := range lines {
		m := mdHeadingRegex.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		if start >= 0 {
			if len(m[1]) <= level {
				return strings.TrimSpace(strings.Join(lines[start:i], "\n"))
			}
			continue
		}
		if a := mdAnchorRegex.FindStringSubmatch(m[2]); a != nil && a[1] == anchor {
			start, level = i, len(m[1])
		}
	}
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:], "\n"))
}

func readRepoOrEmbedded(root, rel string) ([]byte, error) {
	clean, err := scan.CleanRel(rel)
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean))); err == nil {
		return data, nil
	}
	return fs.ReadFile(sddstudio.EmbeddedAssets, clean)
}

// ---------- CheckIntent ----------

// Intent answers "what do I need before changing these paths?" — the pre-implementation view.
type Intent struct {
	Feature       string   `json:"feature,omitempty"`
	Exceeded      bool     `json:"threshold_exceeded"`
	Reason        string   `json:"reason,omitempty"`
	Constitution  bool     `json:"constitution_touched"`
	HasPRD        bool     `json:"has_prd"`
	HasTechSpec   bool     `json:"has_tech_spec"`
	Required      []string `json:"required"`
	NextCommands  []string `json:"next_commands"`
	ThresholdNote string   `json:"threshold_note,omitempty"`
}

// CheckIntent evaluates the size threshold for the given paths (globs allowed) and what the feature still needs.
func CheckIntent(root string, paths []string, feature string) (*Intent, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		if _, err := scan.CleanRel(p); err != nil {
			return nil, err
		}
	}
	if feature != "" && !scan.IsSlug(feature) {
		return nil, fmt.Errorf("ungültiger Feature-Slug %q", feature)
	}
	th := scan.EvaluateThreshold(cfg.Thresholds, paths)
	in := &Intent{Feature: feature, Exceeded: th.Exceeded, Reason: th.Reason, Required: []string{}, NextCommands: []string{}}
	if len(cfg.Thresholds.Layers) == 0 && len(cfg.Thresholds.FeatureArtifactsRequiredWhen.TouchedPaths) == 0 {
		in.ThresholdNote = "Die Größenschwelle ist leer konfiguriert und greift nie."
	}
	if c, err := scan.CleanRel(cfg.Artifacts.Constitution); err == nil {
		for _, p := range paths {
			if scan.Overlaps(p, c) {
				in.Constitution = true
			}
		}
	}
	if feature != "" {
		sets, err := activeFeatures(root, cfg)
		if err != nil {
			return nil, err
		}
		if s := sets[feature]; s != nil {
			in.HasPRD, in.HasTechSpec = len(s.prds) > 0, len(s.designs) > 0
		}
	}
	slug := feature
	if slug == "" {
		slug = "<slug>"
	}
	if th.Exceeded {
		in.Required = append(in.Required, "PRD (SDD-020)", "Tech Spec (SDD-030)")
		if !in.HasPRD || !in.HasTechSpec {
			in.NextCommands = append(in.NextCommands, "sdd new --feature "+slug)
		}
	}
	if in.Constitution {
		in.Required = append(in.Required, "entschiedenes ADR mit references auf die Verfassung (SDD-010)")
		in.NextCommands = append(in.NextCommands, "sdd new --type adr --title <Titel>")
	}
	return in, nil
}

// ---------- Context ----------

// Governing is one artifact that applies to a file, with its rank in the precedence order.
type Governing struct {
	Rank    int    `json:"rank"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Status  string `json:"status,omitempty"`
	Feature string `json:"feature,omitempty"`
	Reason  string `json:"reason"`
}

// Context lists the artifacts that govern a file, ordered by precedence (sdd_process_3 §3.6):
// constitution, ADRs whose references overlap it, then Tech Specs, PRDs and plans whose touches overlap it.
func Context(root, file string) ([]Governing, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	target, err := scan.CleanRel(file)
	if err != nil {
		return nil, err
	}
	out := []Governing{}
	if c, err := scan.CleanRel(cfg.Artifacts.Constitution); err == nil {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c))); err == nil {
			out = append(out, Governing{Rank: 1, Kind: "Verfassung", Path: c, Reason: "gilt systemweit und gewinnt gegen alles"})
		}
	}

	if adrDir := strings.TrimSpace(cfg.Artifacts.ADR); adrDir != "" {
		list, err := scan.ListArtifacts(root, adrDir, cfg.Exclude)
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			if !scan.IsADRFileName(path.Base(a.Path)) {
				continue
			}
			refs := adrReferences(root, a.Path)
			for _, ref := range refs {
				if scan.Overlaps(ref, target) {
					out = append(out, Governing{Rank: 2, Kind: "ADR", Path: a.Path, Status: a.Frontmatter.Status, Reason: "references: " + ref})
					break
				}
			}
		}
	}

	sets, err := activeFeatures(root, cfg)
	if err != nil {
		return nil, err
	}
	var features []string
	for slug := range sets {
		features = append(features, slug)
	}
	sort.Strings(features)
	for _, slug := range features {
		s := sets[slug]
		for _, group := range []struct {
			rank int
			kind string
			list []scan.Artifact
		}{{3, "Tech Spec", s.designs}, {4, "PRD", s.prds}, {5, "Plan", s.plans}} {
			for _, a := range group.list {
				for _, t := range a.Frontmatter.Touches {
					if scan.Overlaps(t, target) {
						out = append(out, Governing{Rank: group.rank, Kind: group.kind, Path: a.Path, Status: a.Frontmatter.Status, Feature: slug, Reason: "touches: " + t})
						break
					}
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out, nil
}

func adrReferences(root, rel string) []string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return nil
	}
	var fm struct {
		References []string `yaml:"references"`
	}
	if yaml.Unmarshal([]byte(text[4:4+end]), &fm) != nil {
		return nil
	}
	return fm.References
}

// ---------- Trace ----------

// Trace follows the feature's acceptance criteria to Tech Specs, tests and tasks.
func Trace(root, feature string) ([]rules.ACTrace, error) {
	if !scan.IsSlug(feature) {
		return nil, fmt.Errorf("ungültiger Feature-Slug %q", feature)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	return rules.Trace(&rules.Context{RepoRoot: root, Config: cfg}, feature)
}

// ---------- ValidateDraft ----------

// DraftCheck is the result of checking a not yet written artifact.
type DraftCheck struct {
	Path     string           `json:"path"`
	Findings []report.Finding `json:"findings"`
	Errors   int              `json:"errors"`
	Warnings int              `json:"warnings"`
}

// ValidateDraft checks content as if it were already stored at path: configuration, constitution and all artifact
// trees are copied into a temporary directory, the draft is placed there and `sdd lint` runs on the copy. Test
// coverage rules (SDD-022/025) are skipped — tests are written after the draft.
func ValidateDraft(root, file, content string) (*DraftCheck, error) {
	rel, err := scan.CleanRel(file)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(path.Ext(rel), ".md") && !strings.EqualFold(path.Ext(rel), ".sdd") {
		return nil, fmt.Errorf("nur Markdown- oder .sdd-Artefakte lassen sich prüfen: %s", rel)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "sdd-draft-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	copies := []string{config.DefaultConfigFileName, catalog.DefaultCatalogRelPath, cfg.Artifacts.Constitution}
	for _, c := range copies {
		if err := copyFile(root, tmp, c); err != nil {
			return nil, err
		}
	}
	a := cfg.Artifacts
	for _, spec := range []string{a.PRD, a.Design, a.Plans, a.Archive, a.ADR} {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		dir := strings.TrimSuffix(spec, "/")
		if scan.HasWildcard(spec) {
			dir = strings.TrimSuffix(scan.StaticPrefix(spec), "/")
		}
		if err := copyTree(root, tmp, dir); err != nil {
			return nil, err
		}
	}
	target := filepath.Join(tmp, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return nil, err
	}

	rep, err := sddcore.RunCheckWithConfig(sddcore.RunOptions{RepoRoot: tmp}, cfg)
	if err != nil {
		return nil, err
	}
	res := &DraftCheck{Path: rel, Findings: []report.Finding{}}
	for _, f := range rep.Findings {
		if f.Path != rel || f.RuleID == "SDD-022" || f.RuleID == "SDD-025" {
			continue
		}
		res.Findings = append(res.Findings, f)
		switch f.Severity {
		case "error":
			res.Errors++
		case "warn":
			res.Warnings++
		}
	}
	return res, nil
}

func copyFile(root, dst, rel string) error {
	clean, err := scan.CleanRel(rel)
	if err != nil {
		return nil // unconfigured or invalid: nothing to copy
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return nil // missing files are reported by the rules on the copy
	}
	target := filepath.Join(dst, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func copyTree(root, dst, dir string) error {
	clean, err := scan.CleanRel(dir)
	if err != nil || clean == "." {
		return nil
	}
	base := filepath.Join(root, filepath.FromSlash(clean))
	return filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		return copyFile(root, dst, filepath.ToSlash(rel))
	})
}
