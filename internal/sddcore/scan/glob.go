// Package scan provides the language-neutral file, frontmatter and Git primitives the rule checks build on.
package scan

import (
	"regexp"
	"strings"
	"sync"
)

// Glob semantics (handbook section "Glob-Syntax"):
//   - Patterns are matched against slash-separated paths relative to the repository root and are anchored at the root.
//   - `**` matches any number of path segments (including none), `*` any run of characters within one segment,
//     `?` exactly one character within a segment, `{a,b}` one of the alternatives, `[...]` a character class.
//   - A pattern without `/` therefore only matches files in the root directory; use `**/` for any depth.

var (
	globCacheMu sync.Mutex
	globCache   = map[string]*regexp.Regexp{}
)

// Match reports whether path matches the glob pattern. Invalid patterns never match.
func Match(pattern, path string) bool {
	re := compileGlob(pattern)
	return re != nil && re.MatchString(normalizePath(path))
}

// MatchAny reports whether path matches at least one of the patterns.
func MatchAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if Match(p, path) {
			return true
		}
	}
	return false
}

// HasWildcard reports whether the pattern contains glob meta characters.
func HasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[{")
}

// StaticPrefix returns the directory prefix of a pattern up to (excluding) the first segment that contains a wildcard,
// always ending in "/" or empty. For a pattern without wildcards it returns the pattern itself.
func StaticPrefix(pattern string) string {
	pattern = normalizePath(pattern)
	idx := strings.IndexAny(pattern, "*?[{")
	if idx < 0 {
		return pattern
	}
	cut := strings.LastIndex(pattern[:idx], "/")
	if cut < 0 {
		return ""
	}
	return pattern[:cut+1]
}

// Overlaps reports whether two patterns (or literal paths) can denote a common file:
//   - both literal: equal paths;
//   - one literal: the other pattern matches it;
//   - both with wildcards: one static prefix is a prefix of the other (deliberately conservative).
func Overlaps(a, b string) bool {
	a, b = normalizePath(a), normalizePath(b)
	aw, bw := HasWildcard(a), HasWildcard(b)
	switch {
	case !aw && !bw:
		return a == b
	case !aw:
		return Match(b, a)
	case !bw:
		return Match(a, b)
	default:
		pa, pb := StaticPrefix(a), StaticPrefix(b)
		return strings.HasPrefix(pa, pb) || strings.HasPrefix(pb, pa)
	}
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	return strings.TrimPrefix(p, "./")
}

func compileGlob(pattern string) *regexp.Regexp {
	pattern = normalizePath(pattern)
	globCacheMu.Lock()
	defer globCacheMu.Unlock()
	if re, ok := globCache[pattern]; ok {
		return re
	}
	re, err := regexp.Compile("^" + globToRegex(pattern) + "$")
	if err != nil {
		re = nil
	}
	globCache[pattern] = re
	return re
}

func globToRegex(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch c {
		case '*':
			if i+1 < len(p) && p[i+1] == '*' {
				if (i == 0 || p[i-1] == '/') && i+2 < len(p) && p[i+2] == '/' {
					b.WriteString(`(?:.*/)?`)
					i += 2
				} else {
					b.WriteString(`.*`)
					i++
				}
				continue
			}
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		case '[':
			end := strings.IndexByte(p[i+1:], ']')
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			class := p[i+1 : i+1+end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + class + "]")
			i += end + 1
		case '{':
			end := strings.IndexByte(p[i+1:], '}')
			if end < 0 {
				b.WriteString(`\{`)
				continue
			}
			var alts []string
			for _, alt := range strings.Split(p[i+1:i+1+end], ",") {
				alts = append(alts, globToRegex(alt))
			}
			b.WriteString("(?:" + strings.Join(alts, "|") + ")")
			i += end + 1
		case '/':
			if p[i+1:] == "**" {
				// trailing `dir/**` also matches `dir` itself
				b.WriteString(`(?:/.*)?`)
				i += 2
				continue
			}
			b.WriteString("/")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return b.String()
}
