---
feature: config-weakening
status: active
archive-after:
touches:
  - catalog/**
  - internal/sddcore/catalog/**
  - internal/sddcore/rules/**
  - handbuch/**
  - docs/sdd_konzept.md
  - README.md
  - CHANGELOG.md
---

# PRD — Abschwächung der Konfiguration braucht ein ADR (SDD-012)

## Problem

Außerhalb des KI-Desktops bemerkt `sdd` keine Abschwächung von `sdd.config.yaml`. `profile: minimal` schaltet SDD-011, SDD-050 und SDD-051 ab, obwohl die Baseline sie nicht unterdrücken darf. Eine geänderte `gates.<gate>.override` nimmt Regeln aus einem Gate, ein neues Muster in `exclude` nimmt Pfade aus jeder Prüfung. Im Pull Request ist das jeweils eine unscheinbare Zeile. ADR 0005 hat entschieden: Eine eindeutig abschwächende Änderung braucht, wie eine Änderung der Verfassung (SDD-010), ein ADR im selben Diff. Die Regel dafür darf sich durch genau diese Änderung nicht abschalten lassen.

## Ziel

`sdd gate pre-merge` und `pre-implementation` melden eine abschwächende Änderung der Konfiguration als `error`, solange kein ADR im selben Diff sie begründet. Weder `profile` noch `override` schalten diese Prüfung ab.

## Nutzer

- **Teams**, die `sdd` in CI nutzen und Abschwächungen nicht dem Review überlassen wollen.
- **Reviewer**, die zu einer Abschwächung das Warum finden wollen.

## Umfang

- Neues Katalogattribut `pinned`.
- Neue Regel SDD-012 mit Prüfvertrag.
- Katalog `1.2.0`, Handbuch, Konzept, README, CHANGELOG.

## Nicht-Ziele

- Erfassen von `tests.include`, `artifacts.*`, `base_ref` und `constitution_sections` (ADR 0005, Entscheidung 3).
- Bestehende Abschwächungen melden. Die Regel greift nur auf Änderungen im Diff gegen die Merge-Basis.
- Prüfen, ob das ADR eine gute Begründung enthält.
- Weitere `pinned`-Regeln. Jede braucht ein eigenes ADR.

## Akzeptanzkriterien

- **AC-1** Eine Katalogregel mit `pinned: true` gehört in jedem Gate ihrer `gate_default` zur effektiven Regelmenge, auch wenn `gates.<gate>.override` gesetzt ist und sie nicht nennt. Andere Regeln verhalten sich unverändert.
- **AC-2** Der Katalog lehnt `pinned: true` bei einer Regel ab, die nicht das Profil `minimal` und nicht `baseline: forbidden` trägt. Eine solche Regel ließe sich sonst über `profile` oder die Baseline doch abschalten.
- **AC-3** SDD-012 meldet in `pre-implementation` und `pre-merge` einen `error` an `sdd.config.yaml`, wenn die Datei sich gegenüber der Merge-Basis abschwächend ändert. Das ist der Fall, wenn `profile` in der Ordnung `minimal` < `standard` < `regulated` sinkt, `gates.<gate>.override` sich ändert, `exclude` ein neues Muster erhält, `thresholds.layers` oder `thresholds.feature_artifacts_required_when.touched_paths` einen Eintrag verlieren oder `min_layers_touched` steigt. Die Meldung nennt jede erkannte Abschwächung.
- **AC-4** SDD-012 ist erfüllt, wenn ein ADR im selben Diff `sdd.config.yaml` in `references` oder `adr` nennt (wie SDD-010). Ohne Konfiguration an der Merge-Basis (erstmaliges Anlegen) und bei nicht abschwächenden Änderungen (z. B. nur `tests.include`, eine zusätzliche Schicht in `thresholds.layers` oder Kommentare) meldet sie nichts.
- **AC-5** SDD-012 trägt Profil `minimal`, `baseline: forbidden` und `pinned: true`. Die Katalogversion ist `1.2.0`.
- **AC-6** Handbuch (Abschnitt und Prüfvertrag für SDD-012, `pinned` beim Katalog, SDD-072), Konzept §9.6, README und CHANGELOG beschreiben die Regel und `pinned`.
