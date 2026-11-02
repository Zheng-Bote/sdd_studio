---
feature: sdd-archive
status: active
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
---

# Tech Spec — Features archivieren mit `sdd archive`

Setzt das PRD `docs/prd/sdd-archive.md` um, auf Grundlage von ADR 0003 (Archivieren gehört zur Stufe „schreibend“) und Handbuch „Archivierung (SDD-050)“.

## Architektur

```text
internal/sddcore/scaffold/plan.go     Op.MoveFrom + Expect: Ziel anlegen, dann Quelle entfernen
internal/sddcore/scaffold/archive.go  ArchivePlan(root, cfg, feature, release): Zielpfade, Frontmatter
internal/sddcore/guide/archive.go     Archive(root, feature, release): Vorbedingung „jedes AC getestet“, dann ArchivePlan
cmd/sdd/archive.go                    sdd archive --feature <slug> [--release <v>] [--dry-run]
internal/mcp/tools.go                 Werkzeug sdd_archive
internal/desk/agent.go                writeTools: sdd_archive
```

### `ArchivePlan`

1. `artifacts.archive` muss gesetzt und ein Ordner sein (kein Platzhalter). Sonst kommt `ErrInvalidInput`.
2. Für `prd`, `design` und `plans` liefert `scan.ListArtifacts` die Dateien der Typspezifikation. Eine Datei gehört zum Feature, wenn ihr `feature:` gleich dem Slug ist, oder, ohne `feature:`, wenn `scan.SlugInSpec` den Slug liefert. Dateien unter dem Archiv zählen nicht.
3. **Ziel:** Basis ist der Elternordner der Typspezifikation. Bei einem Ordner ist das der Ordner selbst (`docs/prd`), bei einem Glob der statische Präfix (`specs/features/` → `specs/features`). Das Ziel ist `<archive>/<Pfad relativ zum Elternordner dieser Basis>`: `docs/prd/x.md` → `docs/archive/prd/x.md`, `specs/features/x/spec.md` → `specs/archive/features/x/spec.md`.
4. **Frontmatter:** `setFrontmatter(content, "status", "archived")` und mit `release` auch `setFrontmatter(…, "release", v)`. Eine vorhandene Zeile `key:` im Frontmatter-Block wird ersetzt, sonst wird die Zeile vor dem schließenden `---` eingefügt. Der Rest bleibt byteweise gleich. Ohne Frontmatter-Block kommt `ErrInvalidInput`.
5. Je Datei entsteht eine `Op{Path: ziel, Content: neu, MoveFrom: quelle, Expect: alt}`. Keine Datei bedeutet `ErrInvalidInput` mit „keine aktiven Artefakte für Feature …“.

### `Op.MoveFrom`

- **Vorschau:** Weicht die Quelle von `Expect` ab, wird die Operation übersprungen („Datei seit der Planung verändert“). Existiert das Ziel, wird sie ebenfalls übersprungen.
- **Apply:** Erst wird das Ziel mit `O_EXCL` angelegt (Elternordner innerhalb der Wurzel wie bisher). Dann wird die Quelle nach erneutem Vergleich mit `Expect` entfernt. Der Status ist `moved`.
- Ein Fehler beim Entfernen ist ein Fehler. Das Ziel bleibt dann bestehen, nichts geht verloren.

### `guide.Archive`

`rules.Trace(feature)`: Jedes AC braucht mindestens einen Testverweis, sonst kommt `ErrInvalidInput` mit „nicht archivierbar: AC-1, AC-3 ohne Testverweis (SDD-022)“. Das gilt nur, wenn das Feature ein aktives PRD mit ACs hat; ohne PRD (nur Tech Spec oder Plan) entfällt die Prüfung. Danach folgt `ArchivePlan`. Außerdem werden alle `Skip` des Plans geprüft: Eine übersprungene Operation (Ziel existiert) ist für `archive` ein Fehler, damit nicht nur ein Teil verschoben wird.

### CLI und MCP

- **CLI:** `sdd archive` gibt je Operation `würde verschieben <quelle> -> <ziel>` bzw. `verschoben …` aus. Darunter kommen die neuen Frontmatter-Zeilen. Fehler führen zu Exit-Code `2`.
- **MCP:** `sdd_archive` liefert in der Vorschau `ops: [{path: ziel, action: "move", from: quelle, diff}]`. Der `diff` vergleicht alten und neuen Inhalt unter dem Zielpfad. Mit `apply` kommen `results` mit Status.
- **Desktop:** Die Freigabekarte zeigt `move` als „verschieben“ mit Quelle und Diff.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `archive.go`, `plan.go` (`MoveFrom`) | `TestArchivePlanMoves` |
| AC-2 | `archive.go`, `guide/archive.go` | `TestArchiveRefuses` |
| AC-3 | `cmd/sdd/archive.go`, Vorschau | `TestArchivePlanMoves` (Vorschau) |
| AC-4 | `tools.go`, `agent.go`, Freigabekarte | `TestArchiveTool`, `TestArchiveIsWrite`, `approval-card.spec.ts` |
| AC-5 | Zusammenspiel mit SDD-050/SDD-022 | `TestArchiveThenLint` |
