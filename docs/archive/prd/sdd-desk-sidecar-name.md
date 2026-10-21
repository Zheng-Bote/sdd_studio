---
feature: sdd-desk-sidecar-name
status: archived
release: v0.3.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - apps/desk/src-tauri/**
  - apps/desk/scripts/**
  - handbuch/**
---

# PRD — Eindeutiger Name für das mitgelieferte `sdd` der Desktop-Anwendung

## Problem

Die Desktop-Anwendung bringt `sdd` als Sidecar mit. Das `.deb`-Paket legt Sidecars nach `/usr/bin`, also als `/usr/bin/sdd` — ein kurzer, verbreiteter Name. Ein anderes Paket mit einem `sdd` kollidiert bei der Installation, oder eine von Hand installierte, andere Version von `sdd` im `PATH` wird überdeckt bzw. überdeckt die mitgelieferte.

## Ziel

Das mitgelieferte `sdd` trägt einen eindeutigen Namen, `sdd-studio-sdd`, und `sdd-desk` findet es neben sich. Eine eigene Installation von `sdd` bleibt unberührt und funktioniert weiter wie bisher.

## Nutzer

- **Nutzer, die die Desktop-Anwendung als Paket installieren**, besonders neben einer eigenen Installation von `sdd`.

## Umfang

Name des Sidecars in Bündel und Build-Skript, die Suche von `sdd-desk` nach `sdd` und das Handbuch.

## Nicht-Ziele

- Umbenennen von `sdd` selbst oder von `sdd-desk`.
- Ablage der Sidecars außerhalb von `/usr/bin` (bestimmt Tauri).
- Abgleich der Versionen zwischen mitgeliefertem und installiertem `sdd`.

## Akzeptanzkriterien

- **AC-1** `sdd-desk` nimmt ohne `--sdd` zuerst `sdd-studio-sdd` neben sich, sonst `sdd` neben sich, sonst `sdd` aus dem `PATH`; mit `--sdd` genau diesen Pfad.
- **AC-2** Das Bündel der Desktop-Anwendung enthält das Sidecar als `sdd-studio-sdd`, nicht als `sdd`, und das Build-Skript erzeugt es unter diesem Namen.
