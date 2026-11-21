# Backlog — Phase 7: Modelle und lokale CLIs im Desktop

Aufgenommen am 2026-11-20, nach dem Merge von PR #13 (`v0.8.0`). Das war Phase 7
des damaligen Plans und ist **nicht** in 0.8.0 enthalten.

Ziel: Der KI-Desktop soll alle lokal verfügbaren Modelle wirklich anbieten — und
auf Wunsch auch eine lokal installierte Agenten-CLI benutzen können, ohne die
Freigabezusagen aus ADR 0003 zu verlieren.

## Teil A — Ollama, das ehrlich „verfügbar“ ist

Der lokale Anbieter `ollama-local` (`internal/desk/config.go`, `providerSpecs`)
ist vorhanden, und `handleMeta` (`internal/desk/server.go`) mischt die Modelle
aus `GET /api/tags` (`ListLocalModels` in `internal/desk/provider.go`) in die
Auswahlliste. Drei Lücken bleiben:

1. **Keine Auffrischung.** Die Liste entsteht nur beim Laden von `/api/meta`.
   Ein gerade geholtes Modell (`ollama pull`) fehlt bis zum Neustart des
   Desktops. Nötig: ein Weg, die Anbieterliste neu zu lesen — z. B.
   `GET /api/providers/refresh` oder ein Knopf im Kopfbereich neben der
   Modellauswahl (`apps/desk/src/app/app.ts`).

2. **Kein Filter nach Fähigkeiten.** `nomic-embed-text` und ähnliche Modelle
   erscheinen als wählbar und fallen erst beim Test durch, weil sie keine
   Werkzeuge aufrufen. Nötig: `POST /api/show` je Modell und Auswertung von
   `capabilities` (`tools`) — Modelle ohne Werkzeugaufruf entweder ausblenden
   oder sichtbar als „nicht für das Gespräch" kennzeichnen. Der Modelltest in
   `internal/desk/modeltest.go` prüft das schon, aber erst auf Klick.

3. **Keine Empfehlung je Ablauf.** Die README hält fest, dass `qwen3:14b` und
   `qwen3:32b` an `sdd-conflicts` scheitern (erfundene Zitate, Widersprüche
   nicht gefunden). Der Desktop bietet den Ablauf trotzdem mit jedem Modell an.
   Nötig: die Ablaufkarte nennt „mit diesem Modell nicht belegt", statt ein
   Ergebnis zu versprechen, das der belegte Stand nicht deckt.

Nebenpunkt: `ollamaChat` sendet kein `think` und liest `thinking` nur aus
Antworten. Für Reasoning-Modelle wäre ein Schalter sinnvoll; die
Stream-Pipeline (`type: "thinking"`) kann das bereits.

## Teil B — Lokale Agenten-CLIs (`claude`, `agy`, `codewhale`)

Heute spricht der Desktop ausschließlich HTTP: `desk.NewChat` kennt drei
Protokolle (`ollama`, `openai`, `claude`). Eine CLI ist kein weiterer Eintrag in
`providerSpecs`, weil die CLIs **selbst Agenten** sind: Sie führen Werkzeuge
aus, statt sie als `tool_calls` zurückzugeben — und der Desktop orchestriert den
Werkzeugkreislauf selbst, mit seinen Freigabekarten dazwischen
(`internal/desk/agent.go`: `readTools`, `writeTools`, `criticalTools`).

Drei Varianten wurden erwogen:

- **A — CLI als reine Textmaschine.** Klein im Aufwand, aber der Desktop müsste
  den Werkzeugkatalog in den Prompt schreiben und Antworten parsen. Brüchig;
  taugt höchstens als „zweite Meinung", nicht als Agent.
- **B — CLI als eigener Agent, Desktop nur Hülle.** Die Freigaben wandern in das
  fremde CLI; die Karten des Desktops sehen nichts. Damit wären die Zusagen aus
  ADR 0003 (jede kritische Aktion einzeln bestätigt) nicht mehr wahr. Bräuchte
  ein eigenes ADR, das ADR 0003 für diesen Betriebsmodus einschränkt.
- **C — Desktop hostet den MCP-Zugang als Proxy (empfohlen).** `sdd-desk`
  betreibt einen MCP-Endpunkt auf `127.0.0.1` (Token wie heute), das CLI wird
  darauf gezeigt; jeder `tools/call` wird zur Freigabekarte im Desktop und geht
  erst nach der Entscheidung an `sdd mcp` weiter. Damit bleiben die Freigaben
  gültig und `--allow-critical` muss das CLI nie bekommen. Größter Aufwand:
  Proxy, Kartenkorrelation über Prozessgrenzen, Sitzungszuordnung, Timeouts.

Zwei Punkte gelten unabhängig von der Variante:

- **`claude` ist nicht lokal.** Der Prozess läuft lokal, das Modell nicht. Er
  muss als Cloud-Anbieter klassifiziert werden (Hinweis „Inhalte verlassen das
  Gerät" bleibt). Dasselbe gilt für `agy` und `codewhale`, solange deren Routing
  nicht belegbar lokal ist.
- **Der Headless-Vertrag von `agy` und `codewhale` ist unbekannt.** Für den Plan
  gebraucht: Aufruf ohne Interaktion, Ausgabeformat (JSON-Stream oder Text),
  Abbruchsignal, und ob sie einen MCP-Server annehmen. Für `claude` ist das
  bekannt, für die beiden anderen nicht — nicht raten, sondern beim Hersteller
  nachsehen.

## Art des Vorhabens

Ein eigenes Feature mit PRD und Tech Spec (`docs/prd/`, `docs/design/`), weil es
Konfiguration, Kern und Oberfläche berührt. Teil A ist klein und unabhängig
lieferbar; Teil B braucht vorher ein ADR zur Frage, wem die Freigabe gehört,
wenn ein fremder Agent die Werkzeuge aufruft.

Teil A ist als `desk-model-availability` umgesetzt (v0.9-Kandidat). Für Teil B
liegt der Vorschlag als [ADR 0010](../adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md)
vor: MCP-Proxy im Desktop, damit ADR 0003 unverändert gilt. Er ist `proposed` —
entschieden wird er vom Menschen, nicht durch dieses Dokument.
