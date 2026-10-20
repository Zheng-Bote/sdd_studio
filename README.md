# SDD Studio (`sdd`)

Sprachneutrales CLI-Werkzeug (`sdd`) und Go-Kernbibliothek (`internal/sddcore`) zur maschinellen Durchsetzung von **Spec-Driven Development** (`sdd_process_3`) über einen gemeinsamen, selbstprüfenden Regelkatalog und ein eingebettetes Handbuch.

## Leitprinzip

> Es gibt **einen Regelkatalog** ([`catalog/catalog.yaml`](catalog/catalog.yaml)). Das [Handbuch](handbuch/03-mechanismen.md) erklärt ihn, das Programm setzt ihn durch. Beide sind Ausgaben derselben Quelle — und diese Quelle wird über `SDD-070` bis `SDD-073` maschinell geprüft.

## Aktueller Stand (Roadmap Schritt 1 und 2)

- **Entscheidungen & Konzept:** [`docs/sdd_konzept.md`](docs/sdd_konzept.md) und [`docs/sdd_process_3.md`](docs/sdd_process_3.md)
- **Dogfooding-Konfiguration & Verfassung:** [`sdd.config.yaml`](sdd.config.yaml), [`constitution.md`](constitution.md)
- **Selbstschutz-Regeln (`SDD-070` – `SDD-073`):** Katalog, Handbuchabschnitte, Prüfverträge und Laufzeit-Checks aktiv
- **Verfassungsregeln `SDD-001`, `SDD-002a`:** Verfassung vorhanden und mit Pflichtabschnitten (Phase 1C)
- **Feature-Regeln `SDD-020`, `SDD-030`:** PRD und Tech Spec bei Schwellenüberschreitung, mit Feature-Auflösung (`--feature`, Frontmatter im Diff, Branch-Name) und fail-closed-Verhalten (Phase 1C)
- **AC-Regeln `SDD-021`, `SDD-022`:** Akzeptanzkriterien in PRDs tragen eine stabile Kennung `AC-<n>` und sind in Testdateien referenziert (Phase 1C)
- **Archivregel `SDD-050`:** abgeschlossene Features liegen vor dem Release im Archiv (Phase 1C)
- **Baseline & Ratchet:** Gate-gebundene Baseline (`.sdd/baseline.json`) mit `content_hash`, deterministischem `ordinal` und Schutz für `baseline: forbidden`
- **CLI-Kommandos:**
  - `go run ./cmd/sdd lint [--json] [--gate <gate>]`
  - `go run ./cmd/sdd gate <pre-implementation|pre-merge|pre-release> [--json] [--feature <slug>[,<slug>…]] [--paths <globs>]`
  - `go run ./cmd/sdd baseline [--gate <gate>] [--prune] [--accept <rule_id>]`
  - `go run ./cmd/sdd init [--profile minimal|standard|regulated] [--dry-run] [--no-ci]`
  - `go run ./cmd/sdd new --feature <slug> [--type prd,design,plan] [--title …] [--touches …] [--archive-after YYYY-MM-DD]` bzw. `new --type adr|doc --title …`
- **GitHub Action:** [`action.yml`](action.yml) baut `sdd` und führt `sdd gate <gate>` aus (siehe [Handbuch: CI-Einbindung](handbuch/03-mechanismen.md#ci-einbindung))
- **Feature-Artefakte des Werkzeugs selbst:** `sdd-core-v1` und `sdd-init-new` unter [`docs/prd`](docs/prd) und [`docs/design`](docs/design)

## Nächste Schritte (beim nächsten Mal)

- **Schritt 3:** `sdd adopt` — Inventar, Reifegrad, Baseline und Probelauf für Bestandsprojekte (Pilot: `md-indexer`).
