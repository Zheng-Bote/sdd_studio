---
feature: desk-model-availability
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - README.md
  - CHANGELOG.md
  - docs/arc42.md
---

# Tech Spec — Modelle im Desktop, die ehrlich verfügbar sind (`desk-model-availability`)

Setzt das PRD `docs/prd/desk-model-availability.md` um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

Alles bleibt in der Desk-Schicht; `internal/sddcore`, `internal/mcp` und `cmd/sdd` werden nicht berührt. `sdd` bleibt KI-frei und offline.

```text
internal/desk/provider.go     parseTags, parseCapabilities (reine Funktionen),
                              ModelCapabilities (POST /api/show)
internal/desk/server.go       modelView/providerView, mergeLocalModels, /api/meta
internal/desk/evidence.go     Belegstand-Tabelle, JudgmentFlows, EvidenceFor
apps/desk/src/app.ts          Knopf „Modelle aktualisieren", Kennzeichnung in der Auswahl
apps/desk/src/app/core/*      Meta-Typen, ChatSession.refreshMeta, Vorauswahl
apps/desk/src/app/process/*   Belegstand auf den Karten von sdd-conflicts und sdd-review
```

Datenfluss: Der Desktop liest `/api/meta` (ein Aufruf je Sitzung und je Klick auf den Knopf). Der Handler fragt `/api/tags` und je Modell `/api/show` beim lokalen Ollama ab, mischt die Fähigkeiten in die Modellliste und legt die statische Belegstand-Tabelle dazu. Die Oberfläche rechnet daraus die Kennzeichnung und den Belegstand; sie schätzt nichts.

Das Auffrischen braucht keinen neuen Endpunkt: `handleMeta` liest `/api/tags` schon bei jedem Aufruf, eingefroren war allein die Oberfläche, die `/api/meta` nur beim Start holte. Ein Knopf, der `meta()` erneut aufruft, ist die kleinste Mechanik, die das Problem löst.

## Schnittstellen

**`GET /api/meta`** — `providers[].models` wechselt von `string[]` auf Objekte; beide Enden liegen in diesem Repository und werden gemeinsam ausgeliefert:

```json
{
  "judgment_flows": ["sdd-conflicts", "sdd-review"],
  "providers": [
    {
      "id": "ollama-local", "name": "Ollama (lokal)", "protocol": "ollama",
      "endpoint": "http://localhost:11434", "cloud": false, "available": true,
      "models": [
        {
          "name": "qwen3:14b",
          "tools": true,
          "evidence": {
            "sdd-conflicts": {
              "flow": "sdd-conflicts", "protocol": "ollama", "model_prefix": "qwen3:",
              "status": "failed",
              "note": "qwen3:14b erfand Zitate; qwen3:32b fand nach 47 Minuten nichts",
              "source": "README, Nächste Schritte"
            }
          }
        },
        { "name": "nomic-embed-text", "tools": false },
        { "name": "neu:1b" }
      ]
    }
  ]
}
```

Der Server rechnet den Belegstand je Modell aus (`annotateEvidence`), damit die Sicherheitszusage an einer Stelle liegt und in Go geprüft wird: Ein Modell ohne Eintrag trägt kein `evidence`, die Oberfläche zeigt dann „nicht belegt". `judgment_flows` sagt der Oberfläche, für welche Abläufe sie überhaupt etwas anzeigt.

`tools` fehlt, wenn der Wert unbekannt ist — nicht `false`. Der Unterschied ist die ganze Aussage: „kann keine Werkzeuge" ist ein Befund, „weiß ich nicht" ist keiner.

**`POST /api/show` (Ollama)** — Anfrage `{"model": "<name>"}`; ausgewertet wird `capabilities`. Fehlt der Schlüssel (Ollama vor 0.5), gilt der Wert als unbekannt. Fehler und Zeitüberschreitung ebenso; ein nicht erreichbarer Dienst macht die Auswahl nicht leer.

**Budget** — `/api/meta` bleibt bedienbar: `/api/tags` mit 1,5 s wie bisher, die `/api/show`-Aufrufe parallel unter einer Gesamtfrist von 2 s. Was nicht rechtzeitig antwortet, bleibt ohne `tools`.

## Datenmodell

```go
// internal/desk/server.go
type modelView struct {
    Name     string                   `json:"name"`
    Tools    *bool                    `json:"tools,omitempty"` // nil = unbekannt
    Evidence map[string]EvidenceEntry `json:"evidence,omitempty"`
}

type providerView struct { // ersetzt das Einbetten von Provider: models ist jetzt eine Objektliste
    ID, Name, Protocol, Endpoint string
    Cloud, Available             bool
    Models                      []modelView
    Note                        string
}

// internal/desk/evidence.go
const (
    EvidenceProven   = "proven"
    EvidenceFailed   = "failed"
    EvidenceUnproven = "unproven"
)

type EvidenceEntry struct {
    Flow        string `json:"flow"`
    Protocol    string `json:"protocol"`
    ModelPrefix string `json:"model_prefix,omitempty"`
    Status      string `json:"status"`
    Note        string `json:"note"`
    Source      string `json:"source"`
}
```

`FlowEvidence` ist eine feste Tabelle im Code, keine Konfiguration und keine Ableitung aus Modellnamen. Sie wächst nur, wenn ein Test belegt ist — Einträge ohne `Source` sind ungültig und werden von `EvidenceFor` nicht geliefert.

`EvidenceFor(protocol, model, flow)` wählt den Treffer mit dem längsten passenden `ModelPrefix`; ohne Treffer meldet es „nichts belegt". `annotateEvidence` schreibt das Ergebnis an das Modell, sodass die Oberfläche nur nachschlägt und nichts selbst entscheidet. `JudgmentFlows` begrenzt, wo das überhaupt angezeigt wird.

Persistenz: keine. Es gibt keinen Cache, keine Datei und keine Migration; die Tabelle ist Teil des Binaries.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `ChatSession.refreshMeta()` ruft `meta()` erneut, behält Anbieter/Modell bei, wenn sie noch angeboten werden, sonst Vorauswahl neu; Knopf im Kopfbereich (`app.ts`). | `chat-session.spec.ts` „adds a pulled model on refresh and keeps the selection", „falls back to a remaining model when the selected one is gone" |
| AC-2 | `parseCapabilities` + `ModelCapabilities` in `provider.go`; `modelView.Tools` als `*bool`; unbekannt bleibt `nil`. | `provider_test.go` `TestParseCapabilities`, `desk_test.go` `TestMetaLocalCapabilities` |
| AC-3 | `preferredModel` in `chat-session.ts` wählt erst `tools === true`, dann unbekannt, dann `false`; `modelLabel` hängt „(ohne Werkzeuge)" an den Eintrag. | `chat-session.spec.ts` „prefers a model that calls tools and labels one that does not" |
| AC-4 | `annotateEvidence` schreibt den Belegstand an das Modell; `ProcessPanel` zeigt ihn je Karte von `JudgmentFlows`. | `evidence_test.go` `TestAnnotateEvidence`, `desk_test.go` `TestMetaEvidence`, `process-panel.spec.ts` „shows the evidence for the selected model on judgment flows only" |
| AC-5 | `FlowEvidence` mit `Source`; `EvidenceFor` nimmt den längsten Präfix und ignoriert Einträge ohne Quelle oder mit unbekanntem Status. | `evidence_test.go` `TestEvidenceFor`, `TestEvidenceInMostSpecific`, `TestFlowEvidenceIsValid` |
| AC-6 | `mergeLocalModels` behält die konfigurierten Modelle bei, wenn `/api/tags` scheitert; `available` und Hinweis bleiben wie bisher (wahr, solange Modelle konfiguriert sind). | `provider_test.go` `TestMergeLocalModels`, `desk_test.go` `TestMetaOllamaUnreachable` |

## Sicherheit

- **Keine neue Angriffsfläche nach außen.** Neu ist ein `POST` an den lokalen Ollama-Dienst; Adresse und Modellname kommen aus der Konfiguration bzw. aus `/api/tags`, nie aus dem Gespräch. Der Endpunkt bleibt `127.0.0.1`-gebunden und tokenpflichtig wie jeder andere Desk-Aufruf.
- **Keine neuen Schreibwege.** Der Knopf liest; er ändert weder Modellbestand noch Projektdateien.
- **Keine Inhalte an neue Empfänger.** `/api/show` überträgt nur den Modellnamen; der Hinweis für Cloud-Anbieter („Inhalte verlassen das Gerät") bleibt unverändert.
- **Keine erfundene Auskunft.** Der Belegstand ist eine Tabelle mit Quelle, keine Aussage des Modells; `tools` fehlt, wenn es unbekannt ist, statt auf `true` zu fallen.

## Offene Punkte

- **Nachführung der Belegstände.** Die Tabelle lebt im Code; ein neuer Test mit einem lokalen Modell wird zu einem Commit. Ein externer Ort (Datei, Konfiguration) wäre bequemer, aber auch ein Weg, Belege ohne Prüfung zu ändern. Bis auf Weiteres: Code.
- **`sdd-review` hat keinen Beleg.** Der Ablauf steht in `JudgmentFlows`, aber kein Modell wurde damit getestet; die Karte sagt folgerichtig „nicht belegt". Ein Test würde das ändern.
- **Ältere Ollama-Versionen** liefern keine `capabilities`. Dann bleibt `tools` für alle Modelle unbekannt und die Kennzeichnung aus — die Auswahl verhält sich wie vor dieser Änderung.
