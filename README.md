# SDD Studio (`sdd`)

Sprachneutrales CLI-Werkzeug (`sdd`) und Go-Kernbibliothek (`internal/sddcore`) zur maschinellen Durchsetzung von **Spec-Driven Development** (`sdd_process_3`) über einen gemeinsamen, selbstprüfenden Regelkatalog und ein eingebettetes Handbuch.

## Leitprinzip

> Es gibt **einen Regelkatalog** ([`catalog/catalog.yaml`](catalog/catalog.yaml)). Das [Handbuch](handbuch/03-mechanismen.md) erklärt ihn, das Programm setzt ihn durch. Beide sind Ausgaben derselben Quelle — und diese Quelle wird über `SDD-070` bis `SDD-073` maschinell geprüft.

## Aktueller Stand (Roadmap Schritt 1 — Bootstrap)

- **Entscheidungen & Konzept:** [`docs/sdd_konzept.md`](docs/sdd_konzept.md) und [`docs/sdd_process_3.md`](docs/sdd_process_3.md)
- **Dogfooding-Konfiguration & Verfassung:** [`sdd.config.yaml`](sdd.config.yaml), [`constitution.md`](constitution.md)
- **Selbstschutz-Regeln (`SDD-070` – `SDD-073`):** Katalog, Handbuchabschnitte, Prüfverträge und Laufzeit-Checks aktiv
- **Baseline & Ratchet:** Gate-gebundene Baseline (`.sdd/baseline.json`) mit `content_hash`, deterministischem `ordinal` und Schutz für `baseline: forbidden`
- **CLI-Kommandos:**
  - `go run ./cmd/sdd lint [--json] [--gate <gate>]`
  - `go run ./cmd/sdd gate <pre-implementation|pre-merge|pre-release> [--json] [--feature <slug>] [--paths <globs>]`
  - `go run ./cmd/sdd baseline [--gate <gate>] [--prune] [--accept <rule_id>]`

## Nächste Schritte (beim nächsten Mal)

- **Phase 1C:** Verbleibende 7 Regeln des v1-Schnitts ergänzen (`SDD-001`, `SDD-002a`, `SDD-020`, `SDD-030`, `SDD-021`, `SDD-022`, `SDD-050`) — jeweils inklusive Handbuchabschnitt, Prüfvertrag und Tests im selben Commit.
- **Schritt 2:** `sdd init`, `sdd new` (Templates) und GitHub-Action-Wrapper.
