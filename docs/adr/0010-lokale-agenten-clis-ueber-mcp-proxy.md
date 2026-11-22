---
status: accepted
date: 2026-11-21
supersedes: []
references:
  - constitution.md
  - docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md
  - docs/notes/backlog-phase-7-modelle-und-clis.md
  - docs/notes/headless-vertraege-agenten-clis.md
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
   - **Ihre eigenen Schreibzugriffe sind mechanisch verhindert**, nicht bloß unerwünscht. Die Tech Spec nennt den Mechanismus: abgeschaltete Datei- und Shell-Werkzeuge, ein Nur-Lese-Mount des Projektverzeichnisses, eine Sandbox ohne Schreibrecht. Geprüft wird nicht die Abwesenheit eigener Werkzeuge, sondern die Wirksamkeit der Sperre.

     Erwogen war, eine CLI nur anzubieten, wenn sie gar keine eigenen Werkzeuge mitbringt, die am Proxy vorbeischreiben. Verworfen: Für die genannten CLIs ist das nicht erfüllbar — Datei- und Shell-Werkzeuge sind ihr Kern, nicht ihre Beigabe; man könnte sie nicht entfernen, ohne die CLI auf einen Textgenerator zu reduzieren (Variante 1). Eine Bedingung, die die Entscheidung selbst aufhebt, ist keine Bedingung. Geprüft wird deshalb die Sperre, nicht die Abwesenheit.
6. **Cloud bleibt Cloud.** `claude` und — bis zum Beleg des Gegenteils — `agy` und `codewhale` werden als Cloud-Anbieter klassifiziert: Hinweis vor dem ersten Zug, keine stillschweigende Lokalität.

## Konsequenzen

- **Kein neues ADR für eine Ausnahme.** Variante 2 hätte ADR 0003 einschränken müssen; diese Entscheidung braucht das nicht. Der Satz „jede kritische Aktion wird einzeln bestätigt" bleibt ohne Zusatz wahr.
- **Neuer Baustein in `internal/desk`:** ein MCP-Server-Endpunkt auf `127.0.0.1`, der `tools/list` aus dem Kindprozess durchreicht und `tools/call` erst nach der Entscheidung ausführt. Sicherheitsregeln wie für die bestehende Desk-API: Bindung an Loopback, `Host`-Prüfung, Sitzungstoken, Fristen je Aufruf.
- **Der Werkzeugkatalog bleibt eine Quelle.** Die CLI sieht genau die Werkzeuge, die `sdd mcp` anbietet; der Desktop schreibt keinen zweiten Katalog.
- **Risiko: Kartenflut.** Ein Agent, der in einem Zug zwanzig Werkzeuge aufruft, erzeugt zwanzig Karten. Gegenmaßnahme ist die bestehende Stufenlogik (lesend ohne Karte, Vorschau vor dem Schreiben), nicht eine pauschale Freigabe.
- **Risiko: Abbruch und Timeout über Prozessgrenzen.** Stirbt die CLI, muss der Proxy offene Karten zurücknehmen, statt auf eine Entscheidung zu warten, die niemand mehr sieht. Das braucht eine explizite Frist je Aufruf.
- **Risiko: Belegstand.** Ob eine CLI die Abläufe wirklich trägt, ist danach so unbelegt wie bei den lokalen Modellen. Die Belegstand-Tabelle aus `desk-model-availability` (`internal/desk/evidence.go`) gilt entsprechend: ohne Test „nicht belegt", nie „belegt".
- **Die Zusage im README stimmt danach nicht mehr, wenn sie bleibt, wie sie ist.** Dort steht: „**Nur die Werkzeuge von `sdd mcp`**, keine eigenen, kein Dateizugriff daran vorbei." Für ein gehostetes CLI gilt das nur für Aktionen **über `sdd`**; was die CLI mit ihren eigenen Werkzeugen schreibt, sieht keine Karte, keine Regelprüfung und kein Gate. Vor der Umsetzung wird die Aussage deshalb eingeschränkt („kein Dateizugriff an `sdd` vorbei") und der CLI-Modus in der Oberfläche sichtbar gekennzeichnet. Eine Zusage, die stillschweigend nicht mehr stimmt, ist der größere Schaden als die Einschränkung.
- **Nachtrag 2026-11-22:** Beide Kandidaten werden genommen, `claude` und `codewhale`. Damit ist auch die in den Konsequenzen offen gelassene Frage entschieden, welche CLI zuerst kommt: nicht zuerst, sondern beide. Grundlage ist der Belegstand unten; `agy` bleibt zurückgestellt.

## Vor der Entscheidung

Dieses ADR blieb `proposed`, bis zwei Dinge geklärt waren. Sie entschieden, ob die Entscheidung überhaupt einen Gegenstand hat:

1. **Headless-Verträge.** Für `agy` und `codewhale` war unbelegt, ob sie ohne Interaktion starten, welches Ausgabeformat sie haben, wie sie abbrechen und ob sie einen MCP-Server annehmen. Für `claude` war das dokumentiert — aber `claude` ist Cloud, das Lokalitätsargument trägt dort nicht.
2. **Wirksame Schreibsperre.** Je CLI ist zu belegen, mit welchem Mechanismus eigene Schreibzugriffe verhindert werden (Entscheidung 5b). Ein Mechanismus, der auf gutem Willen oder auf einer Einstellung beruht, die die CLI selbst ändern kann, zählt nicht.

Kommt kein Kandidat durch dieses Tor, lautet die richtige Entscheidung **„keine CLI-Integration"**. Auch das ist ein Ergebnis, und es ist billiger als ein Proxy ohne Nutznießer. Der Bau des Proxys beginnt erst nach dieser Klärung, nicht davor.

### Belegstand vom 2026-11-22

Die Recherche ist gelaufen; Belege und Versuchsaufbau stehen in [`docs/notes/headless-vertraege-agenten-clis.md`](../notes/headless-vertraege-agenten-clis.md). Geprüft wurden `claude` 2.1.330, `agy` 1.5.9 und `codewhale` 0.10.3 gegen die lokal installierten Programme; der codewhale-Lauf mit Modell wurde nachgeholt, sobald ein Anbieterschlüssel bereitstand.

| CLI | 5a headless + MCP | 5b wirksame Schreibsperre | Ergebnis |
|---|---|---|---|
| `claude` | belegt | belegt (`--allowedTools` ist eine Allowlist; Shell-Schreibversuch als `permission_denials` abgewiesen, Datei nicht angelegt) | **besteht** |
| `agy` | belegt, Abbruchsignal nur Exit 1 (nicht unterscheidbar) | **nicht belegt** — Standardmodus führt Shell-Schreibbefehle ohne Freigabe aus; `--mode plan` sperrt den Schreibzugriff, aber auch MCP | besteht nicht |
| `codewhale` | belegt (`exec` headless, Exit 0; MCP-Werkzeug aufgerufen: `PROBE-MCP-OK`; `SIGTERM` → Exit 143) | belegt (`--sandbox read-only` verweigert den Shell-Schreibversuch mit `Operation not permitted`, Datei nicht angelegt) | **besteht** |

Damit ist die Lage nicht mehr „unbekannt", sondern entscheidbar:

- **`claude` ist der erste Kandidat.** Beide Bedingungen belegt. Es bleibt Cloud: der Hinweis „Inhalte verlassen das Gerät" gilt, und die README-Zusage aus den Konsequenzen ist vor der Umsetzung einzuschränken.
- **`codewhale` ist der passendste Vertrag für den Proxy** (`--parent-death-watch` beantwortet das Abbruchrisiko, `--max-turns`/`--max-tool-calls` die Kartenflut). Beide Bedingungen sind belegt: MCP-Aufruf und Shell-Schreibversuch liefen im selben Lauf gegeneinander, der Schreibversuch scheiterte an der OS-Sandbox. Offen bleibt eine Design-Frage für den Proxy: `--allowed-tools` nimmt als Allowlist auch den Aufrufpfad (`tool_search`) weg — die Liste muss ihn einschließen, sonst kann selbst das erlaubte MCP-Werkzeug nicht geladen werden.
- **`agy` wird zurückgestellt.** Ohne belegte Schreibsperre kein Angebot.

Über Annahme, Verwerfung oder Einschränkung entscheidet der Mensch. **Angenommen am 2026-11-22**: Beide Kandidaten werden genommen — `claude` und `codewhale`; `agy` bleibt zurückgestellt.

## Offene Punkte

- **Sitzungszuordnung.** Ob eine CLI-Sitzung einer Desktop-Sitzung entspricht oder mehrere erlaubt, entscheidet der Proxy; die Karten müssen auch dann eindeutig bleiben, wenn zwei CLIs gleichzeitig laufen.
