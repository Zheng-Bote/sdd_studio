---
status: proposed
date: 2026-11-05
supersedes: []
references:
  - docs/sdd_konzept.md
  - handbuch/03-mechanismen.md
---

# ADR 0007 — Qualitätsanforderungen im PRD mit stabiler Kennung `QA-<n>`

## Kontext

Ein PRD hält heute Problem, Ziel, Nutzer, Umfang, Nicht-Ziele und Akzeptanzkriterien mit Kennung `AC-<n>`. Für kleine Features reicht das. Bei mittleren und größeren Vorhaben fehlen gerade die Anforderungen, die später teuer werden: messbare Qualitätsanforderungen (Leistung, Zuverlässigkeit, Sicherheit …), Randbedingungen, Annahmen und das Verhalten im Fehlerfall. Im Live-Test vom 2026-11-05 kamen die Grenzfälle erst nach mehreren Runden zutage, das PRD wuchs von vier auf zehn Kriterien.

Ein erweiterter, optionaler Ablauf zum Anforderungsmanagement soll das früh erfragen. Offen ist, wie Qualitätsanforderungen im PRD stehen. Als Akzeptanzkriterien `AC-<n>` passen sie schlecht: Sie gelten oft für das ganze Feature und sind nicht immer durch einen einzelnen Test belegbar. SDD-022 würde für jede einen Testverweis verlangen. Ohne Kennung wiederum lassen sie sich nicht aus arc42 (Abschnitt 10), aus Tech Specs oder Tests verfolgen.

Erwogen wurden:

1. **Als `AC-<n>` führen.** Verworfen: SDD-022 verlangt dann für „Verfügbarkeit 99,5 %“ einen Testverweis, der oft nur vorgetäuscht wäre.
2. **Ohne Kennung als Fließtext.** Verworfen: nicht verfolgbar, nicht stabil zitierbar.
3. **Eigene Kennung `QA-<n>`** in einem eigenen Abschnitt — vorgeschlagen.

## Entscheidung

1. Ein PRD **kann** einen Abschnitt `## Qualitätsanforderungen` haben. Jede Anforderung dort trägt eine Kennung `QA-<n>`, ist messbar formuliert (Merkmal, Maß, Zielwert, Bedingung) und wird wie `AC-<n>` nie umnummeriert; entfallene werden als ~~QA-n~~ markiert.
2. Weitere **optionale** Abschnitte: `## Randbedingungen`, `## Annahmen und Abhängigkeiten`. Grenzfälle und Fehlerverhalten werden zu Akzeptanzkriterien `AC-<n>`, weil sie testbar sind.
3. **Keine neue Pflichtregel.** SDD-021 bis SDD-023 bleiben unverändert; `QA-<n>` braucht keinen Testverweis. Eine Prüfung auf vage Formulierungen kommt, wenn überhaupt, später als heuristische `warn`-Regel (Verfassung, Strikte Regel 2).
4. Tech Specs, Tests und die Architekturdoku (arc42 Abschnitt 10) **dürfen** `QA-<n>` zitieren; `sdd` wertet das zunächst nicht aus.

## Konsequenzen

- Neuer, optionaler Ablauf `sdd-requirements` (eigenes Feature mit PRD und Tech Spec); `sdd-idea` bietet ihn bei mittleren und größeren Vorhaben an.
- Handbuch: Abschnitt zu den optionalen PRD-Abschnitten und `QA-<n>`.
- `sdd-arc42` füllt Qualitätsziele und -szenarien aus den `QA-<n>` der PRDs.
- Spätere Erweiterung möglich: `sdd_trace` zeigt `QA-<n>` mit Fundstellen, eine Regel prüft eindeutige Kennungen.
- Risiko: Anforderungen wachsen zu Pflichtenheften. Gegenmaßnahme: alles optional, der Ablauf fragt je Merkmal nur, was für dieses Vorhaben zählt, und das Konzept schließt Projektverwaltung (Prioritäten, Schätzungen) weiter aus.
