---
feature: desk-guided-flow
status: archived
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# PRD — Geführter Ablauf: Technik vor touches, Reihenfolge, Tests je Sprache

## Problem

Im Praxistest führte `sdd-idea` von der Idee direkt zu Akzeptanzkriterien und touches, ohne zu klären, ob ein Shell-Skript, ein C++-, Rust- oder Go-Programm oder nur Markdown entsteht. Die touches waren deshalb erfundene Pfade (`src/main.xyz`, `tests/…_test.xyz`), und `sdd_check_intent` prüfte die Größenschwelle auf dieser Grundlage. Das Modell kann die Sprache eines bestehenden Projekts auch nicht erkennen, denn `sdd_read` liest keinen Code.

Außerdem kennt das Modell die Reihenfolge des Prozesses nicht. Es hat ein Feature archiviert, für das es noch keinen Code gab, weil ein überschrittenes `archive-after` SDD-050 auslöste. Für Testdateien gibt es keinen geführten Schritt: Das Modell wich auf Markdown-Notizen aus, statt Tests in der Projektsprache anzulegen.

## Ziel

Der geführte Ablauf klärt die Technik, bevor er nach Pfaden fragt. Pfade und Tests folgen den Konventionen der gewählten Sprache. Jeder geführte Schritt kennt die Reihenfolge des Prozesses, und Tests je Kriterium haben einen eigenen geführten Schritt.

## Nutzer

- **Nutzer des KI-Desktops**, neu oder in einem bestehenden Projekt.
- **Andere MCP-Clients**, die die Prompts von `sdd mcp` nutzen.

## Umfang

- Ein lesendes Werkzeug `sdd_stack`.
- Der neu geordnete Prompt `sdd-idea`.
- Die Prozess-Reihenfolge in allen geführten Prompts.
- Ein neuer Prompt `sdd-tests`.
- Die Einstufung von `sdd_stack` im Desktop.
- Handbuch und README.

## Nicht-Ziele

- Code oder Tests ausführen, Abhängigkeiten installieren.
- Eine Empfehlung der Sprache durch `sdd`. Die Empfehlung formuliert das Modell, die Entscheidung trifft der Mensch.
- Eine Erkennung über Dateiinhalte. Es zählen nur Dateinamen und Marker-Dateien.
- Änderungen an `sdd next` oder an Katalogregeln.

## Akzeptanzkriterien

- **AC-1** `sdd_stack` nennt die erkannten Sprachen des Repositorys, sortiert nach Zahl der Quelldateien. Je Sprache nennt es die gefundenen Marker-Dateien (z. B. `go.mod`, `Cargo.toml`, `CMakeLists.txt`), die üblichen Testdatei-Muster, die Kommentarsyntax und ein Beispiel für die Testverweis-Zeile. Dazu kommt das aktuelle `tests.include`. `exclude`, `.git/` und `.sdd/` zählen nicht mit. Ein Repository ohne Quelldateien liefert eine leere Liste. Im KI-Desktop ist `sdd_stack` lesend.
- **AC-2** `sdd-idea` klärt nach Umfang und Nicht-Zielen und vor den Akzeptanzkriterien die Technik. Mit `sdd_stack` wird eine erkannte Sprache bestätigt; ohne erkannte Sprache schlägt das Modell eine mit Begründung vor und fragt. Bei einem neuen Projekt bietet es ein ADR für die Wahl an. Danach fragt es nach den touches in den Konventionen dieser Sprache, ohne Platzhalter-Endungen.
- **AC-3** Jeder geführte Prompt nennt die Reihenfolge des Prozesses: Idee, PRD, Tech Spec (wenn nötig), Plan, Tests je Kriterium, Umsetzung, Gate, Merge, Release, Archiv. Er sagt außerdem, dass ein überschrittenes `archive-after` vor der Umsetzung einen falschen Stichtag bedeutet und kein Archivieren.
- **AC-4** Der Prompt `sdd-tests` (Feature) führt so: Kriterien ohne Test mit `sdd_trace`, Sprache mit `sdd_stack`, wenn nötig `tests.include` mit `sdd_read`/`sdd_write` setzen. Dann je Kriterium ein Test in der Projektsprache, mit der Zeile `<Kommentar> <slug> AC-<n>: …` direkt über dem Test, angelegt mit `sdd_write`. Zum Schluss Prüfung mit `sdd_trace` und `sdd_next`.
