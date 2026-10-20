---
feature: sdd-desk-streaming
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
---

# PRD — Gestreamte Modellantworten in `sdd-desk`

## Problem

`sdd-desk` zeigt eine Antwort erst, wenn das Modell fertig ist. Bei lokalen Modellen und langen Entwürfen (PRD, Tech Spec) wartet der Nutzer oft eine Minute oder länger vor einem Drehsymbol, ohne zu sehen, ob das Modell in die richtige Richtung schreibt. Abbrechen lässt sich ein Zug nicht.

## Ziel

Der Text einer Antwort und, falls das Modell ihn liefert, sein Gedankengang erscheinen, während das Modell arbeitet; Werkzeugergebnisse erscheinen, sobald sie vorliegen; ein laufender Zug lässt sich abbrechen. Freigaben, Hinweis für Cloud-Anbieter und alle Sicherheitsgrenzen von `sdd-desk` bleiben unverändert.

## Nutzer

- **Einsteiger und Erfahrene** im Gespräch mit `sdd-desk`, besonders mit lokalen Modellen.

## Umfang

Gestreamte Antworten für alle Anbieter von `sdd-desk` (Ollama lokal und Cloud, OpenAI-kompatibel, Claude), ein gestreamter Gesprächszug in der HTTP-API und die Anzeige samt Abbrechen in der Oberfläche.

## Nicht-Ziele

- Gestreamte Werkzeugaufrufe: ein Aufruf wird erst zur Freigabe vorgelegt, wenn er vollständig ist.
- Darstellung als Markdown; Text bleibt Text.
- Fortsetzen eines abgebrochenen Zuges.
- Änderungen an `sdd` oder `sdd mcp`.

## Akzeptanzkriterien

- **AC-1** Alle drei Protokolle (Ollama, OpenAI-kompatibel, Claude) liefern Text stückweise, während das Modell schreibt; das Ergebnis eines gestreamten Schritts — Text und Werkzeugaufrufe — gleicht dem eines nicht gestreamten.
- **AC-2** `POST /api/turn/stream` liefert einen Gesprächszug als Folge von Ereignissen: Textstücke, jede fertige Nachricht (Antwort oder Werkzeugergebnis) und zum Schluss Verlauf und offene Freigaben; ein Fehler mitten im Zug kommt als Ereignis.
- **AC-3** Freigaben gelten im gestreamten Zug genau wie im nicht gestreamten: kein Werkzeug ohne Erlaubnis, schreibende Aufrufe mit Vorschau; ohne bestätigten Hinweis lehnt der Endpunkt einen Cloud-Anbieter ab, bevor etwas gestreamt wird.
- **AC-4** Bricht der Nutzer ab, beendet das Backend die Anfrage an den Anbieter; die Oberfläche behält die fertigen Nachrichten und verwirft einen Werkzeugaufruf ohne Ergebnis, damit das Gespräch fortgesetzt werden kann.
- **AC-5** Die Oberfläche zeigt den entstehenden Text fortlaufend als Text an und bietet während eines Zuges „Abbrechen“ an.
- **AC-6** Liefert ein Modell seinen Gedankengang gesondert (Ollama `thinking`, OpenAI-kompatibel `reasoning`/`reasoning_content`, Claude `thinking_delta`), zeigt die Oberfläche ihn während des Zuges aufklappbar an; er wird nicht Teil des Verlaufs und geht nicht zurück an das Modell.
