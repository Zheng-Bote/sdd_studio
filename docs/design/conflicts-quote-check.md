---
feature: conflicts-quote-check
status: active
archive-after:
touches:
  - internal/mcp/**
  - internal/desk/**
  - handbuch/**
  - docs/sdd_konzept.md
---

# Tech Spec — Zitate aus sdd-conflicts maschinell prüfen

Setzt das PRD `docs/prd/conflicts-quote-check.md` um. `sdd` bleibt KI-frei (ADR 0003, Entscheidung 1): Das Werkzeug vergleicht Zeichenketten, es bewertet nichts.

## Architektur

```text
internal/mcp/conflicts.go    sdd_conflicts_check: Zulässigkeit, Zitatprüfung, Regelliste, Markdown-Tabelle
internal/mcp/tools.go        Werkzeugliste und Verteilung (lesend, ohne Schalter)
internal/mcp/prompts.go      sdd-conflicts, Schritte 5 und 6
internal/desk/agent.go       readTools um sdd_conflicts_check ergänzt
handbuch/04-agenten.md       SDD-060-Absatz, Zahl der Werkzeuge (zwanzig)
docs/sdd_konzept.md          §9.6, Absatz „Assistierter Abgleich“
```

### Eingabe (AC-1, AC-3)

```json
{
  "feature": "billing",
  "conflicts": [{
    "constitution": {"path": "constitution.md", "line": 12, "quote": "…"},
    "spec": {"path": "docs/design/billing.md", "line": 40, "quote": "…"},
    "kind": "Widerspruch | Spannung | unklar",
    "suggestion": "Tech Spec anpassen | ADR für eine Verfassungsänderung"
  }],
  "rules": [{"path": "constitution.md", "line": 12, "result": "Treffer | kein Widerspruch"}]
}
```

Ein unbekannter Wert in `kind`, `suggestion` oder `result` führt zu einem Werkzeugfehler mit den erlaubten Werten. Fehlt `feature`, ebenso.

### Zulässige Dateien (AC-2)

- `constitution`: genau `artifacts.constitution`.
- `spec`: eine Tech Spec mit `feature: <feature>` aus `artifacts.design` oder ein ADR aus `artifacts.adr`. Die Tech Spec kann ein verlinktes ADR zitieren, das die Abweichung erlauben soll.

Jede andere Datei bekommt den Status `nicht zulässig`. Gelesen wird über `readArtifact`, also mit denselben Pfadregeln wie `sdd_read`.

### Zitatprüfung (AC-2)

`norm(s)`: Markdown-Hervorhebungen (`*`, `_`, `` ` ``) entfernen, jeden Leerraum zu einem Leerzeichen zusammenfassen, außen trimmen. Ein vom Modell mitkopiertes Zeilenpräfix aus `sdd_read` (`  12| `) wird vorher entfernt.

- `len(norm(quote)) < 12` → `zu kurz`.
- `norm(line[n])` enthält `norm(quote)` → `belegt`.
- Sonst: alle Zeilen mit Treffer suchen. Genau eine → `andere Zeile` mit `found_line`; keine oder mehrere → `nicht gefunden` (mehrdeutig zählt nicht als belegt).

Die Prüfung arbeitet zeilenweise. Ein Zitat über einen Zeilenumbruch hinweg wird nicht gefunden; der Prompt verlangt Zitate aus einer Zeile.

### Regelliste (AC-4)

Für eine Verfassung mit Endung `.md`: Zeilen außerhalb von Code-Blöcken, die auf `^\d+\.\s` passen und nach einer Überschrift `## ` stehen, sind Regeln. `missing_rules` sind die Regeln ohne Eintrag in `rules` (Vergleich über die Zeile), `unknown_rules` die Einträge, deren Zeile keine Regel ist. Bei `.sdd`: `rules_checkable: false` mit dem Hinweis „Regelliste nicht prüfbar“.

### Ergebnis (AC-5)

```json
{
  "conflicts": [{"index": 0, "verified": true, "kind": "…", "suggestion": "…",
                 "constitution": {"path": "…", "line": 12, "status": "belegt"},
                 "spec": {"path": "…", "line": 41, "status": "andere Zeile", "found_line": 41}}],
  "summary": {"verified": 1, "rejected": 0, "missing_rules": 0},
  "missing_rules": [{"path": "constitution.md", "line": 14, "text": "…"}],
  "unknown_rules": [],
  "rules_checkable": true,
  "markdown": "| Verfassung | Tech Spec | Einschätzung | Vorschlag |\n…"
}
```

Die Tabelle in `markdown` enthält nur belegte Meldungen. Die Zitate erscheinen mit den geprüften Zeilen (`found_line`, wo vorhanden). Senkrechte Striche in Zitaten werden maskiert.

### Prompt (AC-6)

`sdd-conflicts` Schritt 5 wird zu: „Rufe sdd_conflicts_check mit allen Meldungen (je Seite path, line und ein wörtliches Zitat aus einer Zeile) und der vollständigen Regelliste auf. Nennt das Ergebnis fehlende Regeln, prüfe sie nach und rufe das Werkzeug erneut auf.“ Schritt 6: „Antworte mit der Tabelle aus markdown. Darunter: verworfene Meldungen mit Grund (nicht gefunden, zu kurz, nicht zulässig), dann die Regelliste.“

### Desktop

`readTools` erhält `sdd_conflicts_check`. `TestToolTiersCoverAllTools` und `TestAgentUsesOnlySddTools` sichern die Einstufung und die Zahl (20).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `conflicts.go`, `tools.go` | `TestConflictsCheckTool` |
| AC-2 | `conflicts.go` (`norm`, Zitatprüfung, Zulässigkeit) | `TestConflictsCheckQuotes` |
| AC-3 | `conflicts.go` (Aufzählungen, `verified`) | `TestConflictsCheckQuotes`, `TestConflictsCheckRefusesValues` |
| AC-4 | `conflicts.go` (`constitutionRules`) | `TestConflictsCheckRules` |
| AC-5 | `conflicts.go` (Ergebnis, `markdown`) | `TestConflictsCheckQuotes` |
| AC-6 | `prompts.go` | `TestConflictsPromptUsesCheck` |
| AC-7 | `handbuch/04-agenten.md`, `docs/sdd_konzept.md` | `TestSDD060NotInCatalog` (erweitert) |

## Risiken

- **Zu strenge Prüfung:** Ein inhaltlich richtiges Zitat mit Tippfehler gilt als nicht gefunden. Das ist gewollt, denn wörtlich heißt wörtlich. Die Antwort nennt solche Meldungen als verworfen, der Mensch sieht sie also trotzdem.
- **Mehrdeutige Zitate** (dieselbe Wendung in mehreren Zeilen, Angabe falsch) gelten als nicht belegt. Das Modell kann mit längerem Zitat erneut prüfen.
