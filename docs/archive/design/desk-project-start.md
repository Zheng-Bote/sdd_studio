---
feature: desk-project-start
status: archived
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# Tech Spec — Nicht erfasste Projekte im KI-Desktop erkennen und starten

Setzt das PRD `docs/archive/prd/desk-project-start.md` um. Grundlagen sind `docs/archive/design/desk-guided-flow.md` (`sdd_stack`), `docs/archive/design/constitution-guide.md` (`sdd-constitution`), `docs/archive/design/agent-files.md` und `docs/archive/design/desk-critical-tools.md` (`sdd_adopt_apply`).

## Architektur

```text
internal/sddcore/guide/stack.go     StackInfo.Files: Zahl aller gezählten Dateien
internal/desk/server.go             handleMeta: project_state aus sdd_stack (Aufruf des Desktops, nicht des Modells)
apps/desk/src/app/core/types.ts     Meta.project_state
apps/desk/src/app/chat/project-start.ts  Hinweis mit Schaltfläche (Komponente ProjectStart)
apps/desk/src/app/chat/chat-panel.ts  holt sdd-adopt bzw. sdd-start und legt den Text ins Eingabefeld
internal/mcp/prompts.go             Prompts sdd-adopt und sdd-start
internal/mcp/server.go              Anweisung für ein Projekt ohne Konfiguration
```

**Status (AC-2):**
- `handleMeta` ruft `s.Agent.Runner.CallTool("sdd_stack", {})` auf (der MCP-Client des Desktops) und liest `configured` und `files` aus dem Ergebnis. Daraus ergibt sich `project_state`: `configured`, `existing` (bei `files > 0`) oder `empty`.
- Ein Fehler lässt das Feld weg.
- Der Aufruf ist lesend und vom Desktop selbst ausgelöst. Die Freigabestufen aus ADR 0003 gelten für Aufrufe, die das Modell anfordert; dieser Aufruf erscheint nicht im Gespräch.

**Oberfläche (AC-3):** Ist `project_state` gleich `existing` oder `empty`, zeigt die Komponente `ProjectStart` in `ChatPanel` über dem leeren Gespräch einen Hinweis („Dieses Projekt ist noch nicht mit sdd erfasst.“) mit einer Schaltfläche. Sie ruft `api.prompt('sdd-adopt' | 'sdd-start', {})` und legt den Text mit `prefill` ins Eingabefeld.

**Prompts (AC-4):**

`sdd-adopt`:
1. `sdd_stack`: Was ist das für ein Projekt?
2. `sdd_adopt_apply` ohne `apply`: den Bericht erklären (Reifegrad, Schritte einschließlich Agenten-Dateien, Konflikte, Archivhinweise).
3. Fragen, dann `sdd_adopt_apply` mit `apply` (kritisch, der Mensch bestätigt einzeln). Arbeitsbaum und Branch prüft `adopt` selbst.
4. Danach steht der Branch der Übernahme. Mit `sdd-constitution` die Verfassung hier erarbeiten, vor dem Merge (SDD-010).
5. Hinweis: die Verfassung auf dem Branch committen, den Branch prüfen und mergen; danach `sdd_next`.

`sdd-start`:
1. `sdd_init` mit `git: true`, zuerst die Vorschau, dann `apply` nach Bestätigung.
2. Technik klären: `sdd_stack` und Vorschlag mit Begründung, ADR anbieten (`sdd-adr`).
3. `sdd-constitution`.
4. `sdd-idea` für das erste Feature.

**Anweisungen:** Aus „Fehlt die Projektkonfiguration, schlage sdd_init vor (für ein neues Projekt mit git: true).“ wird: „Fehlt die Projektkonfiguration: Hat der Ordner schon Inhalt (sdd_stack: files > 0), schlage den Ablauf sdd-adopt vor, sonst sdd-start (sdd_init mit git: true).“

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `stack.go` | `TestStack` |
| AC-2 | `desk/server.go` | `TestMetaProjectState` |
| AC-3 | `project-start.ts`, `chat-panel.ts` | `project-start.spec.ts` |
| AC-4 | `prompts.go`, `server.go` | `TestStartPrompts` |
