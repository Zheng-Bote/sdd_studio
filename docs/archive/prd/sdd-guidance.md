---
feature: sdd-guidance
status: archived
release: v0.2.0
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - internal/mcp/**
  - handbuch/**
  - constitution.md
  - .gitignore
---

# PRD — Prozessführung: sdd next und sdd mcp (`sdd-guidance`)

## Problem

Wer `sdd` nutzt, muss die Reihenfolge des Prozesses kennen: erst PRD mit Nicht-Zielen und Akzeptanzkriterien, dann die erklärte Absicht, bei Überschreiten der Schwelle die Tech Spec, bei Verfassungsänderungen ein ADR, dann Plan, Tests, Gate und Archiv. Die Regeln melden Verstöße, sagen aber nicht, *was als Nächstes dran ist*. Agenten — und der geplante KI-Desktop (ADR 0002) — haben keinen maschinenlesbaren Zugang zu diesem Wissen.

## Ziel

`sdd next` nennt für jedes Feature den nächsten Prozessschritt mit Begründung und passendem Befehl. `sdd mcp` stellt diesen und die übrigen Prüfungen Agenten als Werkzeuge, Ressourcen und geführte Abläufe zur Verfügung — KI-frei, offline, ohne Zugriff auf Baseline, Lockfile oder Git-Schreibvorgänge.

## Nutzer

- **Menschen**, die ein Feature Schritt für Schritt durch den Prozess bringen.
- **Agenten** (Claude Code, Claude Desktop, `sdd-desk`), die vor einer Änderung fragen, was gilt und was fehlt.

## Umfang

Befehl `sdd next` und MCP-Server `sdd mcp` über stdio mit Werkzeugen, Ressourcen und Prompts.

## Nicht-Ziele

- LLM-Aufrufe oder Netzwerkzugriff in `sdd` — das bleibt dem getrennten `sdd-desk` vorbehalten (ADR 0002).
- MCP über HTTP oder andere Transporte als stdio.
- Werkzeuge für Baseline, Lockfile, `adopt --apply` oder das Entscheiden von ADRs.
- Automatisches Ausfüllen von Artefakten; die Prompts fragen, statt Text zu erfinden.

## Akzeptanzkriterien

- **AC-1** `sdd next --feature <slug>` zeigt die Prozessschritte des Features in fester Reihenfolge mit Zustand (erledigt, offen, nicht nötig, manuell) und benennt genau einen nächsten Schritt mit Begründung und Befehl; ohne `--feature` je aktivem Feature den nächsten Schritt, ohne Feature den Einstieg mit `sdd new`.
- **AC-2** Die Schritte werden aus den Dateien und denselben Regeln abgeleitet wie `sdd lint` (PRD, Nicht-Ziele, Akzeptanzkriterien, Absicht, Schwelle, Tech Spec, AC-Bezug, Plan, Tests, Archivierung); `--json` liefert dieselben Daten maschinenlesbar.
- **AC-3** `sdd mcp` spricht JSON-RPC 2.0 über stdio (`initialize`, `ping`, `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list`, `prompts/get`), handelt die Protokollversion aus und meldet unbekannte Methoden als Fehler, ohne abzubrechen.
- **AC-4** Die Werkzeuge `sdd_next`, `sdd_lint`, `sdd_gate`, `sdd_report`, `sdd_explain`, `sdd_check_intent`, `sdd_context`, `sdd_trace` und `sdd_validate_draft` liefern dieselben Ergebnisse wie die zugehörigen Bibliotheksfunktionen; `sdd_context` ordnet die für eine Datei geltenden Artefakte nach Präzedenz (Verfassung, ADR, Tech Spec, PRD, Plan).
- **AC-5** `sdd_new` zeigt ohne `--allow-write` nur die Vorschau; erst ein mit `--allow-write` gestarteter Server legt Dateien an, und nur über den Plan/Apply-Pfad von `sdd new`.
- **AC-6** Kein Werkzeug, keine Ressource und kein Prompt verändert Baseline, ADR-Lockfile oder Git oder setzt ein ADR auf `accepted`; Pfade außerhalb des Repositorys werden abgelehnt.
- **AC-7** Die Prompts `sdd-idea`, `sdd-techspec`, `sdd-adr`, `sdd-plan` und `sdd-review` führen durch den jeweiligen Schritt, nennen die geltenden Regeln und verlangen Prüfung mit `sdd_validate_draft` vor dem Schreiben.
- **AC-8** Kern, CLI, `sdd ui` und `sdd mcp` enthalten keinen Code für ausgehende Verbindungen; ein Test prüft die Quelltexte.
