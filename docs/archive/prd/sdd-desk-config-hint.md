---
feature: sdd-desk-config-hint
status: archived
release: v0.3.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - apps/desk/src-tauri/**
  - handbuch/**
---

# PRD — Richtiger Ort für `llm.env` in der Warnung von `sdd-desk`

## Problem

Fehlt `llm.env`, warnt `sdd-desk` mit dem Pfad `config/llm.env` neben der ausführbaren Datei. In der Desktop-Anwendung ist das der Ordner des Sidecars — bei einem installierten Bündel z. B. `/usr/bin/config/llm.env`, ein Ort, an den die Datei nicht gehört und an dem sie ohne Root-Rechte nicht anzulegen ist. Der richtige Ort, der Konfigurationsordner der Anwendung, erscheint nirgends.

## Ziel

Fehlt die Konfiguration, nennt die Warnung den Ort, an dem der Nutzer die Datei anlegen soll: in der Desktop-Anwendung `llm.env` im Konfigurationsordner der Anwendung, sonst wie bisher `config/llm.env` neben `sdd-desk`.

## Nutzer

- **Nutzer der Desktop-Anwendung**, die Cloud-Anbieter zum ersten Mal einrichten.

## Umfang

Ein Startargument von `sdd-desk` für den empfohlenen Ort, das die Desktop-Anwendung setzt, und der Text der Warnung.

## Nicht-Ziele

- Anlegen der Datei aus der Anwendung heraus.
- Ändern der Reihenfolge, in der Orte gelesen werden: eine vorhandene Datei im Konfigurationsordner hat Vorrang, sonst gilt `config/llm.env` neben `sdd-desk`.
- Änderungen an `sdd`.

## Akzeptanzkriterien

- **AC-1** Startet `sdd-desk` ohne `--config` und mit `--config-hint <pfad>` und gibt es am Standardort keine Datei, nennt die Warnung `<pfad>` als Ort für `llm.env` samt Rechten `0600`; ohne `--config-hint` nennt sie wie bisher den Standardort. Mit `--config` hat `--config-hint` keine Wirkung.
- **AC-2** Die Desktop-Anwendung übergibt `<app_config_dir>/llm.env` als `--config-hint`, wenn sie dort keine Datei findet, und wie bisher als `--config`, wenn die Datei existiert.
