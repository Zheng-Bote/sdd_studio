---
status: proposed
date: 2026-11-21
supersedes: []
references:
  - constitution.md
  - docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md
  - docs/notes/backlog-phase-7-modelle-und-clis.md
---

# ADR 0010 — Lokale Agenten-CLIs nur über einen MCP-Proxy des Desktops

## Kontext

Phase 7 Teil B der [Backlog-Notiz](../notes/backlog-phase-7-modelle-und-clis.md) will lokal installierte Agenten-CLIs (`claude`, `agy`, `codewhale`) im KI-Desktop nutzbar machen. Das ist keine Frage des Anbieter-Adapters: `internal/desk` spricht heute ausschließlich HTTP (`desk.NewChat` kennt `ollama`, `openai`, `claude`), und der Desktop führt den Werkzeugkreislauf selbst — er stuft jeden Aufruf ein (`readTools`, `writeTools`, `criticalTools` in `internal/desk/agent.go`), zeigt die Freigabekarte und reicht erst nach der Entscheidung an `sdd mcp` weiter.

Eine CLI ist dagegen **selbst ein Agent**: Sie plant, ruft Werkzeuge auf und schreibt Dateien in ihrer eigenen Schleife. Damit stellt sich nicht die Frage nach einem weiteren Protokoll, sondern die nach der Freigabehoheit: Wessen Karte sieht der Mensch, wenn ein fremder Agent `sdd_write` aufruft?

Die Zusage aus [ADR 0003](0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md) ist eng: drei Stufen, kritische Aktionen einzeln bestätigt, nie für die Sitzung. Sie gilt für den Desktop, weil der Desktop jeden Aufruf sieht.

Erschwerend, und für die Entscheidung wesentlich:

- **`sdd mcp` spricht nur stdio** (`srv.Serve(os.Stdin, stdout)` in `cmd/sdd/mcp.go`). Es gibt keinen HTTP-Transport, an den man eine CLI einfach zeigen könnte.
- **`claude` ist nicht lokal.** Der Prozess läuft lokal, das Modell nicht. Für den bestehenden Desktop heißt das: Cloud-Anbieter, Hinweis „Inhalte verlassen das Gerät". Dasselbe gilt für `agy` und `codewhale`, solange deren Routing nicht belegbar lokal ist.
- **Der Headless-Vertrag von `agy` und `codewhale` ist unbekannt:** Aufruf ohne Interaktion, Ausgabeformat, Abbruchsignal, Annahme eines MCP-Servers. Für `claude` ist er dokumentiert, für die beiden anderen nicht. Hier wird nicht geraten.

Erwogen wurden drei Varianten:

1. **CLI als reine Textmaschine.** Klein im Aufwand — der Desktop schreibt seinen Werkzeugkatalog in den Prompt und parst die Antwort. Brüchig: Die CLI ist dann kein Agent, sondern ein Textgenerator, dessen Werkzeugwünsche man aus Prosa zurückgewinnen muss. Als „zweite Meinung" denkbar, als Agent nicht.
2. **CLI als eigener Agent, Desktop nur Hülle.** Die Freigaben wandern in das fremde CLI; die Karten des Desktops sehen nichts. Damit wäre ADR 0003 für diesen Betriebsmodus nicht mehr wahr — es bräuchte ein ADR, das die Zusage einschränkt.
3. **Der Desktop hostet den MCP-Zugang als Proxy.** `sdd-desk` stellt einen MCP-Endpunkt auf `127.0.0.1` bereit (Token wie heute), die CLI wird darauf gezeigt, und jeder `tools/call` wird zur Freigabekarte im Desktop. Erst nach der Entscheidung geht der Aufruf an `sdd mcp`.

## Entscheidung

1. **Variante 3, und nur sie.** Der Desktop wird für Agenten-CLIs zum MCP-Proxy. Damit bleibt ADR 0003 unverändert gültig: Der Mensch bestätigt weiterhin im Desktop, und `--allow-critical` bekommt die CLI nie — der Kindprozess `sdd mcp` läuft unter der Kontrolle von `sdd-desk`.
2. **Der Proxy ist Teil von `sdd-desk`, nicht von `sdd`.** `sdd` bleibt KI-frei und offline (ADR 0003, Satz 1). Der Proxy lebt dort, wo schon die Freigabekarten und das Sitzungstoken liegen; `internal/mcp` bekommt keinen zweiten Transport.
3. **Jeder `tools/call` ist ein Kartenvorgang.** Der Proxy stuft den Aufruf mit derselben Einordnung ein wie die eigene Schleife (`internal/desk/agent.go`) und liefert ohne Freigabe keinen Inhalt, sondern wartet. Eine Sitzungsfreigabe gilt wie bisher nur für die Stufe der Karte, auf der sie erteilt wurde; kritische Aktionen nur einzeln.
4. **Aufrufe werden über eine Kennung korreliert, nicht über die Reihenfolge.** Über Prozessgrenzen hinweg ist „der nächste Aufruf gehört zu Karte X" nicht belastbar: Die CLI kann parallel arbeiten, wiederholen oder abbrechen. Der Proxy vergibt je Aufruf eine Kennung, führt sie in der Karte und im Protokoll und beantwortet genau den Aufruf, für den entschieden wurde.
5. **Eine CLI wird nur angeboten, wenn zwei Bedingungen belegt sind.** Beides wird geprüft, nicht angenommen:
   - **Sie lässt sich headless starten** und nimmt einen MCP-Server an (Aufruf ohne Interaktion, Ausgabeformat, Abbruchsignal dokumentiert).
   - **Sie bringt keine eigenen Werkzeuge mit, die am Proxy vorbeischreiben.** Ein Agent mit eigenem Dateizugriff kann die Zusage aus ADR 0003 nicht halten, gleichgültig wie sauber der Proxy ist. Lässt sich der Eigenzugriff nicht abschalten oder auf das Projektverzeichnis ohne Schreibrecht beschränken, wird die CLI nicht angeboten.
6. **Cloud bleibt Cloud.** `claude` und — bis zum Beleg des Gegenteils — `agy` und `codewhale` werden als Cloud-Anbieter klassifiziert: Hinweis vor dem ersten Zug, keine stillschweigende Lokalität.

## Konsequenzen

- **Kein neues ADR für eine Ausnahme.** Variante 2 hätte ADR 0003 einschränken müssen; diese Entscheidung braucht das nicht. Der Satz „jede kritische Aktion wird einzeln bestätigt" bleibt ohne Zusatz wahr.
- **Neuer Baustein in `internal/desk`:** ein MCP-Server-Endpunkt auf `127.0.0.1`, der `tools/list` aus dem Kindprozess durchreicht und `tools/call` erst nach der Entscheidung ausführt. Sicherheitsregeln wie für die bestehende Desk-API: Bindung an Loopback, `Host`-Prüfung, Sitzungstoken, Fristen je Aufruf.
- **Der Werkzeugkatalog bleibt eine Quelle.** Die CLI sieht genau die Werkzeuge, die `sdd mcp` anbietet; der Desktop schreibt keinen zweiten Katalog.
- **Risiko: Kartenflut.** Ein Agent, der in einem Zug zwanzig Werkzeuge aufruft, erzeugt zwanzig Karten. Gegenmaßnahme ist die bestehende Stufenlogik (lesend ohne Karte, Vorschau vor dem Schreiben), nicht eine pauschale Freigabe.
- **Risiko: Abbruch und Timeout über Prozessgrenzen.** Stirbt die CLI, muss der Proxy offene Karten zurücknehmen, statt auf eine Entscheidung zu warten, die niemand mehr sieht. Das braucht eine explizite Frist je Aufruf.
- **Risiko: Belegstand.** Ob eine CLI die Abläufe wirklich trägt, ist danach so unbelegt wie bei den lokalen Modellen. Die Belegstand-Tabelle aus `desk-model-availability` (`internal/desk/evidence.go`) gilt entsprechend: ohne Test „nicht belegt", nie „belegt".
- **Nicht entschieden und nicht beauftragt:** welche CLI zuerst kommt, wann Teil B umgesetzt wird und wie der Belegstand je CLI erhoben wird. Dieses ADR entscheidet den Weg, nicht den Fahrplan. Die Umsetzung braucht ein eigenes Feature mit PRD und Tech Spec.

## Offene Punkte

- **Headless-Verträge von `agy` und `codewhale`.** Beim Hersteller nachsehen, nicht raten; ohne Beleg keine Aufnahme in die Anbieterliste.
- **Eigene Werkzeuge der CLIs.** Je CLI prüfen, ob sich der Eigenzugriff abschalten lässt; das Ergebnis gehört in die Tech Spec des Features, nicht in dieses ADR.
- **Sitzungszuordnung.** Ob eine CLI-Sitzung einer Desktop-Sitzung entspricht oder mehrere erlaubt, entscheidet der Proxy; die Karten müssen auch dann eindeutig bleiben, wenn zwei CLIs gleichzeitig laufen.
