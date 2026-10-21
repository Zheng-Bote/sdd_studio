# Changelog

Alle nennenswerten Änderungen an SDD Studio (`sdd`). Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung [Semantic Versioning](https://semver.org/lang/de/). Der Regelkatalog trägt eine eigene Version (`catalog/catalog.yaml`).

## [0.3.0] — 2026-10-21

Regelkatalog unverändert `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`.

### Hinzugefügt

- **„LLM testen“ in `sdd-desk`** (`POST /api/model/test`): schickt eine feste Testnachricht samt Werkzeugbeschreibungen an das gewählte Modell und meldet, ob es antwortet und ob es Werkzeuge annimmt — mit der Meldung des Anbieters, z. B. bei einem nicht mehr angebotenen Modell. Keine Gesprächs- oder Projektinhalte; bei Cloud-Anbietern erst nach dem Hinweis.
- **Eigenes Icon** der Desktop-Anwendung und der Web-Oberfläche von `sdd-desk`.

### Geändert

- Die Desktop-Anwendung bringt `sdd` als `sdd-studio-sdd` mit; ein Paket installiert kein zweites `/usr/bin/sdd` mehr. `sdd-desk` sucht zuerst `sdd-studio-sdd`, dann `sdd` neben sich, dann im `PATH`.
- Fehlt `llm.env`, nennt die Warnung in der Desktop-Anwendung den Konfigurationsordner der Anwendung statt eines Pfads neben dem Sidecar (`sdd-desk --config-hint`).
- Handbuch: Änderungen über der Größenschwelle gehören vor dem Release-Branch auf den Haupt-Branch, weil dort nach dem Archivieren kein Feature mehr auflösbar ist.
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.3.0`.

### Behoben

- Lokal ermittelte Merge-Basis: Nach dem Push eines Feature-Branches war `@{upstream}` dessen eigene entfernte Kopie und der Diff für `pre-merge` leer; diese Kopie gilt nicht mehr als Basis, stattdessen `main` bzw. `master`.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.3.0`): `sdd-desk-model-test`, `sdd-desk-config-hint`, `sdd-desk-sidecar-name`.

## [0.2.0] — 2026-10-21

Regelkatalog `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`.

### Hinzugefügt

- **Regeln:** `SDD-003` (Verfassung ohne Feature-Inhalte, heuristisch) und `SDD-040` (technischer Plan vor der ersten Task).
- **`sdd ui`:** lokale Oberfläche auf `127.0.0.1` mit Übersicht, Artefakten, Befunden (live), neuem Artefakt, Übernahme und Handbuch; `Host`-, `Origin`- und Token-Prüfung, Schreiben nur über Plan/Apply.
- **Prozessführung:** `sdd next` nennt je Feature den nächsten Prozessschritt; `sdd explain <regel-id>` zeigt Regel, Begründung und Prüfvertrag.
- **`sdd mcp`:** MCP-Server über stdio für Agenten mit zehn Werkzeugen, Ressourcen und geführten Prompts (Idee → PRD, Tech Spec, ADR, Plan, Review); ohne KI und offline, Schreiben nur mit `--allow-write` und nur neue Artefakte.
- **KI-Desktop `sdd-desk`** (optional, getrennt, [ADR 0002](docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)): führt im Gespräch mit einem wählbaren Sprachmodell (Ollama lokal/Cloud, OpenRouter, NVIDIA NIM, Claude, Gemini) durch den Prozess; Werkzeuge nur über `sdd mcp` und nur mit Freigabe, schreibende mit Vorschau; Hinweis vor Cloud-Anbietern; Schlüssel in `llm.env` mit Rechten `0600`; keine Telemetrie.
- **Desktop-Anwendung** (Tauri v2) mit `sdd-desk` und `sdd` als Sidecars: Projektwahl und -wechsel, Navigationsregel, keine Tauri-Rechte für Seiten; `sdd-desk --exit-with-stdin` endet mit der Anwendung.
- **Gestreamte Antworten** in `sdd-desk` (`POST /api/turn/stream`): Text und Gedankengang erscheinen, während das Modell arbeitet; Abbrechen beendet auch die Anfrage an den Anbieter.

### Geändert

- `SDD-022`/`SDD-025`: Testverweise nennen den Feature-Slug, sobald es mehr als ein Feature gibt (archivierte eingeschlossen).
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.2.0`.

### Behoben

- `sdd next` verlangte ein ADR für eine Verfassungsänderung, obwohl ein verlinktes, angenommenes ADR sie abdeckte und `SDD-010` im Gate nichts meldete.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.2.0`): `sdd-rules-003-040`, `sdd-ui`, `sdd-guidance`, `sdd-desk`, `sdd-desk-tauri`, `sdd-desk-streaming`.

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

[0.3.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.3.0
[0.2.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.2.0
[0.1.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.1.0
