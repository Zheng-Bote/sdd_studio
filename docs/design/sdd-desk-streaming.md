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

# Tech Spec — Gestreamte Modellantworten in `sdd-desk`

Setzt das PRD `docs/prd/sdd-desk-streaming.md` um. Grundlage ist die Tech Spec `docs/design/sdd-desk.md`; Gesprächsschleife, Freigaben und Datenmodell bleiben, wie sie dort beschrieben sind.

## Architektur

```text
internal/desk/
  ├── stream.go      Streamer-Schnittstelle, Zeilenleser für NDJSON und SSE, Stream je Protokoll
  ├── agent.go       TurnStream: wie Turn, meldet Ereignisse; Check vor dem ersten Byte
  └── server.go      POST /api/turn/stream (application/x-ndjson)
apps/desk/src/app/
  ├── core/desk-api.ts      turnStream: fetch mit ReadableStream, AbortSignal
  ├── core/chat-session.ts  streaming-Text, cancel(), Aufräumen nach Abbruch
  └── chat/chat-panel.ts    entstehender Text, „Abbrechen“
```

Ein Anbieter kann zusätzlich zu `Chat` die Schnittstelle `Streamer` erfüllen:

```go
type Streamer interface {
    Stream(ctx, model, system string, msgs []Message, tools []ToolDef, onText func(string)) (Message, error)
}
```

`TurnStream` nutzt `Stream`, wenn der Anbieter ihn bietet, sonst `Complete`. `Turn` (und damit `POST /api/turn`) bleibt unverändert nicht gestreamt.

## Schnittstellen

**Protokolle** (jeweils `stream: true`):

| Protokoll | Format | Text | Werkzeugaufrufe |
|---|---|---|---|
| Ollama | NDJSON, `done: true` am Ende | `message.content` | `message.tool_calls`, vollständig in einem Stück |
| OpenAI-kompatibel | SSE `data: {…}`, Ende `data: [DONE]` | `choices[0].delta.content` | `delta.tool_calls[]` nach `index` zusammengesetzt, `arguments` als Textstücke |
| Claude | SSE mit `event:`/`data:` | `content_block_delta` / `text_delta` | `content_block_start` (`tool_use`) und `input_json_delta` je Block-`index` |

Ein `error`-Objekt im Stream wird zum Fehler des Schritts. Zeilen bis 1 MiB.

**`POST /api/turn/stream`**: gleicher Body wie `/api/turn`. Prüfungen von Anbieter, Hinweis und Modell laufen vorher (`Agent.Check`) und antworten wie bisher mit `400`/`428`. Danach `200`, `Content-Type: application/x-ndjson`, eine JSON-Zeile je Ereignis, nach jeder Zeile `Flush`:

| `type` | Felder | Bedeutung |
|---|---|---|
| `delta` | `text` | Textstück der entstehenden Antwort |
| `message` | `message` | fertige Nachricht (Antwort mit Werkzeugaufrufen oder Werkzeugergebnis) |
| `done` | `messages`, `pending` | wie die Antwort von `/api/turn` |
| `error` | `error` | Fehler mitten im Zug; danach endet der Stream |

Der Zug läuft im Kontext der Anfrage: bricht die Oberfläche `fetch` ab, wird die Anfrage an den Anbieter abgebrochen.

## Oberfläche

`DeskApi.turnStream(req, onEvent, signal)` liest den Body zeilenweise. `ChatSession` hält den entstehenden Text in `streaming`, hängt jede `message` an und übernimmt bei `done` Verlauf und offene Freigaben. `cancel()` bricht ab; danach bleiben die fertigen Nachrichten, ein angefangener Text wird als Antwort behalten, und eine Antwort mit Werkzeugaufrufen ohne Ergebnis wird samt allem danach verworfen (`trimOpenCalls`), damit kein Anbieter einen unvollständigen Verlauf sieht.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `stream.go` | `TestStreamProviders` |
| AC-2 | `agent.go` (`TurnStream`), `server.go` | `TestTurnStreamEvents`, `TestTurnStreamEndpoint` |
| AC-3 | `agent.go` (`Check`, `decide`) | `TestTurnStreamKeepsApprovals` |
| AC-4 | Anfragekontext, `chat-session.ts` (`cancel`, `trimOpenCalls`) | `TestTurnStreamCancel`, `chat-session.spec.ts` |
| AC-5 | `chat-panel.ts`, `chat-session.ts` | `chat-session.spec.ts` |

## Sicherheit

- Der Endpunkt liegt unter `/api/`: `Host`-Prüfung, Sitzungstoken, `Content-Type`-Prüfung und Größenlimit gelten wie für alle Aufrufe.
- Gestreamt wird nur, was `/api/turn` ohnehin zurückgibt; Freigaben laufen über denselben `decide`.
- Textstücke werden als Text gerendert, nie als HTML.
