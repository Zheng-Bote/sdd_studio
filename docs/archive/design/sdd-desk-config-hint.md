---
feature: sdd-desk-config-hint
status: archived
release: v0.3.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - apps/desk/src-tauri/**
  - handbuch/**
---

# Tech Spec — Richtiger Ort für `llm.env` in der Warnung von `sdd-desk`

Setzt das PRD `docs/archive/prd/sdd-desk-config-hint.md` um. Grundlage sind die Tech Specs `docs/archive/design/sdd-desk.md` (Konfiguration) und `docs/archive/design/sdd-desk-tauri.md` (Sidecar).

## Architektur

```text
cmd/sdd-desk/main.go              Flag --config-hint, an LoadConfig weitergereicht
internal/desk/config.go           LoadConfig(path, explicit, hint, getenv): Warnung nennt hint
apps/desk/src-tauri/src/sidecar.rs  config_arg liefert Config::File oder Config::Hint; desk_args setzt das Flag
```

`LoadConfig` erhält einen zusätzlichen Parameter `hint`. Nur wenn die Datei am Standardort fehlt (`!explicit`), lautet die Warnung:

`keine Konfiguration — llm.env unter <hint> anlegen (Rechte 0600); nur lokales Ollama verfügbar`

Ist `hint` leer, steht dort der Standardort. Gelesen wird `hint` nie; es ist nur ein Text.

In der Shell wird aus `config_arg(dir) -> Option<PathBuf>` ein `config_arg(dir) -> ConfigArg` mit `File(path)` (Datei vorhanden → `--config`) und `Hint(path)` (fehlt → `--config-hint`). Der Pfad ist immer `<app_config_dir>/llm.env`.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.go`, `main.go` | `TestLoadConfigHint` |
| AC-2 | `sidecar.rs` (`config_arg`, `desk_args`) | `config_is_passed_only_when_present`, `desk_args_bind_project_and_free_port` |
