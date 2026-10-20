package scan

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter holds the artifact fields the rule checks evaluate (concept section 9.4).
type Frontmatter struct {
	Feature      string   `yaml:"feature"`
	Status       string   `yaml:"status"`
	ArchiveAfter string   `yaml:"archive-after"`
	Touches      []string `yaml:"touches"`
}

// ParseFrontmatter extracts a YAML frontmatter block delimited by `---` lines at the very start of the file.
// It returns ok=false if the file has no frontmatter block or the block is not valid YAML.
func ParseFrontmatter(data []byte) (Frontmatter, bool) {
	var fm Frontmatter
	text := strings.TrimPrefix(string(data), "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return fm, false
	}
	rest := text[len("---\n"):]
	var block string
	switch {
	case strings.HasPrefix(rest, "---\n") || rest == "---":
		block = ""
	default:
		end := strings.Index(rest, "\n---\n")
		if end < 0 {
			if !strings.HasSuffix(rest, "\n---") {
				return fm, false
			}
			end = len(rest) - len("\n---")
		}
		block = rest[:end]
	}
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		return Frontmatter{}, false
	}
	fm.Feature = strings.TrimSpace(fm.Feature)
	fm.Status = strings.TrimSpace(fm.Status)
	return fm, true
}

var adrFileRegex = regexp.MustCompile(`^[0-9]{4}-.*\.md$`)

// IsADRFileName reports whether a file name follows the ADR convention NNNN-*.md.
func IsADRFileName(name string) bool {
	return adrFileRegex.MatchString(name)
}

// NormalizeText unifies line endings, drops whitespace-only lines and collapses whitespace runs to one space
// (SDD-010 formatting exception, SDD-011 content hash).
func NormalizeText(s string) string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			out = append(out, strings.Join(fields, " "))
		}
	}
	return strings.Join(out, "\n")
}

// Body returns the file content after a leading frontmatter block (or the whole content if there is none).
func Body(data []byte) string {
	text := strings.ReplaceAll(strings.TrimPrefix(string(data), "\uFEFF"), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	if end := strings.Index(text[4:], "\n---\n"); end >= 0 {
		return text[4+end+5:]
	}
	if strings.HasSuffix(text, "\n---") {
		return ""
	}
	return text
}
