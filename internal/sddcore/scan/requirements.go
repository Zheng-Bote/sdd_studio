package scan

import "regexp"

// requirementIDRegex reads the stable requirement IDs of an initiative (ADR 0009). A requirement is a list item that
// starts with its ID, optionally bold ("- **R1** …", "- R1 …"); prose that merely mentions an ID does not count, and
// neither does a template placeholder ("- <…>").
var requirementIDRegex = regexp.MustCompile(`(?m)^[ \t]*(?:[-*+]|[0-9]+[.)])[ \t]+(?:\*\*)?(R[1-9][0-9]*)\b`)

// RequirementIDs returns the requirement IDs of an initiative in order of appearance, each once.
func RequirementIDs(data []byte) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range requirementIDRegex.FindAllSubmatch(data, -1) {
		id := string(m[1])
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
