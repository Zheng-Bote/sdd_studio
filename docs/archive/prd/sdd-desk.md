---
feature: sdd-desk
status: archived
release: v0.2.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - apps/desk/**
  - .gitignore
  - handbuch/**
  - sdd.config.yaml
---

# PRD — KI-Desktop (`sdd-desk`)

## Problem

Der Prozess von der Produkt-Idee über PRD, Tech Spec, ADR und Plan bis zu Tests, Gate und Archiv ist für Einsteiger schwer zu überblicken. `sdd` prüft mechanisch und nennt mit `sdd next` den nächsten Schritt, führt aber kein Gespräch: Niemand fragt nach Nicht-Zielen, schlägt prüfbare Akzeptanzkriterien vor oder formuliert eine Tech Spec mit. `sdd` selbst soll dafür KI-frei und offline bleiben (ADR 0002).

## Ziel

Ein getrenntes, optionales Programm führt im Gespräch mit einem frei wählbaren Sprachmodell durch den Prozess und darf `sdd`-Werkzeuge ausführen — jedes nur mit Erlaubnis des Nutzers, schreibende erst nach Vorschau.

## Nutzer

- **Einsteiger**, die eine Idee zu einem prüfbaren PRD und weiter durch den Prozess bringen wollen.
- **Erfahrene**, die Entwürfe schneller formulieren und gegen die Regeln prüfen lassen wollen.

## Umfang

Go-Backend `sdd-desk` (Anbieter, Gesprächsschleife, MCP-Client für `sdd mcp`, lokaler Server) und Angular-Oberfläche unter `apps/desk/`, die später in eine Tauri-v2-Anwendung wandert.

## Nicht-Ziele

- Die Tauri-Hülle selbst — sie folgt später; die Oberfläche ist dafür vorbereitet.
- Schwärzung von Inhalten (ADR 0002); der Nutzer wählt den Anbieter bewusst.
- Zugriff der KI auf Baseline, ADR-Lockfile, `adopt --apply` oder das Entscheiden von ADRs.
- Eigene Werkzeuge neben `sdd mcp`, Dateizugriff am MCP-Server vorbei, Telemetrie.

## Akzeptanzkriterien

- **AC-1** `sdd-desk` liest die Anbieter aus `config/llm.env` neben der ausführbaren Datei (oder `--config`); Prozessvariablen haben Vorrang, Anbieter ohne Schlüssel sind abgeschaltet, und eine Datei mit Rechten für Gruppe oder andere wird abgelehnt.
- **AC-2** Lokales Ollama, Ollama Cloud, OpenRouter, NVIDIA NIM, Claude und Gemini lassen sich mit Werkzeugaufrufen ansprechen; Anbieter und Modell sind in der Oberfläche wählbar.
- **AC-3** Werkzeuge laufen ausschließlich über einen von `sdd-desk` gestarteten `sdd mcp`; angeboten werden genau dessen Werkzeuge, jeder andere Name wird abgelehnt.
- **AC-4** Jeder Werkzeugaufruf braucht eine Erlaubnis — einmal, für die Sitzung oder Ablehnung; ein schreibender Aufruf zeigt vorher die Vorschau mit Dateien und Inhalt.
- **AC-5** Vor dem ersten Aufruf eines Cloud-Anbieters in einer Sitzung müssen Anbieter und Hinweis „Inhalte verlassen das Gerät“ bestätigt sein; ohne Bestätigung lehnt das Backend den Aufruf ab.
- **AC-6** Die geführten Abläufe von `sdd mcp` (Idee → PRD, Tech Spec, ADR, Plan, Review) lassen sich starten, und die Oberfläche zeigt den Prozessstand aus `sdd_next`.
- **AC-7** Der lokale Server bindet nur an `127.0.0.1`, prüft `Host`, verlangt für Anfragen an die Programmierschnittstelle das Sitzungstoken und setzt eine Content-Security-Policy; es gibt keine Telemetrie.
- **AC-8** `sdd` — Kern, CLI, `sdd ui`, `sdd mcp` — importiert kein Paket des Desktops.
