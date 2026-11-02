---
feature: sdd-archive
status: archived
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# PRD — Features archivieren mit `sdd archive`

## Problem

Ein Feature zu archivieren (SDD-050) heißt heute: PRD, Tech Spec und Plan von Hand nach `artifacts.archive` verschieben und in jeder Datei `status: archived` setzen. Einen Befehl dafür gibt es nicht. Im KI-Desktop ist das Modell deshalb dreimal auf `sdd_new` mit Typ `doc` ausgewichen und hat Dateien unter `docs/notes/` angelegt. Am Ende musste der Nutzer im Terminal verschieben. Das Modell hat außerdem ein Feature archiviert, für das es noch keine Zeile Code und keinen Test gab.

## Ziel

Ein Feature lässt sich mit einem Schritt archivieren, in der CLI und im KI-Desktop, mit Vorschau. Ein Feature, dessen Akzeptanzkriterien noch nicht getestet sind, lässt sich nicht archivieren.

## Nutzer

- **Wer ein Release vorbereitet** und die Features des Releases archiviert.
- **Nutzer des KI-Desktops.**

## Umfang

- Der CLI-Befehl `sdd archive`.
- Das MCP-Werkzeug `sdd_archive`.
- Die Einstufung als schreibender Aufruf im Desktop.
- Handbuch und README.

## Nicht-Ziele

- Git-Operationen (`git mv`, Commit). Verschoben wird im Arbeitsbaum.
- Mehrere Features in einem Aufruf.
- Archivieren über `sdd ui`.
- Ein Archiv, das als Datei-Glob konfiguriert ist.
- ADRs, Verfassung und Doku archivieren.
- Verweise in anderen Artefakten nachführen. SDD-051 meldet sie wie bisher.

## Akzeptanzkriterien

- **AC-1** `sdd archive --feature <slug> [--release <version>]` verschiebt jede Feature-Datei (PRD, Tech Spec, Plan) außerhalb des Archivs nach `<artifacts.archive>/<Typordner>/<Rest des Pfades>`, z. B. `docs/prd/x.md` nach `docs/archive/prd/x.md`. Im Frontmatter steht danach `status: archived`, mit `--release` auch `release: <version>`. Der übrige Inhalt bleibt unverändert, und die Originaldatei ist entfernt.
- **AC-2** `sdd archive` bricht mit Exit-Code `2` ab und schreibt nichts, wenn das Feature keine Datei außerhalb des Archivs hat, wenn `artifacts.archive` fehlt oder ein Datei-Glob ist, wenn eine Zieldatei schon existiert, oder wenn ein Akzeptanzkriterium des Features in keiner Testdatei referenziert ist. Im letzten Fall nennt die Meldung die Kennungen.
- **AC-3** `--dry-run` zeigt je Datei Quelle, Ziel und die geänderten Frontmatter-Zeilen, ohne zu schreiben.
- **AC-4** Das MCP-Werkzeug `sdd_archive` (`feature`, `release`, `apply`) verhält sich wie AC-1 bis AC-3. `apply` ist nur mit `--allow-write` erlaubt. Im KI-Desktop gilt es mit `apply` als schreibender Aufruf mit Vorschau.
- **AC-5** Nach `sdd archive` meldet `sdd lint` für das Feature kein SDD-050 mehr, und SDD-022 zählt seine Testverweise weiter mit (archivierte Features bleiben bekannt).
