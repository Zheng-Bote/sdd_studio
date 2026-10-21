---
feature: sdd-rules-003-040
status: archived
release: v0.2.0
archive-after:
touches:
  - catalog/**
  - internal/sddcore/rules/**
  - handbuch/**
---

# Tech Spec — Regeln SDD-003 und SDD-040 (`sdd-rules-003-040`)

Setzt das PRD `docs/archive/prd/sdd-rules-003-040.md` um. Maßgeblich sind die Prüfverträge im Handbuch.

## Architektur

```text
internal/sddcore/rules
  ├── constitution.go   checkSDD003 neben SDD-001/002a/002b
  └── traceability.go   checkSDD040 neben SDD-041 (gleiche Task-Erkennung)
```

## Schnittstellen

Keine neuen Befehle; beide Regeln laufen in `sdd lint` und `pre-merge`.

## Datenmodell

| Regel | Schwere | Profil | Gates | Baseline |
|---|---|---|---|---|
| SDD-003 | warn | minimal | pre-merge | allowed |
| SDD-040 | warn | standard | pre-merge | allowed |

**SDD-003, Muster** (je Zeile außerhalb von Codeblöcken und Frontmatter, erster Treffer zählt):

- User Story: `Als <…> möchte ich` bzw. `As a/an <…> I want` (Groß-/Kleinschreibung egal);
- Task: Checkbox-Zeile wie bei `SDD-041`;
- Akzeptanzkriterium: Kennung `AC-<n>`;
- Planung: Überschrift mit `Sprint`, `Backlog`, `Roadmap`, `Meilenstein`, `Milestone`, `User Stor…` (als Wortanfang).

Für eine `.sdd`-Verfassung gelten dieselben Muster; Kommentarzeilen (`#`, `//`) zählen dort nicht als Überschrift.

**SDD-040, Auslegung von „Plan“:** Der Plan vor den Tasks ist das technische Vorgehen, also die Tech Spec (in Spec Kit `plan.md`). Ein Feature mit Tasks in `artifacts.plans` (Spec Kit `tasks.md`) braucht daher eine aktive Tech Spec mit demselben `feature:`. Liegen Tasks und Vorgehen in einer Datei unter `docs/plans`, gilt dieselbe Anforderung.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `checkSDD003` | `TestSDD003` |
| AC-2 | `checkSDD040` | `TestSDD040` |

## Sicherheit

Beide Regeln lesen nur Dateien innerhalb des Repositorys.

## Offene Punkte

- Ob die Planungs-Stichworte von `SDD-003` in der Praxis zu viele Fehlalarme erzeugen (etwa eine Verfassungsregel „Die Roadmap liegt im Issue-Tracker“), zeigt der Gebrauch; es ist bewusst nur ein `warn`.
