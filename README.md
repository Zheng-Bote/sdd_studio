# SDD Studio (`sdd`)

Sprachneutrales CLI-Werkzeug (`sdd`) und Go-Kernbibliothek (`internal/sddcore`) zur maschinellen Durchsetzung von **Spec-Driven Development** (`sdd_process_3`) über einen gemeinsamen, selbstprüfenden Regelkatalog und ein eingebettetes Handbuch. Dazu kommt der optionale KI-Desktop **`sdd-desk`**, der im Gespräch durch den Prozess führt.

## Leitprinzip

> Es gibt **einen Regelkatalog** ([`catalog/catalog.yaml`](catalog/catalog.yaml)). Das [Handbuch](handbuch/03-mechanismen.md) erklärt ihn, das Programm setzt ihn durch. Beide sind Ausgaben derselben Quelle — und diese Quelle wird über `SDD-070` bis `SDD-073` maschinell geprüft.

`sdd` selbst ist **KI-frei und offline** ([ADR 0002](docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)): Kern, CLI, `sdd ui` und `sdd mcp` enthalten keinen LLM-Code und bauen keine Netzwerkverbindung nach außen auf.

## Stand

Letzte Version: **0.1.0** ([CHANGELOG](CHANGELOG.md)). Seitdem auf `main`: `SDD-003`, `SDD-040`, `sdd ui`, `sdd next`, `sdd explain`, `sdd mcp` und `sdd-desk`, auch als Tauri-v2-Anwendung.

- **Entscheidungen & Konzept:** [`docs/sdd_konzept.md`](docs/sdd_konzept.md), [`docs/sdd_process_3.md`](docs/sdd_process_3.md), ADRs unter [`docs/adr`](docs/adr)
- **Dogfooding:** [`sdd.config.yaml`](sdd.config.yaml), [`constitution.md`](constitution.md); das Werkzeug prüft sich selbst
- **Feature-Artefakte des Werkzeugs:** aktiv unter [`docs/prd`](docs/prd) und [`docs/design`](docs/design) (`sdd-rules-003-040`, `sdd-ui`, `sdd-guidance`, `sdd-desk`, `sdd-desk-tauri`), mit 0.1.0 archiviert unter [`docs/archive`](docs/archive)

### Regeln

| Bereich | Regeln |
|---|---|
| Verfassung | `SDD-001` vorhanden, `SDD-002a` Pflichtabschnitte, `SDD-002b` `.sdd`-Grammatik, `SDD-003` keine Feature-Inhalte (heuristisch) |
| ADR | `SDD-010` ADR bei Verfassungsänderung, `SDD-011` unveränderliche ADRs (`.sdd/adr-lock.json`) |
| Feature | `SDD-020` PRD und `SDD-030` Tech Spec ab Größenschwelle; Feature-Auflösung `--feature` → Frontmatter im Diff → Branch, fail-closed |
| PRD & Akzeptanzkriterien | `SDD-021` stabile Kennung `AC-<n>`, `SDD-023` Nicht-Ziele genannt, `SDD-022` in Tests referenziert (mit Feature-Slug, sobald es mehr als ein Feature gibt), `SDD-025` Kennung in Testdeklaration oder -rumpf, `SDD-031` Tech Spec nennt die umgesetzten AC-Kennungen |
| Umfang & Plan | `SDD-024` Diff weicht nicht von der erklärten Absicht (`touches`) ab, `SDD-040` Plan vor der ersten Task, `SDD-041` jede Task nennt ein Akzeptanzkriterium |
| Archiv | `SDD-050` abgeschlossene Features vor dem Release archiviert, `SDD-051` aktive Artefakte verweisen nicht auf archivierte |
| Selbstschutz | `SDD-070` – `SDD-073`: Katalog, Handbuchabschnitte, Prüfverträge, Laufzeit-Checks |

Jede Regel erklärt `sdd explain <regel-id>`. **Baseline & Ratchet:** gate-gebundene Baseline (`.sdd/baseline.json`) mit `content_hash`, deterministischem `ordinal` und Schutz für `baseline: forbidden`.

## Bauen

Voraussetzung: Go 1.27 (für `sdd-desk` zusätzlich Node.js und npm).

```text
go build -o bin/sdd ./cmd/sdd
```

## `sdd` — Kommandos

| Kommando | Zweck |
|---|---|
| `sdd lint [--json] [--gate <gate>]` | Befunde für das ganze Projekt |
| `sdd gate <pre-implementation\|pre-merge\|pre-release> [--json] [--feature <slug>[,…]] [--paths <globs>]` | Gate prüfen, Exit-Code `0`/`1`/`2` |
| `sdd baseline [--gate <gate>] [--prune] [--accept <regel-id>]` | Bestand einfrieren, quittieren (`--accept SDD-011` schreibt das ADR-Lockfile) |
| `sdd init [--profile minimal\|standard\|regulated] [--dry-run] [--no-ci]` | Neues Projekt: Konfiguration, Verfassung, Artefaktordner, CI-Workflow |
| `sdd new --feature <slug> [--type prd,design,plan] [--title …] [--touches …] [--archive-after YYYY-MM-DD]` | Feature-Artefakte aus Templates; `new --type adr\|doc --title …` |
| `sdd adopt [--profile …] [--apply] [--branch sdd/adopt]` | Bestandsprojekt übernehmen: Probelauf-Bericht; mit `--apply` auf neuem Branch, ein Commit je Schritt ([ADR 0001](docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md)) |
| `sdd report [--format markdown\|json] [--since <ref>]` | Reifegrad, Befunde, Gates, Schuldenstand mit Trend |
| `sdd next [--feature <slug>]` | nächster Prozessschritt je Feature |
| `sdd explain <regel-id>` | Regel, Begründung und Prüfvertrag |
| `sdd ui [--port N] [--open]` | lokale Oberfläche auf `127.0.0.1`: Übersicht, Artefakte, Befunde (live), Neues Artefakt, Übernahme, Handbuch |
| `sdd mcp [--allow-write]` | MCP-Server über stdio für Agenten ([Handbuch: Für Agenten](handbuch/04-agenten.md)) |

Artefaktpfade dürfen Ordner oder Globs sein, z. B. GitHub Spec Kit `specs/features/*/spec.md` ohne Umzug.

### CI

Die [GitHub Action](action.yml) baut `sdd`, führt `sdd gate <gate>` aus und schreibt den Bericht in die Job-Zusammenfassung:

```yaml
- uses: Zheng-Bote/sdd_studio@v0.1.0
  with:
    gate: pre-merge
```

Vollständiges Beispiel im [Handbuch: CI-Einbindung](handbuch/03-mechanismen.md#ci-einbindung); `sdd init` legt den Workflow an.

### Agenten (MCP)

`sdd mcp` stellt zehn Werkzeuge (`sdd_next`, `sdd_lint`, `sdd_gate`, `sdd_report`, `sdd_explain`, `sdd_check_intent`, `sdd_context`, `sdd_trace`, `sdd_validate_draft`, `sdd_new`), Ressourcen (`sdd://constitution`, `sdd://catalog`, `sdd://handbook`, `sdd://artifact/<pfad>`) und geführte Prompts (`sdd-idea`, `sdd-techspec`, `sdd-adr`, `sdd-plan`, `sdd-review`) bereit. Einbindung, z. B. in Claude Code oder Claude Desktop:

```json
{ "mcpServers": { "sdd": { "command": "sdd", "args": ["mcp"] } } }
```

Kein Agent kann die Baseline ändern, Befunde quittieren, das ADR-Lockfile schreiben, `sdd adopt --apply` ausführen oder ein ADR auf `accepted` setzen — dafür gibt es kein Werkzeug.

## KI-Desktop `sdd-desk` (optional)

`sdd-desk` führt im Gespräch mit einem frei wählbaren Sprachmodell von der Produkt-Idee über PRD, Tech Spec, ADR und Plan bis zum Gate. Es ist ein **getrenntes Programm**: Werkzeuge erreicht es nur über einen selbst gestarteten `sdd mcp --allow-write`, jeder Aufruf braucht deine Erlaubnis.

### Als Desktop-Anwendung (Tauri v2)

Voraussetzung zusätzlich: Rust und die [Tauri-Systemabhängigkeiten](https://v2.tauri.app/start/prerequisites/).

```text
cd apps/desk && npm ci
npm run tauri:dev        # Oberfläche und Sidecars bauen, Anwendung starten
npm run tauri:build      # Bündel (deb, rpm, AppImage, dmg, msi …) unter src-tauri/target/release/bundle/
```

Die Anwendung bringt `sdd-desk` und `sdd` als Sidecars mit. Sie öffnet das Projekt aus dem Startargument, sonst das zuletzt genutzte, sonst einen Ordnerdialog; **Datei → Projekt öffnen …** (Strg+O) wechselt es. Die Modelle stehen in `llm.env` im Konfigurationsordner der Anwendung, unter Linux `~/.config/io.github.zhengbote.sdddesk/llm.env` (Rechte `0600`).

### Als lokale Web-Oberfläche: bauen

```text
cd apps/desk && npm ci && npm run build:desk && cd ../..   # Oberfläche bauen und in das Go-Programm einbetten
go build -o bin/sdd ./cmd/sdd
go build -o bin/sdd-desk ./cmd/sdd-desk
```

`npm run build:desk` muss vor `go build` laufen; die gebaute Oberfläche (`internal/desk/webdist`) ist nicht eingecheckt.

### Modelle einrichten

Für die Web-Oberfläche stehen die Anbieter in `config/llm.env` **neben der ausführbaren Datei**, beim Bau oben also in `bin/config/llm.env` — nie im Projekt:

```text
mkdir -p bin/config
cp config/llm.env.example bin/config/llm.env
chmod 600 bin/config/llm.env      # sdd-desk lehnt Dateien ab, die Gruppe oder andere lesen dürfen
```

Unterstützt: Ollama lokal, Ollama Cloud, OpenRouter, NVIDIA NIM, Claude und Gemini. Je Anbieter `<ANBIETER>_API_KEY`, optional `<ANBIETER>_ENDPOINT` und `<ANBIETER>_LLM_01…`; Prozessvariablen haben Vorrang. Ohne Datei steht nur lokales Ollama zur Verfügung (Modelle über `/api/tags`). Mit `--config <pfad>` lässt sich eine andere Datei angeben.

### Starten

Im Repository, das du bearbeiten willst:

```text
bin/sdd-desk --project . --open
```

| Option | Bedeutung |
|---|---|
| `--project <pfad>` | Projektverzeichnis mit `sdd.config.yaml` (Standard: `.`) |
| `--open` | Oberfläche im Standardbrowser öffnen |
| `--config <pfad>` | `llm.env` (Standard: `config/llm.env` neben `sdd-desk`) |
| `--sdd <pfad>` | `sdd`-Programm (Standard: neben `sdd-desk`, sonst `PATH`) |
| `--port <n>` | Port auf `127.0.0.1` (Standard: `0` = freier Port) |
| `--exit-with-stdin` | beenden, sobald stdin schließt (setzt die Desktop-Anwendung für ihren Sidecar) |

`sdd-desk` gibt Adresse und URL mit Sitzungstoken aus; ohne `--open` diese URL im Browser öffnen. Beenden mit Strg+C.

In der Oberfläche Anbieter und Modell wählen, links einen geführten Ablauf starten (z. B. **Idee → PRD**) oder frei fragen. Die Seitenleiste zeigt den Prozessstand aus `sdd next`.

### Was der Desktop darf

- **Nur die Werkzeuge von `sdd mcp`**, keine eigenen, kein Dateizugriff daran vorbei.
- **Jeder Aufruf braucht eine Freigabe:** einmal, für die Sitzung (nur lesende Werkzeuge) oder ablehnen. `sdd_new` mit `apply` zeigt vorher Dateien und Inhalt und lässt sich nur einzeln freigeben; bestehende Dateien ändert es nie.
- **Cloud-Anbieter erst nach Bestätigung**, dass Inhalte das Gerät verlassen. Inhalte werden **nicht geschwärzt**; mit lokalem Ollama bleibt alles auf dem Rechner.
- **Lokaler Server** nur auf `127.0.0.1`, `Host`-Prüfung, Sitzungstoken, Content-Security-Policy ohne Inline-Skripte; keine Telemetrie.
- **Desktop-Anwendung:** Das Fenster zeigt nur die eigene Startseite und den laufenden Sidecar, andere Web-Adressen öffnet der Standardbrowser; keine Seite erhält Tauri-Rechte. Stirbt die Anwendung, enden `sdd-desk` und `sdd mcp` mit ihr.

Entwicklung der Oberfläche mit Live-Reload: `bin/sdd-desk --port 47900` starten, dann `cd apps/desk && npm start` (leitet `/api` an `127.0.0.1:47900` weiter) und die ausgegebene `?token=…` an `http://localhost:4200/` anhängen. Details: [`apps/desk/README.md`](apps/desk/README.md), [Handbuch: KI-Desktop](handbuch/04-agenten.md#sdd-desk).

## Tests

```text
go test ./...
cd apps/desk && npm test
cd apps/desk/src-tauri && cargo test
go run ./cmd/sdd lint
```

## Nächste Schritte

- **Streaming der Modellantworten** in `sdd-desk`.
- **Entscheidung offen:** `SDD-060` (Widersprüche Verfassung ↔ Tech Spec) ist laut Konzept nur assistiert über ein lokales Sprachmodell möglich (§9.6) — Kandidat für `sdd-desk`, nicht für `sdd`.
