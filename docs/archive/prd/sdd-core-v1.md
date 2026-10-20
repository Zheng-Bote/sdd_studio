---
feature: sdd-core-v1
status: archived
release: v0.1.0
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

# PRD — SDD-Prüfkern v1 (`sdd-core-v1`)

## Problem

Spec-Driven Development nach `sdd_process_3` lebt von Mechanismen statt Phasen. Ohne ein Werkzeug, das diese Mechanismen in CI und lokal durchsetzt, bleiben Verfassung, PRD, Tech Spec und Akzeptanzkriterien Behauptungen: Niemand merkt, wenn ein Feature die Größenschwelle überschreitet, ohne spezifiziert zu sein, oder wenn ein Akzeptanzkriterium nie getestet wird.

## Ziel

Ein sprachneutrales CLI `sdd` mit Go-Kernbibliothek prüft ein Repository gegen einen einzigen, selbstprüfenden Regelkatalog. Die Ergebnisse sind deterministisch, maschinenlesbar und über drei Gates (`pre-implementation`, `pre-merge`, `pre-release`) in CI einsetzbar. Bestandsprojekte lassen sich über eine gate-gebundene Baseline schrittweise heben.

## Nutzer

- **Entwicklerinnen und Entwickler**, die vor und während der Implementierung wissen wollen, welche Artefakte fehlen.
- **CI-Pipelines**, die über Exit-Codes Merges und Releases blockieren.
- **LLM-Agenten**, die Befunde als JSON lesen und Handbuchverweisen folgen.

## Umfang

Prüfkern (`internal/sddcore`), CLI-Befehle `lint`, `gate` und `baseline`, Regelkatalog mit eingebettetem Handbuch sowie die Regeln des v1-Schnitts (Konzept, Abschnitt 15): `SDD-001`, `SDD-002a`, `SDD-020`, `SDD-021`, `SDD-022`, `SDD-030`, `SDD-050` und `SDD-070` bis `SDD-073`.

## Nicht-Ziele

- `sdd init`, `sdd new` und der GitHub-Action-Wrapper — eigenes Feature in Roadmap-Schritt 2.
- `sdd adopt`, `sdd report`, `sdd ui`, MCP-Server, KI-Assistenz, Index-Anbindung, `sdd staleness`.
- Die Regeln des v1.1-Schnitts (`SDD-002b`, `SDD-010`, `SDD-011`, `SDD-023` bis `SDD-025`, `SDD-031`, `SDD-041`, `SDD-051`).
- Sprachspezifische Parser: Alle Prüfungen arbeiten auf Pfad-Globs und Textmustern.

## Akzeptanzkriterien

- **AC-1** Fehlt `sdd.config.yaml`, bricht jeder Prüflauf mit einem Ausführungsfehler (Exit-Code `2`) ab, statt mit Standardwerten zu prüfen.
- **AC-2** Eine Katalogregel ohne Handbuchabschnitt, ohne registrierte Prüffunktion, ohne effektives Gate oder — bei `error` — ohne Prüfvertrag erzeugt einen Befund (`SDD-070` bis `SDD-073`).
- **AC-3** `sdd lint` meldet auf dem `sdd`-Repository selbst keine Befunde (Dogfooding).
- **AC-4** Die Baseline unterdrückt bekannte `error`-Befunde gate-gebunden mit deterministischem `ordinal`; Regeln mit `baseline: forbidden` werden nie unterdrückt.
- **AC-5** Fehlt die konfigurierte Verfassung, ist sie leer, keine reguläre Datei oder liegt sie außerhalb des Repositorys, meldet `SDD-001` einen Befund.
- **AC-6** Fehlt in einer Markdown-Verfassung einer der Pflichtabschnitte Architektur/Layering, Security oder Strikte Regeln, meldet `SDD-002a` je fehlendem Abschnitt einen Befund.
- **AC-7** Überschreitet ein Feature die Größenschwelle ohne PRD bzw. Tech Spec mit passendem `feature:`-Feld, melden `SDD-020` bzw. `SDD-030` einen `error`; ist die Absicht nicht erklärt, ein `warn`; ist kein Feature auflösbar, aber die Schwelle überschritten, einen `error` (fail-closed).
- **AC-8** Jedes Akzeptanzkriterium in einem Feature-Artefakt trägt eine stabile Kennung `AC-<n>`; fehlt sie, meldet `SDD-021` einen Befund.
- **AC-9** Jede AC-Kennung eines aktiven Feature-Artefakts kommt in mindestens einer Testdatei vor; fehlt sie, meldet `SDD-022` einen Befund.
- **AC-10** Ein Feature-Artefakt mit `status: implemented` oder überschrittenem `archive-after` außerhalb des Archivs erzeugt in `pre-release` einen `error` (`SDD-050`).
