# 3. Mechanismen, Gates und Regelkatalog-Referenz

Dieser Teil des Handbuchs dokumentiert die Prüfmechanismen von `sdd`, das Berichtsformat sowie die Referenz und die verbindlichen Prüfverträge aller Regeln im Katalog (`catalog/catalog.yaml`).

---

## Projektanlage, Templates, Übernahme, Bericht, Prozessführung, Agenten und Oberfläche (`sdd init`, `sdd new`, `sdd adopt`, `sdd report`, `sdd next`, `sdd explain`, `sdd mcp`, `sdd ui`)

`init`, `new` und `adopt` schreiben ausschließlich **additiv**; `report` schreibt nichts: Eine bestehende Datei wird nie überschrieben, sondern als `übersprungen` gemeldet. Alle Zielpfade werden vor dem ersten Schreibzugriff geprüft; Pfade außerhalb des Repositorys (absolut, `..`, symbolischer Link nach außen) brechen mit Exit-Code `2` ab. `--dry-run` zeigt den Plan, ohne zu schreiben.

### `sdd init` {#sdd-init}

```text
sdd init [--profile minimal|standard|regulated] [--dry-run] [--no-ci]
```

Legt an: `sdd.config.yaml` (Profil, Standardpfade, leere Schwellen), `constitution.md` mit den Pflichtabschnitten aus `SDD-002a`, die Verzeichnisse `docs/{adr,prd,design,plans,archive,notes}` (je mit `.gitkeep`, nur wenn das Verzeichnis fehlt), den Workflow `.github/workflows/sdd.yml` und eine leere `.sdd/baseline.json` — ein neues Projekt startet ohne Altlasten.

**Danach:** `thresholds.layers` und `thresholds.feature_artifacts_required_when.touched_paths` an das Projekt anpassen. Solange beide leer sind, ist die [Größenschwelle](#groessenschwelle) nie überschritten.

### `sdd new` {#sdd-new}

```text
sdd new --feature <slug> [--type prd,design,plan] [--title <text>] [--touches <globs>] [--archive-after YYYY-MM-DD]
sdd new --type adr|doc --title <text> [--slug <slug>]
```

| Typ | Ziel | Frontmatter |
|---|---|---|
| `prd` | `<artifacts.prd>/<slug>.md` | `feature`, `status: draft`, `archive-after`, `touches` |
| `design` | `<artifacts.design>/<slug>.md` | wie `prd` |
| `plan` | `<artifacts.plans>/<slug>.md` | `feature`, `status: draft`, `archive-after` |
| `adr` | `<artifacts.adr>/<NNNN>-<slug>.md` | `status: proposed`, `date`, `supersedes`, `references` |
| `doc` | `<artifacts.docs[0]>/<slug>.md` | `status: active` |

- Ohne `--type` legt `--feature` PRD und Tech Spec an — genau das Paar, das `SDD-020`/`SDD-030` verlangen.
- Trägt im Zielverzeichnis bereits eine Datei `feature: <slug>`, wird für diesen Typ nichts angelegt, auch wenn der Dateiname abweicht.
- `archive-after` bleibt ohne Angabe leer (kein Stichtag); `touches` ohne `--touches` eine leere Liste — `pre-implementation` meldet dann „Absicht nicht erklärt".
- Die ADR-Nummer ist die größte vierstellige Präfixnummer im ADR-Verzeichnis plus eins; der Slug wird aus dem Titel abgeleitet (Umlaute transliteriert).
- Das PRD-Template enthält `AC-1` als Platzhalter: Es erfüllt `SDD-021`, verlangt nach `SDD-022` aber einen Test — bei mehreren PRDs mit Slug im Testkommentar (`// <slug> AC-1: …`).
- Ohne `sdd.config.yaml` verwendet `sdd new` die Standardpfade von `sdd init` und weist darauf hin.

### `sdd adopt` {#sdd-adopt}

```text
sdd adopt [--profile minimal|standard|regulated] [--apply] [--branch sdd/adopt]
```

Übernimmt ein bestehendes Projekt. Ohne `--apply` ist es ein **Probelauf**: Der Adopt-Bericht erscheint auf der Standardausgabe, geschrieben wird nichts.

1. **Inventar** — alle Markdown-Dateien (ohne `exclude` und `.sdd/`), CI-Workflows (`.github/workflows/*.yml|yaml`, `.gitlab-ci.yml`), vorhandene Konfiguration, Verfassung und Baseline. Ohne `sdd.config.yaml` wird eine vorgeschlagen: Spec-Kit-Layout, wenn `specs/features/*/spec.md` existiert, sonst die Pfade von `sdd init`; der ADR-Ordner ist der erste aus `docs/adr`, `docs/architecture/adr`, `doc/adr`, `adr` mit einer `NNNN-*.md`.
2. **Klassifikation** — jede Datei genau ein Artefakttyp, erste passende Regel: Verfassung, ADR (`NNNN-*.md` im ADR-Ordner), PRD/Tech Spec/Plan nach `artifacts.*`, Archiv, sonst Doku/Exploration („kein Muster“).
3. **Reifegrad** — L1 Verfassung ohne Befunde; L2 PRD und Tech Spec getrennt, AC-Kennungen ohne `SDD-021`-Befunde; L3 CI ruft `sdd gate` auf, Baseline vorhanden; L4 Feature-Artefakte tragen `feature:` und `status`, Archiv konfiguriert, keine `SDD-050`-Befunde. Vorgeschlagen wird genau das erste nicht erfüllte Kriterium der nächsten Stufe.
4. **Schritte** — je Schritt ein Commit, Schritte ohne Änderung entfallen: *Konfiguration* (falls keine vorhanden), *Verfassung* (aus dem Template, falls keine vorhanden), *Frontmatter* (`feature: <slug>` und `status: active` vor den Inhalt aktiver Feature-Artefakte ohne Frontmatter; der Slug ist das `*`-Segment des Globs bzw. der Dateiname), *Baseline* (alle Gates, nach den vorherigen Schritten, vor dem ersten Gate-Lauf), *Bericht* (`.sdd/adopt-report.md`).

**Mit `--apply`** gelten die Grenzen aus [ADR 0001](../docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md): nur in einem Git-Repository mit sauberem Arbeitsbaum, nicht bei losgelöstem `HEAD`, nur auf einem **neuen** Branch (existiert er, bricht der Lauf ab). Kein Push, kein Merge; der Ausgangsbranch bleibt unverändert. Gelöscht oder ersetzt wird nichts: Frontmatter wird nur vorangestellt, und nur, wenn die Datei noch genau dem gelesenen Stand entspricht. Hat ein Feature-Artefakt Frontmatter ohne `feature:`, bleibt es unverändert und erscheint unter *Konflikte*.

**Ausgewiesen, nicht ausgeführt:** Die Archivierung (`SDD-050`, nicht baselinefähig) listet der Bericht als Aufräumschritt vor dem ersten Release. Der vorangestellte Status `active` ist eine Annahme; längst umgesetzte Features sind auf `implemented` zu setzen.

### `sdd report` {#sdd-report}

```text
sdd report [--format markdown|json] [--since <ref>]
```

Zeigt den Stand des Projekts: Reifegrad L0–L4 mit nächstem Schritt (dieselben Kriterien wie [`sdd adopt`](#sdd-adopt)), die offenen Befunde von `sdd lint`, für jedes Gate die Anzahl der `error`-, `warn`- und baselineten Befunde sowie den **Schuldenstand**: die Einträge in `.sdd/baseline.json` je Regel und Gate, verglichen mit der Baseline am Vergleichsstand.

- **Vergleichsstand:** `--since <ref>`, sonst die Merge-Basis nach `base_ref` (wie `pre-merge`). Die frühere Baseline wird per `git show <ref>:.sdd/baseline.json` gelesen — Git bleibt lesend. Fehlt sie dort, gilt sie als leer.
- **Ratchet:** Die Baseline darf nur schrumpfen. Enthält sie einen Eintrag (Identität `gate + rule_id + path + content_hash + ordinal`), den der Vergleichsstand nicht enthielt, hebt der Bericht das als Verstoß hervor und listet die neuen Einträge. Es ist ein Hinweis für das Review, kein Regelbefund.
- **Robust:** Ist ein Gate nicht ausführbar (etwa ohne Git) oder der Vergleichsstand nicht lesbar, steht der Grund im Bericht. Exit-Code `2` gibt es nur bei fehlender oder ungültiger Konfiguration, ungültigem Format oder nicht lesbarer aktueller Baseline.

### `sdd next` {#sdd-next}

```text
sdd next [--feature <slug>] [--json]
```

Nennt den nächsten Prozessschritt — aus den Dateien und denselben Regeln abgeleitet wie `sdd lint`, ohne etwas zu schreiben. Mit `--feature` alle Schritte des Features, ohne nur den nächsten Schritt je aktivem Feature; gibt es noch keines, den Einstieg (`sdd new --feature <slug> --type prd`).

| Schritt | erledigt, wenn |
|---|---|
| PRD | aktives PRD mit `feature: <slug>` |
| Nicht-Ziele | kein `SDD-023` am PRD |
| Akzeptanzkriterien | mindestens eine Kennung, kein `SDD-021` am PRD |
| Änderungsabsicht | `touches` in einem Artefakt des Features |
| Tech Spec | vorhanden — oder *nicht nötig*, weil die [Größenschwelle](#groessenschwelle) nicht überschritten ist |
| Tech Spec nennt alle Kriterien | kein `SDD-031` |
| ADR | *nicht nötig*, wenn `touches` die Verfassung nicht berührt; sonst erledigt, sobald ein Artefakt des Features auf ein ADR verlinkt, das die Verfassung in `references` nennt und `accepted` ist — bis dahin *manuell* (anlegen, verlinken, entscheiden; `SDD-010` prüft im Gate) |
| Plan | *nicht nötig* ohne Plan; sonst kein `SDD-040`/`SDD-041` |
| Tests | kein `SDD-022` für die Kennungen des PRDs |
| Abschluss | nach dem Merge `status: implemented`; dann vor dem Release archivieren (`SDD-050`) |

Der nächste Schritt ist der erste *offene*, sonst der erste *manuelle*, sonst der Abschluss. Zustände in der Textausgabe: `✓` erledigt, `✗` offen, `?` manuell, `–` nicht nötig, `·` später.

### `sdd explain` und `sdd mcp` {#sdd-mcp}

```text
sdd explain <regel-id> [--json]
sdd mcp [--allow-write]
```

`sdd explain` zeigt den Handbuchabschnitt und den Prüfvertrag einer Regel. `sdd mcp` stellt die Bibliothek Agenten als MCP-Server über stdio bereit — Werkzeuge, Ressourcen und geführte Prompts, KI-frei und ohne Netzwerk ([ADR 0002](../docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)). Einrichtung, Werkzeuge und Grenzen beschreibt [Teil 4 — Für Agenten](04-agenten.md).

### `sdd ui` {#sdd-ui}

```text
sdd ui [--port N] [--open]
```

Startet eine lokale Oberfläche auf `127.0.0.1` (Standard: freier Port) und gibt die URL mit einem Sitzungstoken aus; `--open` öffnet sie im Standardbrowser, `Strg+C` beendet sie. Die Oberfläche ist eine Ansicht auf dieselbe Bibliothek — jede Funktion gibt es auch in der CLI:

| Ansicht | entspricht |
|---|---|
| Übersicht | `sdd report` |
| Artefakte (Typ, Status, Alter, Archivfrist) | Inventar von `sdd adopt` |
| Befunde (Gate, „neu“/„Altbestand“, Export JSON/CSV, Handbuch-Link, Fundstelle in VS Code oder `$EDITOR`) | `sdd lint --json`, `sdd gate <gate> --json` |
| Neues Artefakt (Vorschau, dann Anlegen) | `sdd new` |
| Übernahme (Probelauf, dann Anwenden auf neuem Branch) | `sdd adopt`, `sdd adopt --apply` |

Die Befundliste aktualisiert sich alle drei Sekunden, solange der Tab sichtbar ist. `Strg+K` öffnet die Befehlspalette, `1`–`5` und `H` springen direkt.

**Sicherheit** (Verfassung, Security 2):

- Bindung ausschließlich an `127.0.0.1`. Jede Anfrage mit einem anderen `Host` — auch `localhost` — wird mit `403` abgewiesen (Schutz vor DNS-Rebinding).
- Kein CORS; `Content-Security-Policy: default-src 'self'`; keine externen Schriften, Skripte oder Telemetrie.
- Schreibende Anfragen (`POST`) brauchen das Sitzungstoken (`X-SDD-Token`), `Origin: http://127.0.0.1:<port>` und `Content-Type: application/json`. Lesen geht ohne Token.
- Geschrieben wird nur über den Plan/Apply-Pfad von `sdd new` und `sdd adopt --apply`, jeweils erst nach Vorschau und Bestätigung. Baseline und ADR-Lockfile lassen sich in der Oberfläche nicht verändern.
- `$EDITOR` wird ohne Shell gestartet und nur für Dateien innerhalb des Repositorys.

### CI-Einbindung (GitHub Action) {#ci-einbindung}

Die Action im Wurzelverzeichnis von `sdd_studio` baut `sdd` aus ihrem eigenen Quellstand und führt `sdd gate <gate>` im Arbeitsverzeichnis aus; der Exit-Code des Gates ist der Exit-Code des Schritts.

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0          # git diff <base>...HEAD braucht die Historie
- uses: Zheng-Bote/sdd_studio@v0.4.0   # Release-Tag; @main nur zum Ausprobieren
  with:
    gate: pre-merge         # pre-implementation | pre-merge | pre-release
    feature: ""             # optional, kommagetrennte Slugs
    paths: ""               # optional, erklärte Absicht für pre-implementation
```

Ohne `fetch-depth: 0` fehlt die Merge-Basis, und das Gate endet mit Exit-Code `2`. Nach dem Gate hängt die Action den [Bericht](#sdd-report) an die Job-Zusammenfassung an — auch wenn das Gate fehlschlägt, und ohne dessen Exit-Code zu verändern. Der von `sdd init` erzeugte Workflow führt `pre-merge` für Pull Requests aus und `pre-release` für Pushes auf `release/**` sowie manuell — also vor dem Tag, nicht danach.

---

## Verfassungsregeln (`SDD-001`, `SDD-002a`)

Die Verfassung ist das ranghöchste Artefakt eines Projekts. Diese Regeln stellen sicher, dass sie existiert und ihr Strukturschema erfüllt.

---

### SDD-001 — Verfassung vorhanden {#sdd-001}

- **Schweregrad:** `error` (`default`, `pre-implementation`, `pre-merge`)
- **Profil:** ab `minimal`
- **Baseline:** `allowed`
- **Check:** `constitution_present`

**Begründung:** Ohne Verfassung fehlt die oberste Quelle der Präzedenzordnung. Architektur-, Sicherheits- und Prozessregeln wären nur implizit vorhanden und damit weder prüfbar noch für Agenten auffindbar.

#### SDD-001 Prüfvertrag {#sdd-001-vertrag}

Für den in `sdd.config.yaml` unter `artifacts.constitution` konfigurierten Pfad muss gelten:
1. Der Pfad ist gesetzt (nicht leer).
2. Der Pfad ist relativ und verweist nach Normalisierung nicht außerhalb des Repository-Wurzelverzeichnisses (kein absoluter Pfad, kein `..`-Ausbruch).
3. Unter dem Pfad existiert eine reguläre Datei (kein Verzeichnis).
4. Die Datei enthält mindestens ein Zeichen, das kein Leerraum ist.

Der Inhalt wird nicht geprüft — das ist Aufgabe von `SDD-002`.

---

### SDD-002a — Markdown-Verfassung enthält die Pflichtabschnitte {#sdd-002a}

- **Schweregrad:** `error` (`default`, `pre-implementation`, `pre-merge`)
- **Profil:** ab `minimal`
- **Baseline:** `allowed`
- **Check:** `constitution_markdown_sections`

**Begründung:** Architektur/Layering, Security und Strikte Regeln sind die Abschnitte, auf die Feature-Artefakte, Reviews und Agenten verweisen. Fehlt einer, ist die Verfassung an genau dieser Stelle stumm. Welcher Modus gilt, entscheidet die Dateiendung: `.sdd` ⇒ `SDD-002b` (formale Grammatik), jede andere Endung ⇒ `SDD-002a`.

#### SDD-002a Prüfvertrag {#sdd-002a-vertrag}

Für die unter `artifacts.constitution` konfigurierte Datei gilt:
1. Endet der Pfad (ohne Beachtung der Groß-/Kleinschreibung) auf `.sdd`, ist die Regel nicht anwendbar.
2. Fehlt die Datei oder ist sie nicht lesbar, meldet die Regel nichts; der Befund gehört zu `SDD-001`.
3. Als Überschrift zählt jede ATX-Überschrift (`#` bis `######`) außerhalb von Codeblöcken (```` ``` ```` oder `~~~`). Eine explizite Anker-ID `{#...}` am Ende sowie schließende `#`-Zeichen werden entfernt.
4. Überschrift und Pflichtabschnitt werden normalisiert verglichen: Kleinschreibung, sämtlicher Leerraum entfernt. Die Ebene der Überschrift ist unerheblich.
5. Pflichtabschnitte (für alle Profile gleich): `Architektur/Layering`, `Security`, `Strikte Regeln`. Für jeden fehlenden Abschnitt entsteht genau ein Befund.

---

### SDD-022 — Jedes Akzeptanzkriterium ist in mindestens einer Testdatei referenziert {#sdd-022}

- **Schweregrad:** `error` (`default`, `pre-merge`); `warn`, wenn keine Testdatei gefunden wird
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `acceptance_criteria_tested`

**Begründung:** Ein Akzeptanzkriterium, auf das kein Test verweist, ist eine Behauptung. Die Regel misst **Präsenz, nicht Coverage**: Ein `// TODO: AC-3 testen` erfüllt sie. Das ist der bewusste Preis für Sprachneutralität ohne Parser; die schärfere Aussage „steht in einer Testdeklaration" trifft später `SDD-025` als `warn`.

#### SDD-022 Prüfvertrag {#sdd-022-vertrag}

1. **Definierte Kennungen:** alle Vorkommen von `\bAC-[1-9][0-9]*\b` in den aktiven PRDs (Auswahl, Frontmatter und Codeblöcke wie bei [`SDD-021`](#sdd-021-vertrag)). Durchgestrichene Kennungen (`~~… AC-<n> …~~`) zählen nicht. Maßgeblich ist das erste Vorkommen je Datei und Kennung.
2. **Testdateien:** alle regulären Dateien des Repositorys, die einen Glob aus `tests.include` treffen und keinen aus `exclude`; das Verzeichnis `.git` wird nie gelesen. Ist `tests.include` leer, gilt die Fallback-Liste:

   ```text
   **/*_test.go
   **/test_*.py
   **/*_test.py
   **/*.{test,spec}.{ts,tsx,js,jsx}
   **/*Test.java
   **/*_test.rs
   **/tests/**
   **/__tests__/**
   ```

3. **Referenz:** Eine Kennung `AC-<n>` gilt als referenziert, wenn in mindestens einer Testdatei eines dieser Muster mit genau der Nummer `<n>` vorkommt (die Ziffernfolge wird vollständig gelesen, `AC-12` referenziert also nicht `AC-1`), gleich ob im Testnamen, in einem String oder in einem Kommentar:
   - `\bAC-<n>` — z. B. `// AC-3`, `it("AC-3 …")`;
   - `AC` in Großbuchstaben, optional `_`, dann `<n>`, wobei davor der Dateianfang oder ein Zeichen außer Großbuchstabe und Ziffer steht — z. B. `TestAC3_Login`, `TestAC_3`;
   - `ac` in beliebiger Schreibweise, optional `_`, dann `<n>`, wobei davor der Dateianfang oder ein Zeichen außer Buchstabe und Ziffer steht — z. B. `test_ac_3`, `ac3`.

   Ein `\b`-basiertes Muster genügt nicht: `\b` trennt weder CamelCase- noch snake_case-Teile und träfe `TestAC3_Login` nicht.
4. **Slug-Pflicht:** Kennt das Repository mehr als ein Feature — Feature-Slugs aller PRDs unter `artifacts.prd` und `artifacts.archive`, archivierte eingeschlossen —, zählt für ein PRD nur eine **Zeile** einer Testdatei, die die Referenz enthält **und** den Feature-Slug dieses PRDs als ganzes Token nennt (`feature:` aus dem Frontmatter, ohne Feld der Dateiname ohne Endung). Als Token gilt der Slug, wenn davor und danach kein Zeichen aus `[a-z0-9-]` steht — `billing` in `billing-v2` zählt also nicht. Nur ein Repository mit genau einem Feature kommt ohne Slug aus. Konvention: `// <slug> AC-<n>: …` im Kommentar über dem Test. Maßgeblich ist die Zeile, weil sich Tests mehrerer Features eine Datei teilen können, und archivierte Features zählen mit, weil ihre Tests weiter bestehen und sonst die gleichen Nummern neuer Features erfüllen würden.
5. Für jede definierte, nicht referenzierte Kennung entsteht ein `error` an ihrem ersten Vorkommen im PRD.
6. Trifft die Glob-Liste keine einzige Datei, entsteht statt Einzelbefunden genau ein `warn`: Ein fehlender Testpfad ist ein Konfigurationsproblem, kein Projektverstoß.

---

## ADR-Regeln (`SDD-010`, `SDD-011`)

Die Verfassung hält den *aktuell gültigen Zustand*, das ADR die *unveränderliche Begründung*. Als **ADR** gilt jede Datei `NNNN-*.md` (vier Ziffern) unter `artifacts.adr`.

---

### SDD-010 — Änderung einer Verfassungsregel verweist auf ein ADR {#sdd-010}

- **Schweregrad:** `error` in `pre-implementation` und `pre-merge`; kein `default`, weil die Regel einen Diff braucht
- **Profil:** ab `minimal`
- **Baseline:** `allowed`
- **Check:** `constitution_change_has_adr`

**Begründung:** Eine Regeländerung ohne ADR verliert ihr Warum. Wer den Zustand ändert, ändert die Verfassung und verweist im selben Change auf das ADR, das ihn begründet.

#### SDD-010 Prüfvertrag {#sdd-010-vertrag}

1. Die Regel greift nur, wenn die unter `artifacts.constitution` konfigurierte Datei in `git diff --no-renames <base>...HEAD` vorkommt ([Merge-Basis](#feature-aufloesung)).
2. **Keine Änderung im Sinne der Regel:** Die Datei fehlt am Vergleichsstand (erstmaliges Anlegen) oder ist im Arbeitsbaum gelöscht (dann gilt `SDD-001`), oder beide Fassungen sind nach Normalisierung gleich — Zeilenenden vereinheitlicht, Zeilen nur aus Leerraum entfernt, Leerraumfolgen zu einem Leerzeichen.
3. Sonst muss mindestens ein ADR, das ebenfalls im Diff vorkommt, die Verfassung in seinem Frontmatter nennen: in der Liste `references` oder im Feld `adr` (Text oder Liste), als repository-relativer Pfad oder relativ zum ADR.
4. Fehlt ein solches ADR, entsteht ein `error` an der Verfassung.

---

### SDD-011 — ADRs sind unveränderlich; Ablösung über supersedes {#sdd-011}

- **Schweregrad:** `error` (`default`, `pre-merge`); `warn` für ein entschiedenes, noch nicht gelocktes ADR
- **Profil:** ab `standard`
- **Baseline:** `forbidden` — Abweichungen werden über das Lockfile quittiert (`sdd baseline --accept SDD-011`), nicht über Baseline-Einträge.
- **Check:** `adr_immutable`

**Begründung:** Ein nachträglich umgeschriebenes ADR fälscht die Begründung der Verfassung. Eine Entscheidung wird abgelöst, indem ein neues ADR sie mit `supersedes` ersetzt; das alte bleibt, wie es war.

#### SDD-011 Prüfvertrag {#sdd-011-vertrag}

1. **Lockfile** `.sdd/adr-lock.json`: `{"version": 1, "entries": {"<pfad>": "<sha256>"}}`. Es wird ausschließlich von `sdd baseline` und `sdd adopt` geschrieben, nie von `lint` oder `gate`.
2. **Hash:** SHA-256 über den Text **ohne Frontmatter**, normalisiert wie bei [`SDD-010`](#sdd-010-vertrag). Das Frontmatter bleibt änderbar, weil Statuswechsel (`proposed` → `accepted` → `superseded`) und neue `references` zum Lebenszyklus gehören.
3. **Entschieden** ist ein ADR, dessen `status` nicht `proposed` ist; vorgeschlagene ADRs sind Entwürfe und werden weder gelockt noch geprüft.
4. Für jedes entschiedene ADR gilt: Ist es nicht gelockt, entsteht ein `warn`; weicht sein Hash vom gelockten ab, ein `error`.
5. `sdd baseline` lockt neue entschiedene ADRs, verändert aber nie einen vorhandenen Eintrag. Nur `sdd baseline --accept SDD-011` übernimmt geänderte Hashes — sichtbar als Änderung des Lockfiles im Review. Einträge gelöschter ADRs bleiben erhalten (Historie).

---

### SDD-025 — Jede AC-Kennung steht in einer Testdeklaration oder ihrem Rumpf {#sdd-025}

- **Schweregrad:** `warn` (`default`, `pre-merge`) — die Deklarationsmuster sind sprachadaptiv, also kein `error`
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `acceptance_criteria_in_test_declarations`

**Begründung:** [`SDD-022`](#sdd-022) misst Präsenz irgendwo in einer Testdatei; auch ein `// TODO: AC-3` im Dateikopf genügt dort. `SDD-025` nähert sich der Aussage „ein Test verweist auf das Kriterium“.

#### SDD-025 Prüfvertrag {#sdd-025-vertrag}

1. Geprüft werden die Kennungen, die `SDD-022` als referenziert anerkennt — mit denselben Mustern und derselben Slug-Regel für mehrdeutige Nummern. Nicht referenzierte Kennungen meldet nur `SDD-022`.
2. **Testdeklaration** ist eine Zeile, die einem dieser Muster entspricht: `func Test…(`/`Benchmark…`/`Fuzz…`/`Example…` (Go), `def test…(` (Python), `it(`, `test(`, `describe(` auch mit `.only`/`.skip` (JavaScript/TypeScript), `TEST(`, `TEST_F(`, `TEST_P(` (C++), `#[test]` (Rust), `@Test` (Java/Kotlin).
3. **Bereich** einer Deklaration: vom ersten Zeile des Kommentar- oder Attributblocks unmittelbar darüber (Zeilen, die mit `//`, `#`, `/*`, `*`, `--`, `@`, `"""` oder `'''` beginnen, ohne Leerzeile dazwischen) bis vor die nächste Deklaration gleicher oder geringerer Einrückung — samt deren Block — oder bis zum Dateiende. Ein freistehender Kommentar, von der Deklaration durch eine Leerzeile getrennt, gehört nicht dazu.
4. Kommt eine anerkannte Referenz einer Kennung in keinem Bereich vor, entsteht ein `warn` an ihrem ersten Vorkommen im PRD.

*Präzisierung gegenüber dem Konzept:* Dort zählen Kommentare vor der Deklaration nicht. Der direkt anschließende Kommentarblock gehört hier zur Deklaration, wie ein Go-Doc-Kommentar oder ein Python-Decorator; das ist die Stelle der Konvention `// <slug> AC-<n>: …`.

---

## Feature-Artefakte bei Schwellenüberschreitung (`SDD-020`, `SDD-030`)

Ab einer projektspezifischen Größenschwelle verlangt `sdd` für ein Feature ein PRD (`SDD-020`) und eine Tech Spec (`SDD-030`). Beide Regeln teilen die Feature-Auflösung, die Glob-Syntax und die Schwellenlogik; sie unterscheiden sich nur im geprüften Verzeichnis (`artifacts.prd` bzw. `artifacts.design`).

### Feature-Auflösung {#feature-aufloesung}

Welche Features ein Gate-Lauf prüft, wird in dieser Reihenfolge bestimmt; die erste Stufe mit Ergebnis gewinnt:

1. **`--feature <slug>`** — kommagetrennt mehrere Slugs möglich. Ein Slug besteht ausschließlich aus `[a-z0-9-]`; ein ungültiger Slug bricht mit Exit-Code `2` ab.
2. **Frontmatter im Diff** — alle Markdown-Dateien, die `artifacts.prd`, `artifacts.design` oder `artifacts.plans` auswählen ([Ordner oder Glob](#artefaktpfade)), die in `git diff <base>...HEAD` vorkommen und ein gültiges `feature:`-Feld tragen.
3. **Branch-Name** — `feature/<slug>` oder `feat/<slug>` bzw. `feat/<nummer>-<slug>`. In CI mit losgelöstem `HEAD` gelten `$GITHUB_HEAD_REF`, `$CI_MERGE_REQUEST_SOURCE_BRANCH_NAME` und `$CI_COMMIT_BRANCH`.

Liefert keine Stufe ein Feature, gilt der Lauf als **nicht aufgelöst** (siehe Prüfverträge). Die Merge-Basis ist `base_ref` aus `sdd.config.yaml`; leer bedeutet: `$GITHUB_BASE_REF` bzw. `$CI_MERGE_REQUEST_TARGET_BRANCH_NAME` (bevorzugt als `origin/<name>`), sonst `@{upstream}` — außer es ist die entfernte Kopie des eigenen Feature-Branches (`origin/feature/x` für `feature/x`), denn nach dem Push wäre der Diff leer —, sonst `main`, sonst `master`. Ist Git nicht verfügbar oder keine Basis ermittelbar, endet der Lauf mit Exit-Code `2`.

### Artefaktpfade: Ordner oder Datei-Glob {#artefaktpfade}

`artifacts.prd`, `artifacts.design`, `artifacts.plans` und `artifacts.archive` sind entweder ein **Ordner** (alle `*.md` darunter, rekursiv) oder ein **Datei-Glob** (sobald der Wert einen Platzhalter enthält). Damit lassen sich bestehende Strukturen ohne Umzug einbinden, etwa GitHub Spec Kit:

```yaml
artifacts:
  prd: specs/features/*/spec.md
  design: specs/features/*/plan.md
  plans: specs/features/*/tasks.md
  archive: specs/archive
```

Liegen PRD und Tech Spec im selben Ordner, unterscheidet sie allein der Glob. `sdd new` leitet den Zielpfad aus einem Glob ab, wenn er genau ein `*` als ganzes Pfadsegment enthält (`specs/features/*/spec.md` → `specs/features/<slug>/spec.md`); bei anderen Globs bricht es mit Exit-Code `2` ab.

### Glob-Syntax {#glob-syntax}

Alle Pfadangaben (`thresholds`, `touches`, `--paths`, `exclude`, `tests.include`) sind Globs relativ zum Repository-Wurzelverzeichnis und **an der Wurzel verankert**:

- `**` steht für beliebig viele Pfadsegmente (auch keines); `dir/**` trifft auch `dir` selbst.
- `*` steht für beliebige Zeichen innerhalb eines Segments, `?` für genau ein Zeichen, `{a,b}` für Alternativen, `[...]` für eine Zeichenklasse (`[!...]` negiert).
- Ein Glob ohne `/` trifft nur Dateien im Wurzelverzeichnis; für beliebige Tiefe ist `**/` nötig.

**Überschneidung zweier Globs** (erklärte Absicht gegen Schwellen-Globs): Zwei Literale überschneiden sich, wenn sie gleich sind; ein Literal und ein Glob, wenn der Glob das Literal trifft; zwei Globs, wenn der statische Präfix (alles vor dem ersten Segment mit Platzhalter) des einen ein Präfix des anderen ist. Diese Definition ist bewusst konservativ: Im Zweifel gilt die Schwelle als berührt.

### Größenschwelle {#groessenschwelle}

Die Schwelle ist überschritten, wenn **mindestens einer** der Pfade einen Glob aus `thresholds.feature_artifacts_required_when.touched_paths` überschneidet **oder** mindestens `min_layers_touched` verschiedene Globs aus `thresholds.layers` überschnitten werden (`min_layers_touched ≤ 0` schaltet die Layer-Bedingung ab). Grundlage ist nie der gesamte Arbeitsbaum:

- **`pre-implementation`:** die erklärte Absicht — `touches` aus dem Frontmatter aller Feature-Artefakte mit `feature: <slug>` unter `artifacts.prd`, `artifacts.design` und `artifacts.plans`, vereinigt mit `--paths`.
- **`pre-merge`:** die Dateien aus `git diff --no-renames <base>...HEAD` (Umbenennungen zählen mit altem und neuem Pfad), ohne `exclude`-Treffer.

---

### SDD-020 — PRD vorhanden, wenn die Größenschwelle greift {#sdd-020}

- **Schweregrad:** `error` in `pre-implementation` und `pre-merge`; `warn`, wenn die Schwelle nicht bewertbar ist. Kein `default` — `sdd lint` ohne Gate prüft die Regel nicht, weil sie ohne Gate-Kontext keine Grundlage hat.
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `feature_prd_present`

**Begründung:** Ab der Größenschwelle ist eine Änderung nicht mehr allein aus Issue und Code nachvollziehbar. Das PRD hält fest, was und warum gebaut wird — und zwar bevor implementiert wird.

#### SDD-020 Prüfvertrag {#sdd-020-vertrag}

Die Regel wird nur in `pre-implementation` und `pre-merge` ausgewertet. Mit der [Feature-Auflösung](#feature-aufloesung), der [Glob-Syntax](#glob-syntax) und der [Größenschwelle](#groessenschwelle) gilt:

1. **Feature aufgelöst:** Für jeden Slug wird die Schwelle bewertet. Ist sie überschritten und existiert unter `artifacts.prd` keine Markdown-Datei (`*.md`, rekursiv, ohne `exclude`-Treffer), deren gültiges YAML-Frontmatter `feature: <slug>` enthält, entsteht ein `error`. Dateien ohne oder mit ungültigem Frontmatter zählen nicht. Ist `artifacts.prd` nicht konfiguriert, entsteht ebenfalls ein `error`.
2. **Feature aufgelöst, Absicht nicht erklärt** (nur `pre-implementation`: weder `touches` noch `--paths`): ein `warn` „Änderungsabsicht nicht erklärt; Schwelle nicht bewertbar".
3. **Kein Feature aufgelöst:** In `pre-implementation` ohne `--paths` entsteht ein `warn` (wie 2.). Mit `--paths` bzw. in `pre-merge` mit dem Diff wird die Schwelle bewertet; ist sie überschritten, entsteht ein `error` mit dem Hinweis `sdd new --feature <slug>` (fail-closed).
4. Ist die Schwelle nicht überschritten, entsteht kein Befund.

---

### SDD-030 — Tech Spec vorhanden, wenn die Größenschwelle greift {#sdd-030}

- **Schweregrad:** wie `SDD-020`
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `feature_design_present`

**Begründung:** Eine Änderung über mehrere Layer oder an öffentlichen Schnittstellen braucht ein abgestimmtes technisches Design, bevor implementiert wird. Ein PRD allein beantwortet das Was, nicht das Wie.

#### SDD-030 Prüfvertrag {#sdd-030-vertrag}

Wie der [Prüfvertrag von SDD-020](#sdd-020-vertrag), mit `artifacts.design` an Stelle von `artifacts.prd` und „Tech Spec" an Stelle von „PRD".

---

### SDD-024 — Der Diff weicht nicht von der erklärten Änderungsabsicht ab {#sdd-024}

- **Schweregrad:** `warn`, nur in `pre-merge` (die Regel braucht den Diff)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `diff_within_intent`

**Begründung:** `pre-implementation` prüft die erklärte Absicht, `pre-merge` die Wirklichkeit. Wächst der Umfang während der Umsetzung über `touches` hinaus, beschreibt die Spezifikation nicht mehr, was gemergt wird — meist ein Zeichen, dass sie nachzuführen ist.

#### SDD-024 Prüfvertrag {#sdd-024-vertrag}

1. Die Features werden wie bei `SDD-020` [aufgelöst](#feature-aufloesung). Ist keines auflösbar, prüft die Regel nichts (das meldet `SDD-020`/`SDD-030`).
2. **Erklärte Absicht** ist die Vereinigung von `touches` aller Feature-Artefakte der aufgelösten Features und `--paths`. Ist sie leer, prüft die Regel nichts.
3. **Geprüft** werden die Dateien aus `git diff --no-renames <base>...HEAD` ohne `exclude`-Treffer, außer Verfassung, Feature-Artefakten (`artifacts.prd`, `design`, `plans`, `archive`), Dateien unter `artifacts.adr`, `artifacts.docs` und `.sdd/`.
4. Eine geprüfte Datei liegt in der Absicht, wenn sie sich mit mindestens einem Glob der Absicht [überschneidet](#glob-syntax). Für jede andere entsteht ein `warn` an der Datei.

---

## Akzeptanzkriterien (`SDD-021`, `SDD-022`)

Akzeptanzkriterien sind das Bindeglied zwischen PRD, Tests und Tasks. Sie tragen deshalb eine stabile Kennung `AC-<n>`, die nie umnummeriert wird.

**Tombstone-Prinzip:** Entfällt ein Kriterium, bleibt seine Nummer vergeben. Die Kennung wird durchgestrichen (`~~AC-2~~`); solche Zeilen gelten als entfallen und werden von den AC-Regeln ignoriert.

---

### SDD-021 — Jedes Akzeptanzkriterium hat eine stabile Kennung {#sdd-021}

- **Schweregrad:** `error` (`default`, `pre-merge`)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `acceptance_criteria_ids`

**Begründung:** Ohne stabile Kennung lässt sich ein Akzeptanzkriterium weder in Tests (`SDD-022`) noch in Tasks referenzieren. Die Rückverfolgbarkeit vom Test zur Anforderung wäre dann behauptet, nicht prüfbar.

#### SDD-021 Prüfvertrag {#sdd-021-vertrag}

Geprüft werden alle Markdown-Dateien unter `artifacts.prd` (rekursiv, ohne `exclude`-Treffer), deren Frontmatter-`status` weder `archived` noch `superseded` ist. Frontmatter und Codeblöcke (```` ``` ````, `~~~`) werden übersprungen. Eine **Listenzeile** beginnt (nach optionalem Einzug) mit `-`, `*`, `+` oder `<zahl>.`/`<zahl>)`, gefolgt von Leerraum und Inhalt. Als Akzeptanzkriterium gilt:

1. jede Checkbox-Zeile (`- [ ]`, `- [x]`, `* [ ]`, `1. [ ]` …), unabhängig vom Abschnitt;
2. jede Listenzeile, die die Zeichenfolge `AC-` enthält;
3. jede Listenzeile im Abschnitt unter einer Überschrift, die `Akzeptanzkriterien` oder `Acceptance Criteria` enthält (Groß-/Kleinschreibung egal). Der Abschnitt reicht bis zur nächsten Überschrift gleicher oder höherer Ebene; Unterabschnitte gehören dazu.

Fortsetzungszeilen und Fließtext sind keine Kriterien. Jede erkannte Zeile muss eine Kennung enthalten, die `\bAC-[1-9][0-9]*\b` erfüllt (positive Ganzzahl ohne führende Null), oder eine durchgestrichene Kennung `~~… AC-<n> …~~` (Tombstone). Für jede andere erkannte Zeile entsteht genau ein Befund mit Datei und Zeilennummer.

---

## Zusammenhang der Feature-Artefakte (`SDD-023`, `SDD-031`, `SDD-041`, `SDD-051`)

Diese Regeln prüfen, ob PRD, Tech Spec und Plan eines Features zueinander passen. Als **aktives Feature-Artefakt** gilt jede Markdown-Datei, die `artifacts.prd`, `artifacts.design` bzw. `artifacts.plans` auswählt ([Ordner oder Glob](#artefaktpfade)), nicht unter `artifacts.archive` liegt und deren `status` weder `archived` noch `superseded` ist. Frontmatter und Codeblöcke werden wie bei `SDD-021` übersprungen.

---

### SDD-023 — PRD nennt Nicht-Ziele {#sdd-023}

- **Schweregrad:** `warn` (`default`, `pre-merge`)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `prd_non_goals`

**Begründung:** Ein PRD ohne Nicht-Ziele lädt dazu ein, den Umfang während der Umsetzung still zu erweitern. Was ausgeschlossen ist, muss so sichtbar sein wie das Ziel.

#### SDD-023 Prüfvertrag {#sdd-023-vertrag}

Für jedes aktive PRD gilt:

1. Es gibt eine Überschrift, deren Text nach Entfernen von `{#…}`-Anker und schließenden `#` und in Kleinschreibung genau `nicht-ziele`, `nichtziele`, `non-goals`, `nongoals` oder `out of scope` lautet. Maßgeblich ist die erste solche Überschrift.
2. Zwischen ihr und der nächsten Überschrift beliebiger Ebene steht mindestens eine Zeile, die nicht nur aus Leerraum besteht. Ein Unterabschnitt allein füllt den Abschnitt nicht.
3. Fehlt die Überschrift oder ist der Abschnitt leer, entsteht genau ein `warn` je Datei.

---

### SDD-031 — Tech Spec referenziert die AC-Kennungen, die sie umsetzt {#sdd-031}

- **Schweregrad:** `warn` (`default`, `pre-merge`)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `design_references_acs`

**Begründung:** Ohne Bezug auf die Akzeptanzkriterien bleibt offen, welches Kriterium durch welchen Teil des technischen Designs erfüllt wird. Eine Tabelle „AC → Umsetzung → Test“ genügt.

#### SDD-031 Prüfvertrag {#sdd-031-vertrag}

1. **Kennungen eines Features:** die definierten AC-Kennungen der aktiven PRDs mit ihrem Feature (`feature:` aus dem Frontmatter, ohne Feld der Dateiname ohne Endung) — dieselbe Menge wie bei [`SDD-022`](#sdd-022-vertrag), Tombstones ausgenommen.
2. **Tech Specs eines Features:** aktive Feature-Artefakte unter `artifacts.design` mit `feature: <slug>`.
3. Hat ein Feature mindestens eine Tech Spec, muss jede seiner Kennungen in mindestens einer dieser Tech Specs als `\bAC-<n>\b` vorkommen (außerhalb von Codeblöcken, nicht durchgestrichen). Für jede fehlende Kennung entsteht ein `warn` an der ersten Tech Spec des Features.
4. Features ohne Tech Spec prüft diese Regel nicht; ob eine nötig ist, entscheidet [`SDD-030`](#sdd-030).

---

### SDD-040 — Plan existiert vor der ersten Task {#sdd-040}

- **Schweregrad:** `warn` (`default`, `pre-merge`)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `plan_before_tasks`

**Begründung:** Tasks ohne technischen Plan zerlegen ein Vorgehen, das niemand abgestimmt hat; Schnittstellen und Datenmodell entstehen dann nebenbei. Der „Plan“ ist hier das technische Vorgehen, also die Tech Spec — in Spec Kit die `plan.md` neben der `tasks.md`.

#### SDD-040 Prüfvertrag {#sdd-040-vertrag}

1. Das **Feature** eines Artefakts ist sein `feature:`-Feld, ohne Feld das `*`-Segment des Globs bzw. der Dateiname ohne Endung.
2. Für jedes aktive Feature-Artefakt unter `artifacts.plans`, das mindestens eine Task enthält (Erkennung wie bei [`SDD-041`](#sdd-041-vertrag)), muss eine aktive Tech Spec unter `artifacts.design` mit demselben Feature existieren.
3. Fehlt sie, entsteht genau ein `warn` je Plan, an seiner ersten Task. Geprüft wird der aktuelle Stand, nicht die Reihenfolge in der Git-Historie.

---

### SDD-041 — Jede Task referenziert mindestens ein Akzeptanzkriterium {#sdd-041}

- **Schweregrad:** `error` (`default`, `pre-merge`)
- **Profil:** ab `standard`
- **Baseline:** `allowed`
- **Check:** `tasks_reference_acs`

**Begründung:** Eine Task ohne Akzeptanzkriterium ist Arbeit ohne prüfbares Ziel: Sie erweitert den Umfang, ohne dass das PRD es zeigt.

#### SDD-041 Prüfvertrag {#sdd-041-vertrag}

Geprüft werden die aktiven Feature-Artefakte unter `artifacts.plans` (bei Spec Kit z. B. `specs/features/*/tasks.md`).

1. **Task** ist jede Checkbox-Zeile: nach optionalem Einzug `-`, `*`, `+` oder `<zahl>.`/`<zahl>)`, Leerraum, dann `[ ]`, `[x]` oder `[X]`.
2. **Fortsetzungszeilen** einer Task sind die folgenden Zeilen, die tiefer eingerückt sind als die Task (Tab = vier Leerzeichen). Leerzeilen werden übersprungen. Der Bereich endet an der ersten nicht-leeren Zeile, die gleich tief oder flacher eingerückt ist, an einer Überschrift oder an einer weiteren Checkbox-Zeile — eine eingerückte Unteraufgabe ist eine eigene Task.
3. Task-Zeile und Fortsetzungszeilen zusammen müssen eine Kennung `\bAC-[1-9][0-9]*\b` enthalten, die nicht durchgestrichen ist (`~~…~~` zählt nicht).
4. Für jede andere Task entsteht ein `error` an ihrer Zeile.

---

### SDD-051 — Aktive Feature-Artefakte verweisen nicht auf archivierte Feature-Artefakte {#sdd-051}

- **Schweregrad:** `warn` (`default`, `pre-merge`, `pre-release`)
- **Profil:** ab `standard`
- **Baseline:** `forbidden` — ein Verweis lässt sich jederzeit billig entfernen oder ersetzen.
- **Check:** `no_links_to_archive`

**Begründung:** Wer einem Verweis aus einer gültigen Spezifikation folgt, darf nicht in einer erledigten landen und sie für gültig halten. ADRs, Verfassung, Doku und Runbooks sind ausgenommen: Historisches zu zitieren ist dort ihr Zweck.

#### SDD-051 Prüfvertrag {#sdd-051-vertrag}

1. **Quellen** sind die aktiven Feature-Artefakte (PRD, Tech Spec, Plan).
2. **Verweise** sind Inline-Links und -Bilder `[text](ziel "titel")` sowie Referenzdefinitionen `[name]: ziel` außerhalb von Codeblöcken. Ziele mit Schema (`https:`, `mailto:` …), reine Anker (`#…`) und protokollrelative Ziele (`//…`) werden ignoriert; `#…` und `?…` am Ende werden abgeschnitten, Prozentkodierung wird aufgelöst.
3. Ein Ziel mit führendem `/` gilt relativ zum Repository, sonst relativ zum Ordner der Quelle. Ziele außerhalb des Repositorys werden ignoriert, nie gelesen.
4. **Archiviert** ist ein Ziel, das `artifacts.archive` auswählt, oder eine Datei mit Frontmatter, nicht-leerem `feature:` und `status: archived`.
5. Für jeden Verweis auf ein archiviertes Ziel entsteht ein `warn` an der Zeile des Verweises.

---

## Archivierung (`SDD-050`)

PRD, Tech Spec und Plan gelten bis zum Release. Danach sind sie Historie: Sie werden nach `artifacts.archive` verschoben, nicht gelöscht, und tragen `status: archived`. Der Archivierungsschritt gehört in den Release-Vorbereitungs-Commit, zusammen mit dem Changelog, unmittelbar vor dem Git-Tag.

**Nachträge vor dem Archivieren:** Ändert die Release-Vorbereitung Pfade über der Größenschwelle (z. B. die Version in `catalog/**`), findet `pre-merge` auf dem Release-Branch kein Feature mehr — alle Artefakte liegen dann im Archiv. Solche Änderungen gehören deshalb vorher als Commit des zuständigen Features (dessen `touches` den Pfad nennt) auf den Haupt-Branch; der Release-Branch enthält danach nur Archivierung, Changelog und Versionsangaben außerhalb der Schwelle.

---

### SDD-050 — Abgeschlossene Features sind vor dem Release archiviert {#sdd-050}

- **Schweregrad:** `warn` (`default`, `pre-merge`), `error` in `pre-release`
- **Profil:** ab `standard`
- **Baseline:** `forbidden` — Archivieren ist ein Verschieben, keine Autorschaft; der Zustand lässt sich jederzeit billig herstellen.
- **Check:** `archive_state`

**Begründung:** Ein Gate nach dem Release blockiert nichts. Deshalb wird vor dem Release-Tag geprüft, dass kein abgeschlossenes Feature mehr zwischen den aktiven Artefakten liegt. Sonst lesen Menschen und Agenten erledigte Anforderungen als gültig.

#### SDD-050 Prüfvertrag {#sdd-050-vertrag}

Ein **Feature-Artefakt** ist eine Markdown-Datei unter `artifacts.prd`, `artifacts.design` oder `artifacts.plans` (rekursiv, ohne `exclude`-Treffer) mit gültigem Frontmatter und nicht-leerem `feature:`-Feld. Die Prüfung ist repository-weit; es findet keine Feature-Auflösung statt.

1. Liegt ein Feature-Artefakt nicht unter `artifacts.archive` und gilt mindestens eines der folgenden, entsteht genau ein Befund je Datei:
   - `status: implemented`;
   - `archive-after` ist gesetzt und kein Datum der Form `YYYY-MM-DD` (ein Git-Tag ist kein gültiger Wert);
   - `archive-after` liegt vor dem heutigen lokalen Datum (am Stichtag selbst noch kein Befund).
2. Trägt eine Markdown-Datei unter `artifacts.archive` ein nicht-leeres `feature:`-Feld und einen `status` außer `archived` oder `superseded`, entsteht ein Befund.
3. Ist `artifacts.archive` nicht konfiguriert, entfällt Punkt 2; Befunde nach Punkt 1 verweisen dann auf die fehlende Konfiguration.

---

### SDD-003 — Verfassung enthält keine Feature-Inhalte {#sdd-003}

- **Schweregrad:** `warn` (`default`, `pre-merge`) — die Erkennung ist heuristisch, daher nie `error`
- **Profil:** ab `minimal`
- **Baseline:** `allowed`
- **Check:** `constitution_without_feature_content`

**Begründung:** Die Verfassung hält dauerhafte, systemweite Regeln. User Stories, Tasks, Akzeptanzkriterien und Planung veralten mit jedem Release; stehen sie in der Verfassung, verwässern sie deren Autorität und erzeugen Drift zum PRD.

#### SDD-003 Prüfvertrag {#sdd-003-vertrag}

Geprüft wird jede Zeile der Verfassung außerhalb von Frontmatter und Codeblöcken; bei einer `.sdd`-Verfassung entfallen Kommentarzeilen (`#`, `//`), und Section-Labels gelten als Überschriften. Eine Zeile ist Feature-Inhalt, wenn sie — in dieser Reihenfolge, erster Treffer zählt —

1. eine Überschrift ist, die mit `Sprint`, `Backlog`, `Roadmap`, `Meilenstein`, `Milestone` oder `User Stor…` an einem Wortanfang enthält (Groß-/Kleinschreibung egal);
2. eine User Story ist: `Als … möchte ich` bzw. `As a/an … I want`;
3. eine Checkbox-Task ist (Erkennung wie bei [`SDD-041`](#sdd-041-vertrag));
4. eine AC-Kennung `AC-<n>` enthält.

Für jede solche Zeile entsteht ein `warn`. Fließtext, der diese Begriffe nur erwähnt („User Stories gehören ins PRD“), ist kein Treffer.

---

## Strikter Modus der Verfassung (`SDD-002b`)

### SDD-002b — `.sdd`-Verfassung erfüllt die formale Grammatik {#sdd-002b}

- **Schweregrad:** `error` (`default`, `pre-implementation`, `pre-merge`)
- **Profil:** ab `minimal`
- **Baseline:** `allowed`
- **Check:** `constitution_sdd_grammar`

**Begründung:** Eine Verfassung, die ihre eigene Grammatik nicht erfüllt, ist kein Vertrag, sondern Prosa. Der Wert des `.sdd`-Formats liegt in maschinell auflösbarer Präzedenz und Eigentum (`Owns`, `Can modify`) — das gelingt nur bei fester Struktur.

#### SDD-002b Prüfvertrag {#sdd-002b-vertrag}

Die Regel gilt, wenn `artifacts.constitution` auf `.sdd` endet (Groß-/Kleinschreibung egal); fehlt die Datei, meldet `SDD-001`. Leere Zeilen und Kommentarzeilen (nach optionalem Einzug `#` oder `//`) werden übersprungen. Für jede andere Zeile gilt, je Verstoß ein `error` mit Zeilennummer:

1. Die erste solche Zeile lautet `Spec: <Name>` (nicht-leerer Name). Fehlt jede inhaltliche Zeile, entsteht ein Befund an Zeile 1.
2. Keine Zeile enthält einen Tabulator.
3. Die Einrückung mit Leerzeichen ist ein Vielfaches von zwei.
4. Jede nicht eingerückte Zeile nach `Spec:` ist ein **Section-Label** `Label:` (Buchstaben, Leerzeichen, Bindestriche; danach Zeilenende oder Leerraum), und das Label ist bekannt. Der Vergleich ignoriert Groß-/Kleinschreibung.

**Bekannte Labels:** standardmäßig `Architecture`, `Security`, `Data`, `Standards`, `References`, `Owns`, `Can modify` — die fünf Section-Typen aus `sdd_process_3` §10 und die SpecDD-Eigentumsangaben. Ein Projekt legt seine Grammatik mit `constitution_sections` in `sdd.config.yaml` fest; die Liste ersetzt dann die Standardliste vollständig:

```yaml
constitution_sections: [System Intent, Architecture, Security, Data, Owns, Can modify]
```

---

## Selbstschutz- und Coverage-Regeln (`SDD-070` – `SDD-073`)

Damit Regelkatalog, Handbuch und Prüfkern niemals auseinanderlaufen, prüft `sdd` bei jedem Lauf zuerst die Konsistenz des Katalogs selbst. Alle vier Regeln gelten ab Profil `minimal`, sind in allen Gates aktiv und tragen `baseline: forbidden`.

---

### SDD-070 — Jede Regel ist im Handbuch dokumentiert {#sdd-070}

- **Schweregrad:** `error` (alle Gates sowie `default`)
- **Profil:** ab `minimal`
- **Baseline:** `forbidden`
- **Check:** `catalog_handbook_section`

**Begründung:** Ein Handbuch, das einer Regel hinterherläuft, ist bereits veraltet. Deshalb entstehen Katalogeintrag und Handbuchabschnitt immer im selben Commit.

#### SDD-070 Prüfvertrag {#sdd-070-vertrag}

Für jede Regel im Katalog (`catalog/catalog.yaml`) muss gelten:
1. Das Feld `handbook` ist im Format `<pfad>#<anker>` gesetzt.
2. Die referenzierte Markdown-Datei existiert (im Arbeitsbaum oder im eingebetteten Handbuch).
3. In der Datei existiert eine Markdown-Überschrift (`#` bis `######`), die die exakte Regel-ID (z. B. `SDD-070`) enthält und deren normalisierter Anker (oder explizite `{#...}`-ID) mit `<anker>` übereinstimmt.

---

### SDD-071 — Jede Regel ist im Prüfkern implementiert {#sdd-071}

- **Schweregrad:** `error` (alle Gates sowie `default`)
- **Profil:** ab `minimal`
- **Baseline:** `forbidden`
- **Check:** `catalog_check_registered`

**Begründung:** Eine Regel ohne registrierte Implementierung im Prüfkern ist bloße Dokumentation und täuscht Sicherheit vor.

#### SDD-071 Prüfvertrag {#sdd-071-vertrag}

Für jede Regel im Katalog (`catalog/catalog.yaml`) muss gelten:
1. Das Feld `check` ist nicht leer.
2. Zur Laufzeit existiert in der Check-Registry des Prüfkerns (`internal/sddcore/rules`) eine registrierte Prüffunktion unter exakt diesem `check`-Namen.

---

### SDD-072 — Jede aktive Regel ist in mindestens einem Gate aktiv {#sdd-072}

- **Schweregrad:** `error` (alle Gates sowie `default`)
- **Profil:** ab `minimal`
- **Baseline:** `forbidden`
- **Check:** `catalog_gate_coverage`

**Begründung:** Wenn eine Regel weder über `gate_default` noch über ein `gates.<gate>.override` in `sdd.config.yaml` einem Gate zugeordnet ist, läuft sie in keinem Freigabeschritt.

#### SDD-072 Prüfvertrag {#sdd-072-vertrag}

Für jedes Gate $g \in \{\text{pre-implementation}, \text{pre-merge}, \text{pre-release}\}$ wird die effektive Regelmenge $\text{Eff}(g)$ bestimmt:
1. Ist in `sdd.config.yaml` für das Gate $g$ ein nicht-leeres `override` konfiguriert, gilt $\text{Eff}(g) = \text{override}(g)$.
2. Andernfalls gilt $\text{Eff}(g) = \text{gate\_default}(g)$ aus `catalog/catalog.yaml`.
3. Jede im gewählten Projektprofil (`minimal` $\subset$ `standard` $\subset$ `regulated`) aktive Regel muss in der Vereinigungsmenge $\bigcup_g \text{Eff}(g)$ enthalten sein und in mindestens einem dieser Gates einen effektiven Schweregrad besitzen.

---

### SDD-073 — Jede error-Regel besitzt einen Prüfvertrag im Handbuch {#sdd-073}

- **Schweregrad:** `error` (alle Gates sowie `default`)
- **Profil:** ab `minimal`
- **Baseline:** `forbidden`
- **Check:** `catalog_error_contract`

**Begründung:** Blockierende Befunde (`error`) dürfen niemals auf unbestimmten Heuristiken beruhen. Jede `error`-Regel benötigt einen expliziten, nachprüfbaren Vertrag.

#### SDD-073 Prüfvertrag {#sdd-073-vertrag}

Für jede Regel im Katalog, deren `severity.default` gleich `error` ist oder die mindestens einen Eintrag `severity.by_gate.<gate>: error` besitzt, muss gelten:
1. Das Feld `contract` ist nicht leer und hat die Form `<pfad>#<anker>`.
2. Die Datei `<pfad>` existiert im Handbuch.
3. In `<pfad>` existiert eine Überschrift, deren normalisierter Markdown-Anker (oder explizite `{#...}`-ID) mit `<anker>` übereinstimmt.
