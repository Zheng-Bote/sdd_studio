---
feature: sdd-desk-sidecar-name
status: active
archive-after:
touches:
  - cmd/sdd-desk/**
  - apps/desk/src-tauri/**
  - apps/desk/scripts/**
  - handbuch/**
---

# Tech Spec — Eindeutiger Name für das mitgelieferte `sdd` der Desktop-Anwendung

Setzt das PRD `docs/prd/sdd-desk-sidecar-name.md` um. Grundlage ist die Tech Spec `docs/archive/design/sdd-desk-tauri.md`.

## Architektur

```text
apps/desk/scripts/build-sidecars.mjs   baut ./cmd/sdd als binaries/sdd-studio-sdd-<triple>
apps/desk/src-tauri/tauri.conf.json    externalBin: binaries/sdd-desk, binaries/sdd-studio-sdd
cmd/sdd-desk/main.go                   findSdd(explicit, dir, lookPath): Suchreihenfolge
```

Tauri legt Sidecars ohne Zielsuffix neben die Anwendung (`.deb`: `/usr/bin/sdd-desk`, `/usr/bin/sdd-studio-sdd`). Der Name `sdd-desk` bleibt; er ist durch das Suffix bereits eindeutig.

`findSdd` wird testbar: Statt `os.Executable` und `exec.LookPath` direkt aufzurufen, erhält es das Verzeichnis von `sdd-desk` und eine Suchfunktion. Reihenfolge: `--sdd`, `<dir>/sdd-studio-sdd`, `<dir>/sdd`, `PATH`-Treffer für `sdd`, sonst `sdd`. Unter Windows jeweils mit `.exe`.

Die Entwicklungsumgebung (`bin/sdd` neben `bin/sdd-desk`) funktioniert unverändert über die zweite Stufe.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `main.go` (`findSdd`) | `TestFindSdd` |
| AC-2 | `tauri.conf.json`, `build-sidecars.mjs` | `build_runs_ui_then_sidecars` |
