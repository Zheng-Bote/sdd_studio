package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// conflictsTool checks the structured result of the guided flow sdd-conflicts (SDD-060, assisted). It compares
// strings only — whether a quote is in its file and line, whether every constitution rule was gone through — and never
// judges a conflict; sdd stays AI-free (ADR 0003).
var conflictsTool = tool{Name: "sdd_conflicts_check", Title: "Abgleich prüfen",
	Description: "Prüft das Ergebnis von sdd-conflicts maschinell: steht jedes Zitat wörtlich in der genannten Datei und Zeile (Verfassung, Tech Spec des Features oder ADR), und fehlt eine Regel der Verfassung in der Regelliste? Bewertet keinen Widerspruch. Liefert je Meldung den Prüfstatus, die fehlenden Regeln und eine Tabelle der belegten Meldungen.",
	InputSchema: obj(map[string]any{
		"feature": str("Feature-Slug"),
		"conflicts": map[string]any{"type": "array", "description": "Meldungen; je Seite ein wörtliches Zitat aus genau einer Zeile",
			"items": obj(map[string]any{
				"constitution": quoteSchema, "spec": quoteSchema,
				"kind":       map[string]any{"type": "string", "enum": conflictKinds},
				"suggestion": map[string]any{"type": "string", "enum": conflictSuggestions},
			}, "constitution", "spec", "kind", "suggestion")},
		"rules": map[string]any{"type": "array", "description": "jede Regel der Verfassung mit Datei:Zeile und Ergebnis",
			"items": obj(map[string]any{
				"path": str("Pfad der Verfassung"), "line": map[string]any{"type": "integer"},
				"result": map[string]any{"type": "string", "enum": ruleResults},
			}, "path", "line", "result")},
	}, "feature", "conflicts", "rules"), Annotations: readOnly}

var (
	quoteSchema = obj(map[string]any{
		"path": str("Datei"), "line": map[string]any{"type": "integer", "description": "Zeile aus sdd_read"},
		"quote": str("wörtliches Zitat aus dieser Zeile"),
	}, "path", "line", "quote")
	conflictKinds       = []string{"Widerspruch", "Spannung", "unklar"}
	conflictSuggestions = []string{"Tech Spec anpassen", "ADR für eine Verfassungsänderung"}
	ruleResults         = []string{"Treffer", "kein Widerspruch"}
)

// Quote states (AC-2).
const (
	quoteVerified    = "belegt"
	quoteOtherLine   = "andere Zeile"
	quoteNotFound    = "nicht gefunden"
	quoteTooShort    = "zu kurz"
	quoteNotAllowed  = "nicht zulässig"
	minQuoteRunes    = 12
	conflictsMaxRows = 200
)

type quoteIn struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Quote string `json:"quote"`
}

type quoteOut struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Status    string `json:"status"`
	FoundLine int    `json:"found_line,omitempty"`
	Reason    string `json:"reason,omitempty"`
	quote     string
}

func (q quoteOut) ok() bool { return q.Status == quoteVerified || q.Status == quoteOtherLine }

// line is the verified line: found_line where the quote was found elsewhere.
func (q quoteOut) line() int {
	if q.FoundLine > 0 {
		return q.FoundLine
	}
	return q.Line
}

type conflictOut struct {
	Index        int      `json:"index"`
	Verified     bool     `json:"verified"`
	Kind         string   `json:"kind"`
	Suggestion   string   `json:"suggestion"`
	Constitution quoteOut `json:"constitution"`
	Spec         quoteOut `json:"spec"`
}

type ruleRef struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text,omitempty"`
}

func (s *Server) toolConflictsCheck(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Feature   string `json:"feature"`
		Conflicts []struct {
			Constitution quoteIn `json:"constitution"`
			Spec         quoteIn `json:"spec"`
			Kind         string  `json:"kind"`
			Suggestion   string  `json:"suggestion"`
		} `json:"conflicts"`
		Rules []struct {
			Path   string `json:"path"`
			Line   int    `json:"line"`
			Result string `json:"result"`
		} `json:"rules"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Feature) == "" {
		return nil, errors.New("feature fehlt")
	}
	if len(a.Conflicts) > conflictsMaxRows || len(a.Rules) > conflictsMaxRows {
		return nil, fmt.Errorf("zu viele Einträge (höchstens %d Meldungen und %d Regeln)", conflictsMaxRows, conflictsMaxRows)
	}
	for i, c := range a.Conflicts {
		if !slices.Contains(conflictKinds, c.Kind) {
			return nil, fmt.Errorf("Meldung %d: kind %q ungültig; erlaubt: %s", i, c.Kind, strings.Join(conflictKinds, ", "))
		}
		if !slices.Contains(conflictSuggestions, c.Suggestion) {
			return nil, fmt.Errorf("Meldung %d: suggestion %q ungültig; erlaubt: %s", i, c.Suggestion, strings.Join(conflictSuggestions, ", "))
		}
	}
	for i, r := range a.Rules {
		if !slices.Contains(ruleResults, r.Result) {
			return nil, fmt.Errorf("Regel %d: result %q ungültig; erlaubt: %s", i, r.Result, strings.Join(ruleResults, ", "))
		}
	}

	cfg, err := config.Load(s.Root)
	if err != nil {
		return nil, err
	}
	constitution, err := scan.CleanRel(cfg.Artifacts.Constitution)
	if err != nil {
		return nil, err
	}
	specs, err := s.conflictSources(cfg, a.Feature)
	if err != nil {
		return nil, err
	}
	lines := map[string][]string{} // read once per file
	fileLines := func(rel string) ([]string, error) {
		if l, ok := lines[rel]; ok {
			return l, nil
		}
		_, text, err := s.readArtifact(rel)
		if err != nil {
			return nil, err
		}
		l := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
		lines[rel] = l
		return l, nil
	}
	check := func(q quoteIn, allowed func(string) bool) quoteOut {
		out := quoteOut{Path: q.Path, Line: q.Line, quote: q.Quote}
		rel, err := scan.CleanRel(q.Path)
		if err != nil || !allowed(rel) {
			out.Status, out.Reason = quoteNotAllowed, "nur die Verfassung, die Tech Spec des Features oder ein ADR"
			return out
		}
		out.Path = rel
		want := normQuote(q.Quote)
		if len([]rune(want)) < minQuoteRunes {
			out.Status = quoteTooShort
			return out
		}
		l, err := fileLines(rel)
		if err != nil {
			out.Status, out.Reason = quoteNotAllowed, err.Error()
			return out
		}
		if q.Line >= 1 && q.Line <= len(l) && strings.Contains(normQuote(l[q.Line-1]), want) {
			out.Status = quoteVerified
			return out
		}
		var hits []int
		for i, text := range l {
			if strings.Contains(normQuote(text), want) {
				hits = append(hits, i+1)
			}
		}
		if len(hits) == 1 {
			out.Status, out.FoundLine = quoteOtherLine, hits[0]
			return out
		}
		out.Status = quoteNotFound
		if len(hits) > 1 {
			out.Reason = fmt.Sprintf("mehrdeutig: in %d Zeilen", len(hits))
		}
		return out
	}
	isConstitution := func(rel string) bool { return rel == constitution }
	isSpec := func(rel string) bool { return slices.Contains(specs, rel) }

	conflicts := []conflictOut{}
	verified := 0
	for i, c := range a.Conflicts {
		co := conflictOut{Index: i, Kind: c.Kind, Suggestion: c.Suggestion,
			Constitution: check(c.Constitution, isConstitution), Spec: check(c.Spec, isSpec)}
		co.Verified = co.Constitution.ok() && co.Spec.ok()
		if co.Verified {
			verified++
		}
		conflicts = append(conflicts, co)
	}

	res := map[string]any{"conflicts": conflicts, "markdown": conflictsTable(conflicts)}
	missing, unknown := []ruleRef{}, []ruleRef{}
	if strings.EqualFold(path.Ext(constitution), ".md") {
		l, err := fileLines(constitution)
		if err != nil {
			return nil, err
		}
		rules := constitutionRules(l)
		listed := map[int]bool{}
		for _, r := range a.Rules {
			if rel, err := scan.CleanRel(r.Path); err == nil && rel == constitution {
				listed[r.Line] = true
				if _, ok := rules[r.Line]; !ok {
					unknown = append(unknown, ruleRef{Path: rel, Line: r.Line})
				}
				continue
			}
			unknown = append(unknown, ruleRef{Path: r.Path, Line: r.Line})
		}
		for _, n := range sortedKeys(rules) {
			if !listed[n] {
				missing = append(missing, ruleRef{Path: constitution, Line: n, Text: rules[n]})
			}
		}
		res["rules_checkable"] = true
	} else {
		res["rules_checkable"] = false
		res["rules_note"] = "Regelliste nicht prüfbar: Die Verfassung ist nicht im Markdown-Format."
	}
	res["missing_rules"], res["unknown_rules"] = missing, unknown
	res["summary"] = map[string]int{"verified": verified, "rejected": len(conflicts) - verified, "missing_rules": len(missing)}
	return resultOf(res), nil
}

// conflictSources are the files a spec-side quote may come from: the Tech Specs of the feature and every ADR.
func (s *Server) conflictSources(cfg *config.Config, feature string) ([]string, error) {
	var out []string
	designs, err := scan.ListArtifacts(s.Root, cfg.Artifacts.Design, cfg.Exclude)
	if err != nil {
		return nil, err
	}
	for _, d := range designs {
		if d.Frontmatter.Feature == feature {
			out = append(out, d.Path)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("keine Tech Spec mit feature: %s", feature)
	}
	adrs, err := scan.ListArtifacts(s.Root, cfg.Artifacts.ADR, cfg.Exclude)
	if err != nil {
		return nil, err
	}
	for _, d := range adrs {
		out = append(out, d.Path)
	}
	return out, nil
}

var (
	readPrefix   = regexp.MustCompile(`^\s*\d+\|\s?`)
	markupChars  = strings.NewReplacer("*", "", "_", "", "`", "")
	numberedRule = regexp.MustCompile(`^\d+\.\s`)
)

// normQuote drops a line prefix copied from sdd_read ("  12| "), Markdown emphasis and differences in white space.
func normQuote(s string) string {
	s = readPrefix.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(markupChars.Replace(s)), " ")
}

// cleanQuote is a quote for display: without a copied sdd_read prefix, white space collapsed, emphasis kept.
func cleanQuote(s string) string {
	return strings.Join(strings.Fields(readPrefix.ReplaceAllString(s, "")), " ")
}

// constitutionRules are the numbered list items under a level-2 heading, outside code blocks: line → text.
func constitutionRules(lines []string) map[int]string {
	rules := map[int]string{}
	inSection, inCode := false, false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		switch {
		case strings.HasPrefix(l, "## "):
			inSection = true
		case strings.HasPrefix(l, "# "):
			inSection = false
		case inSection && numberedRule.MatchString(l):
			rules[i+1] = strings.TrimSpace(l)
		}
	}
	return rules
}

func sortedKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// conflictsTable renders the verified conflicts with their verified lines; pipes in quotes are escaped.
func conflictsTable(conflicts []conflictOut) string {
	esc := strings.NewReplacer("|", `\|`, "\n", " ")
	var b strings.Builder
	b.WriteString("| Verfassung | Tech Spec | Einschätzung | Vorschlag |\n|---|---|---|---|\n")
	rows := 0
	for _, c := range conflicts {
		if !c.Verified {
			continue
		}
		rows++
		fmt.Fprintf(&b, "| „%s“ (%s:%d) | „%s“ (%s:%d) | %s | %s |\n",
			esc.Replace(cleanQuote(c.Constitution.quote)), c.Constitution.Path, c.Constitution.line(),
			esc.Replace(cleanQuote(c.Spec.quote)), c.Spec.Path, c.Spec.line(), c.Kind, c.Suggestion)
	}
	if rows == 0 {
		return "Keine belegte Meldung."
	}
	return b.String()
}
