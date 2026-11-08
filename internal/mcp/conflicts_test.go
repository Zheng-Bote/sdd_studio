package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

const conflictsConstitution = "---\nstatus: active\n---\n\n# Verfassung\n\n## Security\n\n" +
	"1. Es wird nur die **Go-Standardbibliothek** verwendet, keine externen Abhängigkeiten.\n" + // line 9
	"2. Jede Verbindung hat ein Timeout von 10 Sekunden.\n" + // line 10
	"\n## Strikte Regeln\n\n" +
	"1. Jedes Akzeptanzkriterium hat mindestens einen Test.\n" + // line 14
	"\n```\n1. kein Regelpunkt im Code-Block\n```\n"

const conflictsSpec = "---\nfeature: billing\nstatus: active\ntouches: [api/export/**]\n---\n\n# Tech Spec\n\n" +
	"Für das Parsen nutzen wir die Bibliothek github.com/x/yaml.\n" + // line 9
	"\n| AC-1 | Export |\n"

// conflictsProject is project(t) with a constitution that has three numbered rules and a Tech Spec for billing.
func conflictsProject(t *testing.T) *Server {
	t.Helper()
	root := project(t)
	write(t, root, "constitution.md", conflictsConstitution)
	write(t, root, "docs/design/billing.md", conflictsSpec)
	return newServer(root, false)
}

type conflictsResult struct {
	Conflicts []conflictOut  `json:"conflicts"`
	Summary   map[string]int `json:"summary"`
	Missing   []ruleRef      `json:"missing_rules"`
	Unknown   []ruleRef      `json:"unknown_rules"`
	Markdown  string         `json:"markdown"`
	Checkable bool           `json:"rules_checkable"`
}

func conflictsCall(t *testing.T, s *Server, a map[string]any) (conflictsResult, map[string]any) {
	t.Helper()
	r := call(t, s, "sdd_conflicts_check", a)
	if r["isError"] == true {
		return conflictsResult{}, r
	}
	data, _ := json.Marshal(structured(t, r))
	var out conflictsResult
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out, r
}

func conflict(cLine int, cQuote, sPath string, sLine int, sQuote string) map[string]any {
	return map[string]any{
		"constitution": map[string]any{"path": "constitution.md", "line": cLine, "quote": cQuote},
		"spec":         map[string]any{"path": sPath, "line": sLine, "quote": sQuote},
		"kind":         "Widerspruch", "suggestion": "Tech Spec anpassen",
	}
}

var allRules = []any{
	map[string]any{"path": "constitution.md", "line": 9, "result": "Treffer"},
	map[string]any{"path": "constitution.md", "line": 10, "result": "kein Widerspruch"},
	map[string]any{"path": "constitution.md", "line": 14, "result": "kein Widerspruch"},
}

// conflicts-quote-check AC-1: sdd_conflicts_check is offered without any switch, read-only, and writes nothing.
func TestConflictsCheckTool(t *testing.T) {
	s := conflictsProject(t)
	found := false
	for _, tl := range toolList(t, s) {
		if tl["name"] == "sdd_conflicts_check" {
			found = true
			if tl["annotations"].(map[string]any)["readOnlyHint"] != true {
				t.Errorf("not read-only: %v", tl["annotations"])
			}
		}
	}
	if !found {
		t.Fatal("sdd_conflicts_check not offered without switches")
	}
	before := readFileT(t, s.Root, "constitution.md")
	if _, r := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{}, "rules": allRules}); r["isError"] == true {
		t.Fatal(errText(r))
	}
	if readFileT(t, s.Root, "constitution.md") != before {
		t.Error("constitution changed")
	}
}

// conflicts-quote-check AC-2, AC-3, AC-5: every quote is checked against its file and line; a conflict counts only when
// both sides are found; the table holds the verified conflicts with the verified lines.
func TestConflictsCheckQuotes(t *testing.T) {
	s := conflictsProject(t)
	res, r := conflictsCall(t, s, map[string]any{"feature": "billing", "rules": allRules, "conflicts": []any{
		// exact line; emphasis and a copied sdd_read prefix are ignored
		conflict(9, "   9| nur die Go-Standardbibliothek verwendet", "docs/design/billing.md", 9, "nutzen wir die Bibliothek github.com/x/yaml"),
		// right text, wrong line on the spec side: found in exactly one other line
		conflict(9, "keine externen Abhängigkeiten", "docs/design/billing.md", 3, "die Bibliothek github.com/x/yaml"),
		// invented quote
		conflict(10, "Timeout von 30 Sekunden ist erlaubt", "docs/design/billing.md", 9, "nutzen wir die Bibliothek github.com/x/yaml"),
		// too short
		conflict(9, "Go", "docs/design/billing.md", 9, "nutzen wir die Bibliothek github.com/x/yaml"),
		// spec side from a file that is neither the Tech Spec of the feature nor an ADR
		conflict(9, "keine externen Abhängigkeiten", "docs/prd/billing.md", 1, "irgendein Zitat aus dem PRD"),
	}})
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	want := []struct {
		verified  bool
		cStatus   string
		sStatus   string
		foundLine int
	}{
		{true, quoteVerified, quoteVerified, 0},
		{true, quoteVerified, quoteOtherLine, 9},
		{false, quoteNotFound, quoteVerified, 0},
		{false, quoteTooShort, quoteVerified, 0},
		{false, quoteVerified, quoteNotAllowed, 0},
	}
	for i, w := range want {
		c := res.Conflicts[i]
		if c.Verified != w.verified || c.Constitution.Status != w.cStatus || c.Spec.Status != w.sStatus || c.Spec.FoundLine != w.foundLine {
			t.Errorf("conflict %d: %+v, want %+v", i, c, w)
		}
	}
	if res.Summary["verified"] != 2 || res.Summary["rejected"] != 3 {
		t.Errorf("summary: %v", res.Summary)
	}
	if !strings.Contains(res.Markdown, "(docs/design/billing.md:9)") || strings.Contains(res.Markdown, "docs/design/billing.md:3") ||
		strings.Contains(res.Markdown, "30 Sekunden") || strings.Contains(res.Markdown, "9|") {
		t.Errorf("table must hold only verified conflicts with verified lines:\n%s", res.Markdown)
	}
}

// conflicts-quote-check AC-3: unknown kind, suggestion or rule result is refused with the allowed values.
func TestConflictsCheckRefusesValues(t *testing.T) {
	s := conflictsProject(t)
	bad := conflict(9, "keine externen Abhängigkeiten", "docs/design/billing.md", 9, "die Bibliothek github.com/x/yaml")
	bad["kind"] = "Fehler"
	if _, r := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{bad}, "rules": allRules}); r["isError"] != true ||
		!strings.Contains(errText(r), "Widerspruch, Spannung, unklar") {
		t.Errorf("kind: %v", r)
	}
	bad["kind"], bad["suggestion"] = "Spannung", "ignorieren"
	if _, r := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{bad}, "rules": allRules}); r["isError"] != true {
		t.Errorf("suggestion: %v", r)
	}
	rules := []any{map[string]any{"path": "constitution.md", "line": 9, "result": "vielleicht"}}
	if _, r := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{}, "rules": rules}); r["isError"] != true {
		t.Errorf("rule result: %v", r)
	}
	// An unknown slug names the features that have a Tech Spec (counter-test with qwen3:14b).
	if _, r := conflictsCall(t, s, map[string]any{"feature": "feature-billing", "conflicts": []any{}, "rules": allRules}); r["isError"] != true ||
		!strings.Contains(errText(r), "Tech Specs gibt es für: billing") {
		t.Errorf("feature without Tech Spec: %v", r)
	}
}

// conflicts-quote-check AC-4: the numbered items under level-2 headings are the rules; missing and unknown entries are
// named; code blocks do not count.
func TestConflictsCheckRules(t *testing.T) {
	s := conflictsProject(t)
	rules := []any{allRules[0], map[string]any{"path": "constitution.md", "line": 12, "result": "kein Widerspruch"}}
	res, r := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{}, "rules": rules})
	if r["isError"] == true {
		t.Fatal(errText(r))
	}
	if !res.Checkable || len(res.Missing) != 2 || res.Missing[0].Line != 10 || res.Missing[1].Line != 14 ||
		!strings.Contains(res.Missing[0].Text, "Timeout") {
		t.Errorf("missing rules: %+v", res.Missing)
	}
	if len(res.Unknown) != 1 || res.Unknown[0].Line != 12 {
		t.Errorf("unknown rules: %+v", res.Unknown)
	}
	if res.Summary["missing_rules"] != 2 {
		t.Errorf("summary: %v", res.Summary)
	}
	full, _ := conflictsCall(t, s, map[string]any{"feature": "billing", "conflicts": []any{}, "rules": allRules})
	if len(full.Missing) != 0 || len(full.Unknown) != 0 {
		t.Errorf("complete list: %+v %+v", full.Missing, full.Unknown)
	}
}

// conflicts-quote-check AC-6: sdd-conflicts checks its result with sdd_conflicts_check before answering, checks missing
// rules again, and shows only verified conflicts; rejected ones are named separately.
func TestConflictsPromptUsesCheck(t *testing.T) {
	text := promptText(t, newServer(project(t), false), "sdd-conflicts", `{"feature":"billing"}`)
	last := -1
	for _, step := range []string{"sdd_read", "sdd_conflicts_check", "fehlende Regeln", "erneut", "markdown", "verworfen"} {
		i := strings.Index(text, step)
		if i <= last {
			t.Errorf("step %q missing or out of order in:\n%s", step, text)
		}
		last = i
	}
}

// An empty call is not a clean result: it says the check is incomplete (counter-test with qwen3:32b). With the full
// rule list and no conflicts it is complete; verified conflicts stay in the table next to the note.
func TestConflictsCheckIncomplete(t *testing.T) {
	s := conflictsProject(t)
	r := structured(t, call(t, s, "sdd_conflicts_check", map[string]any{"feature": "billing", "rules": []any{}}))
	if r["complete"] != false || !strings.HasPrefix(r["markdown"].(string), "Prüfung unvollständig: 3 Regeln") || strings.Contains(r["markdown"].(string), "Keine belegte") {
		t.Errorf("empty call: %v", r)
	}
	full := structured(t, call(t, s, "sdd_conflicts_check", map[string]any{"feature": "billing", "conflicts": []any{}, "rules": allRules}))
	if full["complete"] != true || full["markdown"] != "Keine belegte Meldung." {
		t.Errorf("full list: %v", full)
	}
	c := conflict(9, "keine externen Abhängigkeiten", "docs/design/billing.md", 9, "die Bibliothek github.com/x/yaml")
	part := structured(t, call(t, s, "sdd_conflicts_check", map[string]any{"feature": "billing", "conflicts": []any{c}, "rules": []any{}}))
	if md := part["markdown"].(string); !strings.HasPrefix(md, "Prüfung unvollständig") || !strings.Contains(md, "| Verfassung |") {
		t.Errorf("partial: %s", md)
	}
}
