---
feature: sdd-desk-tauri
status: active
archive-after:
touches:
  - apps/desk/**
  - cmd/sdd-desk/**
  - .gitignore
  - handbuch/**
  - README.md
  - sdd.config.yaml
---

# PRD — KI-Desktop als Tauri-v2-Anwendung

## Problem

`sdd-desk` läuft heute als lokaler Server, der seine Oberfläche im Browser zeigt. Der Nutzer baut drei Teile von Hand (Oberfläche, `sdd`, `sdd-desk`), startet sie im Terminal, kopiert eine URL mit Sitzungstoken und wechselt das Projekt nur durch einen Neustart. Für Einsteiger, die Zielgruppe des Desktops, ist das zu viel.

## Ziel

Eine Desktop-Anwendung, die sich wie ein Programm installieren und starten lässt: Sie bringt `sdd-desk` und `sdd` als Sidecars mit, öffnet das zuletzt genutzte Projekt und erlaubt den Projektwechsel über das Menü — ohne Terminal, ohne kopierte URL und ohne die Sicherheitsgrenzen des Desktops zu lockern.

## Nutzer

- **Einsteiger**, die `sdd-desk` ohne Terminal nutzen wollen.
- **Erfahrene**, die mehrere Projekte nacheinander öffnen.

## Umfang

Tauri-v2-Hülle unter `apps/desk/src-tauri`: Fenster, Menü, Projektwahl, Lebenszyklus der Sidecars und Bündel für Linux, macOS und Windows. Oberfläche und Gesprächslogik bleiben die von `sdd-desk`.

## Nicht-Ziele

- Streaming der Modellantworten — folgt als eigenes Feature.
- Neue Fähigkeiten der KI oder eigene Werkzeuge der Hülle; die Hülle führt keine `sdd`-Werkzeuge aus.
- Zugriff der Oberfläche auf Tauri-Schnittstellen (Dateisystem, Shell, Dialoge).
- Automatische Updates, Code-Signierung und Veröffentlichung der Bündel.
- Schlüssel in der Anwendung speichern oder eingeben.

## Akzeptanzkriterien

- **AC-1** Die Anwendung startet `sdd-desk` als Sidecar mit `--project <projekt>` auf `127.0.0.1` und einem freien Port; `sdd-desk` findet `sdd` als zweiten Sidecar neben sich.
- **AC-2** Das Projekt kommt aus dem Startargument, sonst aus dem zuletzt genutzten Projekt, sonst aus einem Ordnerdialog; „Projekt öffnen…“ im Menü wechselt das Projekt, beendet den alten Sidecar und merkt sich das neue.
- **AC-3** Die Hülle liest die Adresse mit Sitzungstoken aus der Ausgabe von `sdd-desk`, lädt sie ins Fenster und gibt das Token nirgends aus — nicht im Protokoll, nicht auf der Fehlerseite.
- **AC-4** Das Fenster navigiert nur zur eigenen Startseite und zum Ursprung des laufenden Sidecars; andere `http`- und `https`-Adressen öffnet der Standardbrowser, alle übrigen Adressen werden verworfen.
- **AC-5** Keine Seite im Fenster erhält Zugriff auf Tauri-Schnittstellen.
- **AC-6** `config/llm.env` wird wie bei `sdd-desk` neben dem Programm gesucht; liegt eine `llm.env` im Konfigurationsordner der Anwendung, wird sie mit `--config` übergeben. Die Hülle liest keine Schlüssel.
- **AC-7** Beendet sich der Sidecar unerwartet oder meldet er sich nicht rechtzeitig, zeigt die Startseite den Fehler mit den letzten Protokollzeilen; beim Schließen der Anwendung wird der Sidecar beendet — auch wenn die Anwendung abstürzt oder hart beendet wird.
- **AC-8** Ein Befehl baut Oberfläche, Sidecars und Bündel in der richtigen Reihenfolge.
