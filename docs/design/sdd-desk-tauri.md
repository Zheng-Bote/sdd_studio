---
feature: sdd-desk-tauri
status: active
archive-after:
touches:
  - apps/desk/**
  - cmd/sdd-desk/**
  - .gitignore
  - handbuch/**
  - README.md
  - sdd.config.yaml
---

# Tech Spec — KI-Desktop als Tauri-v2-Anwendung

Setzt das PRD `docs/prd/sdd-desk-tauri.md` um. Rahmen: [ADR 0002](../adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md) und die Tech Spec `docs/design/sdd-desk.md`. Vorbild für den Sidecar-Lebenszyklus ist `md_indexer/apps/desktop/src-tauri/src/sidecar.rs`.

## Architektur

```text
apps/desk/src-tauri/
  ├── tauri.conf.json      Bündel, externalBin binaries/sdd-desk + binaries/sdd, Startseite splash/
  ├── splash/              Startseite: Status, Fehler, Hinweis auf „Projekt öffnen…“ (ohne IPC)
  ├── src/lib.rs           Fenster, Menü, Navigationsregel, Projektwahl
  ├── src/sidecar.rs       SidecarManager: starten, Adresse lesen, beenden; Generationen
  ├── src/settings.rs      zuletzt genutztes Projekt in <app_config_dir>/desk.json
  └── binaries/            gebaute Sidecars <name>-<target-triple> (nicht eingecheckt)
apps/desk/scripts/build-sidecars.mjs   go build von sdd und sdd-desk nach binaries/
```

Die Oberfläche lädt das Fenster **direkt vom Sidecar** (`http://127.0.0.1:<port>/?token=…`), genau wie im Browser: gleiche Herkunft, gleiche Content-Security-Policy, gleiche Token- und `Host`-Prüfung. Die Hülle ist nur Fenster und Prozessverwalter.

## Schnittstellen

**Start des Sidecars:** `sdd-desk --project <projekt> --port 0 --exit-with-stdin [--config <app_config_dir>/llm.env]`. Mit `--exit-with-stdin` beendet sich `sdd-desk`, sobald seine Standardeingabe schließt — das geschieht, wenn die Hülle endet, auch durch Absturz oder `SIGTERM`, bei dem Tauri kein Exit-Ereignis mehr auslöst. Tauri legt Sidecars neben das Programm; `findSdd` in `cmd/sdd-desk` findet `sdd` dort.

**Bereitschaft:** `sdd-desk` schreibt die Zeile `http://127.0.0.1:<port>/?token=<hex>` auf stdout. Sie ist damit eine Schnittstelle: die Hülle erkennt sie am Präfix `http://127.0.0.1:` und `/?token=`, nimmt sie nicht ins Protokoll auf und navigiert dorthin. Kommt sie nicht binnen 30 Sekunden, gilt der Start als fehlgeschlagen.

**Startseite:** `splash/index.html?state=starting|error&project=…&message=…` — rein statisch, liest nur ihre Abfrageparameter. Fehlertexte und Protokollzeilen setzt die Hülle ohne Token ein.

**Menü:** „Datei → Projekt öffnen…“ (Ordnerdialog über `tauri-plugin-dialog` in Rust), „Datei → Beenden“.

**Projektwahl:** erstes Programmargument, sonst `desk.json`, sonst Ordnerdialog. Ein gewähltes Projekt wird nach erfolgreichem Start gespeichert.

## Sicherheit

- **Navigationsregel** (`on_navigation`): erlaubt sind nur der Ursprung der Startseite und `http://127.0.0.1:<port>` des laufenden Sidecars. Andere `http`/`https`-Adressen öffnet der Standardbrowser, alles andere wird verworfen.
- **Kein IPC:** keine Capability-Datei, keine eigenen Kommandos; die Sidecar-Seite ist eine entfernte Herkunft ohne Tauri-Rechte, die Startseite braucht keine.
- **Schlüssel:** die Hülle übergibt nur einen Pfad; `sdd-desk` prüft Rechte (`0600`) und liest die Schlüssel selbst.
- Der Sidecar wird bei Projektwechsel und Programmende beendet; stirbt die Hülle, enden `sdd-desk` (`--exit-with-stdin`) und `sdd mcp` mit ihrem Eingabekanal. Kein Token-Server bleibt verwaist zurück.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `sidecar.rs` (`desk_args`), `build-sidecars.mjs` | `desk_args_bind_project_and_free_port` |
| AC-2 | `lib.rs` (`choose_project`), `settings.rs` | `project_from_arg_then_settings`, `settings_roundtrip` |
| AC-3 | `sidecar.rs` (`ready_url`, `redact`) | `ready_url_is_recognised`, `token_never_reaches_the_log` |
| AC-4 | `lib.rs` (`navigation`) | `navigation_allows_only_splash_and_sidecar` |
| AC-5 | keine Capabilities, `tauri.conf.json` | `no_capabilities_are_granted` |
| AC-6 | `sidecar.rs` (`config_arg`) | `config_is_passed_only_when_present` |
| AC-7 | `sidecar.rs` (Generationen, `stop`), `--exit-with-stdin` in `cmd/sdd-desk`, `splash/` | `splash_url_carries_error_without_token`, `sidecar_ends_with_the_shell` |
| AC-8 | `package.json` (`tauri:build`, `desk:sidecars`), `tauri.conf.json` (`beforeBuildCommand`) | `build_runs_ui_then_sidecars` |

## Offene Punkte

- Streaming der Modellantworten (eigenes Feature).
- Icons sind ein schlichter Platzhalter, bis es ein Logo gibt.
