package rules

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	qaIDRegex        = regexp.MustCompile(`\bQA-[1-9][0-9]*\b`)
	qaTombstoneRegex = regexp.MustCompile(`~~[^~]*\bQA-[0-9]+\b[^~]*~~`)
)

// QATrace follows one quality requirement QA-<n> (ADR 0007). Nothing requires these references; they are information.
type QATrace struct {
	ID        string   `json:"id"`
	PRD       string   `json:"prd"`
	Line      int      `json:"line"`
	Text      string   `json:"text"`
	Duplicate bool     `json:"duplicate,omitempty"` // the ID occurs more than once in the section
	TechSpecs []string `json:"tech_specs"`
	Tests     []string `json:"tests"` // "path:line" of test lines naming the ID and the feature slug
}

// TraceQuality lists the quality requirements of the feature with their Tech Specs and tests.
func TraceQuality(ctx *Context, feature string) ([]QATrace, error) {
	prds, err := activePRDs(ctx)
	if err != nil {
		return nil, err
	}
	out := []QATrace{}
	index := map[string]int{}
	for _, prd := range prds {
		if featureOf(prd, ctx.Config.Artifacts.PRD) != feature {
			continue
		}
		lines, err := readLines(ctx, prd.Path)
		if err != nil {
			return nil, err
		}
		level := 0
		for _, l := range lines {
			if l.Heading > 0 {
				switch {
				case strings.Contains(l.Text, "Qualitätsanforderungen"):
					level = l.Heading
				case level > 0 && l.Heading <= level:
					level = 0
				}
				continue
			}
			if level == 0 || !listItemRegex.MatchString(l.Text) {
				continue
			}
			id := qaIDRegex.FindString(qaTombstoneRegex.ReplaceAllString(l.Text, ""))
			if id == "" {
				continue
			}
			if i, seen := index[id]; seen {
				out[i].Duplicate = true
				continue
			}
			index[id] = len(out)
			out = append(out, QATrace{ID: id, PRD: prd.Path, Line: l.Number, Text: strings.TrimSpace(l.Text), TechSpecs: []string{}, Tests: []string{}})
		}
	}
	if len(out) == 0 {
		return out, nil
	}
	designs, err := activeArtifacts(ctx, ctx.Config.Artifacts.Design)
	if err != nil {
		return nil, err
	}
	files, _, err := loadTestFiles(ctx)
	if err != nil {
		return nil, err
	}
	names := func(text, id string) bool {
		for _, found := range qaIDRegex.FindAllString(qaTombstoneRegex.ReplaceAllString(text, ""), -1) {
			if found == id {
				return true
			}
		}
		return false
	}
	for i := range out {
		for _, ds := range designs {
			if featureOf(ds, ctx.Config.Artifacts.Design) != feature {
				continue
			}
			lines, err := readLines(ctx, ds.Path)
			if err != nil {
				return nil, err
			}
			for _, l := range lines {
				if names(l.Text, out[i].ID) {
					out[i].TechSpecs = append(out[i].TechSpecs, ds.Path)
					break
				}
			}
		}
		for _, f := range files {
			for n, line := range f.Lines {
				if names(line, out[i].ID) && anyMentionsSlug([]string{line}, feature) {
					out[i].Tests = append(out[i].Tests, fmt.Sprintf("%s:%d", f.Path, n+1))
				}
			}
		}
	}
	return out, nil
}
