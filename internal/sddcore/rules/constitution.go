package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// checkSDD001 (SDD-001): The file configured in artifacts.constitution exists, is a regular file inside the repository and is not empty.
func checkSDD001(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	rel := strings.TrimSpace(ctx.Config.Artifacts.Constitution)
	if rel == "" {
		return []report.Finding{{
			Path:        config.DefaultConfigFileName,
			Line:        1,
			Message:     "artifacts.constitution ist nicht konfiguriert",
			ContentHash: HashContent("constitution:unconfigured"),
		}}, nil
	}

	cleanRel, ok := repoRelative(rel)
	if !ok {
		return []report.Finding{{
			Path:        config.DefaultConfigFileName,
			Line:        1,
			Message:     fmt.Sprintf("artifacts.constitution %q liegt außerhalb des Repository-Wurzelverzeichnisses", rel),
			ContentHash: HashContent("constitution:outside:" + rel),
		}}, nil
	}

	info, err := os.Stat(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []report.Finding{{
				Path:        cleanRel,
				Line:        1,
				Message:     fmt.Sprintf("Verfassung %s existiert nicht", cleanRel),
				ContentHash: HashContent("constitution:missing:" + cleanRel),
			}}, nil
		}
		return nil, fmt.Errorf("Verfassung %s nicht lesbar: %w", cleanRel, err)
	}
	if !info.Mode().IsRegular() {
		return []report.Finding{{
			Path:        cleanRel,
			Line:        1,
			Message:     fmt.Sprintf("Verfassung %s ist keine reguläre Datei", cleanRel),
			ContentHash: HashContent("constitution:notregular:" + cleanRel),
		}}, nil
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		return nil, fmt.Errorf("Verfassung %s nicht lesbar: %w", cleanRel, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return []report.Finding{{
			Path:        cleanRel,
			Line:        1,
			Message:     fmt.Sprintf("Verfassung %s ist leer", cleanRel),
			ContentHash: HashContent("constitution:empty:" + cleanRel),
		}}, nil
	}
	return nil, nil
}

// repoRelative normalizes a configured path and reports whether it stays inside the repository root.
func repoRelative(p string) (string, bool) {
	clean, err := scan.CleanRel(p)
	return clean, err == nil
}

// requiredConstitutionSections lists the headings every Markdown constitution must contain (SDD-002a).
var requiredConstitutionSections = []string{"Architektur/Layering", "Security", "Strikte Regeln"}

// checkSDD002a (SDD-002a): A Markdown constitution (any extension except .sdd) contains all required section headings.
// Missing or unreadable constitutions are reported by SDD-001 and skipped here.
func checkSDD002a(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	cleanRel, ok := repoRelative(strings.TrimSpace(ctx.Config.Artifacts.Constitution))
	if !ok || strings.HasSuffix(strings.ToLower(cleanRel), ".sdd") {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		return nil, nil
	}

	present := make(map[string]bool)
	for _, h := range markdownHeadings(string(data)) {
		present[normalizeSectionTitle(h)] = true
	}

	var findings []report.Finding
	for idx, section := range requiredConstitutionSections {
		if present[normalizeSectionTitle(section)] {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        cleanRel,
			Line:        1,
			ByteOffset:  idx,
			Message:     fmt.Sprintf("Verfassung %s enthält keinen Pflichtabschnitt %q", cleanRel, section),
			ContentHash: HashContent("constitution:section:" + cleanRel + ":" + section),
		})
	}
	return findings, nil
}

// markdownHeadings returns the text of all ATX headings outside fenced code blocks, without explicit {#anchor} IDs.
func markdownHeadings(markdown string) []string {
	var headings []string
	fence := ""
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			fence = "```"
			continue
		}
		if strings.HasPrefix(trimmed, "~~~") {
			fence = "~~~"
			continue
		}
		m := headingRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		headings = append(headings, strings.TrimSpace(explicitAnchorRegex.ReplaceAllString(m[2], "")))
	}
	return headings
}

// normalizeSectionTitle lowercases a heading and removes all whitespace and trailing closing hashes.
func normalizeSectionTitle(title string) string {
	title = strings.TrimRight(strings.TrimSpace(title), "#")
	return strings.ToLower(strings.Join(strings.Fields(title), ""))
}

// DefaultConstitutionSections are the known section labels of a .sdd constitution (sdd_process_3, section 10:
// the five section types plus SpecDD ownership), overridable via constitution_sections.
var DefaultConstitutionSections = []string{"Architecture", "Security", "Data", "Standards", "References", "Owns", "Can modify"}

var sddLabelRegex = regexp.MustCompile(`^([A-Za-z][A-Za-z -]*[A-Za-z]|[A-Za-z]):(?:\s|$)`)

// checkSDD002b (SDD-002b): A .sdd constitution follows the formal grammar.
func checkSDD002b(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	cleanRel, ok := repoRelative(strings.TrimSpace(ctx.Config.Artifacts.Constitution))
	if !ok || !strings.HasSuffix(strings.ToLower(cleanRel), ".sdd") {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		return nil, nil // SDD-001
	}
	labels := ctx.Config.ConstitutionSections
	if len(labels) == 0 {
		labels = DefaultConstitutionSections
	}
	known := map[string]bool{}
	for _, l := range labels {
		known[strings.ToLower(strings.TrimSpace(l))] = true
	}

	var findings []report.Finding
	add := func(line, offset int, text, msg string) {
		findings = append(findings, report.Finding{
			Path:        cleanRel,
			Line:        line,
			ByteOffset:  offset,
			Message:     msg,
			ContentHash: HashContent(rule.ID + ":" + cleanRel + ":" + msg + ":" + text),
		})
	}
	seenSpec := false
	offset := 0
	for i, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		n, lineOffset := i+1, offset
		offset += len(line) + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, "\t") {
			add(n, lineOffset, line, fmt.Sprintf("Zeile %d enthält einen Tabulator", n))
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if strings.TrimLeft(line, " ") == strings.TrimLeft(line, " \t") && indent%2 != 0 {
			add(n, lineOffset, line, fmt.Sprintf("Zeile %d: Einrückung von %d Leerzeichen ist kein Vielfaches von zwei", n, indent))
		}
		if !seenSpec {
			seenSpec = true
			if !regexp.MustCompile(`^Spec:\s*\S`).MatchString(line) {
				add(n, lineOffset, line, fmt.Sprintf("Erste inhaltliche Zeile %d ist nicht `Spec: <Name>`", n))
			}
			continue
		}
		if indent > 0 {
			continue
		}
		m := sddLabelRegex.FindStringSubmatch(line)
		switch {
		case m == nil:
			add(n, lineOffset, line, fmt.Sprintf("Zeile %d auf oberster Ebene ist kein Section-Label (`Label:`)", n))
		case !known[strings.ToLower(m[1])]:
			add(n, lineOffset, line, fmt.Sprintf("Zeile %d: unbekanntes Section-Label %q (bekannt: %s)", n, m[1], strings.Join(labels, ", ")))
		}
	}
	if !seenSpec {
		add(1, 0, "", "Verfassung enthält keine Zeile `Spec: <Name>`")
	}
	return findings, nil
}
