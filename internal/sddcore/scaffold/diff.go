package scaffold

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines shown around the change.
const diffContext = 3

// UnifiedDiff shows the change from old to new as a unified diff with a single hunk: common leading and trailing lines
// are cut off, everything in between is shown as removed and added. That is not minimal for scattered edits, but
// complete and linear in the file size. Equal contents yield "".
func UnifiedDiff(path string, old, new []byte) string {
	if string(old) == string(new) {
		return ""
	}
	a, b := splitLines(string(old)), splitLines(string(new))
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	start := max(0, pre-diffContext)
	endA := min(len(a), len(a)-suf+diffContext)
	endB := min(len(b), len(b)-suf+diffContext)

	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", path, path)
	fmt.Fprintf(&out, "@@ -%s +%s @@\n", hunkRange(start, endA-start), hunkRange(start, endB-start))
	for _, l := range a[start:pre] {
		out.WriteString(" " + l + "\n")
	}
	for _, l := range a[pre : len(a)-suf] {
		out.WriteString("-" + l + "\n")
	}
	for _, l := range b[pre : len(b)-suf] {
		out.WriteString("+" + l + "\n")
	}
	for _, l := range a[len(a)-suf : endA] {
		out.WriteString(" " + l + "\n")
	}
	return out.String()
}

// splitLines splits text into lines without their line breaks; a final line break adds no empty line.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// hunkRange formats start (0-based) and count the way unified diffs do: 1-based start, an empty range at the line before.
func hunkRange(start, count int) string {
	if count == 0 {
		return fmt.Sprintf("%d,0", start)
	}
	return fmt.Sprintf("%d,%d", start+1, count)
}
