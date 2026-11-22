# Headless-Verträge der Agenten-CLIs — Belegstand für ADR 0010

Untersucht am 2026-11-22 auf macOS (arm64) gegen die lokal installierten Programme. Das ist die Recherche, die [ADR 0010](../adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md) unter „Vor der Entscheidung" verlangt: fünf Fragen je CLI, jede mit Beleg — nicht mit Vermutung.

Untersuchte Stände: `claude` 2.1.330 (Claude Code), `agy` 1.5.9 (Google Antigravity), `codewhale` 0.10.2.

**Die fünf Fragen:** headless startbar · Ausgabeformat · Abbruchsignal · MCP-Annahme · wirksame Schreibsperre.

## Ergebnis in einer Zeile je CLI

- **`claude`** — besteht beide Bedingungen. Headless belegt, MCP belegt, Schreibsperre belegt: der Werkzeugkatalog ist eine Allowlist, alles Ungenannte wird ohne Rückfrage verweigert.
- **`agy`** — besteht **nicht**. Headless und MCP sind belegt, aber die Schreibsperre ist es nicht: im Standardmodus führt es einen Shell-Schreibbefehl ohne Freigabe aus, und der Plan-Modus, der ihn sperrt, sperrt auch MCP — damit ist er für den Proxy unbrauchbar.
- **`codewhale`** — besteht **wahrscheinlich**, ist aber nicht vollständig belegt. Der Mechanismus existiert und ist betriebssystemseitig durchgesetzt (erstbelegt), aber ein vollständiger Lauf mit Modell war hier nicht möglich: es ist kein Anbieterschlüssel hinterlegt.

## `claude` 2.1.330 — Cloud

| Frage | Beleg | Ergebnis |
|---|---|---|
| Headless | `claude -p "…"` → Antwort, Exit 0 | ja |
| Ausgabeformat | `--output-format text\|json\|stream-json`, `--input-format` | ja, JSON mit `result`, `usage`, `total_cost_usd`, `permission_denials` |
| Abbruchsignal | `SIGTERM` während eines Laufs → Exit **143** | ja, unterscheidbar von Fehler (1) |
| MCP-Annahme | `--mcp-config <datei>` mit `mcpServers`; stdio-Server eingebunden und aufgerufen | ja |
| Schreibsperre | `--allowedTools "mcp__probe__probe_read"` | **ja** |

Der entscheidende Versuch: ein MCP-Probeserver mit einem rein lesenden Werkzeug, dazu die Anweisung, dieses Werkzeug aufzurufen **und danach** per Shell eine Datei anzulegen.

```text
Ergebnis: "PROBE-MCP-OK"            ← MCP-Aufruf ging durch
permission_denials: [{ tool_name: "Bash",
                       tool_input: { command: "touch …/probe4.txt" } }]
Datei probe4.txt: existiert nicht
```

Das ist genau die gesuchte Kombination. `--allowedTools` ist eine **Allowlist**: Nur die genannten Werkzeuge stehen dem Modell überhaupt zur Verfügung, jedes andere wird abgewiesen, ohne dass jemand gefragt wird. Der Desktop setzt die Liste — die CLI kann sie nicht selbst erweitern, und `--dangerously-skip-permissions` wird vom Desktop schlicht nicht übergeben.

Ein Nebenbefund, der für den Proxy zählt: Die Verweigerung erscheint im JSON als `permission_denials` mit dem Werkzeugnamen und der Anweisung. Der Desktop kann also belegen, dass die Sperre gegriffen hat, statt es anzunehmen.

## `agy` 1.5.9 — Cloud (Google Antigravity)

`agy models` listet ausschließlich Cloud-Modelle (Gemini, Claude, GPT-OSS). Damit ist die Klassifikation aus ADR 0010 Entscheidung 6 belegt: **Cloud**, nicht lokal.

| Frage | Beleg | Ergebnis |
|---|---|---|
| Headless | `agy -p "…"` → Antwort, Exit 0; `--print-timeout` | ja |
| Ausgabeformat | `--output-format text\|json\|stream-json`, `--input-format`, `--json-schema` | ja, JSON mit `conversation_id`, `status`, `usage` |
| Abbruchsignal | `SIGTERM` während eines Laufs → Exit **1** | nein, nicht von Fehler unterscheidbar |
| MCP-Annahme | `agy mcp add probe python3 …` (stdio) und `--type http` | ja |
| Schreibsperre | drei Varianten geprüft | **nein** |

Die drei Varianten im Einzelnen:

1. **Standard, headless.** Ein Auftrag, `probe.txt` mit dem Datei-Werkzeug anzulegen, endet mit: *„a tool required the "write_file" permission that headless mode cannot prompt for, so it was auto-denied"*. Das Datei-Werkzeug ist also gesperrt. Derselbe Auftrag über das **Shell**-Werkzeug (`touch probe2.txt`) wird dagegen **ausgeführt**, ohne jede Rückfrage. Die Sperre deckt die Dateiwerkzeuge, nicht die Shell.
2. **`--sandbox`.** Beschrieben als „sandbox with terminal restrictions". Der Shell-Schreibbefehl ging auch damit durch — `probe2.txt` wurde angelegt.
3. **`--mode plan`.** Jetzt wird auch `command` verweigert (*„required the "command" permission that headless mode cannot prompt for"*) — aber genauso **`mcp`**: ein Aufruf des Probe-Werkzeugs endet mit *„required the "mcp" permission that headless mode cannot prompt for, so it was auto-denied"*. Eine Allow-Regel `permissions.allow: ["mcp(probe)"]` — global in `settings.json` und als `--settings` pro Aufruf — änderte daran nichts.

Damit schließen sich die beiden Ziele gegenseitig aus: Der Modus, der Schreiben sperrt, sperrt auch den Proxy. `--dangerously-skip-permissions` hebelt ohnehin alles aus. Für die Bedingung 5b aus ADR 0010 gibt es damit **keinen belegten Mechanismus**.

## `codewhale` 0.10.2 — Cloud (Anbieter konfiguriert, hier ohne Schlüssel)

| Frage | Beleg | Ergebnis |
|---|---|---|
| Headless | `codewhale exec` („Run a non-interactive prompt"); ein Lauf scheiterte am fehlenden Schlüssel, nicht am Programm | ja, laut Vertrag |
| Ausgabeformat | `--output-format text\|stream-json`, `--json` | ja, laut Vertrag |
| Abbruchsignal | nicht messbar — ohne Schlüssel endet jeder Lauf sofort | **unbelegt** |
| MCP-Annahme | `codewhale mcp add --command …` (stdio) und `--url` (streamable HTTP/SSE) | ja |
| Schreibsperre | `--sandbox read-only` + `--allowed-tools`/`--disallowed-tools` | Mechanismus belegt, Kombination nicht |

Die Sperre ist zweistufig und beide Stufen sind brauchbar:

- **`--sandbox read-only`** ist eine Betriebssystem-Sperre, kein Vorsatz. Erstbeleg: ein Schreibversuch **innerhalb** des Arbeitsverzeichnisses wurde mit `Operation not permitted` abgewiesen (Exit 1, Datei nicht angelegt). `--auto` öffnet standardmäßig „Full Access"; ein ausdrücklich gesetztes `--sandbox read-only` ist strenger und schaltet den Trust-Modus ab.
- **`--allowed-tools`** ist wie bei `claude` eine Allowlist („all others denied"), `--disallowed-tools` gewinnt gegen sie.

Zwei zusätzliche Schalter sind für den Proxy fast wichtiger als die Sperre, weil sie zwei in ADR 0010 genannte Risiken direkt adressieren:

- **`--parent-death-watch`** — „Shut down when the parent closes this process's stdin." Das ist die Antwort auf „stirbt die CLI, muss der Proxy offene Karten zurücknehmen": der Desktop hält stdin und schließt es, wenn die Sitzung endet.
- **`--max-turns`, `--max-tool-calls`** — Obergrenzen gegen die Kartenflut aus demselben Abschnitt.

Was fehlt, ist der Lauf: `codewhale exec --auto --sandbox read-only --allowed-tools mcp__…` mit einem echten Modell, der belegt, dass ein MCP-Aufruf durchgeht, während ein Shell-Schreibbefehl scheitert — genau der Versuch, der für `claude` oben dokumentiert ist. Dafür braucht es einen Anbieterschlüssel.

## Was das für ADR 0010 bedeutet

Die Bedingung 5a (headless und MCP) erfüllen alle drei, `agy` beim Abbruchsignal nur eingeschränkt. Die Bedingung 5b (wirksame Schreibsperre) erfüllt heute nur `claude` belegt; `codewhale` hat den Mechanismus, aber nicht den Beleg; `agy` erfüllt sie nicht.

Nach der Regel, die das ADR selbst setzt — „Kommt kein Kandidat durch dieses Tor, lautet die richtige Entscheidung *keine CLI-Integration*" — ist damit nicht das ganze Vorhaben erledigt: ein Kandidat ist durch. Die nächsten Schritte sind eine Entscheidung des Menschen, nicht dieses Dokuments:

1. **`claude` als ersten Kandidaten nehmen** — belegt, aber Cloud; der Hinweis „Inhalte verlassen das Gerät" gilt, und die README-Zusage muss eingeschränkt werden (ADR 0010, Konsequenzen).
2. **`codewhale` nachziehen**, sobald ein Anbieterschlüssel für den Beleglauf bereitsteht. Sein Vertrag ist für den Proxy der passendste, weil `--parent-death-watch` und die Aufrufobergrenzen genau die offenen Risiken adressieren.
3. **`agy` zurückstellen.** Ohne belegte Schreibsperre wird es nicht angeboten; ein späterer Stand kann das ändern.

## Wie geprüft wurde

- Alle Vertragsangaben stammen aus `--help` des jeweils installierten Programms, nicht aus Dokumentation im Netz.
- Die Sperrtests liefen in `/tmp/agy-probe` gegen einen minimalen MCP-stdio-Server, der genau ein rein lesendes Werkzeug anbietet.
- `agy` wurde für die Tests ein Server registriert und eine Berechtigungsregel gesetzt; beides wurde danach entfernt, `settings.json` aus der Sicherung wiederhergestellt.
- Die codewhale-Sperre wurde über dieselbe Betriebssystem-Isolation geprüft, die dieses Projekt für schreibgeschützte Läufe nutzt.
