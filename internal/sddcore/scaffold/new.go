package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Artifact types accepted by `sdd new --type`.
const (
	TypePRD    = "prd"
	TypeDesign = "design"
	TypePlan   = "plan"
	TypeADR    = "adr"
	TypeDoc    = "doc"
)

// ErrInvalidInput marks user input errors of `sdd new` (exit code 2).
var ErrInvalidInput = errors.New("ungültige Eingabe")

var (
	adrNumberRegex = regexp.MustCompile(`^([0-9]{4})-`)
	nonSlugRegex   = regexp.MustCompile(`[^a-z0-9]+`)
)

// NewOptions configures `sdd new`.
type NewOptions struct {
	Types        []string // default: prd + design when Feature is set
	Feature      string   // slug for prd, design, plan
	Title        string
	Slug         string // slug for adr, doc (default: derived from Title)
	ArchiveAfter string // YYYY-MM-DD or empty
	Touches      []string
	Today        time.Time
}

type featureTemplateData struct {
	Slug, Title, ArchiveAfter, PRDPath string
	Touches                            []string
}

// NewPlan builds the plan for new artifacts. Feature artifacts that already exist for the slug are planned as skipped.
func NewPlan(root string, cfg *config.Config, opts NewOptions) (Plan, error) {
	types := opts.Types
	if len(types) == 0 {
		if opts.Feature == "" {
			return Plan{}, fmt.Errorf("%w: --feature <slug> oder --type adr|doc erforderlich", ErrInvalidInput)
		}
		types = []string{TypePRD, TypeDesign}
	}
	var featureTypes, otherTypes []string
	for _, t := range types {
		switch t {
		case TypePRD, TypeDesign, TypePlan:
			featureTypes = append(featureTypes, t)
		case TypeADR, TypeDoc:
			otherTypes = append(otherTypes, t)
		default:
			return Plan{}, fmt.Errorf("%w: unbekannter Typ %q (erlaubt: prd, design, plan, adr, doc)", ErrInvalidInput, t)
		}
	}
	if len(otherTypes) > 0 && (len(featureTypes) > 0 || len(otherTypes) > 1) {
		return Plan{}, fmt.Errorf("%w: adr und doc nur einzeln und nicht zusammen mit Feature-Artefakten anlegen", ErrInvalidInput)
	}
	if len(otherTypes) == 1 {
		return otherPlan(root, cfg, otherTypes[0], opts)
	}
	return featurePlan(root, cfg, featureTypes, opts)
}

func featurePlan(root string, cfg *config.Config, types []string, opts NewOptions) (Plan, error) {
	if !scan.IsSlug(opts.Feature) {
		return Plan{}, fmt.Errorf("%w: Feature-Slug %q (erlaubt: [a-z0-9-]+)", ErrInvalidInput, opts.Feature)
	}
	if opts.ArchiveAfter != "" {
		if _, err := time.Parse("2006-01-02", opts.ArchiveAfter); err != nil {
			return Plan{}, fmt.Errorf("%w: archive-after %q ist kein Datum YYYY-MM-DD", ErrInvalidInput, opts.ArchiveAfter)
		}
	}
	var touches []string
	for _, t := range opts.Touches {
		clean, err := CleanRel(t)
		if err != nil {
			return Plan{}, fmt.Errorf("%w: touches: %v", ErrInvalidInput, err)
		}
		touches = append(touches, clean)
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = opts.Feature
	}

	dirs := map[string]string{TypePRD: cfg.Artifacts.PRD, TypeDesign: cfg.Artifacts.Design, TypePlan: cfg.Artifacts.Plans}
	keys := map[string]string{TypePRD: "artifacts.prd", TypeDesign: "artifacts.design", TypePlan: "artifacts.plans"}
	tmpls := map[string]string{TypePRD: "prd.md.tmpl", TypeDesign: "design.md.tmpl", TypePlan: "plan.md.tmpl"}

	existing := map[string]string{} // type -> path of an artifact already carrying the slug
	for _, t := range []string{TypePRD, TypeDesign, TypePlan} {
		dir := strings.TrimSpace(dirs[t])
		if dir == "" {
			continue
		}
		list, err := scan.ListArtifacts(root, dir, cfg.Exclude)
		if err != nil {
			return Plan{}, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", dir, err)
		}
		for _, a := range list {
			if a.Frontmatter.Feature == opts.Feature {
				existing[t] = a.Path
				break
			}
		}
	}

	prdPath := existing[TypePRD]
	if prdPath == "" && strings.TrimSpace(cfg.Artifacts.PRD) != "" {
		prdPath = path.Join(cfg.Artifacts.PRD, opts.Feature+".md")
	}
	data := featureTemplateData{Slug: opts.Feature, Title: title, ArchiveAfter: opts.ArchiveAfter, PRDPath: prdPath, Touches: touches}

	var plan Plan
	for _, t := range dedupe(types) {
		dir := strings.TrimSpace(dirs[t])
		if dir == "" {
			return Plan{}, fmt.Errorf("%w: %s ist nicht konfiguriert", ErrInvalidInput, keys[t])
		}
		target := path.Join(dir, opts.Feature+".md")
		if p, ok := existing[t]; ok {
			plan.Ops = append(plan.Ops, Op{Path: p, Skip: fmt.Sprintf("trägt bereits feature: %s", opts.Feature)})
			continue
		}
		content, err := render(tmpls[t], data)
		if err != nil {
			return Plan{}, err
		}
		plan.Ops = append(plan.Ops, Op{Path: target, Content: content})
	}
	return plan, plan.Validate()
}

func otherPlan(root string, cfg *config.Config, typ string, opts NewOptions) (Plan, error) {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		return Plan{}, fmt.Errorf("%w: --title ist für %s erforderlich", ErrInvalidInput, typ)
	}
	slug := opts.Slug
	if slug == "" {
		slug = Slugify(title)
	}
	if !scan.IsSlug(slug) {
		return Plan{}, fmt.Errorf("%w: Slug %q (erlaubt: [a-z0-9-]+)", ErrInvalidInput, slug)
	}

	var target string
	var content []byte
	var err error
	switch typ {
	case TypeADR:
		dir := strings.TrimSpace(cfg.Artifacts.ADR)
		if dir == "" {
			return Plan{}, fmt.Errorf("%w: artifacts.adr ist nicht konfiguriert", ErrInvalidInput)
		}
		next, nerr := nextADRNumber(root, dir)
		if nerr != nil {
			return Plan{}, nerr
		}
		number := fmt.Sprintf("%04d", next)
		target = path.Join(dir, number+"-"+slug+".md")
		content, err = render("adr.md.tmpl", struct{ Number, Title, Date string }{number, title, opts.Today.Format("2006-01-02")})
	case TypeDoc:
		if len(cfg.Artifacts.Docs) == 0 || strings.TrimSpace(cfg.Artifacts.Docs[0]) == "" {
			return Plan{}, fmt.Errorf("%w: artifacts.docs ist nicht konfiguriert", ErrInvalidInput)
		}
		target = path.Join(cfg.Artifacts.Docs[0], slug+".md")
		content, err = render("doc.md.tmpl", struct{ Title string }{title})
	}
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Ops: []Op{{Path: target, Content: content}}}
	return plan, plan.Validate()
}

// nextADRNumber returns the highest four-digit prefix in dir plus one.
func nextADRNumber(root, dir string) (int, error) {
	clean, err := CleanRel(dir)
	if err != nil {
		return 0, fmt.Errorf("%w: artifacts.adr: %v", ErrInvalidInput, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, fmt.Errorf("ADR-Verzeichnis %s nicht lesbar: %w", clean, err)
	}
	max := 0
	for _, e := range entries {
		if m := adrNumberRegex.FindStringSubmatch(e.Name()); m != nil {
			if n, _ := strconv.Atoi(m[1]); n > max {
				max = n
			}
		}
	}
	return max + 1, nil
}

// Slugify derives a slug from a title (lowercase ASCII, umlauts transliterated, other characters as hyphens).
func Slugify(title string) string {
	s := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "Ä", "ae", "Ö", "oe", "Ü", "ue", "ß", "ss").Replace(title)
	s = nonSlugRegex.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(s, "-")
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
