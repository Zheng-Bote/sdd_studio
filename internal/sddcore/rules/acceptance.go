package rules

import (
	"fmt"
	"os"
	"path"
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

// FallbackTestGlobs is the test file list used when tests.include is empty (concept section 9.5, handbook SDD-022).
var FallbackTestGlobs = []string{
	"**/*_test.go",
	"**/test_*.py",
	"**/*_test.py",
	"**/*.{test,spec}.{ts,tsx,js,jsx}",
	"**/*Test.java",
	"**/*_test.rs",
	"**/tests/**",
	"**/__tests__/**",
}

var (
	acDefRegex = regexp.MustCompile(`\bAC-([1-9][0-9]*)\b`)
	// acRefRegexes recognize AC references in test files. Greedy digits make a trailing boundary unnecessary;
	// the leading context is matched explicitly because \b does not separate camelCase or snake_case parts.
	acRefRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\bAC-([1-9][0-9]*)`),                   // AC-3 in comments, strings, names
		regexp.MustCompile(`(?:^|[^A-Z0-9])AC_?([1-9][0-9]*)`),     // TestAC3_Login, TestAC_3
		regexp.MustCompile(`(?i)(?:^|[^a-z0-9])ac_?([1-9][0-9]*)`), // test_ac_3, ac3, Ac3
	}
)

// acDefinition is the first occurrence of an AC ID in an active PRD.
type acDefinition struct {
	ID         string
	Number     string
	Feature    string // feature: of the PRD, or the file name without extension
	Path       string
	Line       int
	ByteOffset int
}

// definedACs collects the AC IDs of all active PRDs, ignoring tombstoned (struck-through) IDs.
func definedACs(ctx *Context) ([]acDefinition, error) {
	prds, err := activePRDs(ctx)
	if err != nil {
		return nil, err
	}
	var defs []acDefinition
	for _, prd := range prds {
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(prd.Path)))
		if err != nil {
			return nil, fmt.Errorf("PRD %s nicht lesbar: %w", prd.Path, err)
		}
		feature := prd.Frontmatter.Feature
		if feature == "" {
			feature = strings.TrimSuffix(path.Base(prd.Path), path.Ext(prd.Path))
		}
		seen := map[string]bool{}
		for _, l := range markdownLines(data) {
			text := acTombstoneRegex.ReplaceAllString(l.Text, "")
			for _, m := range acDefRegex.FindAllStringSubmatch(text, -1) {
				if seen[m[0]] {
					continue
				}
				seen[m[0]] = true
				defs = append(defs, acDefinition{ID: m[0], Number: m[1], Feature: feature, Path: prd.Path, Line: l.Number, ByteOffset: l.ByteOffset})
			}
		}
	}
	return defs, nil
}

// checkSDD022 (SDD-022): Every AC ID of an active PRD occurs in at least one test file (presence, not coverage).
func checkSDD022(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	defs, err := definedACs(ctx)
	if err != nil || len(defs) == 0 {
		return nil, err
	}

	include, source := ctx.Config.Tests.Include, "tests.include"
	if len(include) == 0 {
		include, source = FallbackTestGlobs, "Fallback-Liste"
	}
	testFiles, err := scan.WalkFiles(ctx.RepoRoot, include, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Testdateien nicht lesbar: %w", err)
	}
	if len(testFiles) == 0 {
		return []report.Finding{{
			Severity:    "warn",
			Path:        "sdd.config.yaml",
			Line:        1,
			Message:     fmt.Sprintf("Keine Testdatei gefunden (%s); AC-Referenzen nicht prüfbar — tests.include konfigurieren", source),
			ContentHash: HashContent(rule.ID + ":notests:" + source),
		}}, nil
	}

	// referencedIn maps an AC number to the contents of all test files referencing it.
	referencedIn := map[string][]string{}
	for _, f := range testFiles {
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(f)))
		if err != nil {
			return nil, fmt.Errorf("Testdatei %s nicht lesbar: %w", f, err)
		}
		numbers := map[string]bool{}
		for _, re := range acRefRegexes {
			for _, m := range re.FindAllStringSubmatch(string(data), -1) {
				numbers[m[1]] = true
			}
		}
		for n := range numbers {
			referencedIn[n] = append(referencedIn[n], string(data))
		}
	}

	// An AC number defined by several active PRDs is ambiguous: the test file must also name the feature slug.
	definedBy := map[string]map[string]bool{}
	for _, d := range defs {
		if definedBy[d.Number] == nil {
			definedBy[d.Number] = map[string]bool{}
		}
		definedBy[d.Number][d.Path] = true
	}

	var findings []report.Finding
	for _, d := range defs {
		files := referencedIn[d.Number]
		ambiguous := len(definedBy[d.Number]) > 1
		if !ambiguous && len(files) > 0 {
			continue
		}
		if ambiguous && anyMentionsSlug(files, d.Feature) {
			continue
		}
		msg := fmt.Sprintf("%s ist in keiner Testdatei (%s) referenziert", d.ID, source)
		if ambiguous {
			msg = fmt.Sprintf("%s ist in %d PRDs vergeben; keine Testdatei (%s) referenziert %s und nennt zugleich den Feature-Slug %s", d.ID, len(definedBy[d.Number]), source, d.ID, d.Feature)
		}
		findings = append(findings, report.Finding{
			Path:        d.Path,
			Line:        d.Line,
			ByteOffset:  d.ByteOffset,
			Message:     msg,
			ContentHash: HashContent(rule.ID + ":" + d.Path + ":" + d.ID),
		})
	}
	return findings, nil
}

// anyMentionsSlug reports whether one of the contents contains slug as a whole token (not as part of a longer slug).
func anyMentionsSlug(contents []string, slug string) bool {
	re := regexp.MustCompile(`(?:^|[^a-z0-9-])` + regexp.QuoteMeta(slug) + `(?:[^a-z0-9-]|$)`)
	for _, c := range contents {
		if re.MatchString(c) {
			return true
		}
	}
	return false
}
