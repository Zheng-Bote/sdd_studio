---
feature: archive-not-draft
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - handbuch/**
  - README.md
---

# PRD — Kein Archiv für ein Feature im Entwurf

## Problem

`sdd archive` verschiebt ein Feature ins Archiv, sobald jedes Akzeptanzkriterium einen Testverweis hat. Mehr prüft es nicht. Im Live-Test vom 2026-11-05 trug das PRD noch `status: draft`. Es gab keine Zeile Code, und alle zehn Tests waren Gerüste mit `t.Skip`. Trotzdem hätte `sdd_archive` das Feature archiviert. Abgehalten hat davon nur das Modell, nicht das Werkzeug. Ein archiviertes Feature verschwindet aus `sdd next` und den Prüfungen der aktiven Features. Ein Entwurf im Archiv täuscht also einen Abschluss vor, den es nicht gibt.

## Ziel

Ein Feature, dessen PRD noch Entwurf ist, lässt sich nicht archivieren, weder im Terminal noch im Desktop. Ein Feature mit fertiger Spezifikation (`active`, `implemented`) archiviert sich wie bisher.

## Nutzer

- **Nutzer von `sdd archive`** im Terminal und in der Release-Vorbereitung.
- **Nutzer des KI-Desktops**, in dem das Modell `sdd_archive` vorschlägt.

## Umfang

- Eine zusätzliche Abbruchbedingung in `sdd archive` und `sdd_archive`.
- Handbuch und README.

## Nicht-Ziele

- Erkennen, ob Tests übersprungen werden oder Code fehlt. Das hängt von der Sprache ab und ist nicht deterministisch prüfbar.
- Eine Pflicht für `--release`.
- Änderungen an SDD-050 oder an der Bedeutung der übrigen Statuswerte.

## Akzeptanzkriterien

- **AC-1** `sdd archive` bricht mit Exit-Code `2` ab, ohne etwas zu verschieben, wenn ein PRD des Features außerhalb des Archivs `status: draft` trägt. Das gilt auch mit `--dry-run`. Die Meldung nennt den Pfad und sagt, dass erst nach Umsetzung und Merge archiviert wird.
- **AC-2** `sdd_archive` lehnt ein solches Feature ebenso ab, als Vorschau wie mit `apply`.
- **AC-3** Ein Feature, dessen PRDs `active`, `implemented` oder keinen Status tragen, archiviert sich unverändert.
- **AC-4** Handbuch (Abschnitt `sdd archive`) und README nennen die neue Abbruchbedingung.
