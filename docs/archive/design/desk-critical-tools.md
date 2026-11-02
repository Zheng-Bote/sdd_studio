---
feature: desk-critical-tools
status: archived
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/*.go
  - internal/sddcore/scaffold/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# Tech Spec — Kritische Aktionen im KI-Desktop mit einzelner Bestätigung

Setzt das PRD `docs/archive/prd/desk-critical-tools.md` um, auf Grundlage von ADR 0003 (Entscheidungen 5 und 6, Stufe „kritisch“), ADR 0001 (`adopt --apply`) und Konzept §9.6.

## Architektur

```text
internal/sddcore/baselineplan.go   PlanBaseline(root, opts) → BaselinePlan{File, Added, Removed, Lock, Locked, Reaccepted}; Apply
cmd/sdd/main.go                    runBaseline nutzt PlanBaseline (Ausgabe unverändert)
internal/sddcore/scaffold/adr.go   AcceptADRPlan(root, cfg, path): Replace-Op mit status: accepted
internal/mcp/critical.go           sdd_baseline, sdd_adr_accept, sdd_adopt_apply; Server.AllowCritical
cmd/sdd/mcp.go                     --allow-critical (schließt --allow-write ein)
internal/desk/mcpclient.go         startet sdd mcp --allow-write --allow-critical
internal/desk/agent.go             criticalTools; Pending.Critical/Summary; nie Sitzungsfreigabe; Sperre nach Ablehnung
apps/desk/src/app/…                Freigabekarte „kritisch“; ChatSession nimmt kritische Werkzeuge nie in allowed_tools auf
```

### `PlanBaseline` (AC-2)

Die Logik von `runBaseline` zieht unverändert nach `sddcore`: Gate-Prüfung, Prüfung von `--accept` (unbekannte Regel, `baseline: forbidden` außer SDD-011), `BaselineEntries`, die vier Fälle `prune` / `accept` / `gate` / Standard, dann `adrlock.Collect` und `adrlock.Update` auf einer Kopie des Lockfiles.

`BaselinePlan` enthält:
- die neue Baseline-Datei,
- `Added` und `Removed` (Mengendifferenz über die Identität aus Gate, Regel, Pfad, Hash und Ordinal),
- das neue Lockfile,
- `Locked` (neu gelockte Pfade) und `Reaccepted` (quittierte Pfade),
- `WriteLock` (wie bisher: fehlendes Lockfile oder Änderungen).

`Apply` schreibt beides wie bisher. `Summary()` liefert z. B. „Baseline: 3 Einträge aufnehmen (SDD-022 ×3), 0 entfernen; ADR-Lockfile: 1 neu locken“. `runBaseline` gibt danach dieselben zwei Zeilen aus wie heute.

### `AcceptADRPlan` (AC-3)

Der Pfad muss bereinigt sein, unter `artifacts.adr` liegen und `scan.IsADRFileName` erfüllen. Die Datei muss existieren und `status: proposed` tragen. Dann entsteht `setFrontmatter(content, "status", "accepted")` als `Op{Replace, Expect: alt}`. Sonst kommt `ErrInvalidInput` mit dem Grund.

### Werkzeuge (AC-1, AC-4)

- **Freischaltung:** `Server.AllowCritical`, gesetzt über `--allow-critical`; der Schalter setzt `AllowWrite` mit. `toolList()` hängt die drei Werkzeuge nur mit `AllowCritical` an. `runTool` lehnt sie ohne den Schalter als unbekannt ab.
- **Vorschau:** Jede Vorschau liefert `{applied: false, critical: true, summary, …}`.
  - `sdd_baseline`: `added`, `removed`, `locked`, `reaccepted`.
  - `sdd_adr_accept`: `ops` mit Diff.
  - `sdd_adopt_apply`: `report` (Markdown von `adopt.RenderReport`) und `steps`.
- **Apply:** `apply` führt die Aktion aus und liefert das Ergebnis.
  - `sdd_baseline`: Eintragszahlen.
  - `sdd_adr_accept`: Status und SHA-256.
  - `sdd_adopt_apply`: Branch und Commits.

### Desktop (AC-5, AC-6)

- **Einstufung:** `criticalTools = [sdd_baseline, sdd_adr_accept, sdd_adopt_apply]`. `tier(tc)` ergibt `critical` bei einem dieser Werkzeuge mit `apply`, `write` bei `writeTools` mit `apply`, sonst `read`.
- **Freigabe:**
  - `read` und `write` laufen wie bisher, wenn der Aufruf einmal freigegeben ist oder das Werkzeug für die Sitzung.
  - `critical` läuft nur, wenn die Kennung des Aufrufs in `ApprovedCalls` steht; `AllowedTools` zählt nicht.
- **`Pending`:** erhält `Critical bool` und `Summary` (aus der Vorschau).
- **Sperre nach Ablehnung:** `deniedSinceUser(msgs, tool)` ist wahr, wenn nach der letzten Nutzernachricht ein Werkzeugergebnis dieses Werkzeugs mit dem Ablehnungstext steht. Dann wird ein kritischer Aufruf dieses Werkzeugs nicht geparkt, sondern sofort beantwortet: „Abgelehnt: Der Nutzer hat diese kritische Aktion in diesem Zug bereits abgelehnt. Frage erst nach einer neuen Nachricht des Nutzers erneut.“
- **Absicherung neuer Werkzeuge:** Ein Test prüft, dass jedes Werkzeug von `sdd mcp --allow-critical` genau einer der Listen (lesend, schreibend, kritisch) zugeordnet ist. Ein neues Werkzeug ohne Einstufung lässt den Test scheitern.

Die Freigabekarte zeigt bei `critical` einen roten Rahmen, das Badge „kritisch“ und den Hinweis „wird nur dieses eine Mal ausgeführt“. Die Hauptschaltfläche trägt die Zusammenfassung, „Für diese Sitzung erlauben“ fehlt. Die Vorschau zeigt Einträge, Diff oder Bericht. `ChatSession.decide` stuft `session` bei `critical` auf `once` herab.

### Prompts (AC-7)

Aus „Baseline, ADR-Lockfile, adopt --apply und das Entscheiden eines ADR sind dem Menschen vorbehalten“ wird in `common` und in den Server-Anweisungen:

- Baseline, ADR-Lockfile, adopt --apply und das Entscheiden eines ADR sind kritisch. Nutze `sdd_baseline`, `sdd_adr_accept` bzw. `sdd_adopt_apply` nur, wenn der Mensch es ausdrücklich verlangt.
- Jede Ausführung bestätigt der Mensch einzeln. Lehnt er ab, frage im selben Zug nicht erneut.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `critical.go`, `cmd/sdd/mcp.go` | `TestCriticalToolsNeedFlag` |
| AC-2 | `baselineplan.go`, `critical.go`, `cmd/sdd/main.go` | `TestBaselineTool`; CLI-Ausgabe, Baseline und Lockfile vor und nach dem Umbau in 9 Szenarien byteweise gleich |
| AC-3 | `scaffold/adr.go`, `critical.go` | `TestADRAcceptTool` |
| AC-4 | `critical.go` | `TestAdoptApplyTool` |
| AC-5 | `agent.go`, Freigabekarte, `chat-session.ts` | `TestCriticalNeedsSingleApproval`, `approval-card.spec.ts`, `chat-session.spec.ts` |
| AC-6 | `agent.go` (`deniedSinceUser`) | `TestCriticalNotRetriedAfterDenial` |
| AC-7 | `prompts.go`, `server.go` | `TestPromptsCriticalRule` |
