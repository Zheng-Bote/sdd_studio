---
feature: desk-write-content
status: archived
archive-after:
touches:
  - internal/mcp/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/config/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# PRD — Arbeitsergebnisse aus dem KI-Desktop mit Inhalt speichern

## Problem

Im KI-Desktop entsteht im Gespräch ein vollständiges PRD, gespeichert wird aber nur das leere Template. `sdd_new` kennt keinen Inhalt, und ein Werkzeug zum Ändern bestehender Dateien gibt es nicht. Der Prompt sagt dem Modell, es solle bestehende Dateien „mit deinen eigenen Werkzeugen“ ändern, aber `sdd-desk` hat keine. Im Praxistest ist das Modell deshalb auf `sdd_new` mit Typ `doc` ausgewichen, hat Dateien an falschen Orten angelegt (`docs/notes/…`) und trotzdem gemeldet, das PRD „enthält alle von dir definierten Informationen“. Testdateien und `tests.include` musste der Nutzer von Hand anlegen. Das Ergebnis des Gesprächs geht verloren.

## Ziel

Was im Gespräch erarbeitet und vom Menschen freigegeben wurde, liegt danach genau so im Projekt: Artefakte mit Inhalt, geänderte Artefakte, Testdateien, `sdd.config.yaml`. Jeder Schreibvorgang ist vorher als Vorschau bzw. Diff sichtbar und wird vor dem Schreiben mit den Regeln geprüft. Das Modell meldet nur, was ein Werkzeugergebnis belegt.

## Nutzer

- **Nutzer des KI-Desktops**, die den Prozess im Gespräch durchlaufen.
- **Andere MCP-Clients** (z. B. Claude Code), die `sdd mcp --allow-write` nutzen.

## Umfang

- `sdd_new` mit Inhalt.
- Ein neues Werkzeug `sdd_write` zum Anlegen und Ändern von Artefakten, Testdateien und `sdd.config.yaml`.
- `sdd_read` für Konfiguration und Testdateien.
- Freigabe schreibender Werkzeuge für die Sitzung (ADR 0003, Stufe „schreibend“).
- Darstellung von Vorschau und Ergebnis im Desktop.
- Angepasste Prompts und Handbuch.

## Nicht-Ziele

- Quellcode außerhalb der Testdateien schreiben. Der Desktop ist kein Coding-Agent.
- Dateien verschieben oder löschen (Archivieren folgt als eigenes Feature).
- Projekt-Init aus dem Desktop (eigenes Feature).
- Die kritischen Werkzeuge aus ADR 0003 (Baseline, Lockfile, `adopt --apply`, ADR entscheiden).
- Ein CLI-Befehl `sdd write` oder Schreiben über `sdd ui`.
- Die Neuordnung des Prompts `sdd-idea` (Sprachwahl vor touches); das folgt in Phase 3.

## Akzeptanzkriterien

- **AC-1** `sdd_new` nimmt optional `content` an, wenn genau ein Typ angelegt wird, und schreibt diesen Inhalt statt des Templates. Vor Vorschau und Anlegen wird der Inhalt wie mit `sdd_validate_draft` geprüft. Bei einem `error`-Befund, bei einem `feature:` im Frontmatter, das nicht zum Feature-Slug passt, oder bei einem `archive-after` vor heute endet der Aufruf mit einem Fehler, der den Grund nennt, und schreibt nichts.
- **AC-2** `sdd_write` legt eine Datei an oder ersetzt ihren Inhalt. Erlaubt sind nur die Verfassung, Markdown-Dateien in den Artefaktverzeichnissen (`adr`, `prd`, `design`, `plans`, `archive`, `docs`), `sdd.config.yaml` und Dateien, die `tests.include` (oder die Fallback-Liste) auswählt. Jeder andere Pfad, `.git/`, `.sdd/`, ein Pfad außerhalb des Repositorys und ein symbolischer Link werden abgelehnt.
- **AC-3** `sdd_write` ersetzt eine bestehende Datei nur mit `base_sha256` gleich dem SHA-256 ihres aktuellen Inhalts. Fehlt der Wert oder weicht er ab, schreibt es nichts und nennt den aktuellen Hash. Markdown-Artefakte werden vorher wie in AC-1 geprüft, `sdd.config.yaml` muss sich laden und validieren lassen. Bei einem Fehler wird nichts geschrieben. Ersetzt wird atomar, die Dateirechte bleiben erhalten.
- **AC-4** `sdd_write` setzt nie ein ADR auf `accepted`: Ein Aufruf, nach dem eine ADR-Datei `status: accepted` trüge, die vorher nicht so hieß (oder neu ist), wird abgelehnt. Das Entscheiden eines ADR ist eine kritische Aktion (ADR 0003).
- **AC-5** Ohne `apply` liefert `sdd_write` eine Vorschau: Aktion (`create`, `update` oder `unchanged`), bei `update` ein Unified Diff, bei `create` den Inhalt, und die Befunde der Prüfung. Mit `apply` liefert es Status, SHA-256, Zeilenzahl und den Diff des tatsächlich Geschriebenen. `sdd_new` mit `apply` liefert je angelegter Datei SHA-256 und Zeilenzahl.
- **AC-6** `sdd_read` liest zusätzlich `sdd.config.yaml` und Testdateien (wie AC-2) und gibt den SHA-256 des vollständigen Dateiinhalts zurück.
- **AC-7** Im Desktop gilt `sdd_write` mit `apply` als schreibend: Die Freigabekarte zeigt Diff oder Inhalt. Schreibende Werkzeuge lassen sich für die Sitzung je Werkzeug freigeben. Ein so freigegebener Aufruf läuft ohne Karte, sein Ergebnis mit Diff erscheint im Gespräch.
- **AC-8** Die geführten Prompts und die Server-Anweisungen verlangen: Artefakte mit `sdd_new` samt geprüftem Inhalt anlegen, bestehende Dateien mit `sdd_read` lesen und mit `sdd_write` ändern, Erfolg nur melden, wenn ein Werkzeugergebnis ihn belegt.
