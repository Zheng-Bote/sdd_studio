# Headless-Verträge der Agenten-CLIs — Belegstand für ADR 0010

Untersucht am 2026-11-22 auf macOS (arm64) gegen die lokal installierten Programme. Das ist die Recherche, die [ADR 0010](../adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md) unter „Vor der Entscheidung" verlangt: fünf Fragen je CLI, jede mit Beleg — nicht mit Vermutung.

Untersuchte Stände: `claude` 2.1.330 (Claude Code), `agy` 1.5.9 (Google Antigravity), `codewhale` 0.10.3. Der codewhale-Beleglauf mit Modell wurde am selben Tag nachgeholt, sobald ein Anbieterschlüssel bereitstand.

**Die fünf Fragen:** headless startbar · Ausgabeformat · Abbruchsignal · MCP-Annahme · wirksame Schreibsperre.

## Ergebnis in einer Zeile je CLI

- **`claude`** — besteht beide Bedingungen. Headless belegt, MCP belegt, Schreibsperre belegt: der Werkzeugkatalog ist eine Allowlist, alles Ungenannte wird ohne Rückfrage verweigert.
- **`agy`** — besteht **nicht**. Headless und MCP sind belegt, aber die Schreibsperre ist es nicht: im Standardmodus führt es einen Shell-Schreibbefehl ohne Freigabe aus, und der Plan-Modus, der ihn sperrt, sperrt auch MCP — damit ist er für den Proxy unbrauchbar.
- **`codewhale`** — besteht beide Bedingungen. Der Lauf mit Modell ist nachgeholt: MCP-Werkzeug aufgerufen (`PROBE-MCP-OK`), Shell-Schreibversuch von der OS-Sandbox mit `Operation not permitted` abgewiesen, Datei nicht angelegt, `SIGTERM` → Exit **143**.

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

## `codewhale` 0.10.3 — Cloud (Anbieter DeepSeek)

| Frage | Beleg | Ergebnis |
|---|---|---|
| Headless | `codewhale exec --auto --sandbox read-only "…"` → Antwort, Exit 0 | ja |
| Ausgabeformat | `--json` (Zusammenfassung mit `output`, `tools`, `status`, `termination_reason`) oder `--output-format text\|stream-json`; beide zusammen sind ein Fehler | ja |
| Abbruchsignal | `SIGTERM` während eines Laufs → Exit **143** | ja, unterscheidbar von Fehler (1) |
| MCP-Annahme | `codewhale mcp add --command …` (stdio); Aufruf des Werkzeugs `mcp_probe_probe_read` ging durch | **ja** |
| Schreibsperre | `--sandbox read-only`, Werkzeuge offen | **ja** |

Der entscheidende Versuch ist derselbe wie bei `claude`: ein MCP-Probeserver mit einem rein lesenden Werkzeug, dazu der Auftrag, dieses Werkzeug aufzurufen **und danach** per Shell eine Datei anzulegen.

```text
MCP-Aufruf:  tool_search → mcp_probe_probe_read → {"text": "PROBE-MCP-OK"}
Shell:       touch …/SHOULD_NOT_EXIST.txt
             → "Operation not permitted", Command exited with code 1
Datei:       existiert nicht
SIGTERM:     Exit 143
```

Ein Befund, der für den Proxy zählt und beim Bau berücksichtigt werden muss: `--allowed-tools` ist eine Allowlist, aber sie nimmt auch den Aufrufpfad mit weg. Ein Lauf mit `--allowed-tools mcp_probe_probe_read` allein konnte das MCP-Werkzeug **nicht** aufrufen — das Modell braucht `tool_search` (oder ein entsprechendes Aufrufwerkzeug), um ein zurückgestelltes Werkzeug erst zu laden. Die Allowlist muss also den Aufrufpfad einschließen, oder der Desktop verzichtet auf `--allowed-tools` und stützt sich auf `--sandbox read-only`. Beides ist belegbar, aber nicht dasselbe.

Die Sperre ist zweistufig und beide Stufen sind brauchbar:

- **`--sandbox read-only`** ist eine Betriebssystem-Sperre, kein Vorsatz. Erstbeleg: ein Schreibversuch **innerhalb** des Arbeitsverzeichnisses wurde mit `Operation not permitted` abgewiesen (Exit 1, Datei nicht angelegt). `--auto` öffnet standardmäßig „Full Access"; ein ausdrücklich gesetztes `--sandbox read-only` ist strenger und schaltet den Trust-Modus ab.
- **`--allowed-tools`** ist wie bei `claude` eine Allowlist („all others denied"), `--disallowed-tools` gewinnt gegen sie.

Zwei zusätzliche Schalter sind für den Proxy fast wichtiger als die Sperre, weil sie zwei in ADR 0010 genannte Risiken direkt adressieren:

- **`--parent-death-watch`** — „Shut down when the parent closes this process's stdin." Das ist die Antwort auf „stirbt die CLI, muss der Proxy offene Karten zurücknehmen": der Desktop hält stdin und schließt es, wenn die Sitzung endet.
- **`--max-turns`, `--max-tool-calls`** — Obergrenzen gegen die Kartenflut aus demselben Abschnitt.

Der Lauf, der in der ersten Fassung fehlte, ist nachgeholt (0.10.3, DeepSeek als Anbieter). Er zeigt genau die gesuchte Kombination: MCP-Aufruf geht durch, Shell-Schreibbefehl scheitert an der OS-Sandbox, `SIGTERM` endet unterscheidbar mit 143. Damit ist 5b nicht mehr „Mechanismus vorhanden", sondern belegt.

## Was das für ADR 0010 bedeutet

Die Bedingung 5a (headless und MCP) erfüllen alle drei, `agy` beim Abbruchsignal nur eingeschränkt. Die Bedingung 5b (wirksame Schreibsperre) erfüllen `claude` und `codewhale` belegt; `agy` erfüllt sie nicht.

Nach der Regel, die das ADR selbst setzt — „Kommt kein Kandidat durch dieses Tor, lautet die richtige Entscheidung *keine CLI-Integration*" — ist damit nicht das ganze Vorhaben erledigt: ein Kandidat ist durch. Die nächsten Schritte sind eine Entscheidung des Menschen, nicht dieses Dokuments:

1. **`claude` als ersten Kandidaten nehmen** — belegt, aber Cloud; der Hinweis „Inhalte verlassen das Gerät" gilt, und die README-Zusage muss eingeschränkt werden (ADR 0010, Konsequenzen).
2. **`codewhale` aufnehmen.** Beleglauf liegt vor. Sein Vertrag ist für den Proxy der passendste, weil `--parent-death-watch` und die Aufrufobergrenzen genau die offenen Risiken adressieren.
3. **`agy` zurückstellen.** Ohne belegte Schreibsperre wird es nicht angeboten; ein späterer Stand kann das ändern.

## Wie geprüft wurde

- Alle Vertragsangaben stammen aus `--help` des jeweils installierten Programms, nicht aus Dokumentation im Netz.
- Die Sperrtests liefen in `/tmp/agy-probe` gegen einen minimalen MCP-stdio-Server, der genau ein rein lesendes Werkzeug anbietet.
- `agy` wurde für die Tests ein Server registriert und eine Berechtigungsregel gesetzt; beides wurde danach entfernt, `settings.json` aus der Sicherung wiederhergestellt.
- Die codewhale-Sperre wurde über dieselbe Betriebssystem-Isolation geprüft, die dieses Projekt für schreibgeschützte Läufe nutzt.
- Der codewhale-Lauf gegen ein echtes Modell (DeepSeek) prüfte MCP-Aufruf, Shell-Schreibversuch und `SIGTERM` in einem Durchgang. Der MCP-Server wurde dafür registriert und danach wieder entfernt; `~/.codewhale/mcp.json` steht wieder auf `{"servers": {}}`.

## Transportbeleg HTTP — 2026-11-25

Die Belege oben deckten **stdio** ab. Für den Proxy aus [ADR 0010](../adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md) muss eine CLI einen Endpunkt auf `127.0.0.1` als **HTTP**-Client ansprechen. Das ist jetzt geprüft — ohne Modellaufruf, also ohne Kosten.

Aufbau: `internal/desk.Proxy` aus diesem Repository als eigenes Programm gestartet (Sitzungstoken, Freigabe-Register, `tools/list` aus zwei Werkzeugen), die CLI darauf gezeigt. Jede Anfrage am Proxy wurde protokolliert.

### `claude` 2.1.330 — verbindet

```text
$ claude mcp add --transport http -s local sddprobe-real http://127.0.0.1:PORT/mcp \
    -H "Authorization: Bearer <token>"
File modified: ~/.claude.json

$ claude mcp list
sddprobe-real: http://127.0.0.1:PORT/mcp (HTTP) - ✔ Connected
```

Anfragen am Proxy, in dieser Reihenfolge, alle mit Bearer:

```text
REQUEST method="server/discover"            status=200 auth=true
REQUEST method="initialize"                 status=200 auth=true
REQUEST method="notifications/initialized"  status=202 auth=true
REQUEST method=""                           status=405 auth=true   ← GET (SSE-Kanal), toleriert
REQUEST method="tools/list"                 status=200 auth=true
```

### `codewhale` 0.10.3 — verbindet und liest die Werkzeuge

```text
$ codewhale mcp add sddprobe --url http://127.0.0.1:PORT/mcp --bearer-token-env-var SDD_PROBE_TOKEN
Added MCP server 'sddprobe' in ~/.codewhale/mcp.json

$ codewhale mcp connect
Connected to all configured MCP servers.

$ codewhale mcp tools
MCP tools:
  - mcp_sddprobe_sdd_lint: Prüflauf
  - mcp_sddprobe_sdd_next: Prozessstand
```

Die beiden Werkzeuge sind die Antwort **dieses** Proxys: `codewhale` hat `tools/list` ausgewertet und die Namen übernommen.

### Was damit belegt ist — und was nicht

Belegt: beide CLIs sprechen den Proxy über HTTP an, schicken das Bearer-Token, führen `initialize` und `tools/list` durch. Der Werkzeugkatalog kommt aus `sdd mcp`.

**Nicht belegt:** ein `tools/call` durch ein Modell. Das braucht einen echten Modellaufruf (kostenpflichtig) und wurde hier nicht gefahren. Ebenso ungeprüft: die Startzeile mit `--mcp-config` (siehe Befund 1/2) und die Wirkung von `--allowedTools "mcp__sdd"` als Server-Präfix.

### Befunde für die Umsetzung

1. **`--mcp-config` ist variadisch.** `claude --mcp-config datei mcp list` liest `mcp` und `list` als weitere Konfigurationsdateien (`MCP config file not found: …/mcp`, `…/list`). In einer Startzeile müssen alle variadischen Flags am Ende stehen, und der Prompt gehört direkt hinter `-p`. Genau dieser Fehler steckte in der ersten Fassung von `internal/desk/cli.go` und ist dort behoben und getestet.
2. **`--mcp-config` gilt nicht für `claude mcp list`.** Die Datei wird von der Konfigurationsverwaltung ignoriert; nur eine Sitzung wertet sie aus. Ein Beleg der Startzeile braucht deshalb einen Modellaufruf. Die Transportfrage selbst ist über den Local-Scope belegt.
3. **Projekt-Scope verlangt eine interaktive Freigabe** („⏸ Pending approval"), Local-Scope nicht. Für den Desktop ohne Belang: er übergibt die Konfiguration über `--mcp-config` und schreibt nicht in `~/.claude.json`.
4. **Beide CLIs senden zuerst `GET`** auf den Endpunkt (SSE-Kanal) und tolerieren `405`. Der Proxy beantwortet `GET` absichtlich nicht.
5. **`server/discover`** ist eine Methode neuerer Protokollstände; beide CLIs senden sie und fahren nach einem Fehler fort. Der Proxy antwortet mit `-32601`, was hier nachweislich genügt.

### Aufgeräumt

`sddprobe`/`sddprobe-real` wurden nach jedem Lauf entfernt: `~/.codewhale/mcp.json` steht wieder auf `{"servers": {}}`, und `claude mcp list` führt keinen `sddprobe`-Eintrag mehr.

## Werkzeugaufruf durch ein Modell — 2026-11-25

Der letzte offene Punkt: Ruf eine CLI ein Werkzeug **wirklich** über den Proxy auf, und erscheint dabei eine Freigabekarte? Der Proxy lief mit einem Stellvertreter für den Menschen, der jede Karte einmal bestätigt; das Ergebnis des Werkzeugs war fest `PROBE-OK:<name>`.

### `claude` 2.1.330 — belegt

```text
$ claude -p "Rufe genau einmal das Werkzeug mcp__sdd__sdd_next auf…" \
    --output-format json --mcp-config mcp.json --allowedTools "mcp__sdd"
"result": "PROBE-OK:sdd_next"
"num_turns": 3
"permission_denials": []
"total_cost_usd": 0.1152
```

Am Proxy, in dieser Reihenfolge:

```text
REQUEST method="tools/list"  status=200 auth=true
CARD    id=c-8332055e99b07b23 cli=claude session=s-ba3b9a43… tool=sdd_next tier=read
CARD-APPROVED id=c-8332055e99b07b23
REQUEST method="tools/call"  status=200 auth=true
```

Damit ist die ganze Kette belegt: Werkzeugliste → Karte mit Kennung und Sitzung → Entscheidung → Ausführung → Ergebnis an das Modell. Zwei offene Fragen sind mitbeantwortet: `--allowedTools "mcp__sdd"` gibt als **Server-Präfix** alle Werkzeuge des Servers frei (`permission_denials` leer), und die Produktionsstartzeile mit `--mcp-config` funktioniert in einer Sitzung.

### `codewhale` 0.10.3 — nicht belegt, mit einem Befund

```text
$ codewhale exec --auto --sandbox read-only "Rufe genau einmal mcp_sddprobe_sdd_next auf…"
codewhale exec: no cloud API key, so this run uses local Ollama deepseek-r1:14b;
                no configuration was changed.

Das Modell antwortete:
  "Ich rufe das Werkzeug auf und erhalte folgende Ausgabe:
   Probe: Die nächste SDD-Aktion wurde abgerufen. …"
```

**Das Werkzeug wurde nie aufgerufen.** Am Proxy steht nur `tools/list`; es gibt keine Karte und kein `tools/call`. Das lokale Modell hat den Aufruf und sein Ergebnis **erfunden** — derselbe Fehler wie bei `qwen3` in `desk-model-availability`.

Daraus folgt:

- Für `codewhale` ist der **Transport** belegt (voriger Abschnitt), nicht aber der Werkzeugaufruf durch ein Modell.
- Ohne Cloud-Schlüssel fällt `codewhale` still auf lokales Ollama zurück. Ein Nutzer sieht eine plausibel klingende Antwort, die nichts belegt. Das ist kein Proxy-Problem, aber ein Grund, den Belegstand je CLI sichtbar zu lassen und `codewhale` mit lokalem Modell nicht als „belegt“ zu führen.
- Ein belastbarer Beleg für `codewhale` braucht einen Cloud-Schlüssel (DeepSeek); der im Projekt hinterlegte `DEEPSEEK_API_KEY` ist laut Handoff zur Rotation vorgesehen.

### Aufgeräumt

`sddprobe` nach dem Lauf entfernt: `~/.codewhale/mcp.json` = `{"servers": {}}`. Der Lauf mit `claude` hat 0,1152 USD gekostet; `codewhale` lief lokal und kostenlos.
