package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// testDeclRegexes recognize test declarations across languages (SDD-025 contract).
var testDeclRegexes = []*regexp.Regexp{
	regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s*)?(?:Test|Benchmark|Fuzz|Example)\w*\s*\(`), // Go
	regexp.MustCompile(`^\s*(?:async\s+)?def\s+test_?\w*\s*\(`),                               // Python
	regexp.MustCompile(`^\s*(?:it|test|describe)(?:\.\w+)*\s*\(`),                             // JS/TS
	regexp.MustCompile(`^\s*TEST(?:_F|_P)?\s*\(`),                                             // C++ GoogleTest
	regexp.MustCompile(`^\s*#\[test\]`),                                                       // Rust
	regexp.MustCompile(`^\s*@Test\b`),                                                         // Java/Kotlin
}

// attachedRegex matches comment and attribute lines that belong to the declaration directly below them.
var attachedRegex = regexp.MustCompile(`^\s*(?://|#|/\*|\*|--|@|"""|''')`)

func isTestDecl(line string) bool {
	for _, re := range testDeclRegexes {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// declRegions returns, per line, whether it lies in the region of a test declaration: from the first line of the
// comment/attribute block directly above the declaration (no blank line in between) to the line before the next
// declaration of the same or a shallower indentation, or to the end of the file.
func declRegions(lines []string) []bool {
	type decl struct{ start, indent int }
	var decls []decl
	for i, l := range lines {
		if !isTestDecl(l) {
			continue
		}
		// An attribute line (#[test], @Test) directly above another declaration line is part of that block.
		if len(decls) > 0 && decls[len(decls)-1].start < i && i > 0 && isTestDecl(lines[i-1]) && attachedRegex.MatchString(lines[i-1]) {
			continue
		}
		start := i
		for start > 0 && strings.TrimSpace(lines[start-1]) != "" && attachedRegex.MatchString(lines[start-1]) && !isTestDecl(lines[start-1]) {
			start--
		}
		decls = append(decls, decl{start: start, indent: indentOf(l)})
	}
	in := make([]bool, len(lines))
	for k, d := range decls {
		end := len(lines)
		for _, next := range decls[k+1:] {
			if next.indent <= d.indent {
				end = next.start
				break
			}
		}
		for i := d.start; i < end; i++ {
			in[i] = true
		}
	}
	return in
}

// checkSDD025 (SDD-025): An AC reference that SDD-022 accepts also lies within a test declaration's region.
func checkSDD025(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	defs, err := definedACs(ctx)
	if err != nil || len(defs) == 0 {
		return nil, err
	}
	files, _, err := loadTestFiles(ctx)
	if err != nil || len(files) == 0 {
		return nil, err // no test files: SDD-022 warns
	}
	regions := make([][]bool, len(files))
	for i, f := range files {
		regions[i] = declRegions(f.Lines)
	}

	m, err := newMatcherFor(ctx, defs)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, d := range defs {
		if !referencedAnywhere(files, m, d) {
			continue // SDD-022 reports it
		}
		inDecl := false
		for i, f := range files {
			for j, line := range f.Lines {
				if regions[i][j] && m.refers(line, d) {
					inDecl = true
				}
			}
		}
		if inDecl {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        d.Path,
			Line:        d.Line,
			ByteOffset:  d.ByteOffset,
			Message:     fmt.Sprintf("%s ist in Testdateien referenziert, aber nicht im Bereich einer Testdeklaration (Name, Kommentarblock direkt darüber oder Rumpf)", d.ID),
			ContentHash: HashContent(rule.ID + ":" + d.Path + ":" + d.ID),
		})
	}
	return findings, nil
}
