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

// testFile is one file from tests.include (or the fallback list) split into lines.
type testFile struct {
	Path  string
	Lines []string
}

// loadTestFiles reads all test files and reports which glob list selected them.
func loadTestFiles(ctx *Context) ([]testFile, string, error) {
	include, source := ctx.Config.Tests.Include, "tests.include"
	if len(include) == 0 {
		include, source = FallbackTestGlobs, "Fallback-Liste"
	}
	paths, err := scan.WalkFiles(ctx.RepoRoot, include, ctx.Config.Exclude)
	if err != nil {
		return nil, source, fmt.Errorf("Testdateien nicht lesbar: %w", err)
	}
	var files []testFile
	for _, p := range paths {
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(p)))
		if err != nil {
			return nil, source, fmt.Errorf("Testdatei %s nicht lesbar: %w", p, err)
		}
		files = append(files, testFile{Path: p, Lines: strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")})
	}
	return files, source, nil
}

// acMatcher decides whether a test file line references an AC definition (SDD-022 contract, items 3 and 4).
type acMatcher struct {
	definedBy map[string]map[string]bool // AC number -> PRD paths defining it
	slugs     []string                   // feature slugs of all PRDs, archived ones included
}

func newACMatcher(defs []acDefinition, slugs []string) acMatcher {
	m := acMatcher{definedBy: map[string]map[string]bool{}, slugs: slugs}
	for _, d := range defs {
		if m.definedBy[d.Number] == nil {
			m.definedBy[d.Number] = map[string]bool{}
		}
		m.definedBy[d.Number][d.Path] = true
	}
	return m
}

func (m acMatcher) ambiguous(d acDefinition) bool {
	return len(m.definedBy[d.Number]) > 1
}

// refers reports whether line references d. As soon as more than one feature is known (archived ones included),
// the line must also name d's feature slug; a single-feature project needs no slug.
func (m acMatcher) refers(line string, d acDefinition) bool {
	found := false
	for _, re := range acRefRegexes {
		for _, match := range re.FindAllStringSubmatch(line, -1) {
			if match[1] == d.Number {
				found = true
			}
		}
	}
	if !found {
		return false
	}
	if len(m.slugs) <= 1 && !m.ambiguous(d) {
		return true
	}
	return anyMentionsSlug([]string{line}, d.Feature)
}

// knownFeatureSlugs returns the feature slugs of all PRDs, active and archived (frontmatter or file name).
func knownFeatureSlugs(ctx *Context) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, spec := range []string{ctx.Config.Artifacts.PRD, ctx.Config.Artifacts.Archive} {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		list, err := scan.ListArtifacts(ctx.RepoRoot, spec, ctx.Config.Exclude)
		if err != nil {
			return nil, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", spec, err)
		}
		for _, a := range list {
			slug := a.Frontmatter.Feature
			if slug == "" {
				slug = strings.TrimSuffix(path.Base(a.Path), path.Ext(a.Path))
			}
			if !seen[slug] {
				seen[slug] = true
				out = append(out, slug)
			}
		}
	}
	return out, nil
}

// newMatcherFor builds the AC matcher for the given definitions with all known feature slugs.
func newMatcherFor(ctx *Context, defs []acDefinition) (acMatcher, error) {
	slugs, err := knownFeatureSlugs(ctx)
	if err != nil {
		return acMatcher{}, err
	}
	return newACMatcher(defs, slugs), nil
}

// checkSDD022 (SDD-022): Every AC ID of an active PRD occurs in at least one test file (presence, not coverage).
func checkSDD022(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	defs, err := definedACs(ctx)
	if err != nil || len(defs) == 0 {
		return nil, err
	}
	files, source, err := loadTestFiles(ctx)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []report.Finding{{
			Severity:    "warn",
			Path:        "sdd.config.yaml",
			Line:        1,
			Message:     fmt.Sprintf("Keine Testdatei gefunden (%s); AC-Referenzen nicht prüfbar — tests.include konfigurieren", source),
			ContentHash: HashContent(rule.ID + ":notests:" + source),
		}}, nil
	}

	m, err := newMatcherFor(ctx, defs)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, d := range defs {
		if referencedAnywhere(files, m, d) {
			continue
		}
		msg := fmt.Sprintf("%s ist in keiner Testdatei (%s) referenziert", d.ID, source)
		if len(m.slugs) > 1 {
			msg = fmt.Sprintf("%s (Feature %s) ist in keiner Zeile einer Testdatei (%s) zusammen mit dem Feature-Slug referenziert — Konvention `// %s %s: …`", d.ID, d.Feature, source, d.Feature, d.ID)
		}
		if m.ambiguous(d) {
			msg = fmt.Sprintf("%s ist in %d PRDs vergeben; keine Zeile einer Testdatei (%s) referenziert %s und nennt zugleich den Feature-Slug %s", d.ID, len(m.definedBy[d.Number]), source, d.ID, d.Feature)
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

func referencedAnywhere(files []testFile, m acMatcher, d acDefinition) bool {
	for _, f := range files {
		for _, line := range f.Lines {
			if m.refers(line, d) {
				return true
			}
		}
	}
	return false
}

// anyMentionsSlug reports whether one of the lines contains slug as a whole token (not as part of a longer slug).
func anyMentionsSlug(contents []string, slug string) bool {
	re := regexp.MustCompile(`(?:^|[^a-z0-9-])` + regexp.QuoteMeta(slug) + `(?:[^a-z0-9-]|$)`)
	for _, c := range contents {
		if re.MatchString(c) {
			return true
		}
	}
	return false
}

// ACTrace is the traceability of one acceptance criterion (used by `sdd mcp` / sdd_trace).
type ACTrace struct {
	ID        string   `json:"id"`
	PRD       string   `json:"prd"`
	Line      int      `json:"line"`
	TechSpecs []string `json:"tech_specs"`
	Tests     []string `json:"tests"` // "path:line", matched exactly like SDD-022
	Tasks     []string `json:"tasks"` // "path:line" of task lines naming the ID
}

// Trace follows each AC of a feature's active PRDs to Tech Specs, tests and tasks.
func Trace(ctx *Context, feature string) ([]ACTrace, error) {
	defs, err := definedACs(ctx)
	if err != nil {
		return nil, err
	}
	files, _, err := loadTestFiles(ctx)
	if err != nil {
		return nil, err
	}
	m, err := newMatcherFor(ctx, defs)
	if err != nil {
		return nil, err
	}
	designs, err := activeArtifacts(ctx, ctx.Config.Artifacts.Design)
	if err != nil {
		return nil, err
	}
	plans, err := activeArtifacts(ctx, ctx.Config.Artifacts.Plans)
	if err != nil {
		return nil, err
	}

	out := []ACTrace{}
	for _, d := range defs {
		if d.Feature != feature {
			continue
		}
		tr := ACTrace{ID: d.ID, PRD: d.Path, Line: d.Line, TechSpecs: []string{}, Tests: []string{}, Tasks: []string{}}
		for _, ds := range designs {
			if featureOf(ds, ctx.Config.Artifacts.Design) != feature {
				continue
			}
			lines, err := readLines(ctx, ds.Path)
			if err != nil {
				return nil, err
			}
			for _, l := range lines {
				if containsID(l.Text, d.ID) {
					tr.TechSpecs = append(tr.TechSpecs, ds.Path)
					break
				}
			}
		}
		for _, f := range files {
			for i, line := range f.Lines {
				if m.refers(line, d) {
					tr.Tests = append(tr.Tests, fmt.Sprintf("%s:%d", f.Path, i+1))
				}
			}
		}
		for _, p := range plans {
			if featureOf(p, ctx.Config.Artifacts.Plans) != feature {
				continue
			}
			lines, err := readLines(ctx, p.Path)
			if err != nil {
				return nil, err
			}
			for _, l := range lines {
				if checkboxRegex.MatchString(l.Text) && containsID(l.Text, d.ID) {
					tr.Tasks = append(tr.Tasks, fmt.Sprintf("%s:%d", p.Path, l.Number))
				}
			}
		}
		out = append(out, tr)
	}
	return out, nil
}

func containsID(text, id string) bool {
	for _, found := range acIDRegex.FindAllString(acTombstoneRegex.ReplaceAllString(text, ""), -1) {
		if found == id {
			return true
		}
	}
	return false
}
