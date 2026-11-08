---
feature: sdd-commit
status: archived
archive-after:
touches:
  - internal/sddcore/scan/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
  - docs/arc42.md
  - CHANGELOG.md
release: v0.7.0
---

# PRD — Commit-Werkzeug `sdd_commit` im KI-Desktop

## Problem

Jeder Ablauf im KI-Desktop endet mit dem Hinweis, dass der Mensch die gespeicherten Dateien selbst committen muss. Das zwingt ins Terminal und lässt Ergebnisse uncommittet liegen. ADR 0008 erlaubt dafür ein kritisches Werkzeug.

## Ziel

Der Mensch kann Artefakte, Testdateien, Konfiguration und Prüfzustand aus dem Desktop committen, nach einzelner Bestätigung, auf einem Feature-Branch und ohne dass Quellcode oder Geheimnisse mitgehen.

## Nutzer

- **Nutzer des KI-Desktops**, die einen Ablauf ohne Wechsel ins Terminal abschließen wollen.

## Umfang

- Werkzeug `sdd_commit` in `sdd mcp`.
- Einstufung und Darstellung im Desktop.
- Prompts, Handbuch, README, Architekturdoku, CHANGELOG.

## Nicht-Ziele

- Branches anlegen, `push`, `merge`, `rebase`, `reset`.
- Ein CLI-Befehl; im Terminal gibt es `git`.
- Quellcode committen.

## Akzeptanzkriterien

- **AC-1** `sdd mcp` bietet `sdd_commit` (`paths`, `message`, `apply`) nur mit `--allow-critical` an, mit `destructiveHint: true`.
- **AC-2** Erlaubt sind nur einzeln genannte Dateien, die `sdd_write` schreiben darf, sowie `sdd.config.yaml`, `.sdd/baseline.json` und `.sdd/adr-lock.json`. Jeder andere Pfad, ein Platzhalter, ein Pfad mit Git-Pathspec-Syntax oder ein symbolischer Link wird abgelehnt, und nichts wird committet. Eine leere Pfadliste, eine leere Nachricht und ein Pfad ohne Änderung werden ebenfalls abgelehnt.
- **AC-3** Auf `main`, `master`, dem Branch aus `base_ref` und ohne Branch lehnt das Werkzeug ab und nennt, dass erst ein Feature-Branch anzulegen ist.
- **AC-4** Ohne `apply` liefert es eine Vorschau mit `critical: true`, Branch, Nachricht, je Datei Aktion und Diff und einer Zusammenfassung mit Dateizahl und Branch. Git und Dateien bleiben unverändert.
- **AC-5** Mit `apply` committet es genau die genannten Pfade, auch gelöschte, und liefert den Commit. Anderes bereits Vorgemerktes bleibt uncommittet.
- **AC-6** Im Desktop ist `sdd_commit` mit `apply` kritisch. Die Freigabekarte zeigt Nachricht, Dateien und Diff.
- **AC-7** Prompts und Server-Anweisungen behandeln `sdd_commit` wie die übrigen kritischen Aktionen.
- **AC-8** Handbuch, README, Architekturdoku und CHANGELOG beschreiben das Werkzeug.
