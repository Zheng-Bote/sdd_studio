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
