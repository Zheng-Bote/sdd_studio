---
type: Note
title: "sdd_konzept"
# Kurzbeschreibung in einem Satz; md-ai zeigt sie als Entitätsbeschreibung.
description: "Konzept für Handbuch und Werkzeug zu sdd_process_3: ein Regelkatalog mit Prüfverträgen, eine gate-gebundene Baseline für Bestandsprojekte, drei Gates, sechs Artefakttypen — eine Quelle, zwei Ausgaben."
# z. B. active, deprecated; bei ADR: proposed, accepted, superseded
status: draft
notebook: "Snippets"
tags:
  - SpecDD
  - Prozess
  - Konzept
  - Werkzeug
# Relationen für den Wissensgraphen: exakte Dateinamen anderer Notizen,
# ohne .md und ohne [[ ]], z. B. uses: [Encryption Service]
uses: []
depends_on: []
contains: []
references: []
adr: []
relation:
  - sdd_process_3
  - sdd_process_2
  - sdd_prozess
  - SDD-Howto_de
  - SpecDD_SpecKit_Integration_DE
# Endpunkt oder Dateiname der API-Doku (leer = keine API dokumentiert)
api: ""
creation date: 2026-10-18 14:00
modification date: Sonntag 18. Oktober 2026 15:33:05
---

# sdd_konzept

----

## Überblick

Dieses Dokument beschreibt zwei Ergebnisse auf einer gemeinsamen Grundlage: ein **Handbuch** für Menschen und ein **Desktop-Programm** für die tägliche Arbeit, beide abgeleitet aus dem Prozess in `sdd_process_3`.

Die tragende Idee ist bewusst schlicht:

> Es gibt **einen Regelkatalog**. Das Handbuch erklärt ihn, das Programm setzt ihn durch. Beide sind Ausgaben derselben Quelle — und diese Quelle wird maschinell geprüft.

Damit wendet das Konzept die zentrale Forderung aus `sdd_process_3` auf sich selbst an: Mechanismen statt Disziplin. Ein Handbuch, das veraltet, und ein Werkzeug, das Regeln prüft, die niemand dokumentiert hat, wären genau der Drift, den der Prozess verhindern soll.

Vier Konsequenzen aus dieser Selbstanwendung prägen die aktuelle Fassung:

- **Keine Regel ohne Prüfvertrag.** Eine Regel, deren Erkennung nicht eindeutig beschrieben ist, ist entweder heuristisch — dann ist sie `warn` — oder unbestimmt. Unbestimmtheit in einem Gate ist ein Fehlalarm mit Ankündigung.
- **Kein Gate ohne Baseline.** Ein Gate, das den Altbestand eines gewachsenen Projekts vollständig gegen neue Regeln prüft, wird nach dem ersten Lauf abgeschaltet. Damit ist es tot, bevor es wirkt.
- **Keine Gate-Aussage ohne Zeitbezug.** Was ein Gate prüfen kann, hängt davon ab, was zum Zeitpunkt des Laufs existiert. Ein Gate vor der Implementierung kann die Implementierung nicht sehen und muss deshalb auf einer erklärten Absicht arbeiten (Abschnitt 5.1.2).
- **Kein Katalog ohne Schnitt.** Ein Regelkatalog wächst, wenn man ihn lässt. Deshalb legt Abschnitt 15 fest, welche Regeln v1 umfasst und was ausdrücklich später kommt. Ohne diesen Schnitt beginnt die Umsetzung mit einem Katalog, der nie fertig wird.

---

## 1. Ausgangslage

`sdd_process_3` trennt fünf Artefakttypen, ordnet sie nach Präzedenz und ersetzt den neunphasigen Wasserfall durch eine Denkreihenfolge mit drei verbindlichen Gates. Der Kern ist dort bereits formuliert:

> Die Reduktion von SpecDD ist nicht die Antwort. Die Durchsetzung der bestehenden Regeln ist es.

Genau an dieser Stelle setzt das Konzept an. Die drei Gates und der Regelkatalog sind ohne Werkzeug nicht durchsetzbar:

- **Gate 1** (`pre-implementation`, vor Implementierung) braucht eine Prüfung, ob ADR und Tech Spec für die *beabsichtigte* Änderung vorliegen — sonst ist es eine Erinnerung.
- **Gate 2** (`pre-merge`, vor Merge) braucht eine maschinelle Prüfung von AC-IDs, Verfassungsänderung und Test-Verweisen — sonst ist es eine Checkliste, die man wegklickt.
- **Gate 3** (`pre-release`, vor dem Release-Tag) braucht eine repository-weite Prüfung, ob abgeschlossene Features nach `archive/` überführt wurden. Ein Gate *nach* dem Release blockiert nichts mehr und bliebe beim Vorsatz.

**Bewusste Abweichung von `sdd_process_3`:** Dort ist Gate 3 als Schritt *nach* dem Release beschrieben. Das Konzept zieht es nach vorn, weil ein Gate, das nichts mehr blockieren kann, kein Gate ist. Diese Änderung ist in Abschnitt 14 als getroffene Entscheidung festgehalten.

Ein weiterer Befund aus der Praxis verstärkt das: Im Referenzprojekt liegt CI als `.github/workflows/ci.yml_` vor — mit Unterstrich, also von GitHub Actions **nicht ausgeführt**, und inhaltlich ein C++-Workflow aus einem anderen Projekt. Es gibt dort faktisch keine Pipeline. Ein Gate, das nicht läuft, ist kein Gate. Das ist die stärkste Begründung dafür, dass der Prüfkern **headless** sein muss.

---

## 2. Leitidee: ein Regelkatalog, zwei Ausgaben

```text
                 +-------------------------+
                 |   Regelkatalog (YAML)   |
                 |   SDD-001 ... SDD-0nn   |
                 |   ID, Schwere, Prüfung, |
                 |   Begründung, Beispiel  |
                 +----+---------------+----+
                      |               |
        +-------------+               +-------------+
        |                                           |
        v                                           v
+-------------------+                     +-------------------+
| Handbuch          |                     | Programm          |
| - Referenzteil je |                     | - Lint / Gates    |
|   Regel-ID        |                     | - Init / Adopt    |
| - Rezepte, Theorie|                     | - Baseline, Editor|
+-------------------+                     +-------------------+
        |                                           |
        +-------------------+-----------------------+
                            v
                 +-------------------------+
                 | Coverage-Check          |
                 | jede Regel dokumentiert?|
                 | jede Regel implementiert?|
                 | jede Regel in einem Gate?|
                 | jede error-Regel mit     |
                 | Prüfvertrag?             |
                 +-------------------------+
```

Der Coverage-Check ist der Selbstschutz des Konzepts:

- **SDD-070** — jede Regel im Katalog hat einen Handbuchabschnitt mit ihrer ID.
- **SDD-071** — jede Regel im Katalog hat eine Implementierung im Prüfkern.
- **SDD-072** — jede Regel ist in mindestens einem Gate effektiv aktiv.
- **SDD-073** — jede `error`-Regel hat einen Prüfvertrag.

Fällt eine der **vier** Prüfungen aus, ist das Produkt unvollständig — und das ist sichtbar, nicht vermutet.

Regel und Handbuchabschnitt entstehen im **selben Commit**. Das ist derselbe Mechanismus, den SDD-010 für Verfassungsänderungen fordert, und es ist die einzige Variante, die den eigenen Prinzipien standhält: Eine Regel ohne Erklärung ist nicht implementierbar, und ein Handbuch, das einer Regel hinterherläuft, ist bereits veraltet.

**Selbstanwendung von Anfang an:** SDD-070 bis SDD-073 sind selbst `error`-Regeln und müssen sich selbst erfüllen. Der erste Commit in Schritt 1 enthält deshalb für alle vier gleichzeitig den Handbuchabschnitt, die Implementierung, die Gate-Zuordnung und den Prüfvertrag — sonst scheitert das Werkzeug an sich selbst, bevor Dogfooding beginnt. Das gilt auch für die Regeln des v1-Schnitts (Abschnitt 15): Jede dort genannte `error`-Regel bekommt Vertrag und Handbuchabschnitt, bevor sie in den Katalog aufgenommen wird.

---

## 3. Ziele und Nicht-Ziele

### Ziele

- Ein Projekt — neu oder bestehend — in einen Zustand bringen, in dem dauerhafte Regeln und temporäre Arbeitsartefakte getrennt und auffindbar sind.
- Die drei Gates aus `sdd_process_3` ohne menschliche Disziplin durchsetzen.
- **Programmiersprachen- und Framework-neutral** für beliebige Zielprojekte (Go, Python, TypeScript, Rust, C/C++, Java sowie Polyglot-Repositories): Alle Prüfungen arbeiten auf Dateipfaden, Git-Diffs, Frontmatter und sprachneutralen Textmustern, ohne sprachspezifische Compiler oder AST-Parser vorauszusetzen (Go dient als Implementierungssprache des Werkzeugs selbst und für Beispiele).
- Das Handbuch so knapp halten, dass es gelesen wird: Theorie nur so viel wie nötig, Rezepte für die häufigen Fälle.
- Bestehende Projekte **inkrementell** heben, nicht umbauen. Kein Big Bang, kein Datenverlust — und keine Wand aus Altbefunden, die das Gate unbenutzbar macht.
- Vollständig offline und ohne Konto nutzbar. Keine Telemetrie.

### Nicht-Ziele

- Kein Projektmanagement-Werkzeug. Keine Sprints, keine Story Points, keine Burndown-Charts.
- Kein Ticket-System und kein Git-Client.
- Keine Dokumentenverwaltung mit eigener Datenbank als Wahrheit.
- Kein Generator, der Inhalte erfindet. Das Programm erzeugt Gerüste und Befunde, keine Prosa.
- Keine Cloud, kein Server, kein Account.
- Kein Ersatz für die Verfassung: das Programm kennt nicht „die Wahrheit", es prüft Dateien.

---

## 4. Produktbild

### 4.1 Handbuch

Zielgruppe sind drei Leser mit unterschiedlichem Bedarf:

- **Einsteiger:** „Wie fange ich an?" — braucht Rezepte, keine Theorie.
- **Erfahrene:** „Welche Regel gilt hier und warum?" — braucht den Referenzteil.
- **Agenten:** „Wie löse ich einen Konflikt auf?" — braucht den Regelkatalog in maschinenlesbarer Form, nicht Prosa.

Aufbau in fünf Teilen:

1. **Grundlagen** — Lebenszyklen, die **sechs** Artefakttypen (Abschnitt 4.3), Pfadvererbung, Präzedenz, Eigentum. Etwa 15 Seiten.
2. **Rezepte** — kleines Feature, großes Feature, Architekturänderung, Bugfix, Hotfix, Archivierung, Umgang mit Explorationsdokumenten. Der meistgelesene Teil; jedes Rezept eine Seite mit Ablauf und Checkliste.
3. **Mechanismen** — Gates, CI-Einbindung, Prüfverträge, Berichtsformat, Regelkatalog-Referenz mit einer Seite je Regel-ID.
4. **Für Agenten** — wie ein Agent die Verfassung liest, Präzedenz auflöst und Gates prüft; identisch mit dem, was das Programm tut.
5. **Anhänge** — Templates, Checklisten, Glossar, Statusvokabular je Artefakttyp, Fallback-Glob-Liste, Änderungshistorie des Katalogs.

Format: Markdown im Repository des Werkzeugs, versioniert. Der Referenzteil wird **nicht** generiert — generierte Prosa liest niemand. Stattdessen verweist jede Regel-ID auf einen handgeschriebenen Abschnitt, und der Coverage-Check stellt sicher, dass keiner fehlt.

**Sprachstrategie:** Regel-IDs, `check`-Namen und Feldnamen sind sprachneutral und stabil. `title` und `rationale` im Katalog sind deutsch; für Agenten zählt ausschließlich die ID plus Prüfvertrag. Eine englische Handbuchfassung ist möglich, aber nicht Teil von v1 — sie wäre eine zweite Quelle, die gepflegt werden müsste.

Versionierung: Der Katalog hat eine eigene Version. Breaking Changes an einer Regel erhöhen die Major-Version und erzeugen einen Migrationshinweis im Handbuch.

### 4.2 Programm („SDD Studio" / CLI `sdd`)

Projekt- und Repository-Name ist **SDD Studio** (`sdd_studio`), das CLI-Kommando heißt **`sdd`**, die Kernbibliothek **`internal/sddcore`**.

Das Programm hat fünf Betriebsarten, alle aus demselben Kern:

- **Init** — neues Projekt anlegen: Verfassung, Verzeichnisstruktur, Templates, CI-Baustein, Projektkonfiguration, Baseline.
- **Adopt** — bestehendes Projekt übernehmen: inventarisieren, klassifizieren, Zuordnung vorschlagen, Probelauf, anwenden.
- **Audit** — laufende Prüfung: Regelkatalog durchsetzen, Befunde mit Ort und Schweregrad, Baseline-Abgleich.
- **Author** — Artefakte aus Templates anlegen (`sdd new`), Live-Prüfung im Watch-Modus und direkter Absprung an die Fundstelle im System-/IDE-Editor (`$EDITOR`, `vscode://`, `obsidian://`).
- **Gate** — Freigabeprüfung: die drei Gates als einzelne, aufrufbare Kommandos.

**Alle fünf Betriebsarten funktionieren ohne grafische Oberfläche.** Die Oberfläche ist eine Ansicht auf dieselbe Bibliothek, kein eigenständiges Produkt. Sonst gälte die Regel nur auf dem Rechner desjenigen, der die App installiert hat.

### 4.3 Artefakttypen

`sdd_process_3` nennt fünf Typen. Für die Praxis fehlt ein sechster: Es gibt Dokumente, die weder Verfassung noch ADR noch Feature-Artefakt sind — Explorationsnotizen, Recherchen, Leitfäden, Design-Skizzen, dieses Dokument selbst.

```text
1  Verfassung        systemweite Regeln, versioniert
2  ADR               unveränderliche Entscheidung mit Begründung
3  PRD               fachliche Anforderungen mit AC-IDs
4  Tech Spec         technische Umsetzung
5  Plan / Tasks      Reihenfolge und Arbeitspakete
6  Doku/Exploration  alles andere: Notizen, Recherche, Leitfäden
```

Der sechste Typ hat bewusst weniger Regeln: kein `feature:`-Feld, kein AC-Zwang, kein Plan. Er hat nur einen Status (`active` | `archived`) und darf nicht unter `prd`, `design` oder `plans` liegen — sonst würde er die Feature-Prüfungen auslösen. Ohne diesen Typ müsste `sdd adopt` in jedem echten Repository Dokumente falsch klassifizieren oder ignorieren, und beides wäre eine Lüge im Bericht.

---

## 5. Funktionen

### P0 — ohne diese ist das Produkt sinnlos

- **Regelkatalog** mit IDs, Schweregraden (auch gate-abhängig), Prüfart, Prüfvertrag, Baseline-Fähigkeit und Begründung.
- **`sdd lint`** — Verfassung und Artefakte gegen den Katalog prüfen, stabil definierte maschinenlesbare Ausgabe (optional `--watch`).
- **`sdd baseline`** — Ist-Stand festschreiben, damit Altbefunde das Gate nicht blockieren (Abschnitt 5.2).
- **`sdd init`** — neues Projekt auf das gewählte Profil bringen, inklusive Baseline.
- **`sdd gate pre-implementation | pre-merge | pre-release`** — die drei Gates.
- **`sdd new`** — Artefakt aus Template anlegen, mit gesetzten Status- und Archivfeldern je Artefakttyp.
- **Handbuch v1** mit Grundlagen, Rezepten, Prüfverträgen und Referenzteil.

### P1 — der eigentliche Nutzen für bestehende Projekte

- **`sdd adopt`** — Inventar, Reifegrad, Vorschlag, Probelauf, Anwendung auf einem Branch.
- **`sdd report`** — Reifegrad, Befunde und Schuldenstand als lesbarer Bericht, auch als PR-Kommentar.
- **GitHub Action** als dünner Wrapper um `sdd gate pre-merge` und `sdd gate pre-release` (siehe Roadmap: Schritt 2, weil sie Voraussetzung für „ab Tag eins in CI prüfbar" ist).
- **Grafische Oberfläche** (`sdd ui` bzw. Desktop-App) mit Projektübersicht, Artefaktbaum, Befundliste, Template-Anlage und Migrationsassistent.

### P2 — Ausbau, nur bei Bedarf

- **Index-Anbindung** für große Bestände: „Wo ist X dokumentiert?", „Welche Dokumente betreffen diese Änderung?" — über einen vorhandenen Suchindex, nicht über eine eigene Datenbank.
- **Agenten-Schnittstelle** (MCP-Server): Regeln, Befunde und Gate-Ergebnisse für LLM-Agenten abrufbar.
- **KI-Assistenz** — lokal, standardmäßig aus, nur Vorschläge (Abschnitt 9.6).
- **`sdd staleness`** — zeitgesteuerter Lauf, der abgelaufene Archivfristen und veraltete Verweise meldet, ohne zu blockieren.

### 5.1 Feature-Auflösung (Scope Resolution bei Gate-Läufen)

Ein Repository enthält über seine Lebensdauer viele PRDs, Tech Specs und Pläne. Damit `sdd gate pre-implementation` und `sdd gate pre-merge` deterministisch wissen, **welches** Feature zum aktuellen Lauf gehört, gilt folgende Auflösungsreihenfolge:

1. **Explizites Flag (`--feature <slug>`):** Überschreibt jede Ableitung (z. B. `sdd gate pre-merge --feature openrouter-provider`). Mehrfach angebbar; dann wird jedes genannte Feature einzeln geprüft.
2. **`feature:`-Feld im Feature-Frontmatter:** Das Identitätsfeld der Artefakte selbst (`feature: billing-export` in `specs/features/billing-export/spec.md` oder `docs/prd/billing-export.md`). Es ist die expliziteste Aussage, die das Projekt über sich trifft: nicht abgeleitet, sondern geschrieben. Weichen Ordner- und Feldname ab, gewinnt das Feld — und `sdd lint` meldet die Abweichung als `warn`.
3. **Git-Diff (`git diff <base>...HEAD`):** Ermittelt alle im aktuellen Branch neu angelegten oder geänderten Dateien unter den konfigurierten Feature-Pfaden (`prd`, `design`, `plans` bzw. `specs/features/<slug>/`) und bestimmt daraus das aktive Feature-Set. Enthält der Diff mehrere Features, wird **jedes** einzeln geprüft; ein Fehler in einem Feature lässt das Gate fehlschlagen.
4. **Branch-Namenskonvention (`feature/<slug>` oder `feat/<id>-<slug>`):** Falls im Diff noch kein Feature-Artefakt existiert (genau der Fehlerfall bei Überschreiten der Größenschwelle!), wird `<slug>` aus dem Branch-Namen abgeleitet und im Befund benannt.

Der **Slug ist die Identität**, nicht der Pfad. Er ist kleingeschrieben, ASCII, `[a-z0-9-]+`. Er steht im `feature:`-Feld, im Ordner- oder Dateinamen, im Branch-Namen und im Bericht. Eine abweichende Schreibweise zwischen diesen Stellen ist ein `warn`, kein Fehler — sie ist meist historisch gewachsen.

### 5.1.1 Wenn keine Zuordnung gelingt: fail-closed (für Feature-Gates)

Der Fall „weder Flag noch Frontmatter noch Diff noch Branch-Konvention liefern ein Feature" ist bei den **Feature-bezogenen Gates** (`pre-implementation` und `pre-merge`) **kein Randfall**, sondern der Normalfall bei `fix/typo`, bei Läufen auf `main` und bei jedem Branch, dessen Name keiner Konvention folgt.

Ohne Regel wäre die wahrscheinlichste Implementierung „kein Feature gefunden → nichts zu prüfen → Erfolg". Das ist stilles Durchwinken und damit genau der Fehler, gegen den dieses Konzept antritt.

Wichtig ist die Unterscheidung zweier Zustände, die leicht verwechselt werden:

- **Absicht unbekannt:** Es existiert kein Feature-Artefakt und keine `--paths`-Angabe. Dann ist die Größenschwelle **nicht bewertbar** — nicht „nicht berührt". Ergebnis: `warn` mit dem Vermerk „keine Feature-Prüfung erfolgt; Änderungsabsicht nicht erklärt".
- **Absicht erklärt, Artefakt fehlt:** Es gibt `--paths` oder ein `touches`-Feld, die Schwelle ist damit bewertbar und überschritten — aber kein Feature-Artefakt ist auflösbar. Ergebnis: `error`. Genau hier greift die Fail-closed-Regel.

Damit verbindlich für `pre-implementation` und `pre-merge`:

- **Absicht erklärt, Schwelle berührt, kein Artefakt auflösbar:** `error`. Das Gate schlägt fehl mit dem Hinweis, `sdd new --feature <slug>` auszuführen.
- **Absicht erklärt, Schwelle nicht berührt:** kein Befund; das Gate prüft die übrigen Regeln.
- **Absicht nicht erklärt (kein Artefakt, kein `--paths`):** `warn` mit ausdrücklichem Vermerk. Kein stilles Durchwinken, aber auch keine Blockade für einen Branch, der noch gar nicht angefangen hat.

**Abgrenzung zu `pre-release`:** Das Gate `pre-release` ist **nicht** Feature-bezogen, sondern prüft **repository-weit** alle Dokumente unter den konfigurierten Pfaden (`prd`, `design`, `plans`, `archive`), da ein Release auf `main` bzw. einem Release-Branch mehrere gemergte Features bündelt. Eine Einzel-Feature-Auflösung nach 5.1 findet bei `pre-release` nicht statt.

Die Branch-Namenskonvention ist selbst eine Konvention und keine Naturgegebenheit. Sie steht deshalb im Handbuch (Abschnitt „Rezepte") und in der Projektkonfiguration, nicht nur in diesem Konzept.

### 5.1.2 Gate 1 kennt die Implementierung nicht — erklärte Änderungsabsicht

`pre-implementation` läuft, bevor der Code geschrieben ist. Ein `git diff` zeigt zu diesem Zeitpunkt meist nur den Branch-Start, nicht die Änderung, die später die Größenschwelle auslösen wird. Würde Gate 1 die Schwelle aus dem Diff ableiten, käme fast immer „Schwelle nicht berührt" heraus — und das Gate würde passieren, bevor es etwas geprüft hat. Der Fehler fiele erst in `pre-merge` auf, also zu spät für die Frage, ob ein PRD überhaupt nötig ist.

Deshalb arbeitet Gate 1 auf einer **erklärten Änderungsabsicht**, nicht auf dem Diff. Um Merge-Konflikte in der zentralen `sdd.config.yaml` bei parallelen Feature-Branches zu vermeiden und die Absicht auch für `pre-merge` (`SDD-024`) sowie spätere Archivierungen dauerhaft nachvollziehbar zu halten, wird die Änderungsabsicht **im Frontmatter des Feature-Artefakts** (erzeugt durch `sdd new`) hinterlegt:

```yaml
---
feature: billing-export
status: active
archive-after: 2026-12-31
touches:
  - openapi.yaml
  - internal/service/**
---
```

Für schnelle Vorabprüfungen auf der Kommandozeile (oder wenn noch kein Artefakt angelegt wurde) lässt sich die Absicht alternativ per Flag übergeben:

```text
sdd gate pre-implementation --feature billing-export --paths openapi.yaml,internal/service/**
```

Ausgewertet werden `touches` (bzw. `--paths`) gegen dieselbe Schwellenlogik wie in Gate 2. Fehlt sowohl ein `touches`-Eintrag im Feature-Artefakt als auch `--paths`, ist das Gate-1-Ergebnis für SDD-020/030 **nicht** „bestanden", sondern `warn`: „Änderungsabsicht nicht erklärt; Schwelle nicht bewertbar". Kein stilles Durchwinken.

Damit ist die Arbeitsteilung eindeutig:

- **Gate 1** prüft Absicht gegen Anforderung: Ist die Schwelle nach der Erklärung berührt, müssen PRD, Tech Spec und (falls die Verfassung berührt wird) ein entschiedenes ADR existieren.
- **Gate 2** prüft Wirklichkeit: Der Diff zeigt, was tatsächlich geändert wurde; die Schwelle wird erneut ausgewertet und muss zur Absicht passen. Weicht der Diff von der in `touches` erklärten Absicht ab, ist das ein eigener Befund (SDD-024, `warn`) — meist ein Zeichen dafür, dass der Scope während der Implementierung gewachsen ist, ohne die Spezifikation nachzuführen.

### 5.2 Baseline und Ratchet — der Mechanismus für Bestandsprojekte

Das übergeordnete Ziel lautet „bestehende Projekte inkrementell heben". Ohne Baseline ist das nicht erreichbar: Ein gewachsenes Projekt verletzt neue Regeln hundertfach, und ein Gate, das den gesamten Altbestand als `error` meldet, wird abgeschaltet — nicht repariert.

**Prüfbefund im Pilotprojekt** (`md-indexer`, Stand 2026-10-18): 22 Feature-Verzeichnisse, davon 11 ohne `plan.md`/`tasks.md`; **keine** `AC-n`-Kennung in `specs/` und **keine** in `*_test.go`; kein `docs/archive`. Ein erster Lauf mit Profil `standard` würde dort Dutzende Fehler für längst gemergte Arbeit erzeugen.

Deshalb:

```text
sdd baseline                       # Ist-Stand für alle Gates aufnehmen
sdd baseline --gate pre-merge      # nur für ein Gate
sdd lint                           # neue Verstöße = error, bekannte = info
sdd baseline --prune               # erledigte Einträge entfernen
```

Regeln des Verfahrens:

- **Die Baseline ist versioniert.** `.sdd/baseline.json` liegt im Repository und wird im Review gesehen. Sie ist kein Cache und nicht wegwerfbar.
- **Sie enthält nur `error`-Befunde.** `warn` und `info` brauchen keine Baseline, weil sie nicht blockieren.
- **Ein Eintrag ist an ein Gate gebunden.** Identität: `gate` + `rule_id` + `path` + `content_hash` + `ordinal`. `line` ist informativ und **nicht** Teil der Identität; sonst macht jede eingefügte Zeile oberhalb alle Einträge derselben Datei ungültig.
- **`ordinal` ist deterministisch definiert:** der nullbasierte Index der Fundstelle innerhalb ihrer Datei, sortiert nach aufsteigendem Byte-Offset. Zwei Funde derselben Regel mit gleichem `content_hash` in derselben Datei werden so unterscheidbar; die Reihenfolge hängt nicht von der Iterationsreihenfolge des Prüfers ab.
- **Textänderung hebt die Unterdrückung auf.** Ändert sich der betroffene Text, passt der `content_hash` nicht mehr; der Eintrag gilt als erledigt und der Befund wird wieder `error`. Das verhindert, dass jemand eine Baseline schreibt und danach ungestört weiterarbeitet.
- **Sie darf nur schrumpfen.** `sdd baseline --prune` entfernt Einträge, die nicht mehr zutreffen. Neue Einträge nur über `sdd baseline --accept <rule> --gate <gate>`; das erzeugt einen eigenen Commit mit Begründung im PR — sichtbar, nicht still.
- **Nicht jede Regel darf gebaselinet werden.** Jede Regel trägt `baseline: allowed | forbidden`. `forbidden` gilt für SDD-050 (Archivierung), SDD-051, SDD-070–073 (Selbstprüfung). Begründung: Diese Regeln sichern einen Zustand, der sich einmalig und billig herstellen lässt — Archivieren ist ein Verschieben, keine Autorschaft. Würde man sie baselinen, wäre der Anspruch „der getaggte Release-Stand ist aufgeräumt" nicht mehr wahr.
- **`sdd report` zeigt den Schuldenstand**: Anzahl baselineter Befunde je Regel und Gate, Veränderung seit dem letzten Lauf. Ohne Trendanzeige ist die Baseline eine Decke.

Konsequenz für die erste Einführung: Der Altbestand wird für die inhaltlichen Regeln (PRD, AC, Plan) baselined, für die Archivierungsregel aber nicht. Das bedeutet genau **einen** zusätzlichen Aufräumschritt vor dem ersten Release nach der Übernahme — Dateien verschieben und `status: archived` setzen. Das ist vertretbar und wird im Adopt-Bericht als eigener Posten ausgewiesen, nicht verschwiegen.

Damit wird aus „inkrementell heben" ein Mechanismus statt einer Absicht — und die Roadmap ändert sich: `sdd baseline` gehört in Schritt 1, nicht in Schritt 3.

---

## 6. Reifegrade für bestehende Projekte

Ein bestehendes Projekt lässt sich nicht an einem Tag umbauen. Deshalb ein Stufenmodell; das Programm stellt fest, wo ein Projekt steht, und schlägt **genau einen** nächsten Schritt vor.

- **L0 — unsortiert:** Regeln liegen verstreut in README, Tickets und Köpfen.
- **L1 — Verfassung:** eine gültige Verfassung existiert, Regeln sind auffindbar, Präzedenz ist benannt.
- **L2 — Trennung:** PRD, Tech Spec und Plan sind getrennte Artefakte, Akzeptanzkriterien haben IDs.
- **L3 — Mechanismen:** Lint und Gates laufen in CI, ADR-Pflicht ist aktiv, Baseline vorhanden und schrumpfend.
- **L4 — Lebenszyklus:** Status, Archivdatum und Archiv vorhanden; die drei Fragen „was gilt, was ist Planung, was ist veraltet" sind beantwortbar.

Der Vorschlag ist bewusst immer nur eine Stufe. Ein Werkzeug, das L0 auf L4 heben will, produziert einen Umbruch, den niemand prüft.

### Adopt-Ablauf

```text
1. Inventar       Dateien, Verzeichnisse, vorhandene Specs, CI-Konfiguration
2. Klassifikation jedes Dokument einem der sechs Artefakttypen zuordnen (Vorschlag, nie automatisch)
3. Reifegrad      L0-L4 bestimmen, mit Begründung je Kriterium
4. Baseline       Ist-Stand aufnehmen, damit Altbefunde nicht blockieren
5. Vorschlag      genau der nächste Schritt, mit erwartetem Aufwand
6. Probelauf      vollständiger Diff, ohne zu schreiben
7. Anwenden       auf einem Branch, ein Commit pro Schritt, Bericht als Datei
8. Prüfen         Gate-Lauf auf dem Branch, Schuldenstand im Bericht
```

Regeln dafür, weil hier am meisten kaputtgehen kann:

- **Nichts wird gelöscht.** Verschieben ja, löschen nein.
- **Nichts wird überschrieben.** Existiert eine Datei, wird sie nie still ersetzt; Konflikte landen im Bericht.
- **Alles auf einem Branch.** Der Hauptzweig bleibt unberührt, bis der Nutzer merged.
- **Jeder Schritt ist einzeln rücknehmbar.** Ein Commit pro Schritt, keine Sammeländerung.
- **Die Baseline wird vor dem ersten Gate geschrieben**, nie danach. Sonst zementiert man Befunde, die man gerade erzeugt hat.
- **Der Archivierungsschritt wird ausgewiesen, nicht ausgeführt.** Der Adopt-Bericht nennt die Features, die vor dem ersten Release nach `archive/` müssen, und die geschätzte Anzahl Verschiebungen. Ausgeführt wird er erst nach Bestätigung.
- **Unklassifizierbares wird als Doku/Exploration ausgewiesen**, nicht stillschweigend einem Feature zugeordnet. Ein Bericht, der „weiß nicht" sagt, ist brauchbarer als einer, der rät.

---

## 7. Profile statt einer Regel für alle

`sdd_process_3` nennt eine Größenschwelle, aber keine Konfiguration. Das Programm macht daraus drei Profile.

- **`minimal`** — Issue plus Feature-Spec. Keine getrennte Tech Spec, kein Plan. Für Solo-Arbeit und kleine Änderungen. Verfassung als Markdown mit Section-Konventionen, ohne `.sdd`-Grammatik.
- **`standard`** — PRD und Tech Spec ab Überschreiten der Schwelle, Plan vor Tasks, AC-IDs, Gate 2 in CI. Das `.sdd`-Format ist optional, aber empfohlen, weil es Präzedenz maschinell auflösbar macht.
- **`regulated`** — wie `standard`, plus ADR-Pflicht für jede Architekturänderung, Review-Gate, Archivpflicht, Security-Abschnitt in der Verfassung.

Die Profile unterscheiden sich **nicht** im Regelkern, sondern darin, welche Regeln aktiv sind. Jede Regel trägt im Katalog, ab welchem Profil sie gilt. Das verhindert zwei getrennte Prozesse, die auseinanderlaufen.

---

## 8. Architektur des Programms

### 8.1 Headless zuerst

```text
+-----------------------------+
|  sdd (CLI)                  |   CI, Skripte, Agenten, Menschen
+--------------+--------------+
               |
+--------------v--------------+
|  sddcore (Go-Bibliothek)    |   Regelkatalog, Analyse, Pläne, Diffs
|  - catalog                  |
|  - scan (Dateien, Git)      |
|  - rules                    |
|  - baseline                 |
|  - plan / apply             |
|  - report                   |
+--------------+--------------+
               |
     +---------+---------+
     |                   |
+----v-----+      +------v------+
| sdd ui   |      | Optional:   |
| (embed)  |      | Suchindex   |
+----------+      +-----------+
```

Begründung: Ein Gate, das nur in einer GUI existiert, kann nicht in CI laufen — und ist damit kein Gate. Die Bibliothek ist die Wahrheit, die Oberfläche eine Ansicht.

### 8.2 Dateien sind der einzige Zustand

- Es gibt **keine Datenbank als Wahrheit**. Kein verstecktes Projektgedächtnis, keine Sitzungsdatei, die zählt.
- Der gesamte Zustand liegt in versionierten Dateien: Verfassung, Artefakte, Projektkonfiguration, Baseline.
- Das Programm kann jederzeit gelöscht werden, ohne dass ein Projekt unbrauchbar wird.
- Ein Cache ist erlaubt, aber wegwerfbar und nie autoritativ.

Das ist dieselbe Regel, die das Referenzprojekt für seinen Markdown-Vault zieht: Quellen sind die Wahrheit, Indizes sind materialisiert. Ein Werkzeug, das „seine" Projekte in einer eigenen Datenbank hält, wäre ein zweiter Wahrheitsanspruch.

Die Baseline ist die einzige Ausnahme, die erklärungsbedürftig ist: Sie ist Zustand, aber **versionierter, überprüfter Zustand** — sie steht im Review und hat eine eigene Änderungsregel (Abschnitt 5.2).

### 8.3 Projektkonfiguration

Eine Datei im Wurzelverzeichnis, Vorschlag `sdd.config.yaml`. Damit die Größenschwelle (`SDD-020`, `SDD-030`) als `error` deterministisch prüfbar ist, werden abstrakte Begriffe auf konkrete Pfad- und Layer-Globs des jeweiligen Projekts abgebildet:

```yaml
version: 1
profile: standard
language: de

# Basis für `git diff <base>...HEAD` in Gate 2 und SDD-024.
# Leer = automatisch ermitteln: in CI $GITHUB_BASE_REF (bzw. $CI_MERGE_REQUEST_TARGET_BRANCH_NAME),
# lokal der Upstream des Branches (`@{upstream}`), sonst `main`, sonst `master`.
base_ref: ""

artifacts:
  constitution: md-indexer.sdd
  adr: docs/architecture/adr
  # Unterstützt sowohl getrennte Ordner (docs/prd, docs/design, docs/plans)
  # als auch bestehende Spec-Kit-Strukturen (specs/features/<slug>/{spec,plan,tasks}.md):
  prd: docs/prd
  design: docs/design
  plans: docs/plans
  archive: docs/archive
  # Doku/Exploration (Artefakttyp 6). Diese Pfade lösen KEINE Feature-Prüfung aus.
  docs:
    - docs/notes
    - docs/research

tests:
  # Prüfbereich für SDD-021/SDD-022 (hier im Beispiel ein Go-Projekt).
  # Ohne Angabe gilt die im Handbuch definierte Fallback-Liste (Abschnitt 9.5).
  include:
    - "**/*_test.go"

thresholds:
  # Beispielhafte Pfad- und Layer-Globs (hier anhand eines Go-Repositories;
  # für Python/TypeScript/Rust/Java frei auf die jeweilige Ordnerstruktur anpassbar):
  layers:
    - internal/watcher/**
    - internal/parser/**
    - internal/repository/**
    - internal/search/**
    - internal/graph/**
    - internal/ai/**
    - internal/api/**
  # Die Schwelle ist überschritten, wenn EINE der Bedingungen zutrifft:
  #   (touched_paths trifft zu) ODER (mindestens min_layers_touched Layer berührt)
  # Diese Verknüpfung ist bewusst disjunktiv: ein Treffer in openapi.yaml
  # allein ist bereits eine öffentlichen Schnittstellenänderung.
  feature_artifacts_required_when:
    min_layers_touched: 2
    touched_paths:
      - "openapi.yaml"
      - "cmd/**"
      - "internal/repository/sqlite.go"
      - "*.sdd"
      - "go.mod"

gates:
  # Der Katalog definiert je Profil die Standardzugehörigkeit (gate_default).
  # Ein override ersetzt diese Liste für das jeweilige Gate vollständig.
  # Überschneidungen mit gate_default sind zulässig; maßgeblich ist
  # ausschließlich die effektive Menge Eff(g) (Abschnitt 9.5, SDD-072).
  pre_implementation:
    override: [SDD-001, SDD-002, SDD-010, SDD-020, SDD-030]
  pre_merge:
    override: [SDD-001, SDD-002, SDD-010, SDD-020, SDD-021, SDD-022, SDD-030, SDD-041]
  pre_release:
    override: [SDD-050, SDD-051]

exclude:
  - vendor/**
  - node_modules/**
```

Damit sind alle Pfade und Schwellen deklarativ konfigurierbar. Das Programm erzwingt keine bestimmte Ordnerstruktur (und kollidiert nicht mit vorhandenen Frameworks wie GitHub Spec Kit, MADR oder arc42), sondern prüft die konfigurierte Zuordnung.

**Gate-Zugehörigkeit hat genau eine Quelle.** Maßgeblich ist die effektive Menge `Eff(g)`: der `override` des Gates, falls gesetzt, sonst `gate_default` aus dem Katalog — nie eine Vereinigung beider. `sdd lint` meldet über SDD-072 jede aktive Regel, die in keinem `Eff(g)` vorkommt; sonst ist eine Regel unsichtbar und läuft nie.

**Ohne `sdd.config.yaml`** bricht jeder Befehl mit Exit-Code `2` und der Meldung „keine Projektkonfiguration gefunden; `sdd adopt` oder `sdd init` ausführen" ab. Es gibt keine stillschweigenden Defaults — ein Prüfer, der ohne Konfiguration rät, prüft das falsche Projekt. Ausnahmen: `sdd init`, `sdd adopt` und `sdd new` legen die Datei selbst an bzw. funktionieren mit einem Vorschlag.

### 8.4 Technologie

- **Kern:** Go. Ein Binary, keine Laufzeitabhängigkeit, gut in CI und auf allen drei Plattformen.
- **Grafische Oberfläche (`sdd ui` vs. Tauri):**
  - *Bevorzugte schlanke Variante:* Das Go-Binary liefert über `sdd ui` (gebunden an `127.0.0.1`) ein per `embed.FS` direkt einkompiliertes Web-Frontend im Browser aus. Das spart die komplette zweite Rust/Tauri-Toolchain und das Sidecar-Management.
  - *Optionale Desktop-Hülle:* Tauri 2 + Angular (wie im Referenzprojekt vorhanden), falls eine native Desktop-Paketierung gewünscht ist.
- **Kein Nachbau eines Volltext-Editors:** Statt eine eigene IDE zu bauen, erzeugt das Werkzeug neue Artefakte aus Templates und öffnet bestehende Dateien bei Klick auf einen Befund direkt an der richtigen Zeile im System-/IDE-Editor (`$EDITOR`, `vscode://file/...`, `obsidian://...`), während `sdd lint --watch` im Hintergrund live prüft.
- **Git-Zugriff:** über die Git-CLI, nicht über eine eingebettete Implementierung. Weniger Code, weniger Überraschungen.
- **Optionaler Index:** Anbindung an einen vorhandenen Suchindex über eine schmale Schnittstelle. Keine Abhängigkeit von dessen Interna, damit das Werkzeug in jedem Projekt läuft.

### 8.5 Verhältnis zu md-indexer / md-ai

Das Werkzeug muss in **jedem** Repository funktionieren, auch ohne md-indexer. Deshalb:

- Eigenes Repository für `sdd` / `sddcore`, keine Import-Abhängigkeit von `internal/…` des Referenzprojekts (`md-indexer` dient als erstes Pilot- und Referenzprojekt).
- Optionale Kopplung ausschließlich über eine stabile Schnittstelle: die CLI `md-ai ask --json` oder `md-api`. Beides existiert bereits.
- Wenn ein Projekt einen md-indexer-Index hat, kann das Werkzeug ihn für „wo ist X dokumentiert" nutzen — muss aber ohne ihn vollständig funktionieren.

### 8.6 Sicherheit der lokalen Oberfläche

`sdd ui` ist ein lokal laufender HTTP-Server mit Schreibrecht auf Projektdateien. Diese Angriffsfläche existiert bei der CLI nicht und braucht eigene Regeln — dieselbe Sorgfalt, die das Referenzprojekt für `md-api` verlangt.

- **Bindung ausschließlich an `127.0.0.1`**, nicht an `localhost` (dual-stack kann auf `::1` und `0.0.0.0` auflösen).
- **Kein `Access-Control-Allow-Origin: *`.** Schreibende Endpunkte prüfen `Host` und `Origin` gegen die erwartete lokale Adresse — ein beliebiger Browser-Tab kann sonst den Port erreichen.
- **Sitzungstoken für Schreiboperationen**, beim Start erzeugt und im ausgegebenen Link mitgeführt. Lesen darf ohne Token möglich sein, Schreiben nie.
- **Schreiben nur über den Plan/Apply-Pfad** derselben Bibliothek, die auch `sdd adopt` nutzt. Kein Dateizugriff direkt aus einem HTTP-Handler.
- **Kein Netzwerkzugriff nach außen.** Die Oberfläche lädt keine externen Schriften, Skripte oder Telemetrie.

---

## 9. Regelkatalog und Mechanismen

### 9.1 Aufbau einer Regel

```yaml
- id: SDD-050
  title: Abgeschlossene Features sind vor dem Release archiviert
  severity:                 # ein Wert oder je Gate; default gilt für `sdd lint` ohne --gate
    default: warn
    by_gate:
      pre-release: error
  profile: standard
  check: archive_state
  gate_default: [pre-merge, pre-release]
  baseline: forbidden
  contract: handbuch/03-mechanismen.md#sdd-050-vertrag
  rationale: >
    Ein Gate nach dem Release blockiert nichts.
    Geprüft wird daher vor dem Release-Tag.
  handbook: handbuch/03-mechanismen.md#sdd-050
```

Zwei Punkte am Schema, die nicht verhandelbar sind:

- **`severity` ist immer ein Objekt**, auch bei einem einzigen Wert (`severity: {default: error}`). Die kürzere Schreibweise `severity: error` ist erlaubte Syntax und wird als `{default: error}` gelesen. Damit gibt es genau eine Quelle für den Schweregrad — die in §9.5 beschriebene Regel „`lint` ohne Gate nimmt `pre-merge`" ist damit hinfällig und durch `default` ersetzt.
- **`gate_default` und `override` sind verschieden.** `gate_default` ist die Aussage des Katalogs, `override` die des Projekts. `Eff(g)` liest nur eine von beiden.

`check` benennt eine Implementierung, nicht eine Absicht. Eine Regel ohne Implementierung ist Dokumentation — und fällt durch SDD-071 auf. Eine Regel ohne `contract` ist ein Versprechen ohne Verfahren und fällt durch SDD-073 (Abschnitt 9.5).

### 9.2 Auszug des Katalogs

- **SDD-001** Verfassung vorhanden — *error* — `baseline: allowed`
- **SDD-002** Verfassung erfüllt ihr Strukturschema; der Modus wird an der Endung entschieden: `.sdd` ⇒ 002b, jede andere Endung ⇒ 002a — *error*
  - **SDD-002a** (Markdown-Verfassung in `minimal`/`standard`): Pflichtabschnitte (Architektur/Layering, Security, Strikte Regeln) vorhanden
  - **SDD-002b** (`.sdd`-Verfassung im strikten Modus): erfüllt die formale `.sdd`-Grammatik (`Spec: Name`, bekannte Section-Labels, 2-Space-Einrückung)
- **SDD-003** Verfassung enthält keine Feature-Inhalte (User Stories, Sprintplanung, Tasklisten) — *warn*, heuristisch
- **SDD-010** Änderung einer Verfassungsregel verweist auf ein ADR — *error* — `baseline: allowed`
- **SDD-011** ADRs sind unveränderlich; Ablösung über `supersedes` — *error* — `baseline: forbidden` (quittiert wird über das ADR-Lockfile, nicht über die Baseline)
- **SDD-020** PRD vorhanden, wenn die Größenschwelle greift — *error* — `baseline: allowed`
- **SDD-021** Jedes Akzeptanzkriterium hat eine stabile ID (`AC-n`) — *error* — `baseline: allowed`
- **SDD-022** Jedes Akzeptanzkriterium ist in mindestens einer Testdatei referenziert — *error* — `baseline: allowed`. **Misst Präsenz, nicht Coverage** (Abschnitt 9.5).
- **SDD-023** PRD nennt Nicht-Ziele — *warn*
- **SDD-024** In `pre-merge` weicht der Diff nicht von der im Feature-Frontmatter (`touches`) erklärten Absicht ab — *warn*
- **SDD-025** Jede AC-Kennung steht in einer Testdeklaration oder ihrem Body (nicht nur irgendwo in der Datei) — *warn*, sprachadaptiv. Ergänzt SDD-022 um die Aussage, die der sprachneutrale Textscan nicht treffen kann.
- **SDD-030** Tech Spec vorhanden, wenn die Größenschwelle greift — *error* — `baseline: allowed`
- **SDD-031** Tech Spec referenziert die AC-IDs, die sie umsetzt — *warn*
- **SDD-040** Plan existiert vor der ersten Task — *warn*
- **SDD-041** Jede Task referenziert mindestens ein AC — *error* im Profil `standard` — `baseline: allowed`
- **SDD-050** Abgeschlossene Features (`status: implemented` oder `archive-after` überschritten) sind vor dem Release nach `archive/` verschoben und tragen `status: archived` — *warn* (default), *error* in `pre-release` — `baseline: forbidden`
- **SDD-051** Aktive **Feature-Artefakte** verweisen nicht auf archivierte Feature-Artefakte — *warn* — `baseline: forbidden`. Nicht betroffen: ADRs, Verfassung und Runbooks, die historische Dokumente zitieren dürfen, weil genau das ihr Zweck ist.
- **SDD-060** Widersprüche zwischen Verfassung und Tech Spec — *info*, assistiert, nicht automatisch entscheidbar
- **SDD-070** Jede Regel ist im Handbuch dokumentiert — *error* — `baseline: forbidden`
- **SDD-071** Jede Regel ist implementiert — *error* — `baseline: forbidden`
- **SDD-072** Jede aktive Regel ist in mindestens einem Gate aktiv — *error* — `baseline: forbidden`
- **SDD-073** Jede `error`-Regel hat einen Prüfvertrag (Abschnitt 9.5) — *error* — `baseline: forbidden`

Der Katalog ist ein Auszug, nicht der Endstand. Entscheidend ist die Disziplin: **keine Regel ohne Prüfung, keine Prüfung ohne Vertrag, keine heuristische Regel als `error`, keine `error`-Regel ohne Determinismus, keine Regel ohne v1-Entscheidung.**

### 9.3 Schweregrade und ihre Bedeutung

- **error** — blockiert das Gate. Nur deterministisch prüfbar, mit Prüfvertrag.
- **warn** — erscheint im Befund, blockiert nicht. Heuristiken leben ausschließlich hier.
- **info** — Hinweis für den Menschen, nie blockierend.

Der Schweregrad ist **gate-abhängig definierbar** (`severity.default` + `severity.by_gate`, §9.1). Gibt es keinen Eintrag für das laufende Gate und keinen `default`, ist die Regel in diesem Gate nicht aktiv — nicht etwa `error`. Diese Regel schützt davor, dass ein fehlendes Feld versehentlich blockiert.

Diese Trennung ist der Schutz gegen die häufigste Werkzeugkrankheit: ein Gate, das wegen eines Fehlalarms abgeschaltet wird, schützt nichts mehr.

### 9.4 Status-, Datums- und Schwellenvokabular

Mehrere Regeln hängen an Feldern, die heute weder definiert noch erzeugt werden. Beides wird verbindlich festgelegt, sonst ist die Regel nicht prüfbar.

**Statuswerte — je Artefakttyp, nicht global:**

```text
Feature-Artefakte (PRD, Tech Spec, Plan):
  draft        angelegt, noch nicht abgestimmt
  active       gültig, in Arbeit
  implemented  umgesetzt, noch nicht archiviert
  archived     nicht mehr gültig, Historie
  superseded   durch ein neueres Dokument abgelöst

ADR:
  proposed     vorgeschlagen, nicht entschieden
  accepted     entschieden und gültig
  rejected     verworfen, bleibt als Historie
  deprecated   galt, gilt nicht mehr
  superseded   durch ein neueres ADR abgelöst

Doku/Exploration:
  active
  archived

Verfassung, Architektur-, Security- und API-Doku, Runbooks:
  active
  archived
```

Ein Prüfer verwendet die Liste des jeweiligen Artefakttyps, nicht eine Vereinigungsmenge. `accepted` bei einem ADR und `active` bei einer Tech Spec sind beide korrekt; ein ADR mit `active` ist es nicht. `sdd new` setzt je Typ den passenden Startwert (`draft` für Feature-Artefakte, `proposed` für ADRs, `active` für Doku).

**Datumsfeld in Feature-Artefakten:**

```yaml
status: active
archive-after: 2026-12-31   # ausschließlich ISO-8601 (YYYY-MM-DD)
touches:                    # erklärte Änderungsabsicht für Gate 1 und SDD-024
  - openapi.yaml
  - internal/service/**
```

Ein Git-Tag ist **kein** gültiges `archive-after`. Ein Tag ist ein Label, kein Datum und ohne `git describe` nicht mit „heute" vergleichbar; beides zu vermischen macht die Regel unbestimmt. Zulässig ist nur ein Datum. Der Bezug zum Release entsteht über die Regel selbst: `pre-release` verlangt, dass kein Feature mehr `implemented` ist — unabhängig vom Datum. `sdd staleness` vergleicht `archive-after` mit dem aktuellen Datum.

**Pflicht:** `sdd new` erzeugt diese Felder. Ein Prüfwerkzeug, das Felder verlangt, die seine eigenen Templates nicht erzeugen, wäre inkonsistent — und SDD-050 nicht prüfbar.

**Größenschwelle (SDD-020/030), verbindlich:**

Zwei Quellen, zwei Gates, eine Logik:

- **Gate 1** wertet die erklärte Änderungsabsicht aus (`touches` im Feature-Frontmatter bzw. `--paths`, Abschnitt 5.1.2).
- **Gate 2** wertet den Diff gegen die Merge-Basis aus (`base_ref`, Abschnitt 8.3).

Die Schwelle ist überschritten, wenn `touched_paths` **oder** `min_layers_touched` zutrifft (disjunktive Verknüpfung, siehe Abschnitt 8.3). Der gesamte Arbeitsbaum ist nie die Grundlage — nur der Diff oder die erklärte Absicht.

### 9.5 Prüfverträge und Berichtsformat

Ein Werkzeug, dessen Kern CI und Agenten sind, braucht einen definierten Berichtsvertrag. „Maschinenlesbare Ausgabe" ist keine Spezifikation.

**Exit-Codes** (stabil, zugesichert):

```text
0  sauber, keine neuen error-Befunde
1  Befunde: mindestens ein neuer error
2  Ausführungsfehler (Konfiguration ungültig oder fehlend, Git nicht verfügbar,
   Baseline nicht lesbar, interner Fehler)
```

**Befundformat (JSON, `--json`):**

```json
{
  "schema_version": 1,
  "tool_version": "0.1.0",
  "catalog_version": "1.0.0",
  "profile": "standard",
  "gate": "pre-merge",
  "feature": "openrouter-provider",
  "baseline": {"applied": true, "suppressed": 42},
  "findings": [
    {
      "rule_id": "SDD-022",
      "severity": "error",
      "status": "new",
      "path": "docs/prd/openrouter-provider.md",
      "line": 37,
      "ordinal": 1,
      "message": "AC-3 wird von keinem Test referenziert",
      "handbook_url": "handbuch/03-mechanismen.md#sdd-022"
    }
  ],
  "summary": {"error": 1, "warn": 4, "info": 12, "suppressed": 42}
}
```

`status` ist `new`, `baselined` oder `resolved`. Das Schema ist versioniert; Änderungen daran folgen derselben Regel wie Katalogänderungen (Major-Version plus Migrationshinweis).

**Testdatei-Fallback-Liste (einmal, hier maßgeblich):**

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

Diese Liste steht **nur hier** und im Handbuch; andere Abschnitte verweisen darauf, statt sie zu wiederholen (Abschnitt 8.3 nennt bewusst nur den Verweis). Zwei divergierende Listen wären derselbe Drift, den das Konzept verhindern soll.

**Prüfverträge — vollständig für alle `error`-Regeln:**

- **SDD-001:** Die in `artifacts.constitution` konfigurierte Datei existiert und ist nicht leer. Kein Inhaltstest — das ist SDD-002.
- **SDD-002a:** Die Verfassungsdatei endet nicht auf `.sdd` und enthält mindestens die Überschriften, die das Profil verlangt (Standard: Architektur/Layering, Security, Strikte Regeln). Vergleich normalisiert (Groß-/Kleinschreibung, führende `#`-Zeichen, Leerraum).
- **SDD-002b:** Die Verfassungsdatei endet auf `.sdd` und erfüllt die formale Grammatik: erste nicht-leere, nicht-kommentierte Zeile (Kommentar: `#` oder `//`) ist `Spec: <Name>`; jede nicht eingerückte Zeile danach ist ein Section-Label `Label:` aus der bekannten Liste; Einrückung ist ein Vielfaches von zwei Leerzeichen; Tabs kommen nicht vor. Die bekannte Liste ist standardmäßig `Architecture`, `Security`, `Data`, `Standards`, `References`, `Owns`, `Can modify` (Abschnitt 10 von `sdd_process_3` plus SpecDD-Eigentum) und wird je Projekt über `constitution_sections` in `sdd.config.yaml` ersetzt.
- **SDD-010:** Eine Änderung an der Verfassungsdatei im Diff erfordert mindestens eine neue oder geänderte ADR-Datei im selben Diff, die die Verfassungsdatei im Feld `references` oder `adr` nennt. Ausnahme ist deterministisch definiert: Ist der Diff der Verfassungsdatei nach Normalisierung (Zeilenenden vereinheitlicht, Zeilen mit ausschließlich Leerraum entfernt, Folgen von Leerraum auf ein Leerzeichen reduziert) identisch mit der Basisversion, gilt die Änderung als Formatierung und es wird kein ADR verlangt.
- **SDD-011:** Ein Lockfile `.sdd/adr-lock.json` speichert je entschiedenem ADR (`status` nicht `proposed`) einen `content_hash` (SHA-256 über den normalisierten Text **ohne Frontmatter**, damit Statuswechsel und neue `references` möglich bleiben). Weicht der Hash einer gelockten ADR ab, ist das ein `error`; ein entschiedenes, noch nicht gelocktes ADR ist ein `warn`. Das Lockfile schreiben ausschließlich `sdd baseline` (neue ADRs) und `sdd adopt` — nie `lint` oder `gate`, denn Prüfen ist lesend. `sdd baseline --accept SDD-011` ist der einzige Weg, eine bestehende Abweichung zu quittieren, und erzeugt eine sichtbare Änderung im Review. Gelöschte ADRs bleiben im Lockfile (Historie).
- **SDD-020:** Die Größenschwelle greift und unter `artifacts.prd` existiert keine Datei, deren Frontmatter `feature: <slug>` den aufgelösten Slug nennt. Dateien ohne `feature:`-Feld zählen nicht — sonst wäre jede beliebige Notiz im Ordner ein PRD.
- **SDD-021:** Geprüft werden die aktiven PRDs unter `artifacts.prd` (Status weder `archived` noch `superseded`); Checkbox-Zeilen in Plänen sind Tasks und fallen unter SDD-041. Für jede Zeile, die als Akzeptanzkriterium erkannt wird (Checkbox-Zeile `- [ ]` / `- [x]`, Aufzählungszeile mit `AC-`, oder Zeile unter einer Überschrift, die `Akzeptanzkriterien` / `Acceptance Criteria` enthält), existiert eine Kennung `AC-<n>` mit `\bAC-[1-9][0-9]*\b`.
- **SDD-022:** Eine AC-Kennung ist `AC-<n>`, `<n>` eine positive Ganzzahl ohne führende Null (`\bAC-[1-9][0-9]*\b`), definiert in den aktiven PRDs unter `artifacts.prd`. Sie gilt als referenziert, wenn in mindestens einer Datei aus `tests.include` (oder der Fallback-Liste oben) eines von drei Mustern mit genau dieser Nummer vorkommt: `\bAC-<n>`; `AC` in Großbuchstaben, optional `_`, dann `<n>`, davor kein Großbuchstabe und keine Ziffer (`TestAC3_Login`); `ac` in beliebiger Schreibweise, optional `_`, dann `<n>`, davor weder Buchstabe noch Ziffer (`test_ac_3`). Ein `\b`-basiertes Muster wie `(?i)\bAC_?<n>\b` reicht dafür nicht, weil `\b` CamelCase- und snake_case-Teile nicht trennt. Kennt das Repository mehr als ein Feature (archivierte eingeschlossen), zählt für ein PRD nur eine Zeile einer Testdatei, die zusätzlich dessen Feature-Slug als ganzes Token nennt (Konvention `// <slug> AC-<n>`); sonst würde ein einziger Test `AC-1` in jedem PRD erfüllen — auch die Tests eines längst archivierten Features. Die Referenz zählt im Testnamen, im Beschreibungsstring oder in einem Kommentar beliebiger Sprachsyntax (`//`, `#`, `/* */`, `--`). Prüfbereich: das ganze Repository, exklusive `exclude`-Globs. Trifft die Glob-Liste (`tests.include` oder Fallback) keine einzige Datei, meldet die Regel genau ein `warn` statt Einzelbefunden — ein fehlender Testpfad ist ein Konfigurationsproblem, kein Projektverstoß. **Der Vertrag misst Präsenz, nicht Coverage.** Ein `// TODO: AC-3 testen` erfüllt ihn. Das ist der bewusste Preis für Sprachneutralität ohne Parser; SDD-025 liefert die schärfere Aussage als `warn`.
- **SDD-025:** In einer Datei aus `tests.include` muss die AC-Kennung innerhalb des Textbereichs einer Testdeklaration stehen. Als Deklaration gelten sprachadaptive Muster: `func Test…`, `def test_…`, `it(`, `test(`, `describe(`, `#[test]`, `@Test`, `TEST(`, `TEST_F(`. Der Bereich reicht vom Kommentar- oder Attributblock unmittelbar über der Deklaration (ohne Leerzeile dazwischen, wie ein Go-Doc-Kommentar oder Python-Decorator) bis zur nächsten Deklaration gleicher oder geringerer Einrückungsebene oder zum Dateiende. Freistehende Kommentare, durch eine Leerzeile getrennt, zählen nicht mit. Geprüft werden nur Kennungen, die SDD-022 als referenziert anerkennt.
- **SDD-030:** Wie SDD-020, mit `artifacts.design` statt `artifacts.prd`.
- **SDD-041:** Als Task gilt eine Checkbox-Zeile (`- [ ]`, `- [x]`, `* [ ]`, `1. [ ]`) in einer Datei unter `artifacts.plans` oder in `tasks.md` eines Feature-Verzeichnisses. Jede solche Zeile muss im selben Zeilenbereich (Zeile oder Fortsetzungszeilen bis zur nächsten Listenebene) eine AC-Kennung enthalten.
- **SDD-050:** Ein Feature mit `status: implemented` (oder einem `archive-after` in der Vergangenheit bzw. einem `archive-after`, das kein ISO-8601-Datum ist) außerhalb von `archive/` erzeugt — ebenso ein Feature-Artefakt unter `archive/`, dessen Status weder `archived` noch `superseded` ist — `warn` im `default`-Gate und `error` in `pre-release`. Die Regel ist nicht baselinet-fähig: Im Release-Vorbereitungs-Commit (zusammen mit dem Changelog-Update unmittelbar vor dem Git-Tag) werden alle `implemented`-Features nach `archive/` verschoben und auf `status: archived` gesetzt. Für den Altbestand eines übernommenen Projekts bedeutet das einen einmaligen, im Adopt-Bericht ausgewiesenen Aufräumschritt (Abschnitt 5.2).
- **SDD-070:** Für jede Regel-ID im Katalog existiert im Handbuch eine Überschrift, die die ID enthält, und die Datei unter `handbook` existiert.
- **SDD-071:** Für jede Regel-ID im Katalog ist ein `check`-Name eingetragen und im Prüfkern registriert (Registry-Lookup zur Laufzeit, nicht Statustext).
- **SDD-072:** Für jedes Gate $g \in \{\text{pre\_implementation}, \text{pre\_merge}, \text{pre\_release}\}$ ist die effektive Regelmenge $\text{Eff}(g) = \text{override}(g)$ (sofern in `sdd.config.yaml` gesetzt), andernfalls $\text{gate\_default}(g)$ aus dem Katalog. Jede im gewählten Profil aktive Regel muss in der Vereinigungsmenge $\bigcup_g \text{Eff}(g)$ enthalten sein; andernfalls wird ein Befund gemeldet.
- **SDD-073:** Für jede Regel mit `severity.default: error` oder mit mindestens einem `severity.by_gate`-Eintrag gleich `error` ist `contract` nicht leer gesetzt und verweist auf einen existierenden Anker im Handbuch.

### 9.6 KI-Assistenz

`sdd_process_3` verlangt Mechanismen statt Disziplin. Ein LLM ist kein Mechanismus — es ist ein Vorschlagswesen. Deshalb gilt seit [ADR 0002](adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md):

- **`sdd` bleibt KI-frei und offline.** Kern, CLI, `sdd ui` und `sdd mcp` enthalten keinen LLM-Code und bauen keine ausgehenden Verbindungen auf.
- **KI-Assistenz ist ein getrenntes, optionales Programm** (`sdd-desk`): Es führt durch den Prozess — von der Produkt-Idee über PRD, Tech Spec, ADR und Plan bis zu Tests, Gate und Archiv —, stellt Fragen und formuliert Entwürfe.
- **Modellwahl:** lokales Ollama oder ein Cloud-Anbieter (Ollama Cloud, OpenRouter, NVIDIA NIM, Claude, Gemini), konfiguriert in `config/llm.env` im Programmverzeichnis des Desktops. Inhalte gehen ungeschwärzt an den gewählten Anbieter; der Desktop zeigt das vor dem ersten Aufruf an.
- **Werkzeuge nur mit Erlaubnis:** Der Desktop führt `sdd`-Werkzeuge über `sdd mcp` aus — jeder Aufruf einzeln bestätigt oder für die Sitzung freigegeben, schreibende nur nach Vorschau.
- **Die KI entscheidet nichts, was ein Gate bestehen lässt.** Sie formuliert; abnehmen muss der Mensch.

**Wichtig für das Ratchet:** Eine KI darf niemals Baseline-Einträge erzeugen, entfernen oder begründen. Sonst wäre die einzige Bremse im System maschinell verhandelbar.

---

## 10. Die grafische Oberfläche (`sdd ui` / Desktop)

Fünf Ansichten, mehr braucht es nicht:

- **Projektübersicht** — Reifegrad, offene Befunde nach Schwere, Schuldenstand (baselinete Befunde und Trend), letzte Änderungen, Status der drei Gates.
- **Artefaktbaum** — Verfassung, ADRs, PRDs, Specs, Pläne, Doku/Exploration, Archiv; je Eintrag Typ, Status, Alter und Archivfrist.
- **Befundliste** — Ort, Regel-ID, Schweregrad, Erklärung, Link in den Handbuchabschnitt sowie Deep-Link zum Öffnen in der eigenen IDE (`$EDITOR` / `vscode://` / `obsidian://`). Filter für „neu" und „Altbestand". Sortierbar, exportierbar.
- **Template-Generator & Live-Vorschau** — neue Artefakte fehlerfrei aus Template anlegen (inklusive typspezifischer Status- und Archivfelder), Live-Prüfung im Watch-Modus bei Änderungen im externen Editor, Verweise als Link.
- **Migrationsassistent** — der Adopt-Ablauf aus Abschnitt 6, mit Probelauf und Diff vor jeder Änderung.

Bewusste Beschränkungen:

- **Ein-Klick-Korrekturen nur additiv.** Fehlende Datei aus Template anlegen ja; Verschieben oder Löschen nie ohne Bestätigung mit Diff.
- **Keine versteckten Schreibvorgänge.** Jede Änderung zeigt vorher, was sie tut.
- **Keine Baseline-Bearbeitung in der Oberfläche.** Einträge entstehen nur über `sdd baseline --accept` in der CLI, mit Commit und Begründung.
- **Tastatur zuerst.** Befehlspalette, keine Menüschlacht.
- **Alles, was die Oberfläche kann, kann die CLI auch.** Keine exklusiven Funktionen.

---

## 11. Roadmap

- **Schritt 1 — Katalog, Lint und Baseline.** Regelkatalog als YAML **mit Prüfvertrag und Handbuchabschnitt je Regel im selben Commit**, `sdd lint` (inkl. `--watch` und `--json`), `sdd baseline` mit Gate-Bindung und `baseline: forbidden`, Berichtsformat v1. Ergebnis: ein Projekt kann geprüft werden, ohne dass Altbestand das Gate blockiert.
- **Schritt 2 — Init, New und Gate.** `sdd init` für die drei Profile, `sdd new` mit typspezifischen Status- und Archivfeldern, `sdd gate` (`pre-implementation` mit erklärter Absicht, `pre-merge`, `pre-release`), GitHub Action. Ergebnis: neue Projekte (und das Pilotprojekt `md-indexer`) sind ab Tag eins in CI prüfbar.
- **Schritt 3 — Adopt.** Inventar, Reifegrad, Baseline, Probelauf, Branch-Anwendung, ausgewiesener Archivierungsschritt. Ergebnis: bestehende Projekte sind hebbar, ohne Umbruch.
- **Schritt 4 — Handbuch v1.** Rezepte und Agententeil ergänzt, Coverage-Check (`SDD-070`/`071`/`072`/`073`) über den gesamten v1-Katalog aktiv. Ergebnis: das Handbuch ist prüfbar vollständig.
- **Schritt 5 — Grafische Oberfläche.** Die fünf Ansichten auf der fertigen Bibliothek (zuerst als eingebettetes `sdd ui` im Go-Binary, bei Bedarf als Tauri-App). Ergebnis: tägliche Arbeit ohne Kommandozeile.
- **Schritt 6 — Ausbau.** Index-Anbindung (`md-api` / `md-ai`), Agenten-Schnittstelle (MCP), KI-Assistenz, `sdd staleness`.

Die Reihenfolge ist nicht verhandelbar: Die Oberfläche kommt nach dem Kern. Eine App ohne Regelkern wäre eine hübsche Hülle um Nichts. Und die Baseline kommt vor dem ersten Gate, nicht danach.

---

## 12. Risiken und Gegenmaßnahmen

- **Zweite Wahrheit durch Werkzeugzustand** → Dateien sind der einzige Zustand; die Baseline ist versioniert und hat eine eigene Änderungsregel.
- **Zeremonie-Ausweitung** → Profile, deterministische Pfad-/Layer-Schwelle, Gate nur bei Überschreitung, v1-Schnitt.
- **Handbuch veraltet** → Coverage-Check SDD-070 (mit 071–073), Regel und Abschnitt im selben Commit.
- **Gates laufen nur lokal** → headless zuerst, CI-Wrapper ab Schritt 2.
- **Fehlalarme machen das Gate unbrauchbar** → Heuristiken nur `warn`, `error` nur mit Prüfvertrag, Altbestand über Baseline, fehlende Gate-Zuordnung blockiert nicht.
- **Baseline wird zur Decke** → Einträge gate-gebunden und nur schrumpfend, `content_hash` + deterministisches `ordinal` pro Fundstelle, `baseline: forbidden` für Lebenszyklus- und Selbstprüfungsregeln, Schuldenstand im Bericht, keine Bearbeitung über Oberfläche oder KI.
- **Stilles Durchwinken bei fehlender Feature-Zuordnung** → zwei getrennte Zustände: unbekannte Absicht = `warn`, erklärte Absicht ohne Artefakt = `error` (5.1.1).
- **Gate 1 prüft nichts, weil der Diff noch leer ist** → erklärte Änderungsabsicht, `warn` statt Erfolg bei fehlender Erklärung (5.1.2).
- **Katalog wächst unbegrenzt** → v1-Schnitt in Abschnitt 15, Regeln außerhalb des Schnitts sind nicht im Katalog.
- **Prüfer rät ohne Konfiguration** → Exit 2 ohne `sdd.config.yaml` (8.3).
- **Lock-in** → Artefakte sind Plaintext; ohne das Werkzeug bleibt alles lesbar und gültig.
- **KI schreibt Dokumente, die niemand liest** → Vorschlag statt Schreiben, Standard aus, kein Zugriff auf die Baseline.
- **Scope-Explosion zum Projektmanagement oder zweiten Texteditor** → Nicht-Ziele in Abschnitt 3, Deep-Link in `$EDITOR` statt eigenem IDE-Nachbau.
- **Lokaler HTTP-Server mit Schreibrecht** → Bindung an `127.0.0.1`, Origin-Prüfung, Sitzungstoken, Schreiben nur über Plan/Apply (8.6).
- **Adopt beschädigt einen Bestand** → Probelauf, Branch, ein Commit pro Schritt, nichts löschen, nichts überschreiben.

---

## 13. Erfolgskriterien

- Ein neues Projekt erreicht Reifegrad L2 in unter zehn Minuten.
- Ein bestehendes Projekt lässt sich ohne Datenverlust eine Stufe heben, mit überprüfbarem Bericht und sinkendem Schuldenstand.
- Der erste Gate-Lauf auf einem gewachsenen Projekt meldet **null neue `error`-Befunde** für Bestandsarbeit in `pre-implementation` und `pre-merge`. In `pre-release` darf einmalig Archivierungsaufwand erscheinen — ausgewiesen, nicht unterdrückt.
- Jede Regel im v1-Katalog hat genau eine Implementierung, einen Prüfvertrag und einen Handbuchabschnitt — maschinell geprüft, nicht behauptet.
- Das Werkzeug ist entfernbar: kein Projekt verliert durch seine Löschung Funktionsfähigkeit.
- Ein neuer Mitwirkender beantwortet die drei Fragen — *was gilt dauerhaft, was ist Planung, was ist veraltet* — in unter fünf Minuten aus dem Bestand.
- Die Gate-Laufzeit bleibt unter dem, was Menschen freiwillig abwarten.

---

## 14. Getroffene Richtungsentscheidungen

- **Eigenes Repository statt Unterordner in `md-indexer`:** `sdd` lebt in einem eigenen Repository (`github.com/Zheng-Bote/sdd_studio`), damit es sprach- und projektübergreifend in CI einsetzbar ist. `md-indexer` ist das erste Pilotprojekt (Ablösung der inaktiven `.github/workflows/ci.yml_` durch eine echte Pipeline mit `sdd gate`).
- **Name, CLI und Paketstruktur:** Das Projekt heißt **SDD Studio** (`sdd_studio`), das CLI-Binary heißt **`sdd`** (`cmd/sdd`), die Kernbibliothek liegt unter **`internal/sddcore`**.
- **Trennung von Konfiguration und Werkzeug-Zustand:** Die menschengeschriebene Projektkonfiguration liegt sichtbar im Repo-Root als **`sdd.config.yaml`**. Maschinenverwaltete, aber in Git mitversionierte Zustandsdateien liegen gebündelt unter **`.sdd/`** (`.sdd/baseline.json` sowie ab v1.1 `.sdd/adr-lock.json`).
- **Handbuch als eingebettetes Markdown (`handbuch/`):** Für v1 wird das Handbuch ausschließlich als Markdown im Repository gepflegt und zusammen mit `catalog.yaml` per `embed.FS` in das Go-Binary eingebettet. Damit funktionieren `SDD-070`/`SDD-073` und `sdd explain <ID>` ohne externe Dateipfade.
- **Tombstone-Prinzip für entfallene `AC-n`-Kennungen:** Entfällt ein Akzeptanzkriterium nachträglich, werden bestehende `AC-n`-Nummern niemals umnummeriert (Schutz von Git-Blame, PR-Historie und Testreferenzen). Durchgestrichene Kennungen (`~~AC-2~~`) gelten im PRD als entfallen und werden von `SDD-021`/`SDD-022` ignoriert.
- **Drei Profile (`minimal`, `standard`, `regulated`) als reine Katalog-Presets:** `regulated` bleibt als drittes Profil bestehen, enthält aber keine Sonderlogik im Code, sondern steuert ausschließlich über den Katalog, welche Regeln aktiv sind.
- **Baseline-Verhalten in CI und Ratchet ohne Ablaufdaten:** Neue, nicht durch `.sdd/baseline.json` abgedeckte `error`-Befunde lassen CI hart fehlschlagen (`Exit 1`). Für Baseline-Einträge gibt es in v1 keine künstlichen Ablaufdaten, sondern einen klaren Schuldenzähler mit Delta-Anzeige.
- **Koexistenz mit bestehenden Frameworks ohne Import-Konverter:** Über `artifacts` in `sdd.config.yaml` lassen sich vorhandene Strukturen (wie `specs/features/<slug>/` aus GitHub Spec Kit, MADR oder arc42) direkt einbinden, ohne funktionierende Ordner umzubenennen oder durch einmalige Konverter umzubauen.
- **Sechs Artefakttypen, nicht fünf:** Doku/Exploration als eigener Typ ohne Feature-Bindung. Ohne ihn müsste `adopt` in jedem echten Repository raten.
- **Adopt-Verhalten bei Konflikten:** Bestehende Dateien werden an Ort und Stelle belassen und im Bericht gemeldet (niemals überschrieben oder still dupliziert).
- **Gate 3 wird als repository-weites `pre-release`-Gate vorgezogen:** Anders als in `sdd_process_3` (Archivierung *nach* dem Release) prüft `pre-release` repository-weit *vor* dem Release-Tag, dass alle abgeschlossenen Features (`status: implemented`) im Release-Vorbereitungs-Commit nach `archive/` verschoben und auf `status: archived` gesetzt wurden. Damit enthält der getaggte Release-Commit bereits den aufgeräumten Bestand.
- **Baseline ist Pflichtbestandteil, nicht Komfort:** `sdd baseline` gehört zu P0 und in Roadmap-Schritt 1. Ohne sie ist das Hauptziel „bestehende Projekte heben" nicht erreichbar. Einträge sind an ein Gate gebunden und an `content_hash` + deterministisches `ordinal` identifiziert, nicht an Zeilennummern.
- **Archivierungs- und Selbstprüfungsregeln sind nicht baselinet-fähig:** `baseline: forbidden` für SDD-050, SDD-051 und SDD-070–073. Der Anspruch „getaggter Release-Stand ist aufgeräumt" bleibt damit wahr; der Altbestand erzeugt genau einen ausgewiesenen Aufräumschritt vor dem ersten Release.
- **Gate 1 arbeitet auf erklärter Änderungsabsicht (`touches` im Feature-Frontmatter bzw. `--paths`), nicht auf dem Diff:** Zum Zeitpunkt von Gate 1 existiert die implementierende Änderung meist noch nicht. Durch die Verortung im Feature-Frontmatter statt in `sdd.config.yaml` entstehen keine Merge-Konflikte bei parallelen Branches und die Absicht bleibt bis `pre-merge` (`SDD-024`) sowie im Archiv erhalten. Ohne erklärte Absicht meldet Gate 1 `warn` („Schwelle nicht bewertbar"), nie Erfolg.
- **Fail-closed gilt nur bei erklärter Absicht:** Unbekannte Absicht ist `warn`, erklärte Absicht ohne Artefakt ist `error` (5.1.1). Die frühere Formulierung war im Hauptfall nicht erreichbar, weil die Schwelle ohne Absicht gar nicht bewertbar ist.
- **Slug, nicht Pfad, ist die Feature-Identität:** Der Slug steht im `feature:`-Feld, im Ordner-/Dateinamen, im Branch-Namen und im Bericht; Abweichungen sind `warn`, nicht `error`.
- **Vollständige Programmiersprachen-Neutralität:** Das Werkzeug benötigt keine sprachspezifischen AST-Parser. Schwellen (`layers`, `touched_paths`) arbeiten auf Pfad-Globs, Test-Referenzen (`SDD-022`) auf Textmustern, und die schärfere Aussage (`SDD-025`) auf sprachadaptiven Deklarationsmustern — als `warn`.
- **SDD-022 misst Präsenz, nicht Coverage:** Der Vertrag ist ehrlich benannt. Die Aussage „der Test prüft wirklich dieses Kriterium" kann ein sprachneutraler Textscan nicht treffen; SDD-025 nähert sich ihr als `warn`.
- **Ein Schweregrad-Schema, nicht drei Aussagen:** `severity.default` + `severity.by_gate` im Katalog; `sdd lint` ohne `--gate` nutzt `default`. Fehlt beides, ist die Regel im Gate nicht aktiv.
- **Regel und Handbuchabschnitt entstehen im selben Commit:** Damit ist Dogfooding ab Schritt 1 möglich; ein `draft`-Status für Regeln entfällt. Das löst den Widerspruch, dass das Werkzeug sich ab Schritt 2 selbst hätte fehlschlagen lassen.
- **Gate-Zugehörigkeit hat eine Quelle:** Maßgeblich ist `Eff(g)` — `override` ersetzt, sonst `gate_default`. Überschneidungen sind zulässig, eine Vereinigung beider Listen findet nicht statt. SDD-072 prüft die Vereinigungsmenge über alle Gates.
- **Prüfvertrag ist Voraussetzung für `error`:** Ohne Vertrag höchstens `warn`. Durchgesetzt über SDD-073, und für **alle** `error`-Regeln des Katalogs in Abschnitt 9.5 hinterlegt.
- **Keine Heuristik in einer `error`-Regel:** Die SDD-010-Ausnahme ist als Whitespace-Normalisierung definiert; `archive-after` ist ausschließlich ISO-8601; Statuswerte gelten je Artefakttyp; Testdateien kommen aus `tests.include` mit einmal definierter Fallback-Liste und `warn` statt `error`, wenn kein Pfad greift.
- **Prüfen bleibt lesend, auch für das ADR-Lockfile:** `lint` und `gate` schreiben `.sdd/adr-lock.json` nie; neue ADRs lockt `sdd baseline`, Abweichungen quittiert nur `--accept SDD-011`. Gehasht wird der Text ohne Frontmatter, gelockt werden nur entschiedene ADRs.
- **Erstmaliges Anlegen der Verfassung verlangt kein ADR (SDD-010):** Sonst schlüge jeder PR aus `sdd init` oder `sdd adopt` fehl; ADR-pflichtig ist die Änderung einer bestehenden Regel.
- **AC-Verweise nennen den Feature-Slug, sobald es mehr als ein Feature gibt:** AC-Nummern sind je PRD vergeben. Kennt das Repository mehrere Features — archivierte eingeschlossen —, muss die referenzierende Testzeile den Feature-Slug nennen (die Zeile, nicht irgendeine Stelle der Datei, weil Tests mehrerer Features eine Datei teilen können). Nach dem ersten Release war jede Nummer unter den aktiven PRDs wieder eindeutig, und die Tests archivierter Features erfüllten die Kriterien neuer (SDD-022, Abschnitt 9.5).
- **AC-Referenzen ohne `\b`-Falle:** Testnamen wie `TestAC3_Login` oder `test_ac_3` müssen als Referenz zählen; die Erkennung in SDD-022 benennt den linken Kontext deshalb explizit statt über `\b` (Abschnitt 9.5).
- **Keine stillen Defaults ohne Konfiguration:** Ohne `sdd.config.yaml` endet jeder Prüflauf mit Exit 2.
- **Merge-Basis ist konfigurierbar:** `base_ref` mit dokumentierter automatischer Ermittlung; ein Prüfer, der die Basis raten muss, ist nicht reproduzierbar.
- **Selbstbezüglichkeit (Dogfooding):** Das `sdd`-Repository prüft sich ab Schritt 1/2 mit Profil `standard`; für das Handbuch selbst genügt der Coverage-Check über SDD-070–073 ohne eigene PRDs. Der erste Commit enthält alle vier Regelabschnitte gleichzeitig.

---

## 15. v1-Schnitt

Ein Katalog wächst, wenn man ihn lässt. Deshalb ist hier festgelegt, was v1 umfasst.

**v1 — implementiert und geprüft:**

```text
SDD-001   Verfassung vorhanden
SDD-002a  Markdown-Verfassung: Pflichtabschnitte
SDD-020   PRD vorhanden bei Schwellenüberschreitung
SDD-021   AC-Kennungen vorhanden
SDD-022   AC-Kennung in Testdatei referenziert
SDD-030   Tech Spec vorhanden bei Schwellenüberschreitung
SDD-050   Archivierung vor Release
SDD-070   Regel dokumentiert
SDD-071   Regel implementiert
SDD-072   Regel in einem Gate aktiv
SDD-073   error-Regel hat Prüfvertrag
```

Dazu: drei Gates, Baseline mit Gate-Bindung, Berichtsformat v1, `sdd init`, `sdd new`, `sdd baseline`.

**v1.1 — direkt danach:**

```text
SDD-002b  .sdd-Grammatik (strikter Modus)
SDD-010   ADR bei Verfassungsänderung
SDD-011   ADR-Unveränderlichkeit mit Lockfile
SDD-023   PRD nennt Nicht-Ziele
SDD-024   Diff weicht nicht von der Absicht ab
SDD-025   AC in Testdeklaration
SDD-031   Tech Spec referenziert AC
SDD-041   Task referenziert AC
SDD-051   Aktive Artefakte verweisen nicht auf archivierte
```

**Später:** SDD-003 (heuristisch), SDD-040, SDD-060 (assistiert).

**Ausdrücklich nicht in v1:** `sdd ui`, MCP-Server, KI-Assistenz, Index-Anbindung, `sdd staleness`. Sie sind in der Roadmap ab Schritt 5/6, aber kein Bestandteil der ersten prüfbaren Version.

Der Schnitt hat einen Zweck: Nach Schritt 2 gibt es ein Werkzeug, das zwölf Regeln wirklich prüft, statt 24 Regeln halb. Ein halb implementierter Katalog wäre die Selbsttäuschung, gegen die SDD-071 existiert.

---

## Offene Punkte

- Trägt das Reifegrad-Modell in der Praxis, oder wird es als Bewertung gelesen und erzeugt Druck statt Orientierung?
- Reicht ein Ratchet über einen einzelnen Schuldenzähler, oder braucht es Fristen je Regel?
- Ist die erklärte Änderungsabsicht (`touches` im Feature-Frontmatter) in der Praxis zumutbar, oder wird sie bei Scope-Änderungen während der Implementierung als lästiger Pflegeaufwand empfunden?
- Ist der v1-Schnitt richtig gezogen, oder fehlt eine Regel, ohne die das Werkzeug im Alltag unbrauchbar ist?
- Braucht der Agententeil des Handbuchs eine eigene, kürzere Fassung — oder reicht der Regelkatalog als YAML mit Prüfverträgen?
- Soll das Handbuch selbst nach dem eigenen Prozess geführt werden (Verfassung, ADRs, PRDs)? Das wäre konsequent, aber ein erheblicher Aufwand für ein Dokument.
