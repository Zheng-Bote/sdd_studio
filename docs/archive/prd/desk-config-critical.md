---
feature: desk-config-critical
status: archived
archive-after:
touches:
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
release: v0.6.0
---

# PRD — Konfiguration im KI-Desktop nur mit einzelner Bestätigung ändern

## Problem

`sdd.config.yaml` bestimmt, was `sdd` prüft: Profil, ausgenommene Pfade, die Regeln je Gate, Schwellen, Testmuster. Heute ändert der KI-Desktop die Datei mit `sdd_write`, also in der Stufe „schreibend“. Ist `sdd_write` für die Sitzung freigegeben, ändert das Modell die Konfiguration ohne Rückfrage. Im Live-Test vom 2026-11-05 ist das mit `tests.include` passiert. Mit derselben Freigabe ließe sich `exclude: ["**"]` oder `profile: minimal` schreiben. `profile: minimal` schaltet SDD-011, SDD-050 und SDD-051 ab, obwohl die Baseline sie nicht unterdrücken darf.

ADR 0004 hat entschieden: Jede Änderung an `sdd.config.yaml` ist im Desktop kritisch und läuft über ein eigenes Werkzeug `sdd_config`.

## Ziel

Die Konfiguration ändert sich im Desktop nur nach einer einzelnen, bewussten Bestätigung. Diese zeigt den Diff und die Wirkung auf die Befunde. Kein Weg über `sdd_write` oder eine Sitzungsfreigabe führt daran vorbei.

## Nutzer

- **Nutzer des KI-Desktops**, die ein Projekt einrichten und dabei z. B. `tests.include` oder `thresholds` setzen.
- **Reviewer**, die nachvollziehen wollen, warum eine Prüfung im Projekt schwächer geworden ist.

## Umfang

- Das kritische Werkzeug `sdd_config` in `sdd mcp`.
- `sdd_write` lehnt `sdd.config.yaml` ab.
- Die Einstufung in `sdd-desk` und die Darstellung der Befund-Differenz auf der Freigabekarte.
- Prompts, Server-Anweisungen, Handbuch.

## Nicht-Ziele

- Eine inhaltliche Sperre in `sdd` für abschwächende Änderungen, z. B. einen Profilwechsel nach unten. Das bleibt einem späteren ADR vorbehalten (ADR 0004, Entscheidung 4).
- Ein CLI-Befehl zum Ändern der Konfiguration.
- Änderungen an `sdd init` bzw. `sdd_init`. Das Anlegen der Konfiguration bleibt schreibend.
- Änderungen an `sdd_read`.

## Akzeptanzkriterien

- **AC-1** `sdd mcp` bietet `sdd_config` (`content`, `base_sha256`, `apply`) nur mit `--allow-critical` an, mit `destructiveHint: true`. Es ändert ausschließlich eine bestehende `sdd.config.yaml`. Dafür muss `base_sha256` dem aktuellen Inhalt entsprechen, und der neue Inhalt muss sich laden und validieren lassen. Sonst schreibt es nichts und nennt den Grund.
- **AC-2** Ohne `apply` liefert `sdd_config` eine Vorschau mit `critical: true`. Sie enthält den Diff und je Prüflauf (Lint, `pre-implementation`, `pre-merge`, `pre-release`) die Befunde, die mit der neuen Konfiguration wegfallen und die hinzukommen. Dazu kommt eine Zusammenfassung mit den geänderten Schlüsseln und der Zahl der wegfallenden und hinzukommenden Befunde.
- **AC-3** Die Zusammenfassung nennt einen Wechsel von `profile` mit altem und neuem Wert und jede Änderung an `gates.<gate>.override` ausdrücklich.
- **AC-4** Mit `apply` ersetzt `sdd_config` die Datei atomar und liefert SHA-256 und Diff des Geschriebenen.
- **AC-5** `sdd_write` lehnt `sdd.config.yaml` ab, als Vorschau wie mit `apply`, und verweist auf `sdd_config`.
- **AC-6** Im Desktop ist `sdd_config` mit `apply` kritisch: einzeln bestätigt, nie für die Sitzung. Die Freigabekarte zeigt die wegfallenden und hinzukommenden Befunde.
- **AC-7** Prompts und Server-Anweisungen nennen `sdd_config` für Änderungen an der Konfiguration. Anders als Baseline, Lockfile, `adopt --apply` und ADR-Entscheid darf das Modell eine Konfigurationsänderung von sich aus vorschlagen, wenn ein Ablauf sie braucht, etwa `tests.include` in `sdd-tests`. Vorher erklärt es sie in Worten. Ausgeführt wird sie nur nach der einzelnen Bestätigung. Lehnt der Mensch ab, fragt das Modell im selben Zug nicht erneut.
- **AC-8** Das Handbuch beschreibt `sdd_config` als kritische Aktion und nennt die Zahl der Werkzeuge von `sdd mcp` richtig.
