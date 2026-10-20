package overview

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RenderJSON writes the report as indented JSON.
func RenderJSON(w io.Writer, o *Overview) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}

// RenderMarkdown writes the report as Markdown, suitable for a terminal or a CI job summary.
func RenderMarkdown(w io.Writer, o *Overview) error {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	p("# SDD-Bericht")
	p("")
	p("Profil `%s` · Reifegrad **L%d**", o.Profile, o.Maturity.Level)
	if o.Maturity.Next != nil {
		p("")
		p("**Nächster Schritt (L%d):** %s.", o.Maturity.Next.Level, o.Maturity.Next.Advice)
	}
	p("")

	p("## Gates")
	p("")
	p("| Gate | error | warn | baselined |")
	p("|---|---|---|---|")
	p("| `lint` | %d | %d | %d |", o.Lint.Error, o.Lint.Warn, o.Lint.Baselined)
	for _, g := range o.Gates {
		if g.Error != "" {
			p("| `%s` | – | – | – (nicht ausführbar: %s) |", g.Gate, g.Error)
			continue
		}
		p("| `%s` | %d | %d | %d |", g.Gate, g.Counts.Error, g.Counts.Warn, g.Counts.Baselined)
	}
	p("")

	p("## Offene Befunde (`sdd lint`)")
	p("")
	if len(o.Findings) == 0 {
		p("Keine.")
	}
	for _, f := range o.Findings {
		p("- **%s** %s `%s:%d` — %s", f.Severity, f.RuleID, f.Path, f.Line, f.Message)
	}
	p("")

	p("## Schuldenstand")
	p("")
	switch {
	case o.SinceError != "":
		p("%d Baseline-Einträge. Trend nicht ermittelbar: %s", o.DebtNow, o.SinceError)
	default:
		p("%d Baseline-Einträge, am Vergleichsstand `%s`: %d (%s).", o.DebtNow, o.Since, o.DebtBefore, delta(o.DebtNow-o.DebtBefore))
	}
	if len(o.Debt) > 0 {
		p("")
		p("| Regel | Gate | jetzt | vorher |")
		p("|---|---|---|---|")
		for _, r := range o.Debt {
			before := fmt.Sprint(r.Before)
			if o.SinceError != "" {
				before = "–"
			}
			p("| %s | `%s` | %d | %s |", r.RuleID, r.Gate, r.Now, before)
		}
	}
	if o.Grew() {
		p("")
		p("> **Verstoß gegen die Ratchet-Regel:** Die Baseline darf nur schrumpfen, enthält aber %d neue Einträge gegenüber `%s`. Neue Befunde sind zu beheben, nicht zu baselinen; Ausnahmen nur über `sdd baseline --accept` mit Begründung im Review.", len(o.Added), o.Since)
		p("")
		for _, e := range o.Added {
			p("> - %s `%s` (Gate `%s`, Zeile %d)", e.RuleID, e.Path, e.Gate, e.Line)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func delta(d int) string {
	switch {
	case d < 0:
		return fmt.Sprintf("%d, gesunken", d)
	case d > 0:
		return fmt.Sprintf("+%d, gewachsen", d)
	default:
		return "unverändert"
	}
}
