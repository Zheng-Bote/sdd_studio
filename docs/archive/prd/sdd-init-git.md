---
feature: sdd-init-git
status: archived
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/scan/**
  - internal/mcp/**
  - internal/desk/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# PRD — Projekt-Init mit Git, auch aus dem KI-Desktop

## Problem

Ein neues Projekt beginnt im KI-Desktop mit einem Fehler. Im Praxistest scheiterte `sdd_check_intent` an „keine Projektkonfiguration gefunden; `sdd adopt` oder `sdd init` ausführen“. Der Nutzer musste ins Terminal wechseln und `sdd init` ausführen. Danach scheiterte `sdd_lint` an Git: Der Ordner war kein Repository, und `pre-implementation` und `pre-merge` brauchen eine Merge-Basis. Auch das ließ sich nur im Terminal beheben (`git init`, erster Commit).

## Ziel

Ein neues Projekt lässt sich in einem Schritt startklar machen: Konfiguration, Verfassung, Verzeichnisse und, auf Wunsch, ein Git-Repository mit erstem Commit. Das geht über die CLI und im KI-Desktop, dort mit Vorschau und Freigabe.

## Nutzer

- **Nutzer des KI-Desktops**, die ein neues Projekt anlegen.
- **CLI-Nutzer**, die `sdd init` in einem leeren Ordner ausführen.

## Umfang

- `sdd init --git` in der CLI.
- Das MCP-Werkzeug `sdd_init` mit Vorschau.
- Die Einstufung als schreibender Aufruf im Desktop.
- Der Hinweis auf `sdd_init` in den Server-Anweisungen.
- Handbuch und README.

## Nicht-Ziele

- `git init` in einem Ordner, der schon in einem Git-Repository liegt, auch in einem übergeordneten.
- `push`, Remote einrichten, Branches anlegen.
- Die Git-Identität setzen.
- Init über `sdd ui`.
- Ein bestehendes Projekt übernehmen (`sdd adopt`).

## Akzeptanzkriterien

- **AC-1** `sdd init --git` legt in einem Ordner ohne Git-Repository die Dateien von `sdd init` an, führt `git init` mit dem Branch `main` aus und erstellt einen Commit, der genau die angelegten Dateien enthält. Andere Dateien im Ordner bleiben unversioniert.
- **AC-2** Liegt der Ordner bereits in einem Git-Repository, legt `sdd init --git` nur die Dateien an. Es führt weder `git init` noch einen Commit aus und meldet das.
- **AC-3** Fehlt Git oder ist keine Git-Identität (Name und E-Mail) gesetzt, bricht `sdd init --git` vor dem ersten Schreibzugriff mit Exit-Code `2` ab. Die Meldung nennt die Abhilfe.
- **AC-4** Das MCP-Werkzeug `sdd_init` (Profil, ohne CI, mit Git) zeigt ohne `apply` die Dateien mit Inhalt und, was mit Git geschieht. Mit `apply` (nur mit `--allow-write`) verhält es sich wie AC-1 bis AC-3 und liefert die angelegten Dateien, den Commit und den nächsten Schritt.
- **AC-5** Im KI-Desktop gilt `sdd_init` mit `apply` als schreibender Aufruf mit Vorschau. Die Server-Anweisungen nennen `sdd_init`, wenn die Projektkonfiguration fehlt.
