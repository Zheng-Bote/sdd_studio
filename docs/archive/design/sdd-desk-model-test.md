---
feature: sdd-desk-model-test
status: archived
release: v0.3.0
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
---

# Tech Spec — Modell testen in `sdd-desk`

Setzt das PRD `docs/archive/prd/sdd-desk-model-test.md` um. Grundlage ist die Tech Spec `docs/archive/design/sdd-desk.md`; Anbieter, Hinweis und Sicherheitsgrenzen bleiben, wie sie dort beschrieben sind.

## Architektur

```text
internal/desk/
  ├── modeltest.go   Agent.TestModel: Testnachricht mit, bei Ablehnung ohne Werkzeuge
  └── server.go      POST /api/model/test
apps/desk/src/app/
  ├── core/desk-api.ts      testModel(req)
  ├── core/chat-session.ts  testModel(), modelTest (Ergebnis für Anbieter und Modell)
  └── app.ts                Button „LLM testen“ und Ergebnis neben der Modellauswahl
```

`Agent.TestModel(ctx, req)` prüft Anbieter, Hinweis und Modell mit `Agent.Check` und ruft dann `Complete` des Anbieters — nicht gestreamt — mit festem Systemtext und fester Nutzernachricht:

- System: `Verbindungstest von sdd-desk. Antworte in einem kurzen Satz und rufe kein Werkzeug auf.`
- Nutzer: `Hallo! Bitte antworte kurz mit „Hello World“.`

Erster Versuch mit `Agent.Tools`, wie in einem Gespräch. Lehnt der Anbieter ihn ab, folgt ein zweiter Versuch ohne Werkzeuge; so lässt sich „Modell nicht erreichbar“ von „Modell ohne Werkzeuge“ trennen. Jeder Versuch hat einen eigenen Kontext mit 120 Sekunden Frist, weil lokale Modelle beim ersten Aufruf erst geladen werden. Läuft die Frist ab oder bricht der Nutzer ab (Anfragekontext), endet der Test ohne zweiten Versuch: Ein langsames Modell hat die Werkzeuge nicht abgelehnt.

## Schnittstellen

**`POST /api/model/test`** mit Body `{provider, model, disclosure_ack}` (Teilmenge von `TurnRequest`). Fehler von `Check` antworten vor dem ersten Versuch wie bei `/api/turn/stream` mit `400` bzw. `428`. Sonst `200`:

| Feld | Bedeutung |
|---|---|
| `ok` | das Modell hat geantwortet (mit oder ohne Werkzeuge) |
| `timeout` | die Frist eines Versuchs lief ab |
| `tools` | `true` erster Versuch gelang, `false` nur ohne Werkzeuge, fehlt wenn keiner gelang |
| `reply` | Anfang der Antwort, höchstens 200 Zeichen; ruft das Modell trotzdem ein Werkzeug auf, dessen Name |
| `error` | Meldung des Anbieters zum ersten fehlgeschlagenen Versuch |
| `error_plain` | Meldung zum zweiten Versuch, wenn auch er fehlschlug |
| `ms` | Dauer des erfolgreichen Versuchs in Millisekunden |

Die Meldungen sind die von `post` (Status und Anfang des Antworttexts, höchstens 500 Zeichen); sie enthalten keine Kopfzeilen, also keinen Schlüssel. Werkzeugaufrufe in der Antwort werden nie ausgeführt.

## Oberfläche

`ChatSession.testModel()` setzt `modelTest` auf `{provider, model, running: true}`, ruft `DeskApi.testModel` und legt das Ergebnis ab. Angezeigt wird es nur, solange `provider` und `model` der Auswahl entsprechen; `selectProvider` und die Modellauswahl setzen es zurück. Der Button steht neben der Modellauswahl und ist gesperrt, wenn kein Modell gewählt ist, ein Zug oder Test läuft oder `needsDisclosure()` gilt. Das Ergebnis erscheint als Zeile unter der Kopfleiste: grün „antwortet, Werkzeuge ok“ mit Dauer und Antwort, gelb „antwortet, aber ohne Werkzeuge“ mit Meldung, rot „antwortet nicht“ bzw. „keine Antwort innerhalb von 120 s“ mit Meldung.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `modeltest.go` | `TestModelTestSendsOnlyFixedText` |
| AC-2 | `modeltest.go` | `TestModelTestWithoutTools` |
| AC-3 | `modeltest.go` (`Check`), `server.go` | `TestModelTestEndpointRefuses` |
| AC-4 | `modeltest.go` (`modelTestTimeout`) | `TestModelTestTimeout` |
| AC-5 | `chat-session.ts`, `app.ts` | `chat-session.spec.ts` |
