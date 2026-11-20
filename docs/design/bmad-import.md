---
feature: bmad-import
status: active
archive-after:
touches:
  - internal/sddcore/config/**
  - internal/sddcore/scan/**
  - internal/sddcore/adopt/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/sddcore/rules/**
  - internal/mcp/**
  - apps/desk/src/**
  - catalog/**
  - templates/**
  - handbuch/**
  - README.md
  - CHANGELOG.md
---

# Tech Spec — BMAD-Bestand übernehmen und als Strecke führen

Setzt das PRD `docs/prd/bmad-import.md` um, auf Grundlage von ADR 0009.

## Architektur

```text
internal/sddcore/config/config.go        pipeline, artifacts.initiative, InitiativePath()
internal/sddcore/scan/frontmatter.go     type, parent, covers, refined, unplanned
internal/sddcore/adopt/inventory.go      detectBMAD, KindInitiative/KindEpic, BMAD-Zweig in propose()
internal/sddcore/adopt/plan.go           Makro-Artefakte bleiben unberührt (nur über IsFeature)
internal/sddcore/scaffold/init.go        InitOptions.Pipeline, initiative.md.tmpl
internal/sddcore/scaffold/new.go         TypeInitiative, TypeEpic
internal/sddcore/guide/next.go           Guide.Initiative, Epic mit refined: false
internal/sddcore/rules/bmad.go           SDD-080, SDD-081, SDD-082
internal/mcp/prompts.go                  sdd-ingest-bmad, sdd-architecture, sdd-epic-refine
apps/desk/src/app/process/process-panel.ts   Etagen „Vorhaben“ und „Features“
catalog/catalog.yaml                     Regeln 1.3.0
handbuch/03-mechanismen.md               Abschnitt und Prüfverträge
```

### Konfiguration (AC-1)

`pipeline` ist ein Feld der Projektkonfiguration mit drei Werten:

| Wert | Bedeutung |
|---|---|
| `""` (fehlt) | bisheriges Verhalten; keine Makro-Prüfungen |
| `sdd` | wie `""`, aber ausdrücklich gewählt |
| `bmad` | Makro-Ebene vorhanden; `SDD-080`–`082` prüfen |

`artifacts.initiative` ist der Pfad der Initiative, Vorgabe `docs/initiative.md`
(`config.DefaultInitiative`). `ArtifactsConfig.InitiativePath()` liefert den
konfigurierten Wert oder die Vorgabe, analog zu `ArchitecturePath()`. Ein
unbekannter `pipeline`-Wert bricht `Validate()` mit Nennung der erlaubten Werte
ab — dieselbe Stelle, die schon das Profil prüft.

### Frontmatter (AC-2)

`scan.Frontmatter` erhält `Type`, `Parent`, `Covers []string`, `Refined *bool`,
`Unplanned []string`. `Refined` ist ein Zeiger, weil „fehlt“ und „false“
verschieden sind: nur ein ausdrückliches `refined: false` markiert ein noch
nicht verfeinertes Epic. Die bestehenden Felder und die Regelprüfungen bleiben
unverändert; unbekannte Felder im Frontmatter stören nicht, weil `yaml.v3`
ohne `KnownFields` dekodiert.

### Erkennung in `sdd adopt` (AC-3, AC-4)

`detectBMAD(root)` ist zustandslos und liest nur Verzeichnisse und Dateinamen:

1. BMAD liegt vor, wenn `_bmad/` oder `_bmad-output/` ein Verzeichnis ist.
2. Initiative: die erste Datei, die `**_bmad-output/**/initiative-*.md` erfüllt.
3. Epics: jede Datei `**_bmad-output/**/tickets/**/epic-*.md`.
4. Ein Index `**_bmad-output/**/tickets/tickets.toml` gilt als vorhanden, wenn er
   existiert; gelesen wird er nur von `SDD-082`.

`Classify` prüft die Makro-Muster **vor** den ADR- und Feature-Pfaden:

- der konfigurierte Pfad `artifacts.initiative` → `KindInitiative`
- `initiative-*.md` unter `_bmad-output` → `KindInitiative`
- `epic-*.md` unter `_bmad-output` → `KindEpic`
- alles andere unter `_bmad-output` (Berichte, Forge, Memlogs) → `KindDoc`

`Kind.IsFeature()` bleibt unverändert `PRD|Design|Plan`. Damit überspringt
`frontmatterStep` Makro-Artefakte von selbst: kein `feature:`-Frontmatter, kein
Konflikt (AC-4).

`Take` ergänzt den Ausschluss um die Werkzeugbäume, sobald BMAD erkannt ist:
`_bmad/**`, `**/.memlog.md` und die Skill-Spiegel der Agenten-CLIs
(`.agent/**`, `.agents/**`, `.claude/**`, `.codewhale/**`, `.cursor/**`). Ohne
`_bmad/` oder `_bmad-output/` bleibt das Inventar wie bisher (AC-3, letzter
Satz). `propose()` setzt in einem BMAD-Bestand `pipeline: bmad` und übernimmt
die Ausschlüsse in die vorgeschlagene Konfiguration; `artifacts.initiative`
bleibt auf der Vorgabe `docs/initiative.md`. Damit ist der Ledger Quelle und
das Artefakt entsteht bei der Übernahme — der Bericht nennt den gefundenen Pfad.
Die Epic-Dateien bleiben, wo sie liegen: der
Ledger von BMAD wird nicht umgeschrieben, und `artifacts.prd` zeigt weiter auf
`docs/prd` — dort entstehen die Feature-Artefakte. Ein Glob mit zwei
`*`-Segmenten wäre von `scan.TargetPath` nicht beschreibbar.

### Gerüst (AC-5)

- `templates/initiative.md.tmpl`: Frontmatter `type: initiative`, Abschnitte
  Description, Outcome, Requirements (`R<n>`), Constraints, Done when,
  Boundaries.
- `templates/epic.md.tmpl`: Frontmatter `type: epic`, `parent`,
  `covers: []`, `refined: false`; Abschnitte Description, Outcome, Done when,
  Boundaries, References.
- `scaffold.TypeInitiative`/`TypeEpic` in `otherPlan`: die Initiative geht nach
  `Artifacts.InitiativePath()`, ein Epic nach `scan.TargetPath(artifacts.prd, slug)`.
  Beide brauchen `--title`; das Epic zusätzlich einen Slug (aus dem Titel, wenn
  keiner genannt ist).
- `InitOptions.Pipeline`; ist er `bmad`, schreibt `InitPlan` die Initiative aus
  dem Template zusätzlich zu Konfiguration und Verfassung.

### Prozesssicht (AC-6, AC-7)

`Guide.Initiative *Step` folgt dem Muster von `Constitution` und `Architecture`:
steht die Datei nicht, ist der Schritt offen (`sdd new --type initiative`);
enthält sie keine `R<n>`-Kennung, ist er offen mit diesem Grund; sonst ist er
erledigt und nennt die Zahl der Anforderungen. Nur bei `pipeline: bmad` oder
vorhandener Initiativdatei gesetzt.

`featureGuide` bricht früh ab, wenn das PRD ein Epic mit `refined: false` ist:
ein einziger Schritt „Epic verfeinern“ (offen) mit Verweis auf den Ablauf
`sdd-epic-refine`. Damit erscheint weder „PRD anlegen“ noch „Tests je
Kriterium“ — ein unverfeinertes Epic ist kein unfertiges Feature.

Die Seitenleiste des Desktops bekommt vor „Prozessstand“ den Block „Vorhaben“
(Initiative und die drei bisherigen Projekt-Schritte) und beschriftet die
Feature-Liste als „Features“.

### Regeln (AC-8)

Alle drei laufen nur bei `pipeline: bmad`; sonst liefern sie keine Befunde. Damit
bleibt jedes andere Projekt unberührt, und `SDD-072` sieht sie trotzdem als
gate-aktiv.

| Regel | Prüfname | Schwere | Gate | Inhalt |
|---|---|---|---|---|
| `SDD-080` | `epic_requirement_anchor` | `error` | pre-implementation, pre-merge | Jedes Epic nennt in `covers` mindestens ein `R<n>`, und jedes genannte existiert in der Initiative |
| `SDD-081` | `requirement_coverage` | `warn` | pre-merge, pre-release | Jedes `R<n>` der Initiative ist von einem Epic gedeckt oder im Frontmatter `unplanned` genannt |
| `SDD-082` | `ticket_index_consistent` | `warn` | pre-merge | `tickets.toml` und die Epic-Artefakte nennen dieselben Slugs |

Epics sind die Artefakte unter `artifacts.prd`, deren Frontmatter `type: epic`
trägt. Gibt es kein solches Artefakt, prüfen `SDD-080` und `SDD-081` nichts: vor
der Übernahme gibt es nichts zu prüfen, und eine Flut „jede Anforderung
uncovered“ wäre falsch. `SDD-082` liest den Index direkt von der Platte
(`_bmad-output/**/tickets/tickets.toml`), weil er kein Artefakt ist.

### Geführte Abläufe (AC-9)

Drei Prompts in `internal/mcp/prompts.go`, nach dem Muster von `sdd-adopt`:
lesen, vorschlagen, einzeln bestätigen lassen. Keiner von ihnen schreibt ohne
`apply`, und die kritischen Werkzeuge (`sdd_config`, `sdd_baseline`) bleiben
kritisch (ADR 0003, ADR 0004).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.go` | `TestPipelineValues`, `TestInitiativePathDefault` |
| AC-2 | `scan/frontmatter.go` | `TestParseFrontmatterBMADFields` |
| AC-3 | `adopt/inventory.go` | `TestBMADInventory`, `TestBMADAbsentUnchanged` |
| AC-4 | `adopt/plan.go` | `TestBMADFrontmatterUntouched` |
| AC-5 | `scaffold/init.go`, `new.go`, `cmd/sdd/scaffold.go` | `TestInitBMAD`, `TestNewInitiativeAndEpic` |
| AC-6 | `guide/next.go` | `TestNextInitiativeStep`, `TestNextInitiativeFromLedger`, `TestNextEpicUnrefined`, `TestNextWithoutBMADHasNoInitiative` |
| AC-7 | `process-panel.ts` | `process-panel.spec.ts` |
| AC-8 | `catalog.yaml`, `rules/bmad.go`, `handbuch/03-mechanismen.md` | `rules/bmad_test.go` |
| AC-9 | `mcp/prompts.go` | `TestBMADPrompts`, `TestBMADSourceReadOnly` |
