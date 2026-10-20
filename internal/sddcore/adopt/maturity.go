package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// Criterion is one maturity criterion with its result and justification.
type Criterion struct {
	Level  int
	Name   string
	Met    bool
	Detail string
	// Advice is the concrete next step if the criterion is not met.
	Advice string
}

// Maturity is the assessed level L0-L4 and exactly one proposed next step.
type Maturity struct {
	Level    int
	Criteria []Criterion
	Next     *Criterion // nil at L4
}

var acIDRegex = regexp.MustCompile(`\bAC-[1-9][0-9]*\b`)

// Assess determines the maturity level from the inventory and the default-lint findings (Tech Spec sdd-adopt).
func Assess(root string, inv *Inventory, findings []report.Finding) Maturity {
	byRule := map[string]int{}
	for _, f := range findings {
		byRule[f.RuleID]++
	}
	count := func(ids ...string) int {
		n := 0
		for _, id := range ids {
			n += byRule[id]
		}
		return n
	}

	var prds, designs, features, withoutFields int
	acs := 0
	for _, f := range inv.Files {
		if !f.Kind.IsFeature() || f.Archived {
			continue
		}
		features++
		switch f.Kind {
		case KindPRD:
			prds++
			if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path))); err == nil {
				acs += len(acIDRegex.FindAllString(string(data), -1))
			}
		case KindDesign:
			designs++
		}
		if f.Frontmatter.Feature == "" || f.Frontmatter.Status == "" {
			withoutFields++
		}
	}

	cs := []Criterion{
		{Level: 1, Name: "Verfassung vorhanden", Met: inv.ConstitutionExists,
			Detail: inv.Config.Artifacts.Constitution,
			Advice: fmt.Sprintf("Verfassung %s anlegen (Schritt „Verfassung“ von sdd adopt --apply)", inv.Config.Artifacts.Constitution)},
		{Level: 1, Name: "Verfassung ohne Befunde (SDD-001, SDD-002a)", Met: inv.ConstitutionExists && count("SDD-001", "SDD-002a") == 0,
			Detail: fmt.Sprintf("%d Befunde", count("SDD-001", "SDD-002a")),
			Advice: "fehlende Pflichtabschnitte Architektur/Layering, Security, Strikte Regeln ergänzen"},
		{Level: 2, Name: "PRD und Tech Spec getrennt vorhanden", Met: prds > 0 && designs > 0,
			Detail: fmt.Sprintf("%d PRD, %d Tech Spec", prds, designs),
			Advice: "für das nächste Feature `sdd new --feature <slug>` verwenden"},
		{Level: 2, Name: "Akzeptanzkriterien mit Kennung", Met: acs > 0 && count("SDD-021") == 0,
			Detail: fmt.Sprintf("%d AC-Kennungen, %d Befunde SDD-021", acs, count("SDD-021")),
			Advice: fmt.Sprintf("Akzeptanzkriterien in den PRDs mit AC-<n> kennzeichnen (%d Zeilen ohne Kennung)", count("SDD-021"))},
		{Level: 3, Name: "CI ruft sdd gate auf", Met: inv.CallsGate,
			Detail: fmt.Sprintf("%d Workflow-Datei(en)", len(inv.Workflows)),
			Advice: "den Workflow aus `sdd init` übernehmen (.github/workflows/sdd.yml) oder `sdd gate pre-merge` in die CI einbauen"},
		{Level: 3, Name: "Baseline vorhanden", Met: inv.BaselineExists,
			Detail: ".sdd/baseline.json",
			Advice: "Baseline aufnehmen (Schritt „Baseline“ von sdd adopt --apply oder `sdd baseline`)"},
		{Level: 4, Name: "Feature-Artefakte tragen feature: und status", Met: features > 0 && withoutFields == 0,
			Detail: fmt.Sprintf("%d von %d ohne beide Felder", withoutFields, features),
			Advice: fmt.Sprintf("Frontmatter ergänzen (%d Dateien; Schritt „Frontmatter“ von sdd adopt --apply)", withoutFields)},
		{Level: 4, Name: "Archiv konfiguriert", Met: inv.Config.Artifacts.Archive != "",
			Detail: inv.Config.Artifacts.Archive,
			Advice: "artifacts.archive in sdd.config.yaml setzen"},
		{Level: 4, Name: "Archivierung aktuell (SDD-050)", Met: count("SDD-050") == 0,
			Detail: fmt.Sprintf("%d Befunde", count("SDD-050")),
			Advice: fmt.Sprintf("%d abgeschlossene Feature-Artefakte nach %s verschieben und status: archived setzen", count("SDD-050"), inv.Config.Artifacts.Archive)},
	}

	m := Maturity{Criteria: cs}
	for level := 1; level <= 4; level++ {
		for i := range cs {
			if cs[i].Level == level && !cs[i].Met {
				m.Next = &cs[i]
				return m
			}
		}
		m.Level = level
	}
	return m
}
