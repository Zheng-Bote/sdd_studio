---
feature: sdd-adopt
status: archived
release: v0.1.0
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - templates/**
  - handbuch/**
  - constitution.md
---

# PRD — Übernahme bestehender Projekte (`sdd-adopt`)

## Problem

`sdd init` hilft nur neuen Projekten. Ein gewachsenes Repository verletzt neue Regeln hundertfach: Feature-Spezifikationen liegen in einer fremden Struktur (z. B. Spec Kit unter `specs/features/<slug>/`), tragen kein `feature:`-Feld, haben keine AC-Kennungen, und eine Konfiguration fehlt. Ein Gate, das diesen Altbestand als `error` meldet, wird abgeschaltet statt repariert.

## Ziel

`sdd adopt` hebt ein bestehendes Projekt ohne Umbruch auf einen prüfbaren Stand: Es inventarisiert, klassifiziert, bestimmt den Reifegrad, schlägt genau einen nächsten Schritt vor und wendet die Übernahme auf Wunsch auf einem eigenen Branch an — ein Commit pro Schritt, nichts gelöscht, nichts überschrieben.

## Nutzer

- **Teams mit Bestandsprojekten**, die `sdd` einführen wollen, ohne den Hauptzweig anzufassen.
- **Reviewer**, die eine Übernahme Schritt für Schritt nachvollziehen und einzeln zurücknehmen können.

## Umfang

Glob-fähige Artefaktpfade in der Konfiguration, `sdd adopt` mit Inventar, Klassifikation, Reifegrad L0–L4, Vorschlag, Probelauf und Anwendung auf einem Branch sowie ein Adopt-Bericht.

## Nicht-Ziele

- KI-gestützte Klassifikation — die Zuordnung folgt festen Pfad- und Namensmustern.
- Ausführen des Archivierungsschritts; er wird nur ausgewiesen.
- `sdd report` mit Schuldenstand-Trend und PR-Kommentar.
- Inhaltliches Ergänzen von Artefakten (Kennungen für Akzeptanzkriterien nachtragen, Abschnitte schreiben).
- Push, Pull Request oder Merge des Adopt-Branches.

## Akzeptanzkriterien

- **AC-1** `artifacts.prd`, `artifacts.design`, `artifacts.plans` und `artifacts.archive` dürfen Ordner oder Datei-Globs sein (z. B. `specs/features/*/spec.md`); alle Regeln und `sdd new` werten beide Formen aus.
- **AC-2** `sdd adopt` ordnet jede Markdown-Datei genau einem der sechs Artefakttypen zu und nennt den Grund; was keinem Muster entspricht, ist Doku/Exploration.
- **AC-3** `sdd adopt` bestimmt den Reifegrad L0–L4 mit Begründung je Kriterium und schlägt genau einen nächsten Schritt vor.
- **AC-4** Ohne `--apply` schreibt `sdd adopt` nichts und zeigt jede geplante Änderung einschließlich der einzufügenden Frontmatter-Blöcke.
- **AC-5** `sdd adopt --apply` verlangt ein Git-Repository mit sauberem Arbeitsbaum, legt einen neuen Branch an (bricht ab, wenn er existiert) und hält jeden Schritt in einem eigenen Commit fest; der Ausgangsbranch bleibt unverändert, keine Datei wird gelöscht oder ersetzt.
- **AC-6** Die Baseline wird nach Konfiguration und Frontmatter, aber vor dem ersten Gate-Lauf geschrieben und enthält nur baselinefähige `error`-Befunde; der Archivierungsschritt wird mit den betroffenen Dateien ausgewiesen, nicht ausgeführt.
- **AC-7** `sdd adopt --apply` schreibt den Bericht nach `.sdd/adopt-report.md`; der Probelauf gibt denselben Bericht aus.
