---
feature: constitution-guide
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - cmd/sdd/**
  - handbuch/**
  - README.md
---

# PRD — Die Verfassung KI-gestützt erarbeiten

## Problem

`sdd init` und `sdd adopt` legen die Verfassung nur als Gerüst mit Platzhaltern an (`1. <Sicherheitsregeln, die für jede Änderung gelten>`). Im Projekt Check_Cert stehen die echten Regeln in `.gemini/GEMINI.md`: nur Go-Standardbibliothek, SPDX-Header, englische Doc-Kommentare, MVC, Geheimnisse in `.env`-Dateien, Einstiegspunkte unter `cmd/`. In die Verfassung gelangen sie nicht, und das Modell im KI-Desktop kann sie nicht lesen, weil `sdd_read` nur Artefakte liest. Weder `sdd next` noch der Desktop sagen, dass die Verfassung noch leer ist. Jedes PRD wird so an einer leeren Verfassung gemessen.

## Ziel

Eine leere Verfassung fällt auf, und es gibt einen geführten Schritt, der sie aus den vorhandenen Quellen des Projekts erarbeitet. Das Modell schlägt Regeln mit Fundstelle vor, der Mensch entscheidet Regel für Regel. Der Schritt steht an der richtigen Stelle: vor dem ersten PRD, bei einer Übernahme noch vor dem Merge des Übernahme-Branches.

## Nutzer

- **Wer ein bestehendes Projekt mit `sdd adopt` übernimmt.**
- **Wer ein neues Projekt mit `sdd init` beginnt.**
- **Nutzer des KI-Desktops.**

## Umfang

- `sdd_read` für Projekt-Kontextdateien.
- Der Schritt „Verfassung ausfüllen“ in `sdd next` und `sdd_next`.
- Der Prompt `sdd-constitution`.
- Der Hinweis in `sdd-idea`.
- Handbuch und README.

## Nicht-Ziele

- Regeln automatisch, ohne Bestätigung, übernehmen.
- Quellcode lesen. Quellen sind Kontextdateien und `sdd_stack`.
- Eine neue Katalogregel. Platzhalter in der Verfassung sind ein Prozessschritt, kein Befund.
- Kontextdateien schreiben (das tut `agent-files`).

## Akzeptanzkriterien

- **AC-1** `sdd_read` liest die vorhandenen Kontextdateien `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.gemini/GEMINI.md`, `gemini/GEMINI.md`, `.github/copilot-instructions.md` und `.cursorrules`, mit Zeilennummern und SHA-256. Ohne `path` nennt es die vorhandenen unter `context`. `sdd_write` schreibt sie weiterhin nicht.
- **AC-2** Enthält die Verfassung eine Listenzeile, die nur aus einem Platzhalter `<…>` besteht (außerhalb von Codeblöcken), meldet `sdd next` (CLI und `sdd_next`) vor allen anderen Schritten den Schritt „Verfassung ausfüllen“. Er nennt den Prompt `sdd-constitution` und die Position: vor dem ersten PRD, nach einer Übernahme noch auf dem Branch der Übernahme, damit das Ausfüllen als erstmaliges Anlegen gilt (SDD-010). Ohne Platzhalter entfällt der Schritt.
- **AC-3** Der Prompt `sdd-constitution` führt so: Verfassung und Kontextdateien mit `sdd_read` lesen und `sdd_stack` aufrufen. Dann je Pflichtabschnitt Regeln vorschlagen, jede mit Fundstelle (Datei:Zeile), nur dauerhafte, systemweite Regeln und keine Feature-Inhalte (SDD-003). Der Mensch bestätigt, streicht oder ergänzt Regel für Regel; Offenes bleibt sichtbar. Prüfen mit `sdd_validate_draft`, speichern mit `sdd_write`. Danach `sdd_gate pre-merge`: Meldet es SDD-010, braucht die Änderung ein ADR (Prompt `sdd-adr`).
- **AC-4** `sdd-idea` beginnt mit `sdd_next`. Meldet es „Verfassung ausfüllen“, schlägt das Modell zuerst `sdd-constitution` vor.
