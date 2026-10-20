package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

var (
	headingRegex        = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	explicitAnchorRegex = regexp.MustCompile(`\{#([a-zA-Z0-9_-]+)\}\s*$`)
	nonSlugCharsRegex   = regexp.MustCompile(`[^a-z0-9-]+`)
)

func registerBuiltins(r *Registry) {
	r.Register("constitution_present", checkSDD001)
	r.Register("constitution_markdown_sections", checkSDD002a)
	r.Register("feature_prd_present", checkSDD020)
	r.Register("feature_design_present", checkSDD030)
	r.Register("acceptance_criteria_ids", checkSDD021)
	r.Register("acceptance_criteria_tested", checkSDD022)
	r.Register("archive_state", checkSDD050)
	r.Register("diff_within_intent", checkSDD024)
	r.Register("adr_immutable", checkSDD011)
	r.Register("constitution_change_has_adr", checkSDD010)
	r.Register("no_links_to_archive", checkSDD051)
	r.Register("tasks_reference_acs", checkSDD041)
	r.Register("design_references_acs", checkSDD031)
	r.Register("prd_non_goals", checkSDD023)
	r.Register("catalog_handbook_section", checkSDD070)
	r.Register("catalog_check_registered", func(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
		return checkSDD071(ctx, rule, r)
	})
	r.Register("catalog_gate_coverage", checkSDD072)
	r.Register("catalog_error_contract", checkSDD073)
}

// checkSDD070 (SDD-070): Every rule in the catalog is documented in the handbook with a heading containing its ID.
func checkSDD070(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	var findings []report.Finding
	for idx, r := range ctx.Catalog.Rules {
		filePart, anchorPart := splitPathAndAnchor(r.Handbook)
		if filePart == "" {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Regel %s hat kein gültiges handbook-Feld", r.ID),
				ContentHash: HashContent(r.ID + ":handbook:empty"),
			})
			continue
		}
		data, err := ctx.ReadHandbookFile(filePart)
		if err != nil {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Regel %s verweist auf nicht lesbare Handbuchdatei %q", r.ID, filePart),
				ContentHash: HashContent(r.ID + ":handbook:missing:" + filePart),
			})
			continue
		}
		if !hasHeadingForRule(string(data), r.ID, anchorPart) {
			findings = append(findings, report.Finding{
				Path:        filePart,
				Line:        1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Regel %s hat keine passende Überschrift (Anker #%s) in %s", r.ID, anchorPart, filePart),
				ContentHash: HashContent(r.ID + ":handbook:heading:" + r.Handbook),
			})
		}
	}
	return findings, nil
}

// checkSDD071 (SDD-071): Every rule in the catalog has a non-empty check name registered in the runtime registry.
func checkSDD071(ctx *Context, selfRule catalog.Rule, reg *Registry) ([]report.Finding, error) {
	var findings []report.Finding
	for idx, r := range ctx.Catalog.Rules {
		if strings.TrimSpace(r.Check) == "" {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Regel %s hat keinen check-Namen eingetragen", r.ID),
				ContentHash: HashContent(r.ID + ":check:empty"),
			})
			continue
		}
		if !reg.Has(r.Check) {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Regel %s verweist auf nicht registrierten Check %q", r.ID, r.Check),
				ContentHash: HashContent(r.ID + ":check:unregistered:" + r.Check),
			})
		}
	}
	return findings, nil
}

// checkSDD072 (SDD-072): Every rule active in the configured profile is included in at least one effective gate Eff(g).
func checkSDD072(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	unionEff := make(map[string]bool)
	for _, g := range catalog.AllGates {
		eff := EffectiveRulesForGate(ctx.Config, ctx.Catalog, g)
		for ruleID := range eff {
			unionEff[ruleID] = true
		}
	}

	var findings []report.Finding
	for idx, r := range ctx.Catalog.Rules {
		if !r.IsActiveInProfile(ctx.Config.Profile) {
			continue
		}
		activeInAnyGate := false
		if unionEff[r.ID] {
			for _, g := range catalog.AllGates {
				eff := EffectiveRulesForGate(ctx.Config, ctx.Catalog, g)
				if eff[r.ID] && r.EffectiveSeverity(g) != "" {
					activeInAnyGate = true
					break
				}
			}
		}
		if !activeInAnyGate {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Aktive Regel %s (Profil %s) ist in keinem Gate effektiv aktiv", r.ID, r.Profile),
				ContentHash: HashContent(r.ID + ":gate:orphaned"),
			})
		}
	}
	return findings, nil
}

// checkSDD073 (SDD-073): Every rule with error severity (in default or by_gate) has a valid contract anchor in the handbook.
func checkSDD073(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	var findings []report.Finding
	for idx, r := range ctx.Catalog.Rules {
		if !r.HasErrorSeverity() {
			continue
		}
		if strings.TrimSpace(r.Contract) == "" {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("error-Regel %s besitzt keinen Prüfvertrag (contract ist leer)", r.ID),
				ContentHash: HashContent(r.ID + ":contract:empty"),
			})
			continue
		}
		filePart, anchorPart := splitPathAndAnchor(r.Contract)
		if filePart == "" || anchorPart == "" {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("error-Regel %s hat ungültiges contract-Format %q (erwartet: <datei>#<anker>)", r.ID, r.Contract),
				ContentHash: HashContent(r.ID + ":contract:format:" + r.Contract),
			})
			continue
		}
		data, err := ctx.ReadHandbookFile(filePart)
		if err != nil {
			findings = append(findings, report.Finding{
				Path:        ctx.CatalogPath,
				Line:        idx + 1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("error-Regel %s verweist im Prüfvertrag auf fehlende Handbuchdatei %q", r.ID, filePart),
				ContentHash: HashContent(r.ID + ":contract:missing:" + filePart),
			})
			continue
		}
		if !hasHeadingAnchor(string(data), anchorPart) {
			findings = append(findings, report.Finding{
				Path:        filePart,
				Line:        1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("error-Regel %s verweist auf nicht existierenden Prüfvertrags-Anker #%s in %s", r.ID, anchorPart, filePart),
				ContentHash: HashContent(r.ID + ":contract:anchor:" + r.Contract),
			})
		}
	}
	return findings, nil
}

func splitPathAndAnchor(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func hasHeadingForRule(markdown, ruleID, expectedAnchor string) bool {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		m := headingRegex.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		headingText := m[2]
		if !strings.Contains(headingText, ruleID) {
			continue
		}
		if expectedAnchor == "" || matchHeadingAnchor(headingText, expectedAnchor) {
			return true
		}
	}
	return false
}

func hasHeadingAnchor(markdown, expectedAnchor string) bool {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		m := headingRegex.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		if matchHeadingAnchor(m[2], expectedAnchor) {
			return true
		}
	}
	return false
}

func matchHeadingAnchor(headingText, expectedAnchor string) bool {
	if am := explicitAnchorRegex.FindStringSubmatch(headingText); am != nil {
		if strings.EqualFold(am[1], expectedAnchor) {
			return true
		}
		headingText = strings.TrimSpace(explicitAnchorRegex.ReplaceAllString(headingText, ""))
	}
	slug := slugifyHeading(headingText)
	target := strings.ToLower(expectedAnchor)
	return slug == target || strings.HasPrefix(slug, target+"-")
}

func slugifyHeading(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
		"—", "-",
		"–", "-",
		" ", "-",
	)
	s = replacer.Replace(s)
	s = nonSlugCharsRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
