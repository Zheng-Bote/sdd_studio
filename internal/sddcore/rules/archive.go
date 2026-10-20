package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// today returns the current local date at midnight; Context.Now allows tests to fix the clock.
func (c *Context) today() time.Time {
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// checkSDD050 (SDD-050): Completed feature artifacts are moved to the archive and carry status archived before a release.
func checkSDD050(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	artifacts, err := ctx.featureArtifacts()
	if err != nil {
		return nil, err
	}
	archiveDir := strings.TrimSuffix(strings.TrimSpace(ctx.Config.Artifacts.Archive), "/")
	today := ctx.today()

	var findings []report.Finding
	for _, a := range artifacts {
		if a.Frontmatter.Feature == "" || (archiveDir != "" && scan.InSpec(a.Path, archiveDir)) {
			continue
		}
		var reasons []string
		if a.Frontmatter.Status == "implemented" {
			reasons = append(reasons, "status: implemented")
		}
		if raw := strings.TrimSpace(a.Frontmatter.ArchiveAfter); raw != "" {
			date, err := time.Parse("2006-01-02", raw)
			switch {
			case err != nil:
				reasons = append(reasons, fmt.Sprintf("archive-after %q ist kein ISO-8601-Datum (YYYY-MM-DD)", raw))
			case date.Before(today):
				reasons = append(reasons, fmt.Sprintf("archive-after %s überschritten", raw))
			}
		}
		if len(reasons) == 0 {
			continue
		}
		target := "nach " + archiveDir + "/ verschieben"
		if archiveDir == "" {
			target = "artifacts.archive konfigurieren und dorthin verschieben"
		}
		findings = append(findings, report.Finding{
			Path:        a.Path,
			Line:        1,
			Message:     fmt.Sprintf("Feature %s ist abgeschlossen (%s), liegt aber nicht im Archiv — %s und status: archived setzen", a.Frontmatter.Feature, strings.Join(reasons, ", "), target),
			ContentHash: HashContent(rule.ID + ":" + a.Path + ":" + strings.Join(reasons, ",")),
		})
	}

	if archiveDir == "" {
		return findings, nil
	}
	archived, err := scan.ListArtifacts(ctx.RepoRoot, archiveDir, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Archiv %s nicht lesbar: %w", archiveDir, err)
	}
	for _, a := range archived {
		if a.Frontmatter.Feature == "" || inactiveStatuses[a.Frontmatter.Status] {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        a.Path,
			Line:        1,
			Message:     fmt.Sprintf("Feature %s liegt im Archiv, trägt aber status: %q statt archived", a.Frontmatter.Feature, a.Frontmatter.Status),
			ContentHash: HashContent(rule.ID + ":" + a.Path + ":status:" + a.Frontmatter.Status),
		})
	}
	return findings, nil
}
