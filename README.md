# SDD Studio (`sdd`)

Sprachneutrales CLI-Werkzeug (`sdd`) und Go-Kernbibliothek (`internal/sddcore`) zur maschinellen Durchsetzung von **Spec-Driven Development** (`sdd_process_3`) über einen gemeinsamen, selbstprüfenden Regelkatalog und ein eingebettetes Handbuch.

## Leitprinzip

> Es gibt **einen Regelkatalog** ([`catalog/catalog.yaml`](catalog/catalog.yaml)). Das [Handbuch](handbuch/03-mechanismen.md) erklärt ihn, das Programm setzt ihn durch. Beide sind Ausgaben derselben Quelle — und diese Quelle wird über `SDD-070` bis `SDD-073` maschinell geprüft.

## Aktueller Stand — Version 0.1.0

Siehe [CHANGELOG](CHANGELOG.md). In CI: `uses: Zheng-Bote/sdd_studio@v0.1.0` (Beispiel im [Handbuch](handbuch/03-mechanismen.md#ci-einbindung)).


- **Entscheidungen & Konzept:** [`docs/sdd_konzept.md`](docs/sdd_konzept.md) und [`docs/sdd_process_3.md`](docs/sdd_process_3.md)
- **Dogfooding-Konfiguration & Verfassung:** [`sdd.config.yaml`](sdd.config.yaml), [`constitution.md`](constitution.md)
- **Selbstschutz-Regeln (`SDD-070` – `SDD-073`):** Katalog, Handbuchabschnitte, Prüfverträge und Laufzeit-Checks aktiv
- **Verfassungsregeln `SDD-001`, `SDD-002a`:** Verfassung vorhanden und mit Pflichtabschnitten (Phase 1C)
- **Feature-Regeln `SDD-020`, `SDD-030`:** PRD und Tech Spec bei Schwellenüberschreitung, mit Feature-Auflösung (`--feature`, Frontmatter im Diff, Branch-Name) und fail-closed-Verhalten (Phase 1C)
- **AC-Regeln `SDD-021`, `SDD-022`:** Akzeptanzkriterien in PRDs tragen eine stabile Kennung `AC-<n>` und sind in Testdateien referenziert (Phase 1C)
- **Archivregel `SDD-050`:** abgeschlossene Features liegen vor dem Release im Archiv (Phase 1C)
- **Seit 0.1.0:** `SDD-003` (Verfassung ohne Feature-Inhalte, heuristisch), `SDD-040` (technischer Plan vor den Tasks); AC-Verweise nennen den Feature-Slug, sobald es mehr als ein Feature gibt
- **Regeln aus v1.1:** `SDD-002b` (`.sdd`-Grammatik), `SDD-010`/`SDD-011` (ADR bei Verfassungsänderung, unveränderliche ADRs mit `.sdd/adr-lock.json`), `SDD-023`, `SDD-024`, `SDD-025`, `SDD-031`, `SDD-041`, `SDD-051`
- **Baseline & Ratchet:** Gate-gebundene Baseline (`.sdd/baseline.json`) mit `content_hash`, deterministischem `ordinal` und Schutz für `baseline: forbidden`
- **CLI-Kommandos:**
  - `go run ./cmd/sdd lint [--json] [--gate <gate>]`
  - `go run ./cmd/sdd gate <pre-implementation|pre-merge|pre-release> [--json] [--feature <slug>[,<slug>…]] [--paths <globs>]`
  - `go run ./cmd/sdd baseline [--gate <gate>] [--prune] [--accept <rule_id>]`
  - `go run ./cmd/sdd init [--profile minimal|standard|regulated] [--dry-run] [--no-ci]`
  - `go run ./cmd/sdd ui [--port N] [--open]` — lokale Oberfläche auf 127.0.0.1: Übersicht, Artefakte, Befunde (live), Neues Artefakt, Übernahme, Handbuch
  - `go run ./cmd/sdd report [--format markdown|json] [--since <ref>]` — Reifegrad, Befunde, Gates, Schuldenstand mit Trend; die Action schreibt ihn in die Job-Zusammenfassung
  - `go run ./cmd/sdd adopt [--profile …] [--apply] [--branch sdd/adopt]` — Bestandsprojekt übernehmen: Probelauf-Bericht, mit `--apply` auf neuem Branch, ein Commit je Schritt
  - `go run ./cmd/sdd new --feature <slug> [--type prd,design,plan] [--title …] [--touches …] [--archive-after YYYY-MM-DD]` bzw. `new --type adr|doc --title …`
- **GitHub Action:** [`action.yml`](action.yml) baut `sdd` und führt `sdd gate <gate>` aus (siehe [Handbuch: CI-Einbindung](handbuch/03-mechanismen.md#ci-einbindung))
- **Artefaktpfade als Ordner oder Glob:** z. B. Spec Kit `specs/features/*/spec.md` ohne Umzug
- **Feature-Artefakte des Werkzeugs selbst:** `sdd-core-v1`, `sdd-init-new`, `sdd-adopt`, `sdd-report` und `sdd-rules-v1-1`, mit Version 0.1.0 archiviert unter [`docs/archive`](docs/archive)

## Nächste Schritte (beim nächsten Mal)

- **Entscheidung offen:** `SDD-060` (Widersprüche Verfassung ↔ Tech Spec) ist laut Konzept nur assistiert über ein lokales Sprachmodell möglich (§9.6).
- **Roadmap Schritt 6:** Agenten-Schnittstelle (MCP).
