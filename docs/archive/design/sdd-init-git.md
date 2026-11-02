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

# Tech Spec — Projekt-Init mit Git, auch aus dem KI-Desktop

Setzt das PRD `docs/archive/prd/sdd-init-git.md` um, auf Grundlage von ADR 0003, Entscheidung 8, und Verfassung Security 4 (zweite Ausnahme vom lesenden Git-Zugriff). Grundlage ist die Tech Spec `docs/archive/design/sdd-init-new.md`.

## Architektur

```text
internal/sddcore/scan/gitwrite.go     CLIGit.InitRepo(branch), CLIGit.HasIdentity()
internal/sddcore/scaffold/gitinit.go  CheckGitInit (vor dem Schreiben), CommitInit (danach)
cmd/sdd/scaffold.go                   sdd init --git
internal/mcp/tools.go                 Werkzeug sdd_init
internal/desk/agent.go                writes(): sdd_init mit apply
internal/mcp/server.go                Anweisung: ohne Konfiguration sdd_init vorschlagen
```

**Git:**
- `InitRepo(branch)` führt `git init --quiet --initial-branch=<branch>` aus.
- `HasIdentity()` ist wahr, wenn `git var GIT_COMMITTER_IDENT` gelingt. Das berücksichtigt Konfiguration und Umgebungsvariablen und funktioniert auch außerhalb eines Repositorys.
- `Commit(message, paths)` (aus ADR 0001) übernimmt nur die genannten Pfade.

**Ablauf** (CLI und MCP gleich):

1. `CheckGitInit(root)` vor dem Plan. Ist `root` in einem Work Tree (`IsRepo`), ist das Ergebnis `GitSkip` mit dem Hinweis „liegt bereits in einem Git-Repository; kein git init, kein Commit“. Fehlt `git` (`exec.LookPath`), endet es mit `ErrGitInit` „git ist nicht installiert oder nicht im PATH“. Fehlt die Identität, endet es mit `ErrGitInit` „Git-Identität fehlt: git config --global user.name …; git config --global user.email …“. Sonst ist das Ergebnis `GitCreate`.
2. Der Plan wird angewandt wie bei `sdd init`.
3. Bei `GitCreate`: `CommitInit(root, results)` ruft `InitRepo("main")` auf, dann `Commit("chore: sdd init", <Pfade mit Status created>)`. Es liefert den kurzen Hash. Schlägt Git danach noch fehl, bleiben die Dateien liegen. Die Meldung nennt den Fehler mit Exit-Code `2`.

Die CLI meldet `git init (main), Commit <hash>` bzw. den Hinweis von `GitSkip`.

**`sdd_init`** (Argumente `profile`, `no_ci`, `git`, `apply`): Der Projektname ist der Ordnername von `root`, wie in der CLI.
- Die Vorschau liefert `{applied: false, ops: [...], git: "<was geschieht>"}`. `ops` hat dieselbe Form wie bei `sdd_new`, damit die Freigabekarte sie zeigt.
- `apply` braucht `--allow-write` und liefert `{applied: true, results, git, commit, next}`. `next` ist der Hinweis von `sdd init` auf `thresholds`.
- Eine fehlgeschlagene Vorbedingung ist ein Werkzeugfehler. In diesem Fall wird nichts geschrieben.

**Desktop:** `writes(tc)` ist wahr für `sdd_new`, `sdd_write` und `sdd_init` mit `apply`.

**Anweisungen:** In den Server-Anweisungen kommt hinzu: „Fehlt die Projektkonfiguration, schlage sdd_init vor (für ein neues Projekt mit git: true).“

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `gitinit.go`, `gitwrite.go`, `cmd/sdd/scaffold.go` | `TestInitWithGit` |
| AC-2 | `gitinit.go` (`GitSkip`) | `TestInitWithGitInsideRepo` |
| AC-3 | `gitinit.go` (Vorbedingungen) | `TestInitWithGitNeedsIdentity` |
| AC-4 | `tools.go` (`sdd_init`) | `TestInitTool` |
| AC-5 | `agent.go`, `server.go` | `TestInitIsWrite`, `TestInitTool` |
