// Package guide derives process guidance from the repository: the next step per feature, the artifacts that govern a
// file, and checks of drafts. It is deterministic and offline (ADR 0002); agents reach it through `sdd mcp`.
package guide

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adrlock"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Step states.
const (
	StateDone      = "done"
	StateOpen      = "open"
	StateNotNeeded = "not-needed"
	StateManual    = "manual"
	StateLater     = "later"
)

// Step is one process step of a feature.
type Step struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Detail  string `json:"detail"`
	Command string `json:"command,omitempty"`
}

// FeatureGuide lists the steps of one feature and the step to do next.
type FeatureGuide struct {
	Feature string `json:"feature"`
	Steps   []Step `json:"steps"`
	Next    *Step  `json:"next,omitempty"`
}

// Guide is the result of `sdd next`.
type Guide struct {
	Features []FeatureGuide `json:"features"`
	// Start is set when no feature exists yet: how to begin.
	Start *Step `json:"start,omitempty"`
}

var acIDRegex = regexp.MustCompile(`\bAC-[1-9][0-9]*\b`)

// artifactSet are the active artifacts of one feature.
type artifactSet struct {
	prds, designs, plans []scan.Artifact
}

func (s artifactSet) all() []scan.Artifact {
	return append(append(append([]scan.Artifact{}, s.prds...), s.designs...), s.plans...)
}

// Next computes the process steps. With feature == "" it covers every feature that has an active artifact.
func Next(root, feature string) (*Guide, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	sets, err := activeFeatures(root, cfg)
	if err != nil {
		return nil, err
	}
	rep, err := sddcore.RunCheckWithConfig(sddcore.RunOptions{RepoRoot: root}, cfg)
	if err != nil {
		return nil, err
	}

	var slugs []string
	if feature != "" {
		if !scan.IsSlug(feature) {
			return nil, fmt.Errorf("ungültiger Feature-Slug %q (erlaubt: [a-z0-9-]+)", feature)
		}
		slugs = []string{feature}
	} else {
		for slug := range sets {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
	}

	g := &Guide{Features: []FeatureGuide{}}
	if len(slugs) == 0 {
		g.Start = &Step{ID: "idea", Title: "Idee als PRD festhalten", State: StateOpen,
			Detail:  "Noch kein Feature. Ein Feature beginnt mit einem PRD: Problem, Ziel, Nutzer, Nicht-Ziele, Akzeptanzkriterien.",
			Command: "sdd new --feature <slug> --type prd"}
		return g, nil
	}
	for _, slug := range slugs {
		g.Features = append(g.Features, featureGuide(root, cfg, slug, sets[slug], rep.Findings))
	}
	return g, nil
}

// activeFeatures groups the active feature artifacts (not archived, superseded or in the archive) by feature.
func activeFeatures(root string, cfg *config.Config) (map[string]*artifactSet, error) {
	sets := map[string]*artifactSet{}
	a := cfg.Artifacts
	for _, k := range []struct {
		spec string
		add  func(*artifactSet, scan.Artifact)
	}{
		{a.PRD, func(s *artifactSet, x scan.Artifact) { s.prds = append(s.prds, x) }},
		{a.Design, func(s *artifactSet, x scan.Artifact) { s.designs = append(s.designs, x) }},
		{a.Plans, func(s *artifactSet, x scan.Artifact) { s.plans = append(s.plans, x) }},
	} {
		if strings.TrimSpace(k.spec) == "" {
			continue
		}
		list, err := scan.ListArtifacts(root, k.spec, cfg.Exclude)
		if err != nil {
			return nil, err
		}
		for _, x := range list {
			if x.Frontmatter.Status == "archived" || x.Frontmatter.Status == "superseded" || (a.Archive != "" && scan.InSpec(x.Path, a.Archive)) {
				continue
			}
			slug := x.Frontmatter.Feature
			if slug == "" {
				slug = scan.SlugInSpec(x.Path, k.spec)
			}
			if sets[slug] == nil {
				sets[slug] = &artifactSet{}
			}
			k.add(sets[slug], x)
		}
	}
	return sets, nil
}

// adrStep decides the ADR step of a feature whose touches include the constitution. Done means an artifact of the
// feature links to an accepted ADR that names the constitution (references or adr, as SDD-010 reads it). Any ADR
// naming the constitution is not enough: it may justify an older, unrelated change.
func adrStep(root string, cfg *config.Config, set *artifactSet, constitution string) Step {
	step := Step{ID: "adr", Title: "ADR für die Verfassungsänderung", State: StateManual,
		Detail:  "touches berührt die Verfassung: ein ADR mit references auf sie anlegen, im PRD oder in der Tech Spec verlinken und entscheiden; SDD-010 prüft im Gate.",
		Command: "sdd new --type adr --title <Titel>"}
	adrs, err := adrlock.Collect(root, cfg)
	if err != nil {
		return step
	}
	linked := map[string]bool{}
	for _, x := range set.all() {
		for _, target := range rules.LinksIn(root, x.Path) {
			linked[target] = true
		}
	}
	var accepted, open, unlinked []string
	for _, a := range adrs {
		if !rules.ADRNames(root, a.Path, constitution) {
			continue
		}
		switch {
		case !linked[a.Path]:
			unlinked = append(unlinked, a.Path)
		case a.Status == "accepted":
			accepted = append(accepted, a.Path)
		case !a.Decided:
			open = append(open, a.Path)
		}
	}
	switch {
	case len(accepted) > 0:
		step.State, step.Command = StateDone, ""
		step.Detail = strings.Join(accepted, ", ") + " (accepted)"
	case len(open) > 0:
		step.Title, step.Command = "ADR entscheiden", ""
		step.Detail = strings.Join(open, ", ") + " ist verlinkt, aber noch nicht entschieden: der Mensch setzt status: accepted (oder rejected)."
	case len(unlinked) > 0:
		step.Detail = "touches berührt die Verfassung. Diese ADRs nennen sie: " + strings.Join(unlinked, ", ") +
			" — das zutreffende im PRD oder in der Tech Spec verlinken, sonst ein neues anlegen; SDD-010 prüft im Gate."
	}
	return step
}

func findingsAt(findings []report.Finding, rule string, paths []scan.Artifact) []report.Finding {
	var out []report.Finding
	for _, f := range findings {
		if f.RuleID != rule {
			continue
		}
		for _, p := range paths {
			if f.Path == p.Path {
				out = append(out, f)
			}
		}
	}
	return out
}

func featureGuide(root string, cfg *config.Config, slug string, set *artifactSet, findings []report.Finding) FeatureGuide {
	if set == nil {
		set = &artifactSet{}
	}
	fg := FeatureGuide{Feature: slug}
	add := func(s Step) { fg.Steps = append(fg.Steps, s) }
	newCmd := func(t string) string { return fmt.Sprintf("sdd new --feature %s --type %s", slug, t) }

	// 1. PRD
	if len(set.prds) == 0 {
		add(Step{ID: "prd", Title: "PRD anlegen", State: StateOpen, Detail: "Kein aktives PRD mit feature: " + slug + ".", Command: newCmd("prd")})
	} else {
		add(Step{ID: "prd", Title: "PRD anlegen", State: StateDone, Detail: set.prds[0].Path})
	}

	// 2. Non-goals, 3. acceptance criteria
	switch {
	case len(set.prds) == 0:
		add(Step{ID: "non-goals", Title: "Nicht-Ziele benennen", State: StateLater, Detail: "nach dem PRD"})
		add(Step{ID: "acceptance", Title: "Akzeptanzkriterien mit Kennung", State: StateLater, Detail: "nach dem PRD"})
	default:
		if fs := findingsAt(findings, "SDD-023", set.prds); len(fs) > 0 {
			add(Step{ID: "non-goals", Title: "Nicht-Ziele benennen", State: StateOpen, Detail: fs[0].Message + " (SDD-023)", Command: "sdd explain SDD-023"})
		} else {
			add(Step{ID: "non-goals", Title: "Nicht-Ziele benennen", State: StateDone, Detail: "Abschnitt vorhanden"})
		}
		acs := 0
		for _, p := range set.prds {
			if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p.Path))); err == nil {
				acs += len(acIDRegex.FindAllString(string(data), -1))
			}
		}
		fs := findingsAt(findings, "SDD-021", set.prds)
		switch {
		case acs == 0:
			add(Step{ID: "acceptance", Title: "Akzeptanzkriterien mit Kennung", State: StateOpen, Detail: "Das PRD enthält keine Kennung AC-<n>."})
		case len(fs) > 0:
			add(Step{ID: "acceptance", Title: "Akzeptanzkriterien mit Kennung", State: StateOpen, Detail: fmt.Sprintf("%d Kriterien ohne Kennung (SDD-021)", len(fs))})
		default:
			add(Step{ID: "acceptance", Title: "Akzeptanzkriterien mit Kennung", State: StateDone, Detail: fmt.Sprintf("%d Kennungen", acs)})
		}
	}

	// 4. Declared intent
	var touches []string
	seenTouch := map[string]bool{}
	for _, x := range set.all() {
		for _, t := range x.Frontmatter.Touches {
			if !seenTouch[t] {
				seenTouch[t] = true
				touches = append(touches, t)
			}
		}
	}
	if len(touches) == 0 {
		add(Step{ID: "intent", Title: "Änderungsabsicht erklären (touches)", State: StateOpen,
			Detail:  "Ohne touches ist die Größenschwelle nicht bewertbar (SDD-020/030 warnen in pre-implementation).",
			Command: "touches im Frontmatter des PRD ergänzen"})
	} else {
		add(Step{ID: "intent", Title: "Änderungsabsicht erklären (touches)", State: StateDone, Detail: strings.Join(touches, ", ")})
	}

	// 5. Tech Spec, 6. AC reference
	th := scan.EvaluateThreshold(cfg.Thresholds, touches)
	switch {
	case len(set.designs) > 0:
		add(Step{ID: "design", Title: "Tech Spec", State: StateDone, Detail: set.designs[0].Path})
		if fs := findingsAt(findings, "SDD-031", set.designs); len(fs) > 0 {
			add(Step{ID: "design-acs", Title: "Tech Spec nennt alle Kriterien", State: StateOpen, Detail: fmt.Sprintf("%d Kriterien fehlen in der Tech Spec (SDD-031)", len(fs))})
		} else {
			add(Step{ID: "design-acs", Title: "Tech Spec nennt alle Kriterien", State: StateDone, Detail: "alle Kennungen referenziert"})
		}
	case len(touches) > 0 && !th.Exceeded:
		add(Step{ID: "design", Title: "Tech Spec", State: StateNotNeeded, Detail: "Größenschwelle nicht überschritten"})
		add(Step{ID: "design-acs", Title: "Tech Spec nennt alle Kriterien", State: StateNotNeeded, Detail: "keine Tech Spec nötig"})
	default:
		detail := "Größenschwelle überschritten: " + th.Reason
		if len(touches) == 0 {
			detail = "Schwelle erst mit touches bewertbar"
		}
		state := StateOpen
		if len(touches) == 0 {
			state = StateLater
		}
		add(Step{ID: "design", Title: "Tech Spec", State: state, Detail: detail, Command: newCmd("design")})
		add(Step{ID: "design-acs", Title: "Tech Spec nennt alle Kriterien", State: StateLater, Detail: "nach der Tech Spec"})
	}

	// 7. ADR when the constitution is touched
	constitution, err := scan.CleanRel(cfg.Artifacts.Constitution)
	constitutionTouched := false
	for _, t := range touches {
		if err == nil && scan.Overlaps(t, constitution) {
			constitutionTouched = true
		}
	}
	if constitutionTouched {
		add(adrStep(root, cfg, set, constitution))
	} else {
		add(Step{ID: "adr", Title: "ADR für die Verfassungsänderung", State: StateNotNeeded, Detail: "Verfassung nicht betroffen"})
	}

	// 8. Plan
	if len(set.plans) == 0 {
		add(Step{ID: "plan", Title: "Plan mit Tasks", State: StateNotNeeded, Detail: "optional; Tasks brauchen Tech Spec (SDD-040) und AC (SDD-041)", Command: newCmd("plan")})
	} else if fs := append(findingsAt(findings, "SDD-040", set.plans), findingsAt(findings, "SDD-041", set.plans)...); len(fs) > 0 {
		add(Step{ID: "plan", Title: "Plan mit Tasks", State: StateOpen, Detail: fmt.Sprintf("%d Befunde (SDD-040/041), z. B.: %s", len(fs), fs[0].Message)})
	} else {
		add(Step{ID: "plan", Title: "Plan mit Tasks", State: StateDone, Detail: set.plans[0].Path})
	}

	// 9. Tests
	if len(set.prds) == 0 {
		add(Step{ID: "tests", Title: "Tests je Kriterium", State: StateLater, Detail: "nach dem PRD"})
	} else if noTests := noTestFiles(findings); noTests != nil {
		add(Step{ID: "tests", Title: "Tests je Kriterium", State: StateOpen, Detail: noTests.Message + " (SDD-022)",
			Command: "tests.include in sdd.config.yaml setzen und Tests anlegen"})
	} else if fs := findingsAt(findings, "SDD-022", set.prds); len(fs) > 0 {
		add(Step{ID: "tests", Title: "Tests je Kriterium", State: StateOpen,
			Detail:  fmt.Sprintf("%d Kriterien ohne Test, z. B.: %s", len(fs), fs[0].Message),
			Command: fmt.Sprintf("Test mit Kommentar // %s AC-<n>: … anlegen", slug)})
	} else {
		add(Step{ID: "tests", Title: "Tests je Kriterium", State: StateDone, Detail: "jede Kennung ist referenziert"})
	}

	// 10. Finish and archive
	implemented := false
	for _, x := range set.all() {
		if x.Frontmatter.Status == "implemented" {
			implemented = true
		}
	}
	if implemented {
		add(Step{ID: "archive", Title: "Archivieren vor dem Release", State: StateOpen,
			Detail:  "status: implemented — vor dem nächsten Tag nach " + cfg.Artifacts.Archive + " verschieben und status: archived setzen (SDD-050)",
			Command: "sdd gate pre-release"})
	} else {
		add(Step{ID: "archive", Title: "Abschließen und archivieren", State: StateLater,
			Detail: "nach dem Merge status: implemented setzen; vor dem Release archivieren (SDD-050)"})
	}

	for _, state := range []string{StateOpen, StateManual} {
		for i := range fg.Steps {
			if fg.Steps[i].State == state {
				fg.Next = &fg.Steps[i]
				return fg
			}
		}
	}
	// Everything specified and tested: the remaining step is finishing (the last step, "later").
	fg.Next = &fg.Steps[len(fg.Steps)-1]
	return fg
}

// noTestFiles returns SDD-022's single warning for a repository without any test file, if present.
func noTestFiles(findings []report.Finding) *report.Finding {
	for i, f := range findings {
		if f.RuleID == "SDD-022" && f.Path == config.DefaultConfigFileName {
			return &findings[i]
		}
	}
	return nil
}
