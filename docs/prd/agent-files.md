---
feature: agent-files
status: active
archive-after:
touches:
  - internal/sddcore/scaffold/**
  - internal/sddcore/adopt/**
  - cmd/sdd/**
  - internal/mcp/**
  - internal/ui/**
  - templates/**
  - handbuch/**
  - README.md
---

# PRD — Agenten-Dateien bei `sdd init` und `sdd adopt`

## Problem

Agenten wie Claude Code, Codex, Gemini CLI oder GitHub Copilot lesen beim Start eine Datei mit Projektanweisungen (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, `.github/copilot-instructions.md`). `sdd init` und `sdd adopt` legen keine davon an. Ein Agent, der in einem sdd-Projekt arbeitet, erfährt deshalb nichts von der Verfassung, der Präzedenz der Artefakte oder dem Ablauf. Im Projekt Check_Cert enthält `.gemini/GEMINI.md` eigene Regeln, verweist aber weder auf die Verfassung noch auf sdd.

## Ziel

Nach `sdd init` oder `sdd adopt --apply` führt jede übliche Agenten-Datei zur Verfassung und zum sdd-Ablauf. Der Inhalt steht einmal in `AGENTS.md`, die anderen Dateien verweisen darauf. Bestehender Inhalt bleibt erhalten.

## Nutzer

- **Entwickler**, die mit Claude Code, Codex, Gemini CLI oder ähnlichen Agenten in einem sdd-Projekt arbeiten.
- **Wer ein Projekt mit `sdd init` oder `sdd adopt` aufsetzt.**

## Umfang

- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` und `.github/copilot-instructions.md` anlegen bzw. mit einem Verweis versehen.
- Gilt für `sdd init`, `sdd_init`, `sdd adopt` und `sdd_adopt_apply`.
- Handbuch und README.

## Nicht-Ziele

- Inhalte vorhandener Agenten-Dateien in die Verfassung übernehmen (eigenes Feature `constitution-guide`).
- Weitere Formate (`.cursorrules`).
- Bestehende Inhalte ändern oder entfernen.

## Akzeptanzkriterien

- **AC-1** Fehlt `AGENTS.md`, wird sie angelegt. Sie nennt die konfigurierte Verfassung als ranghöchstes Artefakt, die Präzedenz (Verfassung → ADR → Tech Spec → PRD → Plan) mit den konfigurierten Pfaden, die Reihenfolge des Prozesses, die Testverweis-Konvention, die Entscheidungen, die dem Menschen vorbehalten sind, und die Einbindung von `sdd mcp`.
- **AC-2** Fehlt `CLAUDE.md`, wird sie mit einem Verweis auf `AGENTS.md` angelegt (`@AGENTS.md`). Gibt es weder `GEMINI.md` noch `.gemini/GEMINI.md` noch `gemini/GEMINI.md`, wird `GEMINI.md` ebenso angelegt. Fehlt `.github/copilot-instructions.md`, wird sie mit einem Markdown-Link angelegt (`Lies zuerst [AGENTS.md](../AGENTS.md) …`), weil offen ist, ob Copilot `@`-Einbindungen auflöst.
- **AC-3** Jede vorhandene Datei unter `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.gemini/GEMINI.md`, `gemini/GEMINI.md` und `.github/copilot-instructions.md` ohne die Markierung `<!-- sdd:agents -->` erhält am Anfang einen Block mit dieser Markierung: in `AGENTS.md` den Verweis auf die Verfassung und den Ablauf, sonst den Verweis auf `AGENTS.md` (relativer Pfad; in `.github/copilot-instructions.md` als Markdown-Link `[AGENTS.md](../AGENTS.md)`). Der übrige Inhalt bleibt byteweise erhalten. Eine Datei mit der Markierung bleibt unverändert, und ein zweiter Lauf ändert nichts. Beginnt eine Datei mit einem Frontmatter-Block, wird sie nicht verändert und gemeldet.
- **AC-4** `sdd init` und `sdd_init` legen die Agenten-Dateien mit den übrigen Dateien an; bei `--git` sind sie Teil des ersten Commits. `sdd adopt --apply` führt dafür einen eigenen Schritt „Agenten-Dateien“ mit eigenem Commit aus, der Probelauf zeigt ihn.
