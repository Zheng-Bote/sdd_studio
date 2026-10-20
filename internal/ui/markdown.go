package ui

import (
	"html"
	"regexp"
	"strings"
)

// RenderMarkdown renders the subset of Markdown the handbook uses: ATX headings with anchors, paragraphs, flat and
// nested lists, fenced code, tables, block quotes, horizontal rules, and inline code, bold, emphasis and links.
// Every text is HTML-escaped before inline markup is applied, so the output never carries raw HTML from the source.
func RenderMarkdown(src []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	var b strings.Builder
	var para []string
	listDepth := 0
	flushPara := func() {
		if len(para) > 0 {
			b.WriteString("<p>" + inline(strings.Join(para, " ")) + "</p>\n")
			para = nil
		}
	}
	closeLists := func(depth int) {
		for listDepth > depth {
			b.WriteString("</li></ul>\n")
			listDepth--
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~"):
			flushPara()
			closeLists(0)
			fence := trimmed[:3]
			var code []string
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), fence); i++ {
				code = append(code, lines[i])
			}
			b.WriteString("<pre><code>" + html.EscapeString(strings.Join(code, "\n")) + "</code></pre>\n")
		case trimmed == "":
			flushPara()
			closeLists(0)
		case headingRe.MatchString(line):
			flushPara()
			closeLists(0)
			m := headingRe.FindStringSubmatch(line)
			level, text := len(m[1]), strings.TrimSpace(strings.TrimRight(m[2], "#"))
			id := ""
			if am := anchorRe.FindStringSubmatch(text); am != nil {
				id, text = am[1], strings.TrimSpace(anchorRe.ReplaceAllString(text, ""))
			} else {
				id = slug(text)
			}
			b.WriteString("<h" + string(rune('0'+level)) + " id=\"" + html.EscapeString(id) + "\">" + inline(text) + "</h" + string(rune('0'+level)) + ">\n")
		case trimmed == "---" || trimmed == "***":
			flushPara()
			closeLists(0)
			b.WriteString("<hr>\n")
		case strings.HasPrefix(trimmed, "|"):
			flushPara()
			closeLists(0)
			var rows []string
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				rows = append(rows, strings.TrimSpace(lines[i]))
			}
			i--
			b.WriteString(table(rows))
		case strings.HasPrefix(trimmed, ">"):
			flushPara()
			closeLists(0)
			b.WriteString("<blockquote>" + inline(strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))) + "</blockquote>\n")
		case listRe.MatchString(line):
			flushPara()
			m := listRe.FindStringSubmatch(line)
			depth := len(strings.ReplaceAll(m[1], "\t", "    "))/2 + 1
			if depth > listDepth+1 {
				depth = listDepth + 1
			}
			switch {
			case depth > listDepth:
				b.WriteString("<ul><li>")
				listDepth = depth
			default:
				closeLists(depth)
				b.WriteString("</li><li>")
			}
			b.WriteString(inline(m[2]))
		case listDepth > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
			b.WriteString(" " + inline(trimmed)) // continuation of a list item
		default:
			closeLists(0)
			para = append(para, trimmed)
		}
	}
	flushPara()
	closeLists(0)
	return b.String()
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	anchorRe  = regexp.MustCompile(`\s*\{#([A-Za-z0-9_-]+)\}\s*$`)
	listRe    = regexp.MustCompile(`^(\s*)(?:[-*+]|[0-9]+[.)])\s+(.*)$`)
	codeRe    = regexp.MustCompile("`([^`]+)`")
	boldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emRe      = regexp.MustCompile(`(^|[^*])\*([^*\s][^*]*)\*`)
	linkRe    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	slugDrop  = regexp.MustCompile(`[^a-z0-9-]+`)
)

// inline escapes text and applies code spans, links, bold and emphasis. Code spans are protected from the rest.
func inline(text string) string {
	var codes []string
	text = codeRe.ReplaceAllStringFunc(text, func(m string) string {
		codes = append(codes, "<code>"+html.EscapeString(m[1:len(m)-1])+"</code>")
		return "\x00" + string(rune(len(codes)-1+'A')) + "\x00"
	})
	text = html.EscapeString(text)
	text = linkRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := linkRe.FindStringSubmatch(m)
		href := html.UnescapeString(sub[2])
		if !safeHref(href) {
			return sub[1]
		}
		return "<a href=\"" + html.EscapeString(href) + "\">" + sub[1] + "</a>"
	})
	text = boldRe.ReplaceAllString(text, "<strong>$1</strong>")
	text = emRe.ReplaceAllString(text, "$1<em>$2</em>")
	for i, c := range codes {
		text = strings.Replace(text, "\x00"+string(rune(i+'A'))+"\x00", c, 1)
	}
	return text
}

// safeHref admits in-page anchors and relative paths only; external and script URLs become plain text.
func safeHref(h string) bool {
	if strings.HasPrefix(h, "#") {
		return true
	}
	return !strings.Contains(h, ":") && !strings.HasPrefix(h, "//")
}

func slug(text string) string {
	s := strings.ToLower(text)
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "—", "-", "–", "-", " ", "-", "`", "").Replace(s)
	s = strings.Trim(slugDrop.ReplaceAllString(s, "-"), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

func table(rows []string) string {
	var b strings.Builder
	b.WriteString("<table>\n")
	for i, r := range rows {
		cells := strings.Split(strings.Trim(r, "|"), "|")
		if i == 1 && strings.Trim(strings.Join(cells, ""), " :-") == "" {
			continue // separator row
		}
		tag := "td"
		if i == 0 {
			tag = "th"
		}
		b.WriteString("<tr>")
		for _, c := range cells {
			b.WriteString("<" + tag + ">" + inline(strings.TrimSpace(c)) + "</" + tag + ">")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String()
}
