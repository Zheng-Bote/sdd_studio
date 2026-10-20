---
feature: sdd-adopt
status: active
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - templates/**
  - handbuch/**
  - constitution.md
---

# Tech Spec — Übernahme bestehender Projekte (`sdd-adopt`)

Setzt das PRD `docs/prd/sdd-adopt.md` um. Die Verfassung hat Vorrang; für schreibenden Git-Zugriff gilt [ADR 0001](../adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md).

## Architektur

```text
internal/sddcore/scan      ArtifactSpec: Ordner oder Datei-Glob; ListArtifacts, InSpec, TargetPath
internal/sddcore           RunCheckWithConfig, BuildBaseline (aus cmd/sdd herausgelöst)
internal/sddcore/adopt
  ├── inventory.go         Markdown-Dateien, CI-Workflows, Layout-Erkennung, Konfigurationsvorschlag
  ├── classify.go          Artefakttyp je Datei mit Begründung
  ├── maturity.go          Kriterien L1–L4, Reifegrad, genau ein Vorschlag
  ├── plan.go              Schritte (Konfiguration, Verfassung, Frontmatter, Baseline, Bericht)
  ├── apply.go             Branch anlegen, Schritte schreiben, je Schritt ein Commit
  └── report.go            Adopt-Bericht (Markdown)
internal/sddcore/scan/gitwrite.go   GitWriter: IsClean, CurrentBranch, CreateBranch, Commit
```

## Schnittstellen

- `sdd adopt [--profile minimal|standard|regulated] [--apply] [--branch <name>]` — ohne `--apply` Probelauf (AC-4); Standardbranch `sdd/adopt`.
- Exit-Codes: `0` Bericht erstellt bzw. angewendet, `2` Bedien- oder Ausführungsfehler (kein Git-Repository, Arbeitsbaum nicht sauber, Branch existiert, Commit fehlgeschlagen).
- **Artefaktpfade (AC-1):** Enthält ein Wert Platzhalter, ist er ein Datei-Glob, sonst ein Ordner (rekursiv, `*.md`). `sdd new` leitet das Ziel aus einem Glob ab, wenn er genau ein `*` enthält, das ein ganzes Pfadsegment bildet und durch den Slug ersetzt wird (`specs/features/*/spec.md` → `specs/features/<slug>/spec.md`); andere Globs sind für `sdd new` ein Bedienfehler.

## Datenmodell

**Klassifikation (AC-2)**, erste passende Regel gewinnt:

1. die konfigurierte bzw. erkannte Verfassung → Verfassung;
2. Datei unter dem ADR-Ordner, Name `NNNN-*.md` → ADR;
3. trifft `artifacts.prd` / `design` / `plans` → PRD / Tech Spec / Plan;
4. trifft `artifacts.archive` → Feature-Artefakt im Archiv (Typ nach Dateiname, sonst PRD);
5. sonst → Doku/Exploration („kein Muster").

**Layout-Erkennung für den Konfigurationsvorschlag:** Existiert mindestens eine `specs/features/*/spec.md`, wird das Spec-Kit-Layout vorgeschlagen (`spec.md` → PRD, `plan.md` → Tech Spec, `tasks.md` → Plan, Archiv `specs/archive`), sonst die Standardpfade von `sdd init`. ADR-Ordner: der erste aus `docs/adr`, `docs/architecture/adr`, `doc/adr`, `adr`, der eine `NNNN-*.md` enthält. Verfassung: die erste vorhandene aus `constitution.md`, `CONSTITUTION.md`, `docs/constitution.md` oder eine einzelne `*.sdd` im Wurzelverzeichnis.

**Reifegrad (AC-3)** — der höchste Grad, dessen Kriterien und die aller niedrigeren erfüllt sind:

| Grad | Kriterien |
|---|---|
| L1 | Verfassung vorhanden, keine Befunde für `SDD-001`/`SDD-002a` |
| L2 | mindestens ein PRD und eine Tech Spec; mindestens eine AC-Kennung; keine Befunde für `SDD-021` |
| L3 | ein CI-Workflow ruft `sdd gate` auf; `.sdd/baseline.json` vorhanden |
| L4 | jedes Feature-Artefakt trägt `feature:` und `status`; `artifacts.archive` konfiguriert; keine Befunde für `SDD-050` |

Der Vorschlag ist das erste nicht erfüllte Kriterium der nächsten Stufe, mit Anzahl betroffener Dateien.

**Schritte und Commits (AC-5, AC-6):**

| Schritt | Inhalt | wenn |
|---|---|---|
| 1 Konfiguration | `sdd.config.yaml` aus dem Vorschlag | Datei fehlt |
| 2 Verfassung | Template aus `sdd init` | keine Verfassung gefunden |
| 3 Frontmatter | `feature: <slug>` und `status: active` als neuer Block vor dem Inhalt | Feature-Artefakt ohne Frontmatter |
| 4 Baseline | `.sdd/baseline.json` für alle Gates | Datei fehlt |
| 5 Bericht | `.sdd/adopt-report.md` | immer |

Schritte ohne Änderung erzeugen keinen Commit. Hat ein Feature-Artefakt bereits Frontmatter, aber kein `feature:`, wird es nicht angefasst, sondern im Bericht als Konflikt gemeldet. Der Slug ist das `*`-Segment des Globs bzw. der Dateiname ohne Endung, nötigenfalls normalisiert (Kleinschreibung, `[a-z0-9-]`). Der Status `active` ist eine Annahme; der Bericht weist darauf hin, dass längst umgesetzte Features auf `implemented` zu setzen sind.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `scan.ArtifactSpec`, `rules`, `scaffold.NewPlan` | `TestArtifactSpecGlob`, `TestRulesWithGlobSpecs`, `TestNewWithGlobSpec` |
| AC-2 | `adopt.Classify` | `TestClassify` |
| AC-3 | `adopt.Assess` | `TestMaturity` |
| AC-4 | `adopt.Plan`, Probelauf | `TestDryRunWritesNothing` |
| AC-5 | `adopt.Apply`, `scan.CLIGitWriter` | `TestApplyCommitsEachStep`, `TestApplyRefusesDirtyTreeAndExistingBranch` |
| AC-6 | Schritt Baseline, Archivposten im Bericht | `TestApplyCommitsEachStep` |
| AC-7 | `adopt.RenderReport` | `TestApplyCommitsEachStep`, `TestDryRunWritesNothing` |

## Sicherheit

- Schreibender Git-Zugriff nur gemäß ADR 0001; alle Git-Aufrufe ohne Shell, Argumente als Liste, Pfade nach `--`.
- Dateien werden ausschließlich über `scaffold.Plan.Apply` geschrieben; das Einfügen von Frontmatter ist eine eigene Operation, die nur greift, wenn die Datei unverändert dem gelesenen Stand entspricht und noch kein Frontmatter hat.

## Offene Punkte

- Ob das Kriterium „CI ruft `sdd gate` auf" als Textsuche in Workflow-Dateien genügt, zeigt erst ein Projekt mit anderer CI.
