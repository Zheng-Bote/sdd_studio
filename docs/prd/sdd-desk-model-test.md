---
feature: sdd-desk-model-test
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
---

# PRD — Modell testen in `sdd-desk`

## Problem

Ob ein in `llm.env` eingetragenes Modell funktioniert, zeigt sich bisher erst mitten im Gespräch: Ein falscher oder nicht mehr angebotener Modellname, ein ungültiger Schlüssel oder ein Modell ohne Werkzeugunterstützung führt zu einem Fehler nach der ersten Nachricht (Beispiel: OpenRouter `HTTP 404: No endpoints found for deepseek/deepseek-r1:free`). Der Nutzer weiß dann nicht, ob das Modell, der Schlüssel oder die Werkzeuge die Ursache sind, und das Gespräch hat schon begonnen.

## Ziel

Neben der Modellauswahl lässt sich das gewählte Modell mit einem Klick prüfen, bevor das Gespräch beginnt. Die Antwort sagt, ob das Modell antwortet, ob es die Werkzeuge von `sdd mcp` annimmt und, wenn nicht, mit welcher Meldung der Anbieter ablehnt.

## Nutzer

- **Alle Nutzer von `sdd-desk`**, besonders beim Einrichten von Cloud-Anbietern und nach Änderungen an `llm.env`.

## Umfang

Ein Endpunkt im Backend von `sdd-desk`, der eine kurze, feste Testnachricht an das gewählte Modell schickt, und ein Button „LLM testen“ neben der Modellauswahl mit Anzeige des Ergebnisses. Für alle Anbieter, auch lokales Ollama.

## Nicht-Ziele

- Abgleich mit Modelllisten der Anbieter (z. B. `/api/v1/models` von OpenRouter): anbieterspezifisch und ein zusätzlicher Netzzugriff; der Test selbst zeigt, ob das Modell erreichbar ist.
- Automatisches Testen beim Start oder beim Wechsel des Modells: jeder Test kostet beim Anbieter.
- Ändern von `llm.env` aus der Oberfläche.
- Änderungen an `sdd` oder `sdd mcp`.

## Akzeptanzkriterien

- **AC-1** Der Test schickt nur eine feste Testnachricht und die Werkzeugbeschreibungen von `sdd mcp` an das gewählte Modell — keine Gesprächs- oder Projektinhalte — und liefert, ob das Modell geantwortet hat, den Anfang der Antwort und die Dauer.
- **AC-2** Lehnt der Anbieter die Anfrage mit Werkzeugen ab, wiederholt der Test sie ohne Werkzeuge und meldet getrennt, ob das Modell überhaupt antwortet und ob es Werkzeuge annimmt, jeweils mit der Meldung des Anbieters.
- **AC-3** Für Cloud-Anbieter gilt der Hinweis „Inhalte verlassen dieses Gerät“ wie für ein Gespräch: ohne Bestätigung lehnt das Backend den Test ab, bevor etwas gesendet wird; unbekannte Anbieter und fehlendes Modell werden ebenso abgelehnt. Der Schlüssel erscheint nie im Ergebnis.
- **AC-4** Ein Test endet nach höchstens 60 Sekunden je Anfrage mit einer Fehlermeldung.
- **AC-5** Die Oberfläche bietet „LLM testen“ neben der Modellauswahl an, solange ein Modell gewählt ist, kein Zug läuft und der Hinweis bestätigt ist; sie zeigt während des Tests einen Hinweis und danach das Ergebnis für genau dieses Modell; ein Wechsel von Anbieter oder Modell entfernt es.
