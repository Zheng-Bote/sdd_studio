---
feature: archive-not-draft
status: active
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - handbuch/**
  - README.md
  - CHANGELOG.md
---

# Tech Spec — Kein Archiv für ein Feature im Entwurf

Setzt das PRD `docs/prd/archive-not-draft.md` um.

## Architektur

```text
internal/sddcore/guide/archive.go   Archive: vor der Testprüfung draftPRDs(root, cfg, feature); bei Treffern ErrInvalidInput
internal/mcp/tools.go               Beschreibung von sdd_archive nennt die Bedingung
handbuch/03-mechanismen.md          Abschnitt sdd archive, Liste „Abbruch“
handbuch/04-agenten.md              Werkzeugtabelle, Zeile sdd_archive
README.md                           Befehlstabelle; „Nächste Schritte“ ohne den erledigten Punkt
```

### `draftPRDs` (AC-1, AC-2, AC-3)

`scan.ListArtifacts(root, cfg.Artifacts.PRD, cfg.Exclude)`. Davon zählen die PRDs, die außerhalb von `artifacts.archive` liegen und zum Feature gehören: über `feature:` oder, wenn das Feld fehlt, über `scan.SlugInSpec`, genau wie `scaffold.ArchivePlan`. Von diesen sind die mit `Frontmatter.Status == "draft"` Entwürfe. Gibt es einen, bricht `guide.Archive` ab, bevor die Testverweise geprüft werden:

`%w: Feature <slug> nicht archivierbar: <pfad> hat status: draft — archiviert wird erst nach Umsetzung und Merge (status: implemented), vor dem Release`

`sdd archive` und `sdd_archive` rufen beide `guide.Archive` auf. Die CLI meldet die Abbruchbedingung wie die übrigen mit Exit-Code `2`, auch bei `--dry-run`. Das Werkzeug meldet sie als Werkzeugfehler, in der Vorschau wie mit `apply`.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `guide/archive.go` | `TestArchiveRefusesDraft` |
| AC-2 | `guide/archive.go` (über `sdd_archive`) | `TestArchiveToolRefusesDraft` |
| AC-3 | `guide/archive.go` | `TestArchiveRefusesDraft` |
| AC-4 | Handbuch, README | `TestArchiveDocsNameDraft` |
