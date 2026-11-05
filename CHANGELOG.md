# Changelog

Alle nennenswerten Änderungen an SDD Studio (`sdd`). Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung [Semantic Versioning](https://semver.org/lang/de/). Der Regelkatalog trägt eine eigene Version (`catalog/catalog.yaml`).

## [0.6.0] — 2026-11-05

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
