---
feature: sdd-rules-v1-1
status: active
archive-after:
touches:
  - catalog/**
  - internal/sddcore/**
  - handbuch/**
  - cmd/**
---

# Tech Spec — Regeln des v1.1-Schnitts (`sdd-rules-v1-1`)

Setzt das PRD `docs/prd/sdd-rules-v1-1.md` um. Jede Regel erhält Katalogeintrag, Handbuchabschnitt und Prüfvertrag im selben Commit wie ihre Prüffunktion (`SDD-070` bis `SDD-073`). Maßgeblich für jede Regel ist ihr Prüfvertrag in `handbuch/03-mechanismen.md`.

## Architektur

```text
internal/sddcore/rules
  ├── traceability.go   SDD-023, SDD-031, SDD-041, SDD-051 (Textregeln über Feature-Artefakte)
  ├── adr.go            SDD-010, SDD-011 (Verfassung im Diff, ADR-Lockfile)
  ├── scope.go          SDD-024 (Diff gegen touches)
  └── testdecl.go       SDD-025 (AC in Testdeklarationen)
internal/sddcore/adrlock   .sdd/adr-lock.json lesen, Hash bilden, schreiben (nur über sdd baseline)
```

## Schnittstellen

- Keine neuen Befehle. `sdd baseline` pflegt zusätzlich `.sdd/adr-lock.json`; `sdd baseline --accept SDD-011` übernimmt geänderte ADRs.
- Die Git-Regeln nutzen die vorhandene lesende Git-Schnittstelle (`ChangedFiles`, `ShowFile`).

## Datenmodell

| Regel | Schwere | Profil | Gates | Baseline |
|---|---|---|---|---|
| SDD-023 | warn | standard | pre-merge | allowed |
| SDD-031 | warn | standard | pre-merge | allowed |
| SDD-041 | error | standard | pre-merge | allowed |
| SDD-051 | warn | standard | pre-merge, pre-release | forbidden |
| SDD-010 | error | minimal | pre-implementation, pre-merge | allowed |
| SDD-011 | error | standard | pre-merge | allowed |
| SDD-024 | warn | standard | pre-merge | allowed |
| SDD-025 | warn | standard | pre-merge | allowed |
| SDD-002b | error | minimal | pre-implementation, pre-merge | allowed |

**ADR-Lockfile** (`.sdd/adr-lock.json`): `{"version": 1, "entries": {"<pfad>": "<sha256>"}}`. Der Hash läuft über den normalisierten Text **ohne Frontmatter** (Zeilenenden vereinheitlicht, Zeilen nur aus Leerraum entfernt, Leerraumfolgen zu einem Leerzeichen). Frontmatter bleibt änderbar, weil Statuswechsel (`proposed` → `accepted` → `superseded`) und neue `references` zum Lebenszyklus gehören. Gelockt werden nur entschiedene ADRs (`status` nicht `proposed`); gelöschte bleiben im Lockfile.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `checkSDD023` | `TestSDD023` |
| AC-2 | `checkSDD031` | `TestSDD031` |
| AC-3 | `checkSDD041` | `TestSDD041` |
| AC-4 | `checkSDD051` | `TestSDD051` |
| AC-5 | `checkSDD010` | `TestSDD010` |
| AC-6 | `checkSDD011`, `adrlock` | `TestSDD011` |
| AC-7 | `checkSDD024` | `TestSDD024` |
| AC-8 | `checkSDD025` | `TestSDD025` |
| AC-9 | `checkSDD002b`, `config.ConstitutionSections` | `TestSDD002b` |

## Sicherheit

- Alle Regeln lesen ausschließlich; Links in `SDD-051` werden nur aufgelöst, wenn sie im Repository bleiben, und nie verfolgt, wenn sie ein Schema (`https:`, `mailto:` …) tragen.
- Das Lockfile wird nur von `sdd baseline` geschrieben, nie von `lint` oder `gate`.

## Offene Punkte

- Die Standardliste der `.sdd`-Section-Labels ist aus `sdd_process_3` §10 abgeleitet; ein echtes `.sdd`-Projekt legt seine Grammatik über `constitution_sections` fest.
