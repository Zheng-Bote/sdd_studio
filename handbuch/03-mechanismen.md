# 3. Mechanismen, Gates und Regelkatalog-Referenz

Dieser Teil des Handbuchs dokumentiert die Prüfmechanismen von `sdd`, das Berichtsformat sowie die Referenz und die verbindlichen Prüfverträge aller Regeln im Katalog (`catalog/catalog.yaml`).

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
4. Für jede definierte, nicht referenzierte Kennung entsteht ein `error` an ihrem ersten Vorkommen im PRD.
5. Trifft die Glob-Liste keine einzige Datei, entsteht statt Einzelbefunden genau ein `warn`: Ein fehlender Testpfad ist ein Konfigurationsproblem, kein Projektverstoß.

**Bekannte Grenze:** Kennungen sind je PRD vergeben, Referenzen aber nicht nach Feature getrennt. Ein Test mit `AC-1` erfüllt `AC-1` in jedem PRD.

---

## Feature-Artefakte bei Schwellenüberschreitung (`SDD-020`, `SDD-030`)

Ab einer projektspezifischen Größenschwelle verlangt `sdd` für ein Feature ein PRD (`SDD-020`) und eine Tech Spec (`SDD-030`). Beide Regeln teilen die Feature-Auflösung, die Glob-Syntax und die Schwellenlogik; sie unterscheiden sich nur im geprüften Verzeichnis (`artifacts.prd` bzw. `artifacts.design`).

### Feature-Auflösung {#feature-aufloesung}

Welche Features ein Gate-Lauf prüft, wird in dieser Reihenfolge bestimmt; die erste Stufe mit Ergebnis gewinnt:

1. **`--feature <slug>`** — kommagetrennt mehrere Slugs möglich. Ein Slug besteht ausschließlich aus `[a-z0-9-]`; ein ungültiger Slug bricht mit Exit-Code `2` ab.
2. **Frontmatter im Diff** — alle Markdown-Dateien unter `artifacts.prd`, `artifacts.design` und `artifacts.plans`, die in `git diff <base>...HEAD` vorkommen und ein gültiges `feature:`-Feld tragen.
3. **Branch-Name** — `feature/<slug>` oder `feat/<slug>` bzw. `feat/<nummer>-<slug>`. In CI mit losgelöstem `HEAD` gelten `$GITHUB_HEAD_REF`, `$CI_MERGE_REQUEST_SOURCE_BRANCH_NAME` und `$CI_COMMIT_BRANCH`.

Liefert keine Stufe ein Feature, gilt der Lauf als **nicht aufgelöst** (siehe Prüfverträge). Die Merge-Basis ist `base_ref` aus `sdd.config.yaml`; leer bedeutet: `$GITHUB_BASE_REF` bzw. `$CI_MERGE_REQUEST_TARGET_BRANCH_NAME` (bevorzugt als `origin/<name>`), sonst `@{upstream}`, sonst `main`, sonst `master`. Ist Git nicht verfügbar oder keine Basis ermittelbar, endet der Lauf mit Exit-Code `2`.

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

## Archivierung (`SDD-050`)

PRD, Tech Spec und Plan gelten bis zum Release. Danach sind sie Historie: Sie werden nach `artifacts.archive` verschoben, nicht gelöscht, und tragen `status: archived`. Der Archivierungsschritt gehört in den Release-Vorbereitungs-Commit, zusammen mit dem Changelog, unmittelbar vor dem Git-Tag.

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
