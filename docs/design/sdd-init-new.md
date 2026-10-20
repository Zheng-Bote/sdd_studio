---
feature: sdd-init-new
status: active
archive-after:
touches:
  - cmd/**
  - internal/sddcore/scaffold/**
  - templates/**
  - embed.go
  - action.yml
  - .github/workflows/**
  - handbuch/**
---

# Tech Spec — Projektanlage, Artefakt-Templates und CI-Baustein (`sdd-init-new`)

Setzt das PRD [`docs/prd/sdd-init-new.md`](../prd/sdd-init-new.md) um und baut auf [`sdd-core-v1`](sdd-core-v1.md) auf.

## Architektur

```text
cmd/sdd                       init/new: Flags parsen, Plan ausgeben, Apply, Exit-Code
templates/*.tmpl              eingebettete Templates (text/template)
internal/sddcore/scaffold
  ├── plan.go                 Plan = geordnete Liste von Operationen; Apply schreibt additiv
  ├── init.go                 InitPlan: Konfiguration, Verfassung, Verzeichnisse, CI, Baseline
  └── new.go                  NewPlan: PRD, Tech Spec, Plan, ADR, Doku
action.yml                    Composite Action um `sdd gate`
.github/workflows/ci.yml      Dogfooding: go test und `sdd gate pre-merge` über die eigene Action
```

**Plan/Apply** ist der einzige Schreibpfad (Verfassung, Security 2): Ein Plan beschreibt Operationen (`mkdir`, `write`) mit repository-relativen Pfaden. `Apply` prüft jeden Pfad gegen die Repository-Wurzel, legt Verzeichnisse an und schreibt Dateien mit `O_CREATE|O_EXCL` — eine bestehende Datei wird nie überschrieben, sondern als `skipped` gemeldet (AC-2). `--dry-run` gibt nur den Plan aus.

## Schnittstellen

- `sdd init [--profile minimal|standard|regulated] [--dry-run] [--no-ci]` — Standardprofil `standard` (AC-1, AC-3).
- `sdd new --feature <slug> [--type prd,design,plan] [--title <text>] [--touches <globs>] [--archive-after YYYY-MM-DD] [--dry-run]` — ohne `--type` entstehen PRD und Tech Spec, also genau das Paar, das `SDD-020`/`SDD-030` verlangen (AC-4).
- `sdd new --type adr|doc --title <text> [--slug <slug>] [--dry-run]` — der Slug wird sonst aus dem Titel abgeleitet (AC-5).
- Exit-Codes: `0` ausgeführt (auch wenn alles übersprungen wurde), `2` Bedienfehler oder Schreibfehler (AC-6).
- `sdd new` ohne `sdd.config.yaml` nutzt die Standardpfade von `sdd init` und weist darauf hin (Konzept 8.3: `new` funktioniert mit einem Vorschlag).

## Datenmodell

| Typ | Ziel | Frontmatter |
|---|---|---|
| PRD | `<artifacts.prd>/<slug>.md` | `feature`, `status: draft`, `archive-after`, `touches` |
| Tech Spec | `<artifacts.design>/<slug>.md` | wie PRD |
| Plan | `<artifacts.plans>/<slug>.md` | `feature`, `status: draft`, `archive-after` |
| ADR | `<artifacts.adr>/<NNNN>-<slug>.md` | `status: proposed`, `date`, `supersedes`, `references` |
| Doku | `<artifacts.docs[0]>/<slug>.md` | `status: active` |

- `archive-after` ist ohne `--archive-after` leer (kein Stichtag) und sonst ein geprüftes ISO-8601-Datum. `touches` ist ohne `--touches` eine leere Liste; `pre-implementation` meldet dann ehrlich „Absicht nicht erklärt".
- „Existiert bereits" heißt für PRD, Tech Spec und Plan: Im Zielverzeichnis trägt eine Datei `feature: <slug>` — unabhängig vom Dateinamen. Für ADR und Doku entscheidet der Zielpfad.
- Die ADR-Nummer ist die größte vierstellige Präfixnummer im ADR-Verzeichnis plus eins.
- Das PRD-Template enthält ein Kriterium `AC-1` als Platzhalter; es erfüllt `SDD-021`, verlangt aber nach `SDD-022` einen Test (AC-6).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `scaffold.InitPlan`, `Apply` | `TestInitCreatesLintCleanProject` |
| AC-2 | `Apply` mit `O_EXCL`, Dry-Run in der CLI | `TestApplyNeverOverwrites`, `TestNewSkipsExistingFeatureArtifact` |
| AC-3 | `InitOptions.Profile`, Validierung | `TestInitRejectsUnknownProfile` |
| AC-4 | `scaffold.NewPlan` für `prd`/`design` | `TestNewFeatureCreatesPRDAndTechSpec`, `TestNewSkipsExistingFeatureArtifact` |
| AC-5 | `scaffold.NewPlan` für `plan`/`adr`/`doc` | `TestNewPlanADRDoc` |
| AC-6 | Templates, Slug-/Datums-/Pfadprüfung | `TestNewArtifactsPassRules`, `TestNewRejectsInvalidInput` |
| AC-7 | `action.yml` | `TestActionPassesInputsViaEnv` |

## Sicherheit

- Alle Zielpfade werden vor dem Schreiben normalisiert; absolute Pfade und `..`-Ausbrüche führen zu Exit `2`, ohne dass etwas geschrieben wird.
- Die Action reicht Eingaben ausschließlich über `env:` an das Skript weiter, nie per `${{ }}`-Interpolation im Skripttext (Schutz vor Skript-Injection über Branch- oder Eingabewerte).
- Die Action baut `sdd` aus ihrem eigenen Checkout (`github.action_path`) und lädt keine vorgebauten Binärdateien.

## Offene Punkte

- Bis zum ersten Release-Tag verweist der erzeugte CI-Workflow auf `Zheng-Bote/sdd_studio@main`; danach soll er auf den Tag zeigen.
