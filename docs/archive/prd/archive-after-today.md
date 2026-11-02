---
feature: archive-after-today
status: archived
archive-after:
touches:
  - internal/desk/**
  - internal/sddcore/scaffold/**
  - handbuch/**
release: v0.5.0
---

# PRD — Heutiges Datum für das Modell, kein Stichtag in der Vergangenheit

## Problem

Im KI-Desktop kennt das Modell das heutige Datum nicht. Im Praxistest (`data/kritik.md`, Punkt 4.3) hat es beim Anlegen eines PRD `archive-after: 2025-12-31` gesetzt, am 2026-11-02. `sdd new` hat das ohne Einwand angenommen. Das frische PRD löste sofort SDD-050 aus („archive-after überschritten“). Das Modell schloss daraus, das Feature sei abgeschlossen, und leitete eine Archivierung ein, bevor es eine Zeile Code gab.

## Ziel

Das Modell rechnet mit dem richtigen Datum. Ein Stichtag, der beim Anlegen schon überschritten wäre, entsteht gar nicht erst, egal ob Mensch oder Modell `sdd new` aufruft.

## Nutzer

- **Nutzer des KI-Desktops**, deren Artefakte das Modell anlegt.
- **Nutzer von `sdd new`** (CLI, `sdd ui`, MCP-Werkzeug `sdd_new`), die sich beim Stichtag vertippen.

## Umfang

Das heutige Datum im Systemtext jedes Modellaufrufs von `sdd-desk`. Eine Prüfung des Stichtags in `sdd new` für alle drei Wege. Die Beschreibung im Handbuch.

## Nicht-Ziele

- Ein Standardwert für `archive-after`. Ohne Angabe bleibt der Stichtag leer wie bisher.
- Prüfung bestehender Artefakte. Das bleibt Aufgabe von SDD-050.
- Uhrzeit oder Zeitzone im Systemtext.

## Akzeptanzkriterien

- **AC-1** Jeder Modellaufruf eines Gesprächszugs in `sdd-desk` (mit und ohne Streaming) erhält im Systemtext das heutige lokale Datum als `YYYY-MM-DD`, ermittelt beim jeweiligen Zug.
- **AC-2** `sdd new` lehnt ein `archive-after` vor dem heutigen lokalen Datum als ungültige Eingabe ab. Die Meldung nennt den Wert und das heutige Datum. Das heutige Datum selbst und spätere Daten sind erlaubt, wie bei SDD-050.
