---
feature: agent-files
status: archived
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
release: v0.5.0
---

# Tech Spec — Agenten-Dateien bei `sdd init` und `sdd adopt`

Setzt das PRD `docs/archive/prd/agent-files.md` um. Grundlage sind die Tech Specs `docs/archive/design/sdd-init-new.md` (Plan/Apply, additiv) und `docs/archive/design/sdd-adopt.md` (Schritte, ein Commit je Schritt).

## Architektur

```text
templates/agents.md.tmpl              Inhalt von AGENTS.md (Pfade aus der Konfiguration)
templates/agents-block.md.tmpl        vorangestellter Block für eine vorhandene AGENTS.md
internal/sddcore/scaffold/agents.go   AgentFilesPlan(root, cfg) → Plan, Details
cmd/sdd/scaffold.go                   init: Plan um AgentFilesPlan ergänzen
internal/mcp/init.go                  sdd_init: ebenso
internal/sddcore/adopt/plan.go        Schritt „Agenten-Dateien“ nach „Verfassung“
```

### `AgentFilesPlan`

Konstante `AgentsMarker = "<!-- sdd:agents -->"`. Für jede Datei wird `Lstat` geprüft: Nur reguläre Dateien zählen; ein Link oder Ordner wird übersprungen und gemeldet.

| Datei | fehlt | vorhanden ohne Markierung |
|---|---|---|
| `AGENTS.md` | anlegen aus `agents.md.tmpl` | `agents-block.md.tmpl` voranstellen |
| `CLAUDE.md` | anlegen: Markierung, `@AGENTS.md`, ein Satz | Verweisblock voranstellen |
| `GEMINI.md`, `.gemini/GEMINI.md`, `gemini/GEMINI.md` | nur `GEMINI.md`, und nur wenn keine der drei existiert | Verweisblock voranstellen |
| `.github/copilot-instructions.md` | anlegen: Markierung, Link-Verweis | Link-Verweisblock voranstellen |

Der **Verweisblock** lautet:

```
<!-- sdd:agents -->
Lies zuerst @<rel> — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).

```

Für `.github/copilot-instructions.md` lautet er als Markdown-Link, weil offen ist, ob Copilot `@`-Einbindungen auflöst:

```
<!-- sdd:agents -->
Lies zuerst [AGENTS.md](../AGENTS.md) — Verfassung, Präzedenz und Ablauf dieses Projekts (sdd).

```

`<rel>` ist der Pfad von der Datei zu `AGENTS.md` (`AGENTS.md`, aus `.gemini/` und `.github/` `../AGENTS.md`). Voranstellen nutzt die bestehende `Op{Prepend, Expect}`; ändert sich die Datei zwischen Planung und Anwendung, wird übersprungen. Beginnt eine Datei mit `---`-Frontmatter, wird sie nicht geändert; die Details nennen „Frontmatter vorhanden — Verweis von Hand ergänzen“.

Die Templates erhalten `Constitution`, `ADR`, `PRD`, `Design` und `Plans` aus `cfg.Artifacts`.

### Einbindung

- **`sdd init`, `sdd_init`:** `plan.Ops = append(plan.Ops, agents.Ops...)`. Weil `CommitInit` nur Ergebnisse mit `created` committet, sind neue Agenten-Dateien Teil des ersten Commits. Eine vorhandene Agenten-Datei, der ein Block vorangestellt wird, nimmt der Commit bewusst nicht mit: Sie gehört dem Nutzer und bleibt unversioniert wie jede andere vorhandene Datei (`sdd-init-git`, AC-1).
- **`adopt`:** In `Propose` folgt nach dem Schritt „Verfassung“ `Step{Title: "Agenten-Dateien", Plan, Detail}`, wenn der Plan Operationen hat. `adopt.Apply` committet je Schritt wie bisher.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `agents.md.tmpl`, `agents.go` | `TestAgentFilesCreate` |
| AC-2 | `agents.go` | `TestAgentFilesCreate`, `TestAgentFilesGeminiVariants`, `TestAgentFilesCopilot` |
| AC-3 | `agents.go` (Prepend, Markierung, Frontmatter) | `TestAgentFilesPrepend`, `TestAgentFilesCopilot` |
| AC-4 | `cmd/sdd/scaffold.go`, `mcp/init.go`, `adopt/plan.go` | `TestInitWithGit`, `TestInitTool`, `TestAdoptAgentStep` |
