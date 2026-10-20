package mcp

import (
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

type promptArg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type prompt struct {
	Name        string      `json:"name"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Arguments   []promptArg `json:"arguments"`
	rules       []string
	body        func(args map[string]string) string
}

// common is appended to every prompt: the boundaries every guided step shares.
const common = `
Grundsätze für diesen Schritt:
- Frage, statt zu erfinden. Stelle eine Frage nach der anderen und formuliere nur aus, was der Mensch beantwortet hat. Offene Punkte bleiben als offene Punkte sichtbar.
- Prüfe jeden Entwurf mit sdd_validate_draft, bevor du ihn speicherst; behebe die Befunde mit dem Menschen.
- Neue Dateien legst du mit sdd_new an (zuerst ohne apply, dann mit apply nach Bestätigung); bestehende Dateien änderst du mit deinen eigenen Werkzeugen und prüfst danach mit sdd_lint.
- Baseline, ADR-Lockfile, adopt --apply und das Entscheiden eines ADR (status: accepted) sind dem Menschen vorbehalten.
- Zum Schluss: sdd_next, um den nächsten Prozessschritt zu nennen.`

func promptList() []prompt {
	return []prompt{
		{Name: "sdd-idea", Title: "Von der Idee zum PRD", Description: "Führt von einer Produkt-Idee zu einem PRD mit Nicht-Zielen und prüfbaren Akzeptanzkriterien.",
			Arguments: []promptArg{{"idea", "die Idee in einem Satz", false}, {"feature", "Feature-Slug, falls schon bekannt", false}},
			rules:     []string{"SDD-020", "SDD-021", "SDD-022", "SDD-023"},
			body: func(a map[string]string) string {
				idea := a["idea"]
				if idea == "" {
					idea = "(noch nicht genannt — frage zuerst danach)"
				}
				return "Begleite mich von einer Produkt-Idee zu einem PRD.\n\nIdee: " + idea + "\n\n" +
					"Vorgehen:\n" +
					"1. Kläre das Problem: Wer hat es, wann, und was kostet es heute?\n" +
					"2. Kläre Nutzer und Ziel: Was ist nach der Umsetzung anders?\n" +
					"3. Kläre den Umfang und frage ausdrücklich nach Nicht-Zielen: Was gehört bewusst nicht dazu?\n" +
					"4. Formuliere mit mir Akzeptanzkriterien, die ein Test prüfen kann, jedes mit Kennung AC-1, AC-2, … (nie umnummerieren).\n" +
					"5. Schlage einen Feature-Slug vor (" + slugHint(a["feature"]) + ") und frage, welche Pfade die Umsetzung voraussichtlich berührt (touches). Prüfe sie mit sdd_check_intent: Greift die Größenschwelle, braucht es auch eine Tech Spec.\n" +
					"6. Hole die Vorschau mit sdd_new (types: [prd]) und fülle die Abschnitte Problem, Ziel, Nutzer, Umfang, Nicht-Ziele, Akzeptanzkriterien.\n"
			}},
		{Name: "sdd-techspec", Title: "Tech Spec zum PRD", Description: "Führt vom PRD zur Tech Spec, die jedes Akzeptanzkriterium einer Umsetzung zuordnet.",
			Arguments: []promptArg{{"feature", "Feature-Slug", true}},
			rules:     []string{"SDD-030", "SDD-031", "SDD-010"},
			body: func(a map[string]string) string {
				return "Begleite mich zur Tech Spec für das Feature " + a["feature"] + ".\n\n" +
					"Vorgehen:\n" +
					"1. Lies das PRD (Ressource sdd://artifact/…, siehe sdd_next) und die Verfassung (sdd://constitution). Die Tech Spec darf die Verfassung präzisieren, nie lockern.\n" +
					"2. Kläre Architektur, Schnittstellen, Datenmodell und Sicherheitsauswirkungen — je Thema eine Frage.\n" +
					"3. Ordne jedes Akzeptanzkriterium einer Umsetzung und einem Test zu (Tabelle AC | Umsetzung | Test).\n" +
					"4. Prüfe mit sdd_context für die wichtigsten Pfade, welche ADRs und Artefakte dort schon gelten. Berührt die Änderung die Verfassung, braucht es ein ADR (Prompt sdd-adr).\n" +
					"5. Vorschau mit sdd_new (types: [design]), dann ausfüllen.\n"
			}},
		{Name: "sdd-adr", Title: "Architekturentscheidung", Description: "Führt zu einem ADR mit Kontext, Optionen, Entscheidung und Konsequenzen.",
			Arguments: []promptArg{{"title", "Titel der Entscheidung", true}},
			rules:     []string{"SDD-010", "SDD-011"},
			body: func(a map[string]string) string {
				return "Begleite mich zu einem ADR: " + a["title"] + ".\n\n" +
					"Vorgehen:\n" +
					"1. Kontext: Welche Kräfte und Randbedingungen erzwingen die Entscheidung?\n" +
					"2. Optionen: mindestens zwei, je mit Vor- und Nachteilen.\n" +
					"3. Entscheidung und Konsequenzen — auch die negativen. Ändert sie eine Verfassungsregel, gehört die Verfassung in references, und die Verfassung wird im selben Change angepasst.\n" +
					"4. Vorschau mit sdd_new (types: [adr]). Das ADR bleibt status: proposed; entscheiden (accepted) und locken (sdd baseline) ist Sache des Menschen. Nach der Entscheidung ist der Text unveränderlich; Ablösung nur über ein neues ADR mit supersedes.\n"
			}},
		{Name: "sdd-plan", Title: "Plan mit Tasks", Description: "Zerlegt ein Feature mit Tech Spec in Tasks, die je ein Akzeptanzkriterium nennen.",
			Arguments: []promptArg{{"feature", "Feature-Slug", true}},
			rules:     []string{"SDD-040", "SDD-041"},
			body: func(a map[string]string) string {
				return "Begleite mich zum Plan für das Feature " + a["feature"] + ".\n\n" +
					"Vorgehen:\n" +
					"1. Prüfe mit sdd_next, dass PRD und Tech Spec stehen — ohne Tech Spec keine Tasks.\n" +
					"2. Zerlege die Umsetzung in kleine, einzeln prüfbare Tasks in sinnvoller Reihenfolge.\n" +
					"3. Jede Task ist eine Checkbox-Zeile und nennt mindestens ein Kriterium: - [ ] AC-2 …\n" +
					"4. Vorschau mit sdd_new (types: [plan]), dann ausfüllen; sdd_trace zeigt, welche Kriterien noch keine Task haben.\n"
			}},
		{Name: "sdd-review", Title: "Stand prüfen", Description: "Prüft den Stand eines Features und nennt Lücken und den nächsten Schritt — ohne etwas zu ändern.",
			Arguments: []promptArg{{"feature", "Feature-Slug", true}},
			rules:     []string{"SDD-022", "SDD-024", "SDD-025", "SDD-050"},
			body: func(a map[string]string) string {
				return "Prüfe den Stand des Features " + a["feature"] + ", ohne Dateien zu ändern.\n\n" +
					"Vorgehen:\n" +
					"1. sdd_next für die Prozessschritte.\n" +
					"2. sdd_trace: Welche Kriterien haben keine Tech-Spec-Stelle, keinen Test, keine Task?\n" +
					"3. sdd_gate mit gate pre-merge: Besteht das Gate? Erkläre jeden Befund mit sdd_explain.\n" +
					"4. Fasse zusammen: was fertig ist, was fehlt, der nächste Schritt. Schlage Korrekturen vor, aber führe sie nur nach Zustimmung aus.\n"
			}},
	}
}

func slugHint(feature string) string {
	if scan.IsSlug(feature) {
		return "vorgeschlagen: " + feature
	}
	return "Kleinbuchstaben, Ziffern, Bindestriche"
}

func (s *Server) getPrompt(name string, args map[string]string) (any, *rpcError) {
	for _, p := range promptList() {
		if p.Name != name {
			continue
		}
		for _, a := range p.Arguments {
			if a.Required && strings.TrimSpace(args[a.Name]) == "" {
				return nil, &rpcError{codeInvalidParams, "Argument fehlt: " + a.Name}
			}
			if a.Name == "feature" && args["feature"] != "" && !scan.IsSlug(args["feature"]) {
				return nil, &rpcError{codeInvalidParams, "ungültiger Feature-Slug " + args["feature"]}
			}
		}
		text := p.body(args) + "\nGeltende Regeln:\n" + s.ruleLines(p.rules) + common
		return map[string]any{
			"description": p.Description,
			"messages":    []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": text}}},
		}, nil
	}
	return nil, &rpcError{codeInvalidParams, "unbekannter Prompt " + name}
}

// ruleLines renders the rules from the live catalog, so prompts never drift from what lint checks.
func (s *Server) ruleLines(ids []string) string {
	cat, _, err := catalog.Load(s.Root)
	if err != nil {
		return "- (Katalog nicht lesbar)\n"
	}
	var b strings.Builder
	for _, id := range ids {
		for _, r := range cat.Rules {
			if r.ID == id {
				fmt.Fprintf(&b, "- %s (%s): %s — Details mit sdd_explain\n", r.ID, r.Severity.Default+byGate(r), r.Title)
			}
		}
	}
	return b.String()
}

func byGate(r catalog.Rule) string {
	if len(r.Severity.ByGate) == 0 {
		return ""
	}
	var parts []string
	for _, g := range catalog.AllGates {
		if s, ok := r.Severity.ByGate[g]; ok {
			parts = append(parts, s+" in "+g)
		}
	}
	prefix := ""
	if r.Severity.Default != "" {
		prefix = ", "
	}
	return prefix + strings.Join(parts, ", ")
}
