# Changelog

Alle nennenswerten Änderungen an SDD Studio (`sdd`). Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung [Semantic Versioning](https://semver.org/lang/de/). Der Regelkatalog trägt eine eigene Version (`catalog/catalog.yaml`).

## [0.1.0] — 2026-10-20

Erste Version. Regelkatalog `1.0.0` mit 20 Regeln, Berichtsformat `schema_version: 1`.

### Hinzugefügt

- **Prüfkern und CLI:** `sdd lint`, `sdd gate pre-implementation|pre-merge|pre-release` und `sdd baseline` mit stabilen Exit-Codes (`0`/`1`/`2`) und JSON-Bericht; gate-gebundene Baseline mit `content_hash` und deterministischem `ordinal`.
- **Regeln des v1-Schnitts:** `SDD-001`, `SDD-002a`, `SDD-020`, `SDD-021`, `SDD-022`, `SDD-030`, `SDD-050` sowie die Selbstprüfung `SDD-070` bis `SDD-073`.
- **Regeln des v1.1-Schnitts:** `SDD-002b`, `SDD-010`, `SDD-011` (mit `.sdd/adr-lock.json`), `SDD-023`, `SDD-024`, `SDD-025`, `SDD-031`, `SDD-041`, `SDD-051`.
- **Projektanlage:** `sdd init` (Konfiguration, Verfassung, Artefaktordner, CI-Workflow, leere Baseline) und `sdd new` (PRD, Tech Spec, Plan, ADR, Doku aus eingebetteten Templates).
- **Übernahme:** `sdd adopt` mit Inventar, Klassifikation in sechs Artefakttypen, Reifegrad L0–L4, Probelauf und `--apply` auf einem neuen Branch mit einem Commit je Schritt ([ADR 0001](docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md)).
- **Bericht:** `sdd report` mit Reifegrad, Gate-Status und Schuldenstand samt Trend gegenüber der Merge-Basis.
- **Artefaktpfade als Ordner oder Glob**, z. B. für GitHub Spec Kit (`specs/features/*/spec.md`).
- **GitHub Action** (`action.yml`): baut `sdd`, führt das Gate aus und schreibt den Bericht in die Job-Zusammenfassung.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.1.0`): `sdd-core-v1`, `sdd-init-new`, `sdd-adopt`, `sdd-report`, `sdd-rules-v1-1`.

[0.1.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.1.0
