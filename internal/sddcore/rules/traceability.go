package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

var nonGoalsHeadingRegex = regexp.MustCompile(`^(?:nicht-?ziele|non-?goals|out of scope)$`)

// activeArtifacts returns the artifacts selected by spec that are neither archived nor superseded nor in the archive.
func activeArtifacts(ctx *Context, spec string) ([]scan.Artifact, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	list, err := scan.ListArtifacts(ctx.RepoRoot, spec, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", spec, err)
	}
	archive := strings.TrimSpace(ctx.Config.Artifacts.Archive)
	var out []scan.Artifact
	for _, a := range list {
		if inactiveStatuses[a.Frontmatter.Status] || (archive != "" && scan.InSpec(a.Path, archive)) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func readLines(ctx *Context, rel string) ([]markdownLine, error) {
	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("%s nicht lesbar: %w", rel, err)
	}
	return markdownLines(data), nil
}

// checkSDD023 (SDD-023): Every active PRD has a non-empty section "Nicht-Ziele" / "Non-Goals" / "Out of Scope".
func checkSDD023(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	prds, err := activeArtifacts(ctx, ctx.Config.Artifacts.PRD)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, prd := range prds {
		lines, err := readLines(ctx, prd.Path)
		if err != nil {
			return nil, err
		}
		found, filled, line := false, false, 1
		for _, l := range lines {
			if l.Heading > 0 {
				if found {
					break // the section ends at the next heading of any level
				}
				if nonGoalsHeadingRegex.MatchString(strings.ToLower(normalizedHeading(l.Text))) {
					found, line = true, l.Number
				}
				continue
			}
			if found && strings.TrimSpace(l.Text) != "" {
				filled = true
				break
			}
		}
		var msg string
		switch {
		case !found:
			msg = fmt.Sprintf("PRD %s hat keinen Abschnitt „Nicht-Ziele“ (Non-Goals, Out of Scope)", prd.Path)
		case !filled:
			msg = fmt.Sprintf("Abschnitt „Nicht-Ziele“ in %s ist leer", prd.Path)
		default:
			continue
		}
		findings = append(findings, report.Finding{Path: prd.Path, Line: line, Message: msg, ContentHash: HashContent(rule.ID + ":" + prd.Path + ":" + fmt.Sprint(found))})
	}
	return findings, nil
}

// normalizedHeading strips the ATX markers, an explicit {#anchor} and closing hashes from a heading line.
func normalizedHeading(line string) string {
	m := headingRegex.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	text := strings.TrimSpace(explicitAnchorRegex.ReplaceAllString(m[2], ""))
	return strings.TrimSpace(strings.TrimRight(text, "#"))
}
