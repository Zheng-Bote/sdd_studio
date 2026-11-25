# Changelog

Alle nennenswerten Änderungen an SDD Studio (`sdd`). Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung [Semantic Versioning](https://semver.org/lang/de/). Der Regelkatalog trägt eine eigene Version (`catalog/catalog.yaml`).

## [Unreleased]

## [0.11.0] — 2026-11-25

Regelkatalog unverändert `1.3.0` mit 26 Regeln. Grundlage ist das Feature `build-linux-macos` ([PRD](docs/archive/prd/build-linux-macos.md), [Tech Spec](docs/archive/design/build-linux-macos.md), [Plan](docs/archive/plans/build-linux-macos.md)).

### Behoben

- **Build-Ziele hängen jetzt an ihren Quellen** (Issue [#35](https://github.com/Zheng-Bote/sdd_studio/issues/35)): `make install` kopierte vorhandene, aber veraltete Dateien — `bin/sdd`, `bin/sdd-desk` und das `.app`-Bundle hatten keine Voraussetzungen, galten damit als aktuell und wurden unverändert installiert. Wer eine Quelle änderte und `make install` fuhr, bekam das alte Ergebnis ohne jeden Hinweis. Jetzt hängen die Programme an ihren Go-Quellen sowie an `catalog/`, `handbuch/` und `templates/` (die `embed.go` einbettet), das Linux-Programm und das Bundle zusätzlich an den Tauri-Quellen; `make install` baut vor dem Kopieren. Ein Test hält die Abhängigkeiten fest.

### Geändert

- **Arbeitsweise über Issues und Feature-Branches** (Issue [#33](https://github.com/Zheng-Bote/sdd_studio/issues/33)): `AGENTS.md` nennt jetzt neben dem SDD-Ablauf, **wie** gearbeitet wird — Issue mit Vorlage, Feature-Branch, Pull Request mit `Closes #<n>`, nie direkt auf dem Standard-Branch; gemergt wird vom Menschen. Die Regel steht im Template `templates/agents.md.tmpl` (also in jedem neuen Projekt, das `sdd init` oder `sdd adopt` anlegt), im Kurzblock für bestehende Agenten-Dateien und in der `AGENTS.md` dieses Repositories. Dazu der Grund: `pre-merge` läuft in CI nur bei einem Pull Request, ein Push auf `main` löst das Gate nicht aus.
- **Issue-Vorlage auf SDD umgestellt** (Issue [#33](https://github.com/Zheng-Bote/sdd_studio/issues/33)): `.github/ISSUE_TEMPLATE/feature.md` fragt Feature-Slug, Problem, Ziel, Umfang, Nicht-Ziele, Akzeptanzkriterien als `AC-<n>` und die betroffenen Schichten ab und verweist auf Verfassung und Arbeitsweise. Die Vorlage `feature_spec.md` stammte aus einem anderen Projekt (EARS-Syntax, PostgreSQL-Schemata, Envelope Encryption, SPDX je Layer) und ist entfernt; ein Test hält ihre Begriffe draußen.

- **Bauen und Installieren auf Linux und macOS** (`build-linux-macos`, Issue [#31](https://github.com/Zheng-Bote/sdd_studio/issues/31), [PRD](docs/archive/prd/build-linux-macos.md), [Tech Spec](docs/archive/design/build-linux-macos.md), [Plan](docs/archive/plans/build-linux-macos.md)): `make` erkennt das System über `uname -s` (`OS=` erzwingt eines, `make print-os` zeigt es). Auf macOS baut es mit `tauri build --bundles app`, also ein startbares „SDD Studio Desktop.app“ statt nur eines nackten Programms, und `make install` kopiert das Bundle nach `$(APPDIR)` (Standard `~/Applications`, überschreibbar). `make install` schreibt auf macOS **keine** `.desktop`-Datei und kein hicolor-Icon-Thema mehr — macOS liest beides nicht — und räumt ein früher dort abgelegtes `sdd-desk-app`-Binary weg. Linux bleibt unverändert: Binaries, Desktop-Verknüpfung, Icons. `make uninstall` entfernt je System genau das, was `make install` angelegt hat. Tests prüfen die Zielpfade beider Systeme über einen Probelauf von `make`.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.11.0`): `build-linux-macos`.

## [0.10.0] — 2026-11-25

Regelkatalog unverändert `1.3.0` mit 26 Regeln. Grundlage ist das Feature `agenten-cli-proxy` ([PRD](docs/archive/prd/agenten-cli-proxy.md), [Tech Spec](docs/archive/design/agenten-cli-proxy.md), [Plan](docs/archive/plans/agenten-cli-proxy.md)) sowie [ADR 0010](docs/adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md).

### Hinzugefügt

- **Agenten-CLIs im Desktop über einen MCP-Proxy** (`agenten-cli-proxy`, [PRD](docs/archive/prd/agenten-cli-proxy.md), [Tech Spec](docs/archive/design/agenten-cli-proxy.md), [Plan](docs/archive/plans/agenten-cli-proxy.md), [ADR 0010](docs/adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md)): `sdd-desk` stellt einen MCP-Endpunkt auf `127.0.0.1` bereit, über den eine lokal installierte Agenten-CLI (`claude`, `codewhale`) die Werkzeuge von `sdd mcp` erreicht. Jeder `tools/call` wird zur Freigabekarte, korreliert über eine Kennung je Aufruf und mit eigener Frist; ohne Entscheidung liefert der Proxy kein Ergebnis, eine kritische Aktion läuft nur einzeln, und ein Ende der CLI-Sitzung nimmt offene Karten zurück. Jede CLI-Sitzung hat ein eigenes Bearer-Token; die CLI erhält nie `--allow-critical`. Angeboten wird nur, was belegt ist: der Belegstand je CLI steht in `internal/desk/evidence.go`, und ohne belegten Lauf gegen den Proxy wird eine CLI als „nicht belegt“ geführt und nicht angeboten — `agy` bleibt wegen der fehlenden Schreibsperre zurückgestellt. Die README-Zusage ist entsprechend auf Aktionen über `sdd` eingeschränkt.

  **Belegt:** `claude` ist durchgehend belegt — Transport über HTTP und ein echter Werkzeugaufruf durch das Modell samt Freigabekarte (`PROBE-OK:sdd_next`, keine Verweigerung, 0,12 USD). **Offen:** `codewhale` hat den Transport belegt, aber keinen Werkzeugaufruf: ohne Cloud-Schlüssel nutzt es lokales Ollama, und das Modell erfand das Ergebnis, ohne das Werkzeug aufzurufen — der Belegstand nennt das auf der Karte. Belege: [`docs/notes/headless-vertraege-agenten-clis.md`](docs/notes/headless-vertraege-agenten-clis.md).

  Die Oberfläche hat dafür einen eigenen Abschnitt in der Seitenleiste: CLI-Auswahl mit Begründung, warum eine CLI nicht angeboten wird, Belegstand je CLI, der Cloud-Hinweis vor dem ersten Lauf und die offenen Aufrufe mit Kennung, Sitzung und Frist; eine Sitzungsfreigabe wird für kritische Karten gar nicht erst angeboten.

### Behoben

- **Startzeile von `claude` hätte den Auftrag verschluckt:** `--mcp-config` und `--allowedTools` sind variadisch; ein dahinter stehender Prompt wäre als weitere Konfigurationsdatei bzw. als weiterer Werkzeugname gelesen worden. Der Prompt steht jetzt direkt hinter `-p`, und ein Lauf ohne Auftrag wird abgelehnt statt mit verrutschten Argumenten gestartet. Gefunden im Transportbeleg, festgehalten in `TestCLIStartFlags`.

- **Der CI-Workflow aus `sdd init` zeigt auf den erzeugenden Release-Tag:** `templates/workflow.yml.tmpl` trug den Action-Pin als Literal (`v0.7.0`) und blieb seit 0.7.0 stehen — ein frisch initialisiertes Projekt prüfte also mit einem zwei Minor-Versionen alten Stand. Das Template setzt den Tag jetzt aus `report.ToolVersion` (`uses: Zheng-Bote/sdd_studio@v<Version>`), und `TestInitWorkflowPinsToolVersion` hält beide Vorkommen im gerenderten Workflow fest.

- **`update-toc` scheiterte an `git push`:** Bei `pull_request` checkt `actions/checkout` den Merge-Commit in einem detached HEAD aus. Der Commit-Schritt committete dort und rief dann `git push` ohne Refspec — `fatal: You are not currently on a branch`, Exit 128. Der Fehler war latent, weil anfangs nie etwas zu committen war. Der Checkout nimmt jetzt den Head-Commit des PR, der Push nennt sein Ziel ausdrücklich, und für Pull Requests aus einem Fork ist der Commit-Schritt ausgesetzt. Belegt auf dem Runner: der Bot hat einen TOC-Commit selbst gepusht.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.10.0`): `agenten-cli-proxy`.

## [0.9.1] — 2026-11-22

Regelkatalog unverändert `1.3.0` mit 26 Regeln. Grundlage sind die Features `web-api-hardening` ([PRD](docs/archive/prd/web-api-hardening.md), [Tech Spec](docs/archive/design/web-api-hardening.md), [Plan](docs/archive/plans/web-api-hardening.md)), `mcpclient-lifecycle` ([PRD](docs/archive/prd/mcpclient-lifecycle.md), [Tech Spec](docs/archive/design/mcpclient-lifecycle.md), [Plan](docs/archive/plans/mcpclient-lifecycle.md)) und `clean-code-refactoring` ([PRD](docs/archive/prd/clean-code-refactoring.md), [Tech Spec](docs/archive/design/clean-code-refactoring.md), [Plan](docs/archive/plans/clean-code-refactoring.md)) sowie [ADR 0010](docs/adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md).

### Behoben

- **Clean Code: Toter Code, Defensives Dispatching und Deduplizierung** (`clean-code-refactoring`, Issue #24): Unerreichbaren `case classConfig:`-Zweig in `internal/mcp/write.go` (`toolWrite`) entfernt (`CC-01`), `sdd_adopt_apply` in `internal/mcp/critical.go` (`runCritical`) explizit gematcht und `default:` mit `errUnknownTool` abgesichert (`CC-03`), Inline-Code-Platzhalter in `internal/ui/markdown.go` auf numerische Indizes umgestellt (`CC-02`) sowie `ui.OpenBrowser` zwischen `cmd/sdd/ui.go` und `cmd/sdd-desk/main.go` dedupliziert (`CC-04`).
- **Robustes Ressourcenmanagement und idempotenter Lifecycle im MCPClient** (`mcpclient-lifecycle`, Issue #23): `MCPClient.Call` (`internal/desk/mcpclient.go`) entfernt ausstehende Request-IDs per `defer` aus `c.pending` und stoppt den Timeout-Timer sofort mit `defer timer.Stop()`, sodass Timeouts oder Schreibfehler keine Einträge leaken (`RES-01`). `MCPClient.Close()` ist durch `sync.Once` idempotent abgesichert und wartet nach `cmd.Process.Kill()` auf die Rückkehr von `cmd.Wait()`, damit kein Zombie-Prozess verbleibt (`RES-02`).
- **Security Hardening für lokale Web-APIs** (`web-api-hardening`, Issue #22): `POST /api/open` in `sdd ui` (`internal/ui/api.go`) prüft Zielpfade mit `os.Lstat` und `filepath.EvalSymlinks` gegen das Repository-Wurzelverzeichnis, sodass symbolische Links oder über verlinkte Elternverzeichnisse ausbrechende Pfade abgewiesen werden und `$EDITOR` nicht starten (`SEC-01`). `sdd-desk` (`internal/desk/server.go`) validiert bei allen `POST /api/*`-Anfragen analog zu `sdd ui` zusätzlich den `Origin`-Header gegen `"http://" + s.Addr` (`SEC-02`).

### Geändert

- **Headless-Verträge der Agenten-CLIs belegt, ADR 0010 angenommen** ([Belegnote](docs/notes/headless-vertraege-agenten-clis.md), [ADR 0010](docs/adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md), Phase 7 Teil B): Die fünf Fragen aus ADR 0010 — headless, Ausgabeformat, Abbruchsignal, MCP-Annahme, wirksame Schreibsperre — sind für `claude` 2.1.330, `agy` 1.5.9 und `codewhale` 0.10.3 gegen die installierten Programme geprüft, mit Versuchsaufbau und Belegen. Ergebnis: `claude` und `codewhale` bestehen beide Bedingungen — bei beiden ging ein MCP-Werkzeugaufruf durch (`PROBE-MCP-OK`), während ein Shell-Schreibversuch abgewiesen wurde (bei `claude` als `permission_denials`, bei `codewhale` durch die Betriebssystem-Sandbox mit `Operation not permitted`; Datei jeweils nicht angelegt), und beide enden auf `SIGTERM` mit Exit 143. `agy` besteht die Schreibsperre nicht (Standardmodus führt Shell-Schreibbefehle ohne Freigabe aus, `--mode plan` sperrt auch MCP) und wird zurückgestellt. ADR 0010 ist damit `accepted` und mit `SDD-011` gelockt. Ein Befund für den Bau: `--allowed-tools` ist bei `codewhale` eine Allowlist, die auch den Aufrufpfad (`tool_search`) wegnimmt — die Liste muss ihn einschließen, sonst ist selbst das erlaubte MCP-Werkzeug nicht ladbar.
- **Diskussionsnotiz um OpenSpec und die Einstiegspunkte erweitert** ([`docs/notes/kombination-specdd-spec-kit-bmad.md`](docs/notes/kombination-specdd-spec-kit-bmad.md)): Die Quelle zur Makro-Ebene beschreibt jetzt vier Bausteine und trennt **Greenfield, Brownfield und BMAD-Vorgeschichte**, weil sie unterschiedliche Reihenfolgen verlangen — im Bestand kommt die Bestandsaufnahme vor der Verfassung. Die Angaben zu OpenSpec sind gegen die Primärquelle geprüft (`openspec/changes/<change>/` mit `proposal.md`, `specs/`, `design.md`, `tasks.md`; `openspec/specs/`; Archiv nach Abschluss). Die Notiz hält außerdem fest, dass BMAD v6 sein Werkzeug unter `.agents/skills/…` installiert — `sdd adopt` erkennt deshalb am Ergebnis `_bmad-output/`, nicht am Werkzeugordner `_bmad/`. Anleitung: der fehlende Abschnitt **C) Projekt mit BMAD-Vorgeschichte** steht jetzt im Inhaltsverzeichnis.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.9.1`): `web-api-hardening`, `mcpclient-lifecycle`, `clean-code-refactoring`.

## [0.9.0] — 2026-11-22

Regelkatalog unverändert `1.3.0` mit 26 Regeln. Grundlage ist das Feature `desk-model-availability` — [PRD](docs/archive/prd/desk-model-availability.md), [Tech Spec](docs/archive/design/desk-model-availability.md).

### Behoben

- **Der Abschluss eines Features widersprach `SDD-050`:** `sdd next`, Handbuch und Anleitung verlangten nach dem Merge `status: implemented` — dieselbe Regel meldet genau diesen Zustand aber als Befund, und `TestSelfDogfoodingClean` verlangt null Befunde auf dem eigenen Repository. Nach dem Merge bleibt der Status jetzt unverändert; archiviert wird im Release-Vorbereitungs-Commit über `sdd archive`, das `status: archived` setzt. `status: implemented` ist ein Erinnerungsstand: er schaltet den Schritt in `sdd next` von „später" auf „offen" und wird bis zur Archivierung gemeldet (`warn`, in `pre-release` `error`). Dieselbe Korrektur im Hinweis des Übernahmeberichts (`sdd adopt`) und in der Meldung, mit der `sdd archive` einen Entwurf ablehnt.

### Hinzugefügt

- **Modelle, die ehrlich verfügbar sind** (`desk-model-availability`, Phase 7 Teil A): Der Desktop nennt zu jedem lokalen Modell, ob es Werkzeuge aufrufen kann (`POST /api/show`); ein Modell ohne Werkzeugaufruf bleibt wählbar, ist in der Auswahl als „ohne Werkzeuge“ gekennzeichnet und wird nicht vorausgewählt. Ein Knopf **Modelle aktualisieren** liest Anbieter und Modellliste neu, sodass ein frisch geholtes `ollama pull` ohne Neustart erscheint. Für die Abläufe, deren Ergebnis ein Urteil des Modells ist (`sdd-conflicts`, `sdd-review`), nennt die Karte den Belegstand zum gewählten Modell — belegt, im Test durchgefallen oder nicht belegt — aus einer Tabelle mit Quelle (`internal/desk/evidence.go`); ohne Testeintrag steht dort „nicht belegt“, nie „belegt“.

### Geändert

- **ADR 0010 überarbeitet, weiterhin `proposed`:** Lokale Agenten-CLIs (`claude`, `agy`, `codewhale`) kämen nur über einen MCP-Proxy im Desktop zum Einsatz; `sdd` bleibt KI-frei und sieht `--allow-critical` nie. Die Bedingung ist jetzt prüfbar gefasst — nicht „die CLI bringt keine eigenen Werkzeuge mit“ (für diese CLIs unerfüllbar), sondern „eigene Schreibzugriffe sind mechanisch verhindert“ — und der Status bleibt `proposed`, bis Headless-Vertrag und wirksame Schreibsperre je CLI belegt sind ([ADR 0010](docs/adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md)).

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.9.0`): `desk-model-availability`.

## [0.8.0] — 2026-11-20

Regelkatalog `1.3.0` mit 26 Regeln. Grundlage ist [ADR 0009](docs/adr/0009-bmad-import-statt-zweitem-prozess.md) — [PRD](docs/archive/prd/bmad-import.md), [Tech Spec](docs/archive/design/bmad-import.md) — und die Notiz [Kombination von SpecDD, Spec Kit und BMAD](docs/notes/kombination-specdd-spec-kit-bmad.md).

### Hinzugefügt

- **Makro-Ebene für BMAD-Bestände** ([ADR 0009](docs/adr/0009-bmad-import-statt-zweitem-prozess.md)): `sdd.config.yaml` kennt `pipeline` (`sdd` oder `bmad`) und `artifacts.initiative` (Vorgabe `docs/initiative.md`). Mit `pipeline: bmad` führt `sdd` eine **Initiative** mit nummerierten Anforderungen (`R1`, `R2`, …) und **Epics**, die in `covers` nennen, welche Anforderungen sie übernehmen.
- **`sdd adopt` erkennt BMAD v6** (`_bmad/`, `_bmad-output/`): Die Initiative wird als Typ Initiative klassifiziert, jede `epic-*.md` als Epic, `_bmad/**` und `**/.memlog.md` bleiben aus dem Inventar, und die vorgeschlagene Konfiguration trägt `pipeline: bmad`. Makro-Artefakte bekommen kein `feature:`-Frontmatter und werden nie umgeschrieben. Ohne `_bmad/` bleibt jedes Verhalten wie bisher.
- **`sdd init --pipeline bmad`** schreibt die Initiative aus einem Template und legt den Epic-Ordner an; **`sdd new --type initiative|epic`** legt beide an (Epic mit `parent`, `covers`, `refined: false`).
- **Drei Regeln für die Makro-Ebene:** `SDD-080` jedes Epic übernimmt mindestens eine existierende Anforderung (`error`, pre-implementation und pre-merge), `SDD-081` jede Anforderung ist gedeckt oder im Frontmatter `unplanned` genannt (`warn`), `SDD-082` `tickets.toml` und die Epic-Dateien nennen dieselben Slugs (`warn`). Alle drei prüfen nur bei `pipeline: bmad`.
- **`sdd next` und der Desktop zeigen zwei Etagen:** die Makrostationen (Initiative, Verfassung, Übernahme, Architektur) vor den Features; ein Epic mit `refined: false` erscheint als ein offener Schritt „Epic verfeinern“ statt als unfertiges Feature, und die Regeln `SDD-021`, `SDD-022`, `SDD-023`, `SDD-031` und `SDD-041` überspringen es.
- **Drei geführte Abläufe:** `sdd-ingest-bmad` (Anforderungen wörtlich übernehmen, Done-when → `AC-<n>`, Constraints mit Fundstelle in die Verfassung, verworfene Optionen als ADR), `sdd-architecture` (Spine aus dem Bestand, arc42, ADR je Entscheidung) und `sdd-epic-refine` (Requirements festlegen, `refined: true`).
- **`sdd_read` liest den BMAD-Ledger** (`_bmad-output/**/*.md`) als Quelle; `sdd_write` und `sdd_commit` lehnen ihn ab. `sdd_new` nimmt `covers`, `sdd_init` nimmt `pipeline`.

### Geändert

- **Notiz als Quelle:** die Diskussionsnotiz liegt jetzt unter [`docs/notes/`](docs/notes) und ist versioniert; `data/` (lokale Bestände, Obsidian-Ablage) steht in `.gitignore`.
- Katalogversion `1.3.0`; der Test, der die Katalogversion prüft, verlangt jetzt mindestens die Version, in der `SDD-012` entstand.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.8.0`): `bmad-import`.

## [0.7.0] — 2026-11-08

Regelkatalog unverändert `1.2.0` mit 23 Regeln. Grundlage von `sdd_commit` ist [ADR 0008](docs/adr/0008-commit-werkzeug-im-ki-desktop.md).

### Hinzugefügt

- `sdd_conflicts_check` meldet `complete` und „Prüfung unvollständig“, solange Regeln der Verfassung in der Liste fehlen.
- `sdd_trace` nennt in `quality` je Qualitätsanforderung `QA-<n>` die Tech Specs und Tests und markiert doppelte Kennungen.
- **`sdd_commit`** ([ADR 0008](docs/adr/0008-commit-werkzeug-im-ki-desktop.md)): Kritisches Werkzeug, das im KI-Desktop einzeln genannte Artefakte, Testdateien, die Konfiguration und den Prüfzustand committet, nach einzelner Bestätigung und nie auf dem Standard-Branch.
- Die Seitenleiste von `sdd-desk` zeigt die Projektschritte aus `sdd next`: Verfassung ausfüllen, Übernahme abschließen, Architekturdoku anlegen bzw. nachführen.
- `sdd next` nennt im Abschlussschritt ein PRD, das noch `status: draft` trägt, weil `sdd archive` Entwürfe ablehnt.

## [0.6.0] — 2026-11-08

Regelkatalog `1.2.0` mit 23 Regeln. Grundlage ist ein Live-Test des KI-Desktops mit Claude Sonnet 5.5 (Szenario cert-expiry-watcher, vom leeren Ordner bis zu den Tests je Kriterium), [ADR 0004](docs/adr/0004-konfiguration-aendern-ist-kritisch.md) und [ADR 0005](docs/adr/0005-abschwaechung-der-konfiguration-braucht-ein-adr.md).

### Hinzugefügt

- **Erweitertes Anforderungsmanagement** ([ADR 0007](docs/adr/0007-qualitaetsanforderungen-im-prd-mit-kennung-qa.md)): Der optionale geführte Ablauf `sdd-requirements` vertieft ein PRD für mittlere und größere Vorhaben — messbare Qualitätsanforderungen mit Kennung `QA-<n>` je Merkmal, Randbedingungen, Annahmen und Abhängigkeiten, Grenzfälle als neue Akzeptanzkriterien (auf Wunsch Gegeben/Wenn/Dann), Hinweise auf vage Formulierungen. `sdd-idea` und `sdd-start` bieten ihn an, `sdd-arc42` nutzt die `QA-<n>`. Keine neue Pflichtregel.
- **Architekturdokumentation nach arc42** ([ADR 0006](docs/adr/0006-arc42-template-unter-cc-by-sa-einbetten.md)): `sdd new --type arc42` (und `sdd_new`) legt `docs/arc42.md` aus einem mitgelieferten deutschen arc42-Template mit Mermaid-Diagrammen an (CC BY-SA 4.0, siehe `NOTICE`); neues Konfigurationsfeld `artifacts.architecture`. `sdd next` erinnert zum Release daran, sie nachzuführen, der geführte Ablauf `sdd-arc42` füllt sie aus den Quellen des Projekts. SDD Studio selbst hat seine Doku in [`docs/arc42.md`](docs/arc42.md).
- **`SDD-012` — Abschwächende Änderung der Konfiguration verweist auf ein ADR** (ADR 0005): In `pre-implementation` und `pre-merge` ein `error`, wenn `sdd.config.yaml` gegenüber der Merge-Basis ein niedrigeres Profil, einen geänderten Override, ein neues `exclude`-Muster oder gelockerte Schwellen bekommt und kein ADR im selben Diff die Datei nennt. Bestehende Konfigurationen sind nicht betroffen.
- **Katalogattribut `pinned`:** Eine so markierte Regel läuft in den Gates ihrer `gate_default` auch neben einem `gates.<gate>.override`; sie muss `profile: minimal` und `baseline: forbidden` tragen. Bisher einzige: `SDD-012`.
- **`sdd_config`:** kritisches Werkzeug (nur mit `--allow-critical`) für Änderungen an `sdd.config.yaml`. Die Vorschau zeigt den Diff und je Prüflauf (Lint und die drei Gates) die Befunde, die mit der neuen Konfiguration wegfallen oder hinzukommen; die Schaltfläche nennt die geänderten Schlüssel, einen Profilwechsel mit altem und neuem Wert und jede Änderung an `override`.
- **`sdd_conflicts_check`:** lesendes Werkzeug, das das Ergebnis von `sdd-conflicts` maschinell prüft: Steht jedes Zitat wörtlich in der genannten Datei und Zeile, und fehlt eine Regel der Verfassung in der Regelliste? Es vergleicht nur Zeichenketten und bewertet nichts. `sdd-conflicts` ruft es vor der Antwort auf und zeigt nur belegte Meldungen.

### Geändert

- **Konfiguration ändern ist im Desktop kritisch** ([ADR 0004](docs/adr/0004-konfiguration-aendern-ist-kritisch.md)): einzeln bestätigt, nie für die Sitzung. `sdd_write` lehnt `sdd.config.yaml` ab; andere MCP-Clients ohne `--allow-critical` können die Konfiguration nicht mehr ändern.
- **Sitzungsfreigabe je Stufe:** Eine Freigabe für die Sitzung gilt nur für die Stufe der Karte, auf der sie erteilt wurde. Auf einer Vorschau („nur Vorschau“) erlaubt sie nur weitere Vorschauen, Schreiben wird erneut gefragt. Bisher gab sie auch spätere Schreibvorgänge frei, ohne dass der Inhalt je gezeigt worden war. API: `allowed_writes` neben `allowed_tools`.
- **`sdd archive` und `sdd_archive` lehnen ein Feature ab, dessen PRD noch `status: draft` trägt.** Bisher genügte ein Testverweis je Kriterium, im Live-Test hätte so ein Entwurf mit lauter übersprungenen Gerüst-Tests archiviert werden können.
- Konzept §9.6: `baseline: forbidden` schützt nur vor der Baseline; `profile: minimal` schaltet SDD-011, SDD-050 und SDD-051 ab.

### Behoben

- **`sdd mcp` schreibt wieder in Unterverzeichnisse:** Weil `sdd` mit dem Wurzelverzeichnis `.` läuft, galt jede bestehende Datei unter z. B. `docs/prd/` als „außerhalb des Repository-Wurzelverzeichnisses (über symbolischen Link)“. `sdd_write` konnte deshalb kein PRD, ADR und keine weitere Testdatei im selben Ordner ändern, und `sdd_adr_accept` setzte das ADR auf `accepted`, meldete danach aber diesen Fehler. Der Schutz vor symbolischen Links aus dem Repository heraus bleibt unverändert.
- `sdd next` zählt bei „Akzeptanzkriterien mit Kennung“ jede Kennung nur einmal; eine in offenen Punkten erneut genannte `AC-n` ergab bisher ein Kriterium zu viel.
- Die Beschreibung von `sdd_write` bot nach der Einführung von `sdd_config` noch `sdd.config.yaml` an.
- `sdd baseline --prune` schrieb `"entries": null`, wenn es den letzten Eintrag entfernte; jetzt steht dort `[]`.
- Ein unbekanntes Argument eines MCP-Werkzeugs wird weiter abgelehnt; die Meldung nennt jetzt die erlaubten Felder und fordert einen neuen Aufruf an, statt nur das Feld zu nennen.
- `sdd_conflicts_check` nennt bei einem unbekannten Feature-Slug die Slugs, für die es eine Tech Spec gibt.

## [0.5.0] — 2026-11-02

Regelkatalog unverändert `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`. Grundlage ist [ADR 0003](docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md), das ADR 0002 ablöst: Der KI-Desktop darf alles, was `sdd` kann, freigegeben in drei Stufen (lesend, schreibend, kritisch).

### Hinzugefügt

- **Arbeitsergebnisse mit Inhalt speichern:** `sdd_new` nimmt mit `content` den geprüften Entwurf statt des Templates. Das neue Werkzeug `sdd_write` legt Artefakte, `sdd.config.yaml` und Testdateien an oder ändert sie: eine bestehende Datei nur mit `base_sha256`, atomar, vorher mit den Regeln geprüft, mit Diff in Vorschau und Ergebnis. Ein ADR setzt es nie auf `accepted`.
- **`sdd init --git`** und das Werkzeug `sdd_init`: in einem Ordner ohne Repository `git init` (Branch `main`) und ein erster Commit genau der angelegten Dateien; vorher geprüft, ob Git und eine Git-Identität vorhanden sind (zweite Ausnahme vom lesenden Git-Zugriff, Verfassung Security 4).
- **`sdd archive --feature <slug> [--release <v>]`** und das Werkzeug `sdd_archive`: verschiebt PRD, Tech Spec und Plan nach `artifacts.archive` und setzt `status: archived`, nur wenn jedes Akzeptanzkriterium einen Testverweis hat.
- **Kritische Werkzeuge** hinter `sdd mcp --allow-critical`: `sdd_baseline`, `sdd_adr_accept`, `sdd_adopt_apply`. `sdd-desk` führt sie nur nach einzelner Bestätigung aus, nie für die ganze Sitzung, und lehnt nach einer Ablehnung weitere Aufrufe bis zur nächsten Nachricht ab.
- **`sdd_stack`:** erkennt Sprachen aus Dateiendungen und Marker-Dateien, mit Testdatei-Mustern und Kommentarsyntax je Sprache.
- **Agenten-Dateien:** `sdd init` und `sdd adopt` legen `AGENTS.md` (Verfassung, Präzedenz, Ablauf) sowie `CLAUDE.md`, `GEMINI.md` und `.github/copilot-instructions.md` mit Verweis darauf an; vorhandene Dateien erhalten den Verweis vorangestellt, ohne dass sich ihr übriger Inhalt ändert.
- **Verfassung erarbeiten:** `sdd next` meldet „Verfassung ausfüllen“, solange Platzhalter des Templates drin sind; der Ablauf `sdd-constitution` schlägt Regeln mit Fundstelle aus README und Agenten-Dateien vor, die `sdd_read` jetzt lesen darf.
- **Neue geführte Abläufe:** `sdd-adopt`, `sdd-start`, `sdd-constitution`, `sdd-tests`. `sdd-desk` erkennt ein nicht erfasstes Projekt beim Öffnen und bietet Übernahme oder Neuanlage an.
- `sdd next` meldet nach `sdd adopt --apply` „Übernahme abschließen“, solange der Branch der Übernahme nicht gemergt ist.
- `Makefile` baut alle Programme und installiert sie lokal (`make`, `make install`); `docs/anleitung.md` beschreibt die Einführung in ein neues oder bestehendes Projekt.

### Geändert

- `sdd-desk` darf schreibende Werkzeuge für die Sitzung freigeben; was sie geschrieben haben, steht mit Diff im Gespräch.
- `sdd-idea` klärt die Technik vor den touches; jeder geführte Ablauf nennt die Reihenfolge des Prozesses und endet mit einer Zusammenfassung in Worten.
- `sdd-desk` nennt dem Modell das heutige Datum; `sdd new` lehnt ein `archive-after` vor heute ab.
- Die Logik von `sdd baseline` liegt in `sddcore` und wird von CLI und `sdd_baseline` geteilt; Ausgabe und Dateien sind unverändert.
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.5.0`.

### Behoben

- Der Chat in `sdd-desk` wird nicht mehr breiter als das Fenster.
- Ein ungültiges Gate nennt die erlaubten Werte; fehlt das Git-Repository oder der erste Commit, sagt die Meldung das statt „base_ref setzen“.
- Eine Antwort des Modells ohne Text erscheint in `sdd-desk` als Hinweis statt gar nicht.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.5.0`): `archive-after-today`, `desk-write-content`, `sdd-init-git`, `sdd-archive`, `desk-critical-tools`, `desk-guided-flow`, `agent-files`, `constitution-guide`, `desk-project-start`, `flow-closing`.

## [0.4.0] — 2026-10-21

Regelkatalog unverändert `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`.

### Hinzugefügt

- **Abgleich mit der Verfassung** (`SDD-060`, assistiert): Der geführte Ablauf `sdd-conflicts` lässt ein Modell in `sdd-desk` die Tech Spec eines Features gegen jede Regel der Verfassung halten und nur Widersprüche melden, die es auf beiden Seiten wörtlich mit Datei und Zeile zitiert. Das Ergebnis ist ein Hinweis, kein Befund: `SDD-060` ist keine Katalogregel und erscheint in keinem Bericht und keiner Baseline.
- **`sdd_read`** in `sdd mcp` (jetzt elf Werkzeuge): liest Artefakte (Verfassung, ADRs, PRDs, Tech Specs, Pläne, Archiv) mit Zeilennummern oder listet sie; nichts anderes, nur `.md`/`.sdd`, über 256 KiB gekürzt.
- **Frist je Anbieter** in `sdd-desk`: `<ANBIETER>_TURN_TIMEOUT` in `llm.env` (1m bis 4h).

### Geändert

- Ein Gesprächszug mit lokalem Ollama darf 30 statt 10 Minuten dauern; Cloud-Anbieter bleiben bei 10 Minuten. Eine einzelne Anfrage an den Anbieter hat keine eigene, kürzere Frist mehr.
- Konzept und Handbuch führen `SDD-060` als assistierten Abgleich statt als Katalogregel.
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.4.0`.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.4.0`): `sdd-conflicts`, `sdd-desk-turn-timeout`.

## [0.3.0] — 2026-10-21

Regelkatalog unverändert `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`.

### Hinzugefügt

- **„LLM testen“ in `sdd-desk`** (`POST /api/model/test`): schickt eine feste Testnachricht samt Werkzeugbeschreibungen an das gewählte Modell und meldet, ob es antwortet und ob es Werkzeuge annimmt — mit der Meldung des Anbieters, z. B. bei einem nicht mehr angebotenen Modell. Keine Gesprächs- oder Projektinhalte; bei Cloud-Anbietern erst nach dem Hinweis.
- **Eigenes Icon** der Desktop-Anwendung und der Web-Oberfläche von `sdd-desk`.

### Geändert

- Die Desktop-Anwendung bringt `sdd` als `sdd-studio-sdd` mit; ein Paket installiert kein zweites `/usr/bin/sdd` mehr. `sdd-desk` sucht zuerst `sdd-studio-sdd`, dann `sdd` neben sich, dann im `PATH`.
- Fehlt `llm.env`, nennt die Warnung in der Desktop-Anwendung den Konfigurationsordner der Anwendung statt eines Pfads neben dem Sidecar (`sdd-desk --config-hint`).
- Handbuch: Änderungen über der Größenschwelle gehören vor dem Release-Branch auf den Haupt-Branch, weil dort nach dem Archivieren kein Feature mehr auflösbar ist.
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.3.0`.

### Behoben

- Lokal ermittelte Merge-Basis: Nach dem Push eines Feature-Branches war `@{upstream}` dessen eigene entfernte Kopie und der Diff für `pre-merge` leer; diese Kopie gilt nicht mehr als Basis, stattdessen `main` bzw. `master`.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.3.0`): `sdd-desk-model-test`, `sdd-desk-config-hint`, `sdd-desk-sidecar-name`.

## [0.2.0] — 2026-10-21

Regelkatalog `1.1.0` mit 22 Regeln, Berichtsformat unverändert `schema_version: 1`.

### Hinzugefügt

- **Regeln:** `SDD-003` (Verfassung ohne Feature-Inhalte, heuristisch) und `SDD-040` (technischer Plan vor der ersten Task).
- **`sdd ui`:** lokale Oberfläche auf `127.0.0.1` mit Übersicht, Artefakten, Befunden (live), neuem Artefakt, Übernahme und Handbuch; `Host`-, `Origin`- und Token-Prüfung, Schreiben nur über Plan/Apply.
- **Prozessführung:** `sdd next` nennt je Feature den nächsten Prozessschritt; `sdd explain <regel-id>` zeigt Regel, Begründung und Prüfvertrag.
- **`sdd mcp`:** MCP-Server über stdio für Agenten mit zehn Werkzeugen, Ressourcen und geführten Prompts (Idee → PRD, Tech Spec, ADR, Plan, Review); ohne KI und offline, Schreiben nur mit `--allow-write` und nur neue Artefakte.
- **KI-Desktop `sdd-desk`** (optional, getrennt, [ADR 0002](docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)): führt im Gespräch mit einem wählbaren Sprachmodell (Ollama lokal/Cloud, OpenRouter, NVIDIA NIM, Claude, Gemini) durch den Prozess; Werkzeuge nur über `sdd mcp` und nur mit Freigabe, schreibende mit Vorschau; Hinweis vor Cloud-Anbietern; Schlüssel in `llm.env` mit Rechten `0600`; keine Telemetrie.
- **Desktop-Anwendung** (Tauri v2) mit `sdd-desk` und `sdd` als Sidecars: Projektwahl und -wechsel, Navigationsregel, keine Tauri-Rechte für Seiten; `sdd-desk --exit-with-stdin` endet mit der Anwendung.
- **Gestreamte Antworten** in `sdd-desk` (`POST /api/turn/stream`): Text und Gedankengang erscheinen, während das Modell arbeitet; Abbrechen beendet auch die Anfrage an den Anbieter.

### Geändert

- `SDD-022`/`SDD-025`: Testverweise nennen den Feature-Slug, sobald es mehr als ein Feature gibt (archivierte eingeschlossen).
- Die GitHub Action und die Workflow-Vorlage von `sdd init` verweisen auf `@v0.2.0`.

### Behoben

- `sdd next` verlangte ein ADR für eine Verfassungsänderung, obwohl ein verlinktes, angenommenes ADR sie abdeckte und `SDD-010` im Gate nichts meldete.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.2.0`): `sdd-rules-003-040`, `sdd-ui`, `sdd-guidance`, `sdd-desk`, `sdd-desk-tauri`, `sdd-desk-streaming`.

## [0.1.0] — 2026-10-20

Erste Version. Regelkatalog `1.0.0` mit 20 Regeln, Berichtsformat `schema_version: 1`.

### Hinzugefügt

- **Prüfkern und CLI:** `sdd lint`, `sdd gate pre-implementation|pre-merge|pre-release` und `sdd baseline` mit stabilen Exit-Codes (`0`/`1`/`2`) und JSON-Bericht; gate-gebundene Baseline mit `content_hash` und deterministischem `ordinal`.
- **Regeln des v1-Schnitts:** `SDD-001`, `SDD-002a`, `SDD-020`, `SDD-021`, `SDD-022`, `SDD-030`, `SDD-050` sowie die Selbstprüfung `SDD-070` bis `SDD-073`.
- **Regeln des v1.1-Schnitts:** `SDD-002b`, `SDD-010`, `SDD-011` (mit `.sdd/adr-lock.json`), `SDD-023`, `SDD-024`, `SDD-025`, `SDD-031`, `SDD-041`, `SDD-051`.
- **Projektanlage:** `sdd init` (Konfiguration, Verfassung, Artefaktordner, CI-Workflow, leere Baseline) und `sdd new` (PRD, Tech Spec, Plan, ADR, Doku aus eingebetteten Templates).
- **Übernahme:** `sdd adopt` mit Inventar, Klassifikation in sechs Artefakttypen, Reifegrad L0–L4, Probelauf und `--apply` auf einem neuen Branch mit einem Commit je Schritt ([ADR 0001](docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md)).
- **Bericht:** `sdd report` mit Reifegrad, Gate-Status und Schuldenstand samt Trend gegenüber der Merge-Basis.
- **Artefaktpfade als Ordner oder Glob**, z. B. für GitHub Spec Kit (`specs/features/*/spec.md`).
- **GitHub Action** (`action.yml`): baut `sdd`, führt das Gate aus und schreibt den Bericht in die Job-Zusammenfassung.

### Archiviert

Die Feature-Artefakte dieser Version liegen unter `docs/archive/` (`status: archived`, `release: v0.1.0`): `sdd-core-v1`, `sdd-init-new`, `sdd-adopt`, `sdd-report`, `sdd-rules-v1-1`.

[0.5.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.5.0
[0.4.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.4.0
[0.3.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.3.0
[0.2.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.2.0
[0.1.0]: https://github.com/Zheng-Bote/sdd_studio/releases/tag/v0.1.0
