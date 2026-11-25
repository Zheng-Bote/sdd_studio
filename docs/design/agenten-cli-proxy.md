---
feature: agenten-cli-proxy
status: draft
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - cmd/sdd-desk/**
  - README.md
  - CHANGELOG.md
  - docs/arc42.md
---

# Tech Spec — Lokale Agenten-CLIs im Desktop, über einen MCP-Proxy (`agenten-cli-proxy`)

Setzt das PRD [`docs/prd/agenten-cli-proxy.md`](../prd/agenten-cli-proxy.md) um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

Alles bleibt in der Desk-Schicht. `internal/mcp`, `internal/sddcore` und `cmd/sdd` werden nicht berührt; `sdd` bleibt KI-frei und offline (ADR 0003, ADR 0010 Entscheidung 2).

```text
internal/desk/cli.go          Registry der angebotenen CLIs: Startflags, Sperrmechanismus,
                              Vorhandensein, Cloud-Kennzeichnung
internal/desk/proxy.go        MCP-Endpunkt (JSON-RPC über streamable HTTP), tools/list,
                              tools/call → Karte → Entscheidung → Ausführung
internal/desk/approval.go     Register offener Aufrufe: Kennung, Frist, Entscheidungskanal
internal/desk/clirun.go       Prozessverwaltung: Start, stdin, Ende, Rücknahme offener Karten
internal/desk/evidence.go     Belegstand je CLI (Tabelle wie für die Modelle)
internal/desk/server.go       Routen /mcp, /api/cli, /api/cli/decide
cmd/sdd-desk/main.go          Verdrahtung: Proxy neben dem bestehenden StartMCP
apps/desk/src/**              CLI-Auswahl, Cloud-Hinweis, Moduskennzeichnung, Karten mit CLI/Sitzung
```

Datenfluss eines Aufrufs:

```text
CLI ──HTTP/JSON-RPC──▶ sdd-desk /mcp ──▶ approval.Register (Kennung, Frist)
                                            │
                             Freigabekarte ◀─┘ (CLI, Sitzung, Werkzeug, Argumente, Stufe)
                                            │
                       Entscheidung ────────┴──▶ Agent.tier-Prüfung ──▶ MCPClient.CallTool
                                                                          (bestehender Kindprozess)
```

Der Proxy erfindet keinen Werkzeugkatalog: `tools/list` reicht den bereits laufenden `sdd mcp`-Kindprozess durch (`MCPClient.Tools()`), `tools/call` benutzt denselben `MCPClient` wie die eigene Schleife des Desktops.

### Voraussetzung: Transport

Die Belege vom 2026-11-22 deckten **stdio** ab. Für den Proxy kommt nur ein Endpunkt auf `127.0.0.1` in Frage, den die CLI als Client anspricht. Beide Kandidaten bieten dafür eine Option:

| CLI | Option (aus `--help` dieser Builds) |
|---|---|
| `claude` 2.1.330 | `mcp add --transport http\|sse <url> --header "Authorization: Bearer …"`; `--mcp-config` nimmt JSON mit `mcpServers` |
| `codewhale` 0.10.3 | `mcp add --url <url> --transport sse --bearer-token-env-var <VAR>` |

**Das ist Optionsvorhandensein, nicht Beleg.** Ein durchgehender Aufruf über HTTP ist bisher nicht gefahren worden — belegt ist stdio. Der erste Schritt der Umsetzung ist deshalb ein Beleglauf je CLI (HTTP-Endpunkt, `tools/list`, ein `tools/call`), protokolliert in [`docs/notes/headless-vertraege-agenten-clis.md`](../notes/headless-vertraege-agenten-clis.md). Fällt er für eine CLI aus, wird sie nicht angeboten, bis er vorliegt — der Rest der Umsetzung hängt nicht davon ab, weil das Register hinter der Transportgrenze liegt.

## Schnittstellen

### `POST /mcp` — MCP über streamable HTTP

Derselbe Loopback-Listener wie die Desk-API, dieselbe `Host`-Prüfung; zusätzlich `Authorization: Bearer <Sitzungstoken>` mit `crypto/subtle`-Vergleich wie bei den übrigen API-Aufrufen. Das Token steht **nicht** in der Kommandozeile der CLI: `claude` bekommt es über eine temporäre `--mcp-config`-Datei (0600, nach dem Start gelöscht), `codewhale` über `--bearer-token-env-var` aus der Umgebung des Kindprozesses.

Unterstützt werden die Methoden, die eine Agenten-CLI für einen Werkzeugserver braucht: `initialize`, `notifications/initialized`, `tools/list`, `tools/call`. Jeder Aufruf ist eine HTTP-Anfrage; die Antwort bleibt offen, bis entschieden ist oder die Frist abläuft (Werkzeugaufrufe wie `sdd_gate` dauern legitim lange).

### `GET /api/cli` — was angeboten wird

```json
{
  "clis": [
    {
      "id": "claude", "name": "Claude Code", "cloud": true,
      "installed": true, "offerable": true,
      "blocked_because": "",
      "evidence": { "status": "unproven", "note": "", "source": "" }
    },
    {
      "id": "agy", "name": "Google Antigravity", "cloud": true,
      "installed": true, "offerable": false,
      "blocked_because": "wirksame Schreibsperre nicht belegt (ADR 0010, Bedingung 5b)",
      "evidence": { "status": "unproven", "note": "", "source": "" }
    }
  ]
}
```

`offerable` ist die UND-Verknüpfung aus: installiert, beide Bedingungen belegt, Transportbeleg vorhanden. `blocked_because` nennt den fehlenden Grund im Klartext, damit `agy` erklärbar fehlt statt stillschweigend zu fehlen.

### `GET /api/cli/cards` und `POST /api/cli/decide`

Offene Karten werden wie die bestehenden `Pending`-Einträge ausgeliefert und ergänzt um `call_id`, `cli`, `session`, `tier`, `deadline`:

```json
{ "call_id": "c-7f31…", "cli": "codewhale", "session": "s-2b9e…",
  "tool": "sdd_write", "tier": "write", "summary": "…",
  "deadline": "2026-11-25T10:41:00Z", "preview": { } }
```

`POST /api/cli/decide` nimmt `{"call_id": "...", "decision": "allow|deny|allow_tool_session|allow_write_session"}`. Die Entscheidung wirkt ausschließlich auf den Aufruf mit dieser Kennung; `allow_tool_session` gilt nur für die Stufe `read` und nur für diese Sitzung und dieses Werkzeug, `allow_write_session` für `write`. Für `critical` wird jede Sitzungsform abgewiesen (HTTP 400) — nicht stillschweigend auf „einmal" heruntergestuft, sondern abgelehnt, damit ein Client den Unterschied nicht verwischen kann.

## Datenmodell

```go
// internal/desk/approval.go
type Call struct {
    ID      string          // Kennung je Aufruf, vom Proxy vergeben
    Session string          // eine CLI-Ausführung = eine Sitzung
    CLI     string
    Tool    string
    Args    json.RawMessage
    Tier    string          // tier() aus agent.go — dieselbe Einordnung wie die eigene Schleife
    Decide  chan Verdict
    Expires time.Time
}

type Verdict struct {
    Allow  bool
    Scope  string // "call", "tool_session", "write_session"
}
```

Das Register hält `map[string]*Call` unter einem Mutex, ein Timer je Aufruf entfernt den Eintrag bei Ablauf und schickt `Verdict{Allow:false}` mit Grund `expired`. Ein Treffer auf eine unbekannte oder abgelaufene Kennung führt nie aus — er wird als „unbekannter Aufruf" beantwortet und protokolliert.

Sitzungszuordnung: Eine CLI-Ausführung ist eine Sitzung. Startet der Nutzer zwei CLIs, laufen zwei Sitzungen mit eigenen Kennungen; Karten tragen beide, und eine Entscheidung wirkt nur in ihrer Sitzung (PRD AC-10).

Der Belegstand je CLI folgt der vorhandenen Tabelle:

```go
type CLIEvidence struct {
    CLI     string // "claude", "codewhale"
    Status  string // EvidenceProven | EvidenceFailed | EvidenceUnproven
    Note    string
    Source  string
}
```

Ein Eintrag ohne `Source` ist ungültig und wird nicht angezeigt — dieselbe Regel wie bei den Modellen, und derselbe Test.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `cli.go`: `Offerable()` = installiert ∧ Belege ∧ Transportbeleg; `agy` ohne Beleg der Sperre → `offerable: false` mit Grund | `TestCLIOfferableOnlyWithProof` |
| AC-2 | `proxy.go`: `tools/call` legt **immer** zuerst einen Registereintrag an und wartet; Ausführung nur nach `Verdict.Allow`; Stufe aus `tier()` in `agent.go` | `TestProxyCallNeedsDecision`, `TestProxyCriticalNeedsSingleApproval` |
| AC-3 | Kennung je Aufruf; `Decide` adressiert `call_id`, nicht Position | `TestProxyCorrelatesByID` (zwei parallele Aufrufe, Antwort vertauscht) |
| AC-4 | `approval.go`: Timer je Aufruf, `Expires`; späte Entscheidung findet keinen Eintrag mehr | `TestProxyCallExpires`, `TestLateVerdictDoesNotExecute` |
| AC-5 | `clirun.go`: Kontext je Sitzung; Prozessende bricht wartende Aufrufe ab und zieht Karten zurück; `--parent-death-watch` wo verfügbar | `TestSessionEndWithdrawsCards` |
| AC-6 | Karte vor dem ersten Lauf je CLI, gleiche Mechanik wie `ErrDisclosure`; Moduskennzeichnung in `apps/desk/src` | `TestCLICloudNoticeRequired` |
| AC-7 | Startflags je CLI (Tabelle unten); kein Durchreichen umgehender Flags | `TestCLIStartFlags`, `TestNoBypassFlags` |
| AC-8 | `evidence.go`: `CLIEvidence`, `EvidenceForCLI`; ohne Eintrag `unproven` | `TestCLIEvidenceRequiresSource` |
| AC-9 | README-Abschnitt „Was der Desktop darf" umformuliert; Abschnitt zum CLI-Modus | Handprüfung (Doku), kein Test |
| AC-10 | `Session` an jeder Karte; Entscheidung nur in ihrer Sitzung | `TestProxyTwoSessionsStaySeparate` |

### Startflags je CLI (Sperrmechanismus, AC-7)

| CLI | Start | Sperre | Warum so |
|---|---|---|---|
| `claude` | `claude -p --output-format stream-json --mcp-config <tmp> --allowedTools "mcp__sdd"` | `--allowedTools` ist eine Allowlist; alles Ungenannte wird ohne Rückfrage verweigert (belegt: `permission_denials`, Datei nicht angelegt) | belegter Mechanismus; `--dangerously-skip-permissions` wird nie übergeben |
| `codewhale` | `codewhale exec --auto --sandbox read-only --output-format stream-json --parent-death-watch` | `--sandbox read-only` ist eine Betriebssystem-Sperre (belegt: `Operation not permitted`, Exit 1) | belegter Mechanismus; **kein** `--allowed-tools`, siehe unten |

**Entscheidung zu `--allowed-tools` bei `codewhale`:** Das ADR lässt die Frage offen. Hier wird sie so entschieden: `--allowed-tools` wird **nicht** übergeben. Grund: Die Allowlist nimmt den Aufrufpfad (`tool_search`) mit weg, und ihn einzuschließen heißt, sich an einen internen Werkzeugnamen zu binden, der nicht Teil des belegten Vertrags ist — bricht er, ist das MCP-Werkzeug still nicht mehr ladbar. `--sandbox read-only` ist dagegen der belegte, OS-gestützte Mechanismus und deckt genau die Bedingung 5b ab. Die Alternative (Aufrufpfad einschließen) bleibt zulässig, braucht aber einen eigenen Beleg und einen Test, der den Aufrufpfad mitprüft.

Die genannten Flags (`--parent-death-watch`, `--max-turns`, `--max-tool-calls`) stehen in der Belegnote, erscheinen aber **nicht** in `--help` dieses Builds; im Binary sind sie vorhanden. Der Start prüft sie deshalb zur Laufzeit: Schlägt der Start an einem unbekannten Flag fehl, wird die CLI als „nicht startbar" gemeldet, statt den Fehler zu verschlucken.

## Sicherheit

- **Bindung und Host:** Der MCP-Endpunkt hängt am bestehenden Listener (`127.0.0.1`), erbt `Host`-Prüfung und CSP; ein zweiter Listener entsteht nicht.
- **Token:** wie die übrige API, `crypto/subtle`-Vergleich. Übergabe über Datei (0600, danach gelöscht) bzw. Umgebungsvariable, nie über `argv` — Prozesslisten sind lesbar.
- **Freigabehoheit:** `critical` nur je Aufruf, Sitzungsfreigabe wird mit HTTP 400 abgewiesen (nicht heruntergestuft). Der Kindprozess `sdd mcp` läuft weiterhin mit `--allow-write --allow-critical` unter der Kontrolle von `sdd-desk`; die CLI sieht ihn nie.
- **Schreibsperre der CLI:** je CLI der belegte Mechanismus, kein umgehendes Flag. Ein Test hält fest, dass `--dangerously-skip-permissions`, `--sandbox danger-full-access` und `--allow-sandbox-elevation` in keiner Startzeile vorkommen.
- **Fristen:** je Aufruf; ohne Frist wäre ein toter Client ein dauerhaft offener Kanal. Zusätzlich Obergrenzen (`--max-turns`, `--max-tool-calls`) gegen Kartenflut.
- **Kein zweiter Katalog:** `tools/list` kommt aus `sdd mcp`; ein Test vergleicht die Liste mit `MCPClient.Tools()`.
- **Neue Angriffsfläche:** ein loopback-gebundener Endpunkt mehr, der ohne Token nichts tut. Er ist nur aktiv, wenn der Nutzer eine CLI gestartet hat.

## Offene Punkte

- **HTTP-Beleglauf** je CLI (siehe „Voraussetzung: Transport") — der einzige Punkt, der das Angebot einer CLI blockieren kann.
- **Kosten und Kontingente.** Beide CLIs sind Cloud; ein Lauf verbraucht fremdes Kontingent. Ob der Desktop das anzeigt (etwa `total_cost_usd` aus `claude --output-format json`), ist nicht Teil dieses PRD.
- **Wie lange darf eine Karte offen bleiben?** Vorschlag: `<CLI>_CALL_TIMEOUT`, Standard 10 Minuten, analog zu `<ANBIETER>_TURN_TIMEOUT`. Zu bestätigen, weil `sdd_gate` über großen Beständen länger laufen kann.
- **`agy` später aufnehmen**, sobald ein Schreibsperrmechanismus belegt ist — dann nur Tabelleneintrag und Startzeile, kein Umbau.
- **CLI-eigene Abläufe.** Ob die CLI die SDD-Prompts (`sdd-idea`, `sdd-tests`, …) über `prompts/list` mitbekommt, ist offen; für dieses PRD genügen Werkzeuge. Nachziehen, wenn eine CLI Prompts nachfragt.
- **Lesende Aufrufe erzeugen ebenfalls eine Karte.** Umgesetzt ist die strenge Lesart von ADR 0010 Entscheidung 3 („jeder `tools/call` ist ein Kartenvorgang“). Die Konsequenz „lesend ohne Karte“ wäre die Alternative; sie würde `sdd_next`/`sdd_lint` im CLI-Modus ohne Rückfrage laufen lassen. Bewusst strenger gebaut, bis der Beleglauf zeigt, ob die Kartenflut im Betrieb stört.
- **Codewhale braucht einen Eintrag in seiner eigenen Konfiguration** (`mcp add … --url`), den die Startzeile für den Lauf setzt und danach entfernt. Das ist ein Eingriff in die Nutzerkonfiguration; im Beleglauf prüfen, ob es einen per-Aufruf-Weg gibt.
- **`claude --allowedTools "mcp__sdd"`** ist als Server-Präfix gesetzt; die Belegnote zeigt nur den vollen Werkzeugnamen (`mcp__probe__probe_read`). Im Beleglauf prüfen, ob das Präfix alle Werkzeuge des Servers freigibt.
- **Variadische Flags (Befund aus dem Transportbeleg 2026-11-25).** `--mcp-config` und `--allowedTools` fressen nachfolgende Wörter. Der Prompt steht deshalb direkt hinter `-p`, und `TestCLIStartFlags` hält die Reihenfolge fest. Wer die Startzeile ändert, muss das erhalten.
- **`--mcp-config` greift nicht in `claude mcp list`**, nur in einer Sitzung. Der Transport ist über den Local-Scope belegt; die Produktionsstartzeile braucht weiterhin einen kostenpflichtigen Modellaufruf als Beleg.
