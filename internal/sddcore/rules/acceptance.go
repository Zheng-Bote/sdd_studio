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

var (
	acIDRegex        = regexp.MustCompile(`\bAC-[1-9][0-9]*\b`)
	acTombstoneRegex = regexp.MustCompile(`~~[^~]*\bAC-[0-9]+\b[^~]*~~`)
	checkboxRegex    = regexp.MustCompile(`^\s*(?:[-*+]|[0-9]+[.)])\s+\[[ xX]\]`)
	listItemRegex    = regexp.MustCompile(`^\s*(?:[-*+]|[0-9]+[.)])\s+\S`)
	acHeadingRegex   = regexp.MustCompile(`(?i)akzeptanzkriterien|acceptance criteria`)
)

// inactiveStatuses are feature artifact statuses whose acceptance criteria are history, not requirements.
var inactiveStatuses = map[string]bool{"archived": true, "superseded": true}

// markdownLine is one line of a Markdown file outside fenced code blocks.
type markdownLine struct {
	Number     int // 1-based
	ByteOffset int
	Text       string
	Heading    int // ATX heading level, 0 if not a heading
}

// markdownLines splits a Markdown document into lines, dropping fenced code blocks and the frontmatter block.
func markdownLines(data []byte) []markdownLine {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	raw := strings.Split(text, "\n")
	var out []markdownLine
	offset := 0
	fence := ""
	inFrontmatter := len(raw) > 0 && raw[0] == "---"
	for i, line := range raw {
		lineOffset := offset
		offset += len(line) + 1
		if inFrontmatter {
			if i > 0 && line == "---" {
				inFrontmatter = false
			}
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		level := 0
		if m := headingRegex.FindStringSubmatch(line); m != nil {
			level = len(m[1])
		}
		out = append(out, markdownLine{Number: i + 1, ByteOffset: lineOffset, Text: line, Heading: level})
	}
	return out
}

// activePRDs returns the PRD artifacts (below artifacts.prd) that are not archived or superseded.
func activePRDs(ctx *Context) ([]scan.Artifact, error) {
	dir := strings.TrimSpace(ctx.Config.Artifacts.PRD)
	if dir == "" {
		return nil, nil
	}
	list, err := scan.ListArtifacts(ctx.RepoRoot, dir, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", dir, err)
	}
	var out []scan.Artifact
	for _, a := range list {
		if !inactiveStatuses[a.Frontmatter.Status] {
			out = append(out, a)
		}
	}
	return out, nil
}

// checkSDD021 (SDD-021): Every line recognized as an acceptance criterion in an active PRD carries an ID AC-<n>.
func checkSDD021(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	prds, err := activePRDs(ctx)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, prd := range prds {
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(prd.Path)))
		if err != nil {
			return nil, fmt.Errorf("PRD %s nicht lesbar: %w", prd.Path, err)
		}
		acSectionLevel := 0 // heading level of the enclosing acceptance-criteria section, 0 if outside
		for _, l := range markdownLines(data) {
			if l.Heading > 0 {
				switch {
				case acHeadingRegex.MatchString(l.Text):
					acSectionLevel = l.Heading
				case acSectionLevel > 0 && l.Heading <= acSectionLevel:
					acSectionLevel = 0
				}
				continue
			}
			if !isAcceptanceCriterionLine(l.Text, acSectionLevel > 0) {
				continue
			}
			if acTombstoneRegex.MatchString(l.Text) || acIDRegex.MatchString(l.Text) {
				continue
			}
			findings = append(findings, report.Finding{
				Path:        prd.Path,
				Line:        l.Number,
				ByteOffset:  l.ByteOffset,
				Message:     fmt.Sprintf("Akzeptanzkriterium ohne gültige Kennung AC-<n>: %q", strings.TrimSpace(l.Text)),
				ContentHash: HashContent(rule.ID + ":" + prd.Path + ":" + l.Text),
			})
		}
	}
	return findings, nil
}

// isAcceptanceCriterionLine implements the recognition part of the SDD-021 contract.
func isAcceptanceCriterionLine(line string, inACSection bool) bool {
	if checkboxRegex.MatchString(line) {
		return true
	}
	if !listItemRegex.MatchString(line) {
		return false
	}
	return inACSection || strings.Contains(line, "AC-")
}
