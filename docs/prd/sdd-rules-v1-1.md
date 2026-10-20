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

# PRD — Regeln des v1.1-Schnitts (`sdd-rules-v1-1`)

## Problem

Der v1-Katalog prüft, ob Artefakte existieren und Akzeptanzkriterien Kennungen tragen. Er prüft nicht, ob die Artefakte zusammenpassen: PRDs ohne Nicht-Ziele, Tech Specs ohne Bezug zu den Kriterien, Tasks ohne Kriterium, aktive Spezifikationen, die auf archivierte verweisen, Verfassungsänderungen ohne ADR, nachträglich umgeschriebene ADRs und ein Diff, der über die erklärte Absicht hinauswächst.

## Ziel

Der Katalog setzt die Rückverfolgbarkeit Verfassung → ADR → PRD → Tech Spec → Plan → Test durch, wie sie das Konzept für v1.1 vorsieht (Abschnitt 15) — deterministisch, sprachneutral und mit `error` nur dort, wo ein Prüfvertrag ohne Heuristik existiert.

## Nutzer

- **Teams**, die Feature-Artefakte pflegen und Widersprüche früh sehen wollen.
- **Reviewer**, die Architekturänderungen nur mit begründetem ADR akzeptieren.

## Umfang

Die Regeln `SDD-023`, `SDD-031`, `SDD-041`, `SDD-051`, `SDD-010`, `SDD-011`, `SDD-024` und `SDD-025`, jeweils mit Katalogeintrag, Handbuchabschnitt, Prüfvertrag und Tests, sowie das ADR-Lockfile `.sdd/adr-lock.json`. `SDD-002b` folgt, sobald die Section-Labels der `.sdd`-Grammatik festgelegt sind.

## Nicht-Ziele

- Inhaltliche Bewertung von Texten (ob ein Nicht-Ziel sinnvoll ist, ob ein Test das Kriterium wirklich prüft).
- Sprachspezifische Parser; Testdeklarationen werden über Textmuster erkannt.
- Automatisches Reparieren von Befunden.

## Akzeptanzkriterien

- **AC-1** `SDD-023` meldet ein `warn` für jedes aktive PRD ohne Abschnitt „Nicht-Ziele“ (oder „Non-Goals“, „Out of Scope“) bzw. mit leerem Abschnitt.
- **AC-2** `SDD-031` meldet ein `warn` für jede Kennung des PRDs eines Features, die in keiner Tech Spec desselben Features vorkommt.
- **AC-3** `SDD-041` meldet einen `error` für jede Task (Checkbox-Zeile in einem Plan), die weder in ihrer Zeile noch in ihren Fortsetzungszeilen eine AC-Kennung enthält.
- **AC-4** `SDD-051` meldet ein `warn` für jeden Link eines aktiven Feature-Artefakts auf ein archiviertes Feature-Artefakt; ADRs, Verfassung und Doku dürfen auf Archiviertes verweisen.
- **AC-5** `SDD-010` meldet einen `error`, wenn der Diff die Verfassung über Formatierung hinaus ändert und kein ADR im selben Diff die Verfassung in `references` oder `adr` nennt.
- **AC-6** `SDD-011` meldet einen `error`, wenn sich der Text eines gelockten ADRs geändert hat, und ein `warn` für ein entschiedenes ADR, das noch nicht gelockt ist; nur `sdd baseline` schreibt das Lockfile, eine Abweichung nur mit `--accept SDD-011`.
- **AC-7** `SDD-024` meldet in `pre-merge` ein `warn` für jede geänderte Datei außerhalb der in `touches` erklärten Absicht der aufgelösten Features.
- **AC-8** `SDD-025` meldet ein `warn` für jede Kennung, die zwar in einer Testdatei, aber nicht im Bereich einer Testdeklaration vorkommt.
