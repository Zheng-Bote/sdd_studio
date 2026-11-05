---
feature: desk-approval-scope
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
---

# Tech Spec — Sitzungsfreigabe gilt nur für die Stufe, die der Mensch gesehen hat

Setzt das PRD `docs/prd/desk-approval-scope.md` um, auf Grundlage von ADR 0003 (Entscheidung 6, Stufen „lesend“ und „schreibend“). Das ADR bleibt unverändert: Es verlangt schon heute, dass vor einer schreibenden Freigabe Vorschau bzw. Diff gezeigt werden. Diese Tech Spec schließt eine Lücke in der Umsetzung.

## Architektur

```text
internal/desk/agent.go             TurnRequest.AllowedWrites; decide prüft je Stufe die passende Liste; Pending.PreviewOnly
apps/desk/src/app/core/types.ts    TurnRequest.allowed_writes, Pending.preview_only
apps/desk/src/app/core/chat-session.ts   allowedWrites; decide('session') wählt die Liste nach pending.writes
apps/desk/src/app/chat/approval-card.ts  Badge und Hinweis „nur Vorschau“
handbuch/04-agenten.md             Abschnitt „Was der Desktop darf“
```

### Backend (AC-1, AC-2)

`TurnRequest` erhält `AllowedWrites []string` (`json:"allowed_writes"`). `allowed_tools` behält Namen und Format, gilt aber nur noch für die Stufe „lesend“.

`decide` bestimmt die Freigabe nach der Stufe aus `tier(tc)`:

| Stufe | läuft ohne Karte, wenn … |
|---|---|
| `read` (auch Vorschau ohne `apply`) | Aufruf in `ApprovedCalls`, oder Werkzeug in `AllowedTools` oder in `AllowedWrites` |
| `write` | Aufruf in `ApprovedCalls`, oder Werkzeug in `AllowedWrites` |
| `critical` | Aufruf in `ApprovedCalls` (unverändert) |

Ein älterer Client, der ein schreibendes Werkzeug nur in `allowed_tools` schickt, bekommt den Aufruf mit Vorschau als `Pending` zurück. Das ist die sichere Richtung: Es wird eher einmal zu oft gefragt, aber nie ohne gezeigte Vorschau geschrieben.

`Pending` erhält `PreviewOnly bool` (`json:"preview_only,omitempty"`). Das Feld ist wahr, wenn die Stufe `read` ist und das Werkzeug in `writeTools` oder `criticalTools` steht.

### Oberfläche (AC-3, AC-4)

`ChatSession` führt neben `allowedTools` die Liste `allowedWrites` und schickt beide mit. `decide(callId, 'session')`:

- bei `critical`: herabgestuft auf `once` (unverändert);
- bei `pending.writes`: Werkzeug in `allowedWrites`;
- sonst: Werkzeug in `allowedTools`.

Die Freigabekarte zeigt bei `preview_only` das Badge „nur Vorschau“ statt „nur lesend“ und den Hinweis: „Das Modell will sehen, was geschrieben würde; geschrieben wird dabei nichts. Für die Sitzung freigegeben, gilt das nur für Vorschauen dieses Werkzeugs, Schreiben wird erneut gefragt.“ Der Text auf schreibenden Karten bleibt.

### Handbuch (AC-5)

Im Abschnitt „Was der Desktop darf“ wird der Punkt zur Freigabe ergänzt. Eine Freigabe für die Sitzung gilt je Werkzeug und je Stufe: Auf einer lesenden Karte oder einer Vorschau erlaubt sie nur Lesen bzw. Vorschauen. Auf einer Karte mit Schreibvorgang erlaubt sie auch das Schreiben.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `agent.go` (`decide`, `TurnRequest.AllowedWrites`) | `TestPreviewSessionDoesNotAllowWrites`, `TestSessionAllowsWrites` |
| AC-2 | `agent.go` (`decide`) | `TestPreviewSessionDoesNotAllowWrites` |
| AC-3 | `chat-session.ts` | `chat-session.spec.ts` |
| AC-4 | `agent.go` (`Pending.PreviewOnly`), `approval-card.ts` | `TestPreviewSessionDoesNotAllowWrites`, `approval-card.spec.ts` |
| AC-5 | `handbuch/04-agenten.md` | `TestHandbookApprovalScope` |

## Risiken

- **Mehr Rückfragen:** Wer bisher eine Vorschau für die Sitzung freigegeben hat, sieht beim ersten Schreiben dieses Werkzeugs wieder eine Karte. Das ist gewollt, die Karte zeigt dann den Inhalt.
- **Vorschau vor dem Schreiben:** Ein Werkzeug in `allowed_writes` führt Vorschauen ohne Karte aus. Die Vorschau schreibt nichts, also ist das unkritisch.
