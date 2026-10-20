package rules

import (
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// isDocumentation reports whether a changed file is an SDD artifact, documentation or tool state rather than an
// implementation change (SDD-024 compares only the latter with the declared intent).
func isDocumentation(ctx *Context, f string) bool {
	a := ctx.Config.Artifacts
	if c, ok := repoRelative(a.Constitution); ok && c == f {
		return true
	}
	for _, spec := range []string{a.PRD, a.Design, a.Plans, a.Archive} {
		if strings.TrimSpace(spec) != "" && scan.InSpec(f, spec) {
			return true
		}
	}
	for _, dir := range append([]string{a.ADR, ".sdd"}, a.Docs...) {
		if strings.TrimSpace(dir) != "" && scan.UnderDir(f, dir) {
			return true
		}
	}
	return false
}

// checkSDD024 (SDD-024): In pre-merge every changed implementation file lies within the declared touches of the
// resolved features.
func checkSDD024(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if ctx.Gate != "pre-merge" {
		return nil, nil
	}
	scope, err := resolveFeatures(ctx)
	if err != nil || len(scope.Slugs) == 0 {
		return nil, err // unresolved features are SDD-020/030's concern
	}
	var touches []string
	for _, slug := range scope.Slugs {
		intent, err := declaredIntent(ctx, slug)
		if err != nil {
			return nil, err
		}
		touches = append(touches, intent...)
	}
	if len(touches) == 0 {
		return nil, nil // no declared intent to compare with
	}
	diff, err := ctx.changedFiles()
	if err != nil {
		return nil, err
	}
	features := strings.Join(scope.Slugs, ", ")
	var findings []report.Finding
	for idx, f := range diff {
		if isDocumentation(ctx, f) || overlapsAny(touches, f) {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        f,
			Line:        1,
			ByteOffset:  idx,
			Message:     fmt.Sprintf("Geänderte Datei liegt außerhalb der erklärten Absicht (touches) von %s — touches im Feature-Frontmatter nachführen oder die Änderung abtrennen", features),
			ContentHash: HashContent(rule.ID + ":" + features + ":" + f),
		})
	}
	return findings, nil
}

func overlapsAny(patterns []string, f string) bool {
	for _, p := range patterns {
		if scan.Overlaps(p, f) {
			return true
		}
	}
	return false
}
