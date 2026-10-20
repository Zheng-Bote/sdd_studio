package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// ReportPath is where `sdd adopt --apply` writes its report.
const ReportPath = ".sdd/adopt-report.md"

// Options configure a take-over.
type Options struct {
	Profile string
	Branch  string
	Project string
	Today   time.Time
}

// Step is one take-over step; each step with changes becomes exactly one commit.
type Step struct {
	Title  string
	Plan   scaffold.Plan
	Detail []string // one line per planned change, shown in the dry run and the report
}

// Proposal is the complete, not yet applied take-over.
type Proposal struct {
	Options   Options
	Inventory *Inventory
	Findings  []report.Finding // default lint against the loaded or proposed configuration
	Maturity  Maturity
	Steps     []Step // configuration, constitution, frontmatter (baseline and report are computed during apply)
	Conflicts []string
	Archive   []report.Finding // SDD-050: reported, never executed
}

// Propose takes the inventory, assesses the maturity and plans the steps. It writes nothing.
func Propose(root string, opts Options) (*Proposal, error) {
	inv, err := Take(root, opts.Profile)
	if err != nil {
		return nil, err
	}
	rep, err := sddcore.RunCheckWithConfig(sddcore.RunOptions{RepoRoot: root}, inv.Config)
	if err != nil {
		return nil, err
	}
	p := &Proposal{Options: opts, Inventory: inv, Findings: rep.Findings}
	p.Maturity = Assess(root, inv, rep.Findings)
	for _, f := range rep.Findings {
		if f.RuleID == "SDD-050" {
			p.Archive = append(p.Archive, f)
		}
	}

	if !inv.ConfigExists {
		content, err := scaffold.RenderConfig(inv.Config)
		if err != nil {
			return nil, err
		}
		p.Steps = append(p.Steps, Step{
			Title:  "Konfiguration",
			Plan:   scaffold.Plan{Ops: []scaffold.Op{{Path: config.DefaultConfigFileName, Content: content}}},
			Detail: []string{fmt.Sprintf("%s anlegen (%s)", config.DefaultConfigFileName, inv.Layout)},
		})
	}
	if !inv.ConstitutionExists {
		content, err := scaffold.RenderConstitution(inv.Config, opts.Project)
		if err != nil {
			return nil, err
		}
		p.Steps = append(p.Steps, Step{
			Title:  "Verfassung",
			Plan:   scaffold.Plan{Ops: []scaffold.Op{{Path: inv.Config.Artifacts.Constitution, Content: content}}},
			Detail: []string{fmt.Sprintf("%s aus dem Template anlegen (Pflichtabschnitte nach SDD-002a, Inhalt ist auszufüllen)", inv.Config.Artifacts.Constitution)},
		})
	}

	fm, err := p.frontmatterStep(root)
	if err != nil {
		return nil, err
	}
	if len(fm.Plan.Ops) > 0 {
		p.Steps = append(p.Steps, fm)
	}
	return p, nil
}

// frontmatterStep plans a frontmatter block for every active feature artifact that has none.
func (p *Proposal) frontmatterStep(root string) (Step, error) {
	a := p.Inventory.Config.Artifacts
	specs := map[Kind]string{KindPRD: a.PRD, KindDesign: a.Design, KindPlan: a.Plans}
	step := Step{Title: "Frontmatter"}
	seen := map[string]string{} // kind|slug -> path
	for _, f := range p.Inventory.Files {
		if !f.Kind.IsFeature() || f.Archived {
			continue
		}
		if f.HasFrontmatter {
			if f.Frontmatter.Feature == "" {
				p.Conflicts = append(p.Conflicts, fmt.Sprintf("%s hat Frontmatter, aber kein `feature:` — nicht verändert, Feld von Hand ergänzen", f.Path))
			}
			continue
		}
		slug := scaffold.Slugify(scan.SlugInSpec(f.Path, specs[f.Kind]))
		if !scan.IsSlug(slug) {
			p.Conflicts = append(p.Conflicts, fmt.Sprintf("%s: kein Slug ableitbar — nicht verändert", f.Path))
			continue
		}
		key := string(f.Kind) + "|" + slug
		if other, dup := seen[key]; dup {
			p.Conflicts = append(p.Conflicts, fmt.Sprintf("%s: Slug %s ist schon %s zugeordnet (%s) — nicht verändert", f.Path, slug, other, f.Kind))
			continue
		}
		seen[key] = f.Path
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
		if err != nil {
			return Step{}, fmt.Errorf("%s nicht lesbar: %w", f.Path, err)
		}
		head := fmt.Sprintf("---\nfeature: %s\nstatus: active\n---\n\n", slug)
		step.Plan.Ops = append(step.Plan.Ops, scaffold.Op{Path: f.Path, Content: []byte(head), Prepend: true, Expect: data})
		step.Detail = append(step.Detail, fmt.Sprintf("%s ← `feature: %s`, `status: active` (%s)", f.Path, slug, f.Kind))
	}
	sort.Strings(p.Conflicts)
	return step, nil
}
