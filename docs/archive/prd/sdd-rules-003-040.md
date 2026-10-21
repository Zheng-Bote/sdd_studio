---
feature: sdd-rules-003-040
status: archived
release: v0.2.0
archive-after:
touches:
  - catalog/**
  - internal/sddcore/rules/**
  - handbuch/**
---

# PRD — Regeln SDD-003 und SDD-040 (`sdd-rules-003-040`)

## Problem

Zwei Verfallsformen bleiben bisher unentdeckt: Die Verfassung sammelt Feature-Inhalte — User Stories, Sprintplanung, Tasklisten — und verliert damit ihre Rolle als dauerhafte, systemweite Regelquelle. Und Features werden in Tasks zerlegt, bevor es einen technischen Plan gibt; im Pilotbestand hatten 11 von 22 Features keine `plan.md`.

## Ziel

Beide Verfallsformen werden früh sichtbar — als `warn`, weil die Erkennung von Feature-Inhalten heuristisch ist und ein fehlender Plan nicht in jedem Projekt blockieren soll.

## Nutzer

- **Verantwortliche der Verfassung**, die sie schlank und dauerhaft halten.
- **Teams**, die Tasks erst nach dem technischen Plan schneiden.

## Umfang

`SDD-003` und `SDD-040` mit Katalogeintrag, Handbuchabschnitt, Prüfvertrag und Tests.

## Nicht-Ziele

- `SDD-060` (Widersprüche zwischen Verfassung und Tech Spec): Laut Konzept nur assistiert über ein lokales Sprachmodell; die KI-Anbindung ist eine eigene Entscheidung.
- Prüfung der Reihenfolge in der Git-Historie (wann Plan und Tasks entstanden sind); geprüft wird der aktuelle Stand.
- Eine verlässliche Erkennung natürlicher Sprache; `SDD-003` arbeitet mit festen Mustern.

## Akzeptanzkriterien

- **AC-1** `SDD-003` meldet ein `warn` für jede Zeile der Verfassung, die nach festen Mustern Feature-Inhalt ist: eine User Story, eine Checkbox-Task, eine AC-Kennung oder eine Überschrift zu Sprint, Backlog, Roadmap, Meilenstein oder User Stories; Codeblöcke werden übersprungen.
- **AC-2** `SDD-040` meldet ein `warn` für jedes Feature, das in einem aktiven Plan mindestens eine Task hat, aber keine aktive Tech Spec mit demselben `feature:`.
