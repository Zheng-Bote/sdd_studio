package scan

import (
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
