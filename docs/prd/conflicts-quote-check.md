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

# PRD — Zitate aus sdd-conflicts maschinell prüfen

## Problem

Der geführte Ablauf `sdd-conflicts` (SDD-060, assistiert) lässt das Modell die Tech Spec eines Features gegen die Verfassung halten. Gemeldet werden darf nur, was das Modell auf beiden Seiten wörtlich mit Datei und Zeile zitiert. Ob die Zitate stimmen, prüft heute niemand außer dem Menschen.

Im E2E-Test vom 2026-10-21 hat das lokale Modell `qwen3:14b` die eingebauten Widersprüche gefunden, sich aber nicht an die Formregeln gehalten: Es hat Zeilen ohne Zitat hinzugefügt, und die Liste der geprüften Regeln war unvollständig. Ein Mensch, der der Tabelle traut, übernimmt dann erfundene oder falsch verortete Zitate. Eine fehlende Regel fällt ihm nicht auf.

## Ziel

Jedes Zitat aus `sdd-conflicts` ist maschinell gegen die Datei geprüft, bevor der Mensch es sieht. Nicht belegte Meldungen erscheinen ausdrücklich als nicht belegt. Regeln der Verfassung, die das Modell nicht durchgegangen ist, werden benannt. Die Einschätzung selbst (Widerspruch, Spannung, unklar) bleibt eine Aussage des Modells, und die Entscheidung bleibt beim Menschen.

## Nutzer

- **Nutzer des KI-Desktops**, die Tech Specs mit `sdd-conflicts` gegen die Verfassung prüfen lassen.
- **Reviewer**, die eine solche Prüfung nachvollziehen wollen.

## Umfang

- Ein lesendes, deterministisches Werkzeug `sdd_conflicts_check` in `sdd mcp`, das das strukturierte Ergebnis des Modells entgegennimmt und prüft.
- Der Prompt `sdd-conflicts` verlangt, das Ergebnis vor der Antwort mit diesem Werkzeug zu prüfen.
- Einstufung im Desktop, Handbuch, Konzept.

## Nicht-Ziele

- Prüfen, ob ein Widerspruch inhaltlich besteht. Das ist nicht deterministisch entscheidbar und bleibt beim Menschen (Konzept §9.6).
- SDD-060 als Katalogregel. Das Ergebnis bleibt ein Hinweis und erscheint in keinem Bericht und keiner Baseline.
- Eine eigene Darstellung des Ergebnisses in der Oberfläche des Desktops. Das Modell antwortet weiter mit Text.
- Regellisten für Verfassungen im SDD-Format (`.sdd`).

## Akzeptanzkriterien

- **AC-1** `sdd mcp` bietet das lesende Werkzeug `sdd_conflicts_check` (`feature`, `conflicts`, `rules`) an, ohne Schalter und mit `readOnlyHint: true`. Es schreibt nichts.
- **AC-2** Jedes Zitat (`path`, `line`, `quote`) wird gegen die Datei geprüft:
  - **belegt**, wenn es in der angegebenen Zeile steht;
  - **belegt, andere Zeile**, wenn es in genau einer anderen Zeile steht (diese Zeile wird genannt);
  - **nicht gefunden** in jedem anderen Fall.
  Der Vergleich ignoriert Unterschiede in Leerraum und Markdown-Hervorhebungen (`*`, `_`, Backticks). Ein Zitat unter 12 Zeichen gilt als **zu kurz**. Zulässig sind nur die Verfassung, die Tech Spec des Features und ADRs, sonst gilt das Zitat als **nicht zulässig**.
- **AC-3** Eine Meldung (`constitution`, `spec`, `kind`, `suggestion`) gilt als belegt, wenn beide Zitate belegt sind, auch mit anderer Zeile. `kind` muss `Widerspruch`, `Spannung` oder `unklar` sein, `suggestion` muss `Tech Spec anpassen` oder `ADR für eine Verfassungsänderung` sein. Andere Werte werden abgelehnt.
- **AC-4** Bei einer Markdown-Verfassung ist jeder nummerierte Listenpunkt unter einer Überschrift der zweiten Ebene eine Regel (Datei:Zeile). Das Ergebnis nennt die Regeln, die in `rules` fehlen, und die Einträge in `rules`, die keine Regel sind. Bei einer Verfassung im SDD-Format steht dort „Regelliste nicht prüfbar“.
- **AC-5** Das Ergebnis enthält je Meldung den Prüfstatus beider Zitate, eine Zusammenfassung (belegt, nicht belegt, fehlende Regeln) und eine Markdown-Tabelle der belegten Meldungen mit den geprüften Zeilen.
- **AC-6** Der Prompt `sdd-conflicts` verlangt:
  - Vor der Antwort ruft das Modell `sdd_conflicts_check` mit allen Meldungen und der vollständigen Regelliste auf.
  - Nennt das Ergebnis fehlende Regeln, prüft das Modell diese nach und ruft das Werkzeug erneut auf.
  - Die Antwort zeigt nur belegte Meldungen, mit den geprüften Zeilen. Nicht belegte Meldungen nennt sie getrennt als verworfen, mit Grund.
- **AC-7** Konzept und Handbuch beschreiben die maschinelle Prüfung der Zitate. Das Ergebnis bleibt ein Hinweis, kein Befund, und SDD-060 steht weiter nicht im Katalog.
