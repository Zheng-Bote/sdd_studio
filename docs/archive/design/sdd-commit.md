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

# Tech Spec — Commit-Werkzeug `sdd_commit` im KI-Desktop

Setzt das PRD `docs/archive/prd/sdd-commit.md` um, auf Grundlage von ADR 0008.

## Architektur

```text
internal/sddcore/scan/gitwrite.go   HeadBranch: symbolic-ref, leer ohne Branch
internal/mcp/commit.go              sdd_commit: Prüfungen, Vorschau, Commit
internal/mcp/critical.go, tools.go  Registrierung und Verteilung
internal/mcp/prompts.go, server.go  Regel für kritische Aktionen
internal/desk/agent.go              criticalTools
apps/desk/src/app/…                 Aktion „löschen“, Commit-Nachricht auf der Karte
```

### Prüfungen (AC-2, AC-3)

Reihenfolge: Nachricht und Pfadliste; Branch; je Pfad `scan.CleanRel`, keine Platzhalter und kein führendes `:` (Git wertet sonst Pathspec-Syntax aus), Klasse Artefakt, Test oder Konfiguration aus `classify` oder einer der beiden `.sdd`-Dateien. `readExisting` lehnt symbolische Links und alles außerhalb der Wurzel ab. Fehlt die Datei, muss sie an `HEAD` existieren (Löschung). Ist der Inhalt gleich dem an `HEAD`, kommt ein Fehler. Die Prüfung läuft für alle Pfade vor dem ersten Git-Aufruf, der schreibt.

`HeadBranch` nutzt `git symbolic-ref --quiet --short HEAD`, nicht `CurrentBranch`, weil dessen CI-Umgebungsvariablen ein detached HEAD verdecken würden. Gesperrt sind `main`, `master` und `base_ref` ohne `origin/`.

### Vorschau und Commit (AC-4, AC-5)

Die Vorschau vergleicht `ShowFile("HEAD", p)` mit der Datei über `scaffold.UnifiedDiff` und verändert auch den Index nicht. `apply` nutzt das vorhandene `CLIGit.Commit`: `git add -- <pfade>` und `git commit -- <pfade>`, das nur diese Pfade aufnimmt. Hooks laufen normal.

### Desktop (AC-6)

`criticalTools` erhält `sdd_commit`. Die Karte zeigt `preview.message` und die Aktion `delete` als „löschen“.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `commit.go`, `critical.go` | `TestCommitToolNeedsFlag` |
| AC-2 | `commit.go` | `TestCommitRefusesPaths` |
| AC-3 | `commit.go`, `gitwrite.go` | `TestCommitRefusesBranches` |
| AC-4 | `commit.go` | `TestCommitPreview` |
| AC-5 | `commit.go` | `TestCommitApply` |
| AC-6 | `agent.go`, Karte | `TestCommitIsCritical`, `approval-card.spec.ts` |
| AC-7 | `prompts.go`, `server.go` | `TestPromptsCommitTool` |
| AC-8 | Handbuch, README, arc42, CHANGELOG | `TestCommitDocumented` |
