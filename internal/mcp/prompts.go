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
- Reihenfolge des Prozesses: Idee → PRD → Tech Spec (wenn sdd_check_intent sie verlangt) → Plan → Tests je Kriterium → Umsetzung → Gate pre-merge → Merge → Release → Archiv. Archiviert wird erst nach dem Release. Ein überschrittenes archive-after vor der Umsetzung heißt: Stichtag mit dem Menschen korrigieren, nicht archivieren.
- Frage, statt zu erfinden. Stelle eine Frage nach der anderen und formuliere nur aus, was der Mensch beantwortet hat. Offene Punkte bleiben als offene Punkte sichtbar.
- Prüfe jeden Entwurf mit sdd_validate_draft, bevor du ihn speicherst; behebe die Befunde mit dem Menschen.
- Neue Artefakte legst du mit sdd_new an und übergibst den geprüften Entwurf als content, nie das leere Template (zuerst ohne apply, dann mit apply nach Bestätigung).
- Bestehende Dateien (Artefakte, Testdateien) liest du mit sdd_read und änderst sie mit sdd_write, mit base_sha256 aus sdd_read; neue Testdateien legst du mit sdd_write an.
- Die Konfiguration (sdd.config.yaml) liest du mit sdd_read und änderst sie nur mit sdd_config (kritisch, ADR 0004). Braucht der Ablauf eine Änderung, erkläre sie in Worten und zeige die Vorschau (Diff, wegfallende und hinzukommende Befunde); ausgeführt wird sie nur nach der einzelnen Bestätigung des Menschen. Lehnt er ab, frage im selben Zug nicht erneut.
- Melde nur, was ein Werkzeugergebnis belegt: Lies nach dem Speichern mit sdd_read gegen, statt Erfolg anzunehmen.
- Baseline, ADR-Lockfile, adopt --apply und das Entscheiden eines ADR (status: accepted) sind kritisch: Der Mensch entscheidet sie. Rufe sdd_baseline, sdd_adr_accept oder sdd_adopt_apply nur auf, wenn er es ausdrücklich verlangt; jede Ausführung bestätigt er einzeln. Lehnt er ab, frage im selben Zug nicht erneut.
- Zum Schluss: sdd_next aufrufen und dann in eigenen Worten zusammenfassen — welche Dateien gespeichert wurden (Pfad), was die Prüfung ergeben hat (in Klartext, nicht nur Regelkennungen), was offen ist und welcher Schritt als Nächstes kommt. Ende nie mit einem bloßen Werkzeugaufruf.`

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
					"0. Rufe sdd_next auf. Meldet es den Schritt „Verfassung ausfüllen“, schlage vor, zuerst den Ablauf sdd-constitution zu gehen, und fahre erst danach fort — jedes PRD wird an der Verfassung gemessen.\n" +
					"1. Kläre das Problem: Wer hat es, wann, und was kostet es heute?\n" +
					"2. Kläre Nutzer und Ziel: Was ist nach der Umsetzung anders?\n" +
					"3. Kläre den Umfang und frage ausdrücklich nach Nicht-Zielen: Was gehört bewusst nicht dazu?\n" +
					"4. Kläre die Technik, bevor es um Dateien geht: Rufe sdd_stack auf. Hat das Projekt schon eine Sprache, nenne sie und lass sie bestätigen. Ist es leer, schlage eine Sprache bzw. Form vor (z. B. Shell-Skript, Go, Rust, C++ oder nur Markdown), begründe sie mit Art des Programms, Zielplattform und Umfang, und frage. Bei einem neuen Projekt biete an, die Wahl als ADR festzuhalten (Prompt sdd-adr).\n" +
					"5. Formuliere mit mir Akzeptanzkriterien, die ein Test prüfen kann, jedes mit Kennung AC-1, AC-2, … (nie umnummerieren).\n" +
					"6. Schlage einen Feature-Slug vor (" + slugHint(a["feature"]) + ") und frage, welche Pfade die Umsetzung voraussichtlich berührt (touches) — in den Konventionen der gewählten Sprache (Quell- und Testpfade wie aus sdd_stack), nie mit Platzhalter-Endungen. Prüfe sie mit sdd_check_intent: Greift die Größenschwelle, braucht es auch eine Tech Spec.\n" +
					"7. Formuliere das PRD mit den Abschnitten Problem, Ziel, Nutzer, Umfang, Nicht-Ziele, Akzeptanzkriterien aus den Antworten, prüfe es mit sdd_validate_draft und lege es mit sdd_new an (types: [prd], content: der geprüfte Entwurf).\n"
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
					"5. Formuliere die Tech Spec, prüfe sie mit sdd_validate_draft und lege sie mit sdd_new an (types: [design], content: der geprüfte Entwurf).\n"
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
					"4. Formuliere das ADR, prüfe es mit sdd_validate_draft und lege es mit sdd_new an (types: [adr], title, content: der geprüfte Entwurf). Das ADR bleibt status: proposed; entscheiden (accepted) und locken entscheidet der Mensch — auf seinen ausdrücklichen Wunsch mit sdd_adr_accept und danach sdd_baseline, jedes einzeln von ihm bestätigt. Nach der Entscheidung ist der Text unveränderlich; Ablösung nur über ein neues ADR mit supersedes.\n"
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
					"4. Formuliere den Plan, prüfe ihn mit sdd_validate_draft und lege ihn mit sdd_new an (types: [plan], content: der geprüfte Entwurf); sdd_trace zeigt, welche Kriterien noch keine Task haben.\n"
			}},
		{Name: "sdd-adopt", Title: "Bestehendes Projekt übernehmen", Description: "Übernimmt ein bestehendes Projekt in sdd: Bericht, Übernahme auf eigenem Branch, dann die Verfassung vor dem Merge.",
			Arguments: []promptArg{},
			rules:     []string{"SDD-001", "SDD-002a", "SDD-010"},
			body: func(map[string]string) string {
				return "Übernimm mit mir dieses bestehende Projekt in sdd.\n\n" +
					"Vorgehen:\n" +
					"1. sdd_stack: Welche Sprachen und wie viele Dateien hat das Projekt?\n" +
					"2. sdd_adopt_apply ohne apply: Erkläre den Bericht — Reifegrad, die geplanten Schritte (Konfiguration, Verfassung, Agenten-Dateien, Frontmatter, Baseline, Bericht), Konflikte und Archivhinweise.\n" +
					"3. Frage, ob ich übernehmen will. Erst dann sdd_adopt_apply mit apply (kritisch: ich bestätige einzeln). Es legt einen neuen Branch an, ein Commit je Schritt; der Ausgangsbranch bleibt unverändert.\n" +
					"4. Danach steht die Verfassung als Gerüst auf dem Branch der Übernahme. Erarbeite sie jetzt mit mir (Ablauf sdd-constitution), noch vor dem Merge — dann gilt das Ausfüllen als erstmaliges Anlegen (SDD-010).\n" +
					"5. Sag mir, dass ich die Verfassung auf dem Branch committen, den Branch prüfen und mergen soll. Danach sdd_next.\n"
			}},
		{Name: "sdd-start", Title: "Neues Projekt anlegen", Description: "Legt ein neues Projekt mit sdd an: Init mit Git, Technik, Verfassung, erstes Feature.",
			Arguments: []promptArg{},
			rules:     []string{"SDD-001", "SDD-002a"},
			body: func(map[string]string) string {
				return "Lege mit mir ein neues Projekt mit sdd an.\n\n" +
					"Vorgehen:\n" +
					"1. sdd_init mit git: true — zuerst die Vorschau zeigen, nach meiner Bestätigung mit apply. Fehlt eine Git-Identität, nenne mir die Befehle.\n" +
					"2. Kläre die Technik: sdd_stack; schlage eine Sprache bzw. Form mit Begründung vor (Art des Programms, Zielplattform, Umfang) und frage. Biete an, die Wahl als ADR festzuhalten (Ablauf sdd-adr), das die Verfassung in references nennt.\n" +
					"3. Erarbeite mit mir die Verfassung (Ablauf sdd-constitution).\n" +
					"4. Dann das erste Feature (Ablauf sdd-idea).\n"
			}},
		{Name: "sdd-constitution", Title: "Verfassung erarbeiten", Description: "Erarbeitet die Verfassung aus den Quellen des Projekts (README, Agenten-Dateien, Technik): Regel für Regel mit Fundstelle, entschieden vom Menschen.",
			Arguments: []promptArg{},
			rules:     []string{"SDD-002a", "SDD-003", "SDD-010"},
			body: func(map[string]string) string {
				return "Erarbeite mit mir die Verfassung dieses Projekts.\n\n" +
					"Wann: vor dem ersten PRD. Nach sdd adopt noch auf dem Branch der Übernahme, vor dem Merge — dann gilt das Ausfüllen als erstmaliges Anlegen und braucht kein ADR (SDD-010).\n\n" +
					"Vorgehen:\n" +
					"1. sdd_read ohne path: Finde die Verfassung (constitution) und die Kontextdateien (context: README, AGENTS.md, CLAUDE.md, GEMINI.md …). Lies die Verfassung und jede Kontextdatei mit sdd_read; rufe sdd_stack für Sprache und Struktur auf.\n" +
					"2. Schlage je Pflichtabschnitt der Verfassung (z. B. Architektur/Layering, Security, Strikte Regeln) Regeln vor. Jede Regel nennt ihre Fundstelle als Datei:Zeile aus sdd_read oder „aus sdd_stack“. Nur dauerhafte, systemweite Regeln — keine Feature-Inhalte wie User Stories, Tasks oder Planung (SDD-003). Wo keine Quelle etwas hergibt, stelle eine Frage statt eine Regel zu erfinden.\n" +
					"3. Gehe die Vorschläge Regel für Regel mit mir durch: übernehmen, streichen oder ändern. Offene Punkte bleiben als offene Punkte sichtbar.\n" +
					"4. Formuliere die Verfassung ohne Platzhalter, prüfe sie mit sdd_validate_draft und speichere sie mit sdd_write (base_sha256 aus sdd_read).\n" +
					"5. Rufe sdd_gate mit gate pre-merge auf. Meldet es SDD-010, existierte die Verfassung schon am Vergleichsstand: Lege mit mir ein ADR an (Prompt sdd-adr), das sie in references nennt.\n" +
					"6. Sage, dass die Verfassung committet werden muss. Kam sie aus einer Übernahme (sdd_next meldet „Übernahme abschließen“), nenne auch: Branch prüfen und mergen, erst danach das erste PRD.\n"
			}},
		{Name: "sdd-tests", Title: "Tests je Kriterium", Description: "Legt für jedes Akzeptanzkriterium eines Features einen Test in der Sprache des Projekts an, mit Testverweis-Zeile.",
			Arguments: []promptArg{{"feature", "Feature-Slug", true}},
			rules:     []string{"SDD-022", "SDD-025"},
			body: func(a map[string]string) string {
				f := a["feature"]
				return "Lege mit mir die Tests für das Feature " + f + " an.\n\n" +
					"Vorgehen:\n" +
					"1. sdd_trace (feature: " + f + "): Welche Kriterien haben noch keinen Test?\n" +
					"2. sdd_stack: Sprache, übliche Testdatei-Muster und Kommentarsyntax des Projekts. Ist die Sprache unklar, frage.\n" +
					"3. Lege mit mir fest, in welche Datei welcher Test kommt. Erfasst tests.include diese Datei nicht (tests_include aus sdd_stack), lies sdd.config.yaml mit sdd_read und ergänze tests.include mit sdd_config um das passende Muster (kritisch: erkläre die Änderung, ich bestätige einzeln).\n" +
					"4. Schreibe je Kriterium einen Test in der Sprache des Projekts: einen echten, wenn das Verhalten schon klar ist, sonst ein Gerüst, das ausdrücklich fehlschlägt oder übersprungen wird, bis es umgesetzt ist. Direkt über jedem Test steht die Zeile <Kommentar> " + f + " AC-<n>: <was der Test prüft>. Neue Dateien legst du mit sdd_write an, bestehende liest du vorher mit sdd_read.\n" +
					"5. Prüfe mit sdd_trace, dass jedes Kriterium einen Test hat, und nenne mit sdd_next den nächsten Schritt.\n"
			}},
		{Name: "sdd-conflicts", Title: "Abgleich mit der Verfassung", Description: "Sucht Widersprüche zwischen der Tech Spec eines Features und der Verfassung — mit wörtlichen Zitaten, ohne etwas zu ändern (SDD-060, assistiert).",
			Arguments: []promptArg{{"feature", "Feature-Slug", true}},
			rules:     []string{"SDD-010"},
			body: func(a map[string]string) string {
				return "Gleiche die Tech Spec des Features " + a["feature"] + " mit der Verfassung ab, ohne Dateien zu ändern.\n\n" +
					"Vorgehen:\n" +
					"1. sdd_read ohne path: Finde Verfassung, Tech Spec (feature: " + a["feature"] + ") und ADRs.\n" +
					"2. Lies mit sdd_read die Verfassung und die Tech Spec vollständig, dann die ADRs, auf die die Tech Spec verweist. Lies nur Pfade aus dieser Liste; verweist die Tech Spec auf kein ADR, überspringe die ADRs.\n" +
					"3. Gehe die Regeln der Verfassung einzeln durch, jede nummerierte Regel jedes Abschnitts — gemeint sind die Regeln der Verfassung, nicht die SDD-Regeln des Katalogs. Suche zu jeder die Stellen der Tech Spec, die etwas vorsehen, das die Regel verbietet. Ein angenommenes ADR, das die Abweichung erlaubt, löst sie auf.\n" +
					"4. Melde nur, was du auf beiden Seiten wörtlich zitieren kannst, mit Datei:Zeile aus sdd_read, je Seite ein Zitat aus genau einer Zeile (ohne die Zeilennummer). Ohne wörtliches Zitat keine Meldung. Fehlt etwas in der Tech Spec, gehört es nicht in die Tabelle.\n" +
					"5. Prüfe dein Ergebnis, bevor du antwortest: Rufe sdd_conflicts_check (feature: " + a["feature"] + ") mit allen Meldungen (constitution und spec mit path, line, quote; kind: Widerspruch, Spannung oder unklar; suggestion: Tech Spec anpassen oder ADR für eine Verfassungsänderung) und der vollständigen Liste aller Regeln der Verfassung auf (path, line, result: Treffer oder kein Widerspruch). Nennt das Ergebnis fehlende Regeln, gehe diese nach und rufe sdd_conflicts_check erneut auf.\n" +
					"6. Antworte mit der Tabelle aus markdown des Ergebnisses; sie enthält nur belegte Meldungen mit den geprüften Zeilen. Darunter die verworfenen Meldungen mit Grund (nicht gefunden, zu kurz, nicht zulässig) — erfinde keine Ersatzzitate. Dann die Regelliste mit Datei:Zeile und je „Treffer in der Tabelle“ oder „kein Widerspruch gefunden“.\n\n" +
					"Das Ergebnis ist ein Hinweis, kein Befund: Es erscheint in keinem Bericht und in keiner Baseline. Ob ein Widerspruch vorliegt und wie er aufgelöst wird, entscheidet der Mensch. Ändere nichts.\n"
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
