---
feature: agenten-cli-proxy
status: draft
archive-after:
---

# Plan — agenten-cli-proxy (`agenten-cli-proxy`)

Reihenfolge der Umsetzung. Jede Task nennt das Akzeptanzkriterium, auf das sie einzahlt.

PRD: [`docs/prd/agenten-cli-proxy.md`](../prd/agenten-cli-proxy.md) · Tech Spec: [`docs/design/agenten-cli-proxy.md`](../design/agenten-cli-proxy.md)

## Vorbemerkung: was zuerst belegt werden muss

Die Tech Spec hängt an einer Annahme, die noch kein Beleg deckt: dass eine CLI den Proxy über HTTP anspricht. Belegt ist bisher nur stdio. **Task 0 ist deshalb ein Beleglauf, nicht Code.** Ohne ihn ist alles Weitere gebaut, aber nicht benutzbar.

## Tasks

- [ ] **AC-1** Transportbeleg erheben: für `claude` 2.1.330 und `codewhale` 0.10.3 je ein Lauf gegen einen lokalen HTTP-MCP-Endpunkt mit `initialize`, `tools/list` und einem `tools/call`; Ergebnis mit Kommandozeile und Ausgabe in `docs/notes/headless-vertraege-agenten-clis.md` nachtragen. Fällt eine CLI durch, wird sie nicht angeboten und der Punkt hier als solcher festgehalten.
- [x] **AC-1, AC-8** `internal/desk/cli.go`: Registry der CLIs (`claude`, `codewhale`, `agy`) mit Name, Cloud-Kennzeichen, Startzeile, Sperrmechanismus, Installationsprüfung (`exec.LookPath`) und Belegstand. `Offerable()` verlangt installiert ∧ beide ADR-Bedingungen belegt ∧ Transportbeleg. `blocked_because` nennt den Grund im Klartext.
- [x] **AC-8** `internal/desk/evidence.go` erweitern: `CLIEvidence`, `EvidenceForCLI`, dieselbe Regel wie bei den Modellen — ohne `Source` ungültig, ohne Eintrag `unproven`.
- [x] **AC-2, AC-3, AC-4** `internal/desk/approval.go`: Register offener Aufrufe mit Kennung je Aufruf, Frist je Aufruf, Entscheidungskanal. Treffer auf unbekannte oder abgelaufene Kennung führt nie aus.
- [x] **AC-2, AC-3, AC-7, AC-10** `internal/desk/proxy.go`: JSON-RPC über HTTP (`initialize`, `notifications/initialized`, `tools/list`, `tools/call`). `tools/list` aus `MCPClient.Tools()`. `tools/call` → Register → Karte → Entscheidung → `tier()`-Prüfung → `MCPClient.CallTool`. Sitzungskennung an jeder Karte.
- [x] **AC-5, AC-6, AC-7** `internal/desk/clirun.go`: CLI als Kindprozess mit der Startzeile aus `cli.go`; Token über Datei (0600, danach gelöscht) bzw. Umgebungsvariable; Prozessende bricht wartende Aufrufe ab und zieht Karten zurück; umgehende Flags sind ausgeschlossen.
- [x] **AC-2, AC-3, AC-4, AC-10** `internal/desk/server.go`: Routen `POST /mcp` (Bearer statt `X-SDD-Token`), `GET /api/cli`, `GET /api/cli/cards`, `POST /api/cli/decide`. Sitzungsfreigabe für `critical` wird mit HTTP 400 abgewiesen.
- [ ] **AC-6** `apps/desk/src/**`: CLI-Auswahl, Cloud-Hinweis vor dem ersten Lauf, dauerhafte Kennzeichnung des CLI-Modus, Karten mit CLI und Sitzung.
- [x] **AC-9** `README.md`: Abschnitt „Was der Desktop darf“ einschränken („kein Dateizugriff an `sdd` vorbei“), CLI-Modus als Betriebsart nennen. `docs/arc42.md` und `CHANGELOG.md` nachziehen.

## Abschluss (gilt für alle Kriterien)

`sdd lint` 0/0/0 (SDD-022 je AC erfüllt), `go test` grün, `sdd gate pre-merge` 0/0/0.

## Stand 2026-11-25

Umgesetzt und getestet: `internal/desk/approval.go` (Register, Kennung, Frist, Sitzungsfreigaben), `internal/desk/cli.go` (Registry, Startzeilen, verbotene Flags), `internal/desk/proxy.go` (JSON-RPC über HTTP, `tools/list`, `tools/call`), `internal/desk/clirun.go` (Kindprozess, Sitzungsende), `internal/desk/cliapi.go` und Routen in `server.go`, `CLIEvidence` in `evidence.go`, Verdrahtung in `cmd/sdd-desk/main.go`, README, arc42, CHANGELOG. Tests: `internal/desk/cli_proxy_test.go`, je Kriterium mindestens eine Zeile mit Kennung.

Offen und bewusst nicht behauptet:

- **Transportbeleg (Task 0).** Ohne ihn bleibt `ProxyProven` falsch, und `claude` wie `codewhale` werden **nicht angeboten** (Grund im Klartext: „der Transportbeleg gegen den Proxy steht aus“). Das ist der gewollte Zustand, solange kein Lauf dokumentiert ist — nicht als fertig zu melden.
- **Oberfläche** (`apps/desk/src`): CLI-Auswahl, Cloud-Hinweis, Kennzeichnung des Modus. Die API steht (`GET /api/cli`, `POST /api/cli/start`, Karten, Entscheidung); die Oberfläche fehlt noch.
- **Codewhale-Konfiguration.** `codewhale` kennt Server nur über seine eigene Konfiguration; die Startzeile fügt sie für den Lauf hinzu und entfernt sie danach. Das berührt die Nutzerkonfiguration und ist im Beleglauf zu prüfen.

## Reihenfolge und Abhängigkeiten

1. Transportbeleg (Task 0) — blockiert nur das Angebot einer CLI, nicht den Bau.
2. `evidence.go` und `cli.go` — reine Daten, keine Prozesse; darauf bauen alle anderen auf.
3. `approval.go` — ohne Netz, ohne Prozess, voll testbar.
4. `proxy.go` — braucht `approval.go`; testbar mit `httptest` und einem gefälschten `ToolRunner`.
5. `server.go` — Verdrahtung; Routentests wie die bestehenden.
6. `clirun.go` — echter Kindprozess; der einzige Teil, der die Sandbox braucht (Loopback, `exec`).
7. Oberfläche und Doku zuletzt; sie folgen der API, nicht umgekehrt.

## Was nicht in diesen Plan gehört

- `agy` anbieten (Sperre nicht belegt).
- Die CLI-eigenen Werkzeuge proxyen.
- Kostenanzeige je Lauf.
- `sdd` selbst ändern.
