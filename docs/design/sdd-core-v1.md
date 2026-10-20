---
feature: sdd-core-v1
status: active
touches:
  - cmd/**
  - catalog/**
  - handbuch/**
  - internal/sddcore/**
  - embed.go
  - go.mod
  - sdd.config.yaml
  - constitution.md
---

# Tech Spec — SDD-Prüfkern v1 (`sdd-core-v1`)

Setzt das PRD [`docs/prd/sdd-core-v1.md`](../prd/sdd-core-v1.md) um. Maßgeblich bleiben die [Verfassung](../../constitution.md) und das [Konzept](../sdd_konzept.md); diese Tech Spec präzisiert sie, lockert sie aber nicht.

## Architektur

```text
cmd/sdd                  CLI: Flag-Parsing, Ausgabe, Exit-Codes 0/1/2
embed.go                 eingebetteter Katalog und Handbuch (embed.FS)
internal/sddcore         RunCheck: lädt Konfiguration, Katalog, Baseline; wertet aus
  ├── config             sdd.config.yaml laden und validieren
  ├── catalog            catalog/catalog.yaml (Arbeitsbaum vor eingebetteter Fassung)
  ├── scan               Globs, Frontmatter, Artefakt-Inventar, Schwelle, Git-CLI
  ├── rules              Check-Registry, Eff(g), Runner, alle Prüffunktionen
  ├── baseline           .sdd/baseline.json: Aufnahme, Abgleich, Prune, Accept
  └── report             Befundformat v1, Sortierung, ordinal, Text/JSON
```

Abhängigkeiten zeigen nur nach unten: `cmd/sdd` → `sddcore` → `rules` → `scan`/`catalog`/`config`/`report`. `scan` kennt keine Regeln.

## Schnittstellen

- **CLI:** `sdd lint [--json] [--gate <g>]`, `sdd gate <g> [--json] [--feature <slug>[,…]] [--paths <globs>]`, `sdd baseline [--gate <g>] [--prune] [--accept <rule_id>]`.
- **Exit-Codes:** `0` keine neuen `error`-Befunde, `1` mindestens ein neuer `error`, `2` Ausführungsfehler (Konfiguration fehlt oder ungültig, Git nicht verfügbar, Baseline nicht lesbar). Bezug: AC-1.
- **Berichtsformat v1 (JSON):** `schema_version`, `tool_version`, `catalog_version`, `profile`, `gate`, `feature`, `baseline`, `findings[]` mit `rule_id`, `severity`, `status`, `path`, `line`, `ordinal`, `message`, `handbook_url`, sowie `summary`.
- **Prüffunktion:** `func(ctx *rules.Context, rule catalog.Rule) ([]report.Finding, error)`, registriert unter dem `check`-Namen des Katalogs. Ein zurückgegebener Fehler ist ein Ausführungsfehler (Exit `2`), kein Befund.

## Datenmodell

- **Katalog:** je Regel `id`, `title`, `severity` (`default` und/oder `by_gate`), `profile`, `check`, `gate_default`, `baseline`, `contract`, `rationale`, `handbook`. `Eff(g)` = `gates.<g>.override` aus der Konfiguration, sonst `gate_default` (AC-2).
- **Baseline:** Einträge je Gate mit `rule_id`, `path`, `ordinal` und `content_hash`; ein geänderter Text hebt die Unterdrückung auf (AC-4).
- **Feature-Frontmatter:** `feature`, `status`, `archive-after` (ISO-8601), `touches` (Globs der erklärten Änderungsabsicht).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.Load` liefert `ErrConfigNotFound`; CLI bildet ihn auf Exit `2` ab | `TestMissingConfigReturnsErrConfigNotFound` |
| AC-2 | `rules/selfcheck.go`: `SDD-070` bis `SDD-073` | `TestSelfChecksDetectViolations` |
| AC-3 | Dogfooding-Lauf über das Repository | `TestSelfDogfoodingClean` |
| AC-4 | `baseline.BuildEntriesFromFindings`, `baseline.Apply` | `TestBaselineRespectsForbiddenAndOrdinal` |
| AC-5 | `rules/constitution.go`: `checkSDD001` | `TestSDD001` |
| AC-6 | `rules/constitution.go`: `checkSDD002a` | `TestSDD002a` |
| AC-7 | `rules/feature.go`: Feature-Auflösung, Schwelle, `checkFeatureArtifact` | `TestSDD020`, `TestSDD030UsesDesignDirectory` |
| AC-8 | `SDD-021` (ausstehend) | — |
| AC-9 | `SDD-022` (ausstehend) | — |
| AC-10 | `SDD-050` (ausstehend) | — |

## Sicherheit

- Keine Netzwerkzugriffe; Git wird ausschließlich lesend über die lokale CLI aufgerufen (`rev-parse`, `diff`).
- Konfigurierte Pfade werden relativ zum Repository aufgelöst; Pfade außerhalb der Wurzel sind ein Befund (`SDD-001`), nie ein Dateizugriff.
- Schreibend arbeitet in v1 nur `sdd baseline`, und zwar ausschließlich auf `.sdd/baseline.json`.

## Offene Punkte

- Die Erkennung von Akzeptanzkriterien (`SDD-021`) zählt jede Aufzählungszeile unter einer Überschrift „Akzeptanzkriterien“; ob das in Bestandsprojekten zu viele Befunde erzeugt, zeigt erst der Pilot `md-indexer`.
