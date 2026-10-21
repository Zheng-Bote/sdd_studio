---
feature: sdd-desk-turn-timeout
status: archived
release: v0.4.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - config/**
  - handbuch/**
---

# PRD — Einstellbare Frist für einen Gesprächszug in `sdd-desk`

## Problem

Ein Gesprächszug in `sdd-desk` endet nach festen 10 Minuten. Mit einem lokalen Modell reicht das für mehrstufige Abläufe nicht: Der Abgleich mit der Verfassung (`sdd-conflicts`) mit `qwen3:14b` liest drei Artefakte und schreibt dann die Antwort — und bricht zweimal mit `context deadline exceeded` ab, bevor die Antwort kommt. Zusätzlich begrenzt der HTTP-Client jede einzelne Anfrage an den Anbieter ebenfalls auf 10 Minuten. Der Nutzer kann daran nichts ändern.

## Ziel

Die Frist eines Zuges passt zum Anbieter: lokal großzügiger als in der Cloud, und für jeden Anbieter in `llm.env` einstellbar. Ein Zug endet weiterhin sicher — durch die Frist oder durch „Abbrechen“.

## Nutzer

- **Nutzer lokaler Modelle**, deren Rechner für mehrstufige Abläufe länger braucht.
- **Nutzer von Cloud-Anbietern** mit langsamen Modellen oder langen Antworten.

## Umfang

Ein Eintrag je Anbieter in `llm.env`, Standardwerte je Art des Anbieters, die Frist im Backend und die Beschreibung in Vorlage und Handbuch.

## Nicht-Ziele

- Einstellen der Frist in der Oberfläche.
- Fristen für einzelne Werkzeugaufrufe von `sdd mcp` oder für „LLM testen“.
- Zug ohne jede Frist.

## Akzeptanzkriterien

- **AC-1** Ohne Eintrag gilt für lokales Ollama eine Frist von 30 Minuten je Zug, für Cloud-Anbieter 10 Minuten.
- **AC-2** `<ANBIETER>_TURN_TIMEOUT` in `llm.env` oder in der Prozessumgebung setzt die Frist dieses Anbieters als Dauer (z. B. `45m`, `1h30m`) zwischen 1 Minute und 4 Stunden; ein ungültiger oder außerhalb liegender Wert erzeugt eine Warnung, und es gilt der Standardwert.
- **AC-3** Beide Endpunkte eines Zuges (`/api/turn`, `/api/turn/stream`) verwenden die Frist des gewählten Anbieters; eine einzelne Anfrage an den Anbieter hat keine eigene, kürzere Frist mehr.
