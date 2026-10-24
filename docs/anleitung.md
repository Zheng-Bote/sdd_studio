# Anleitung — SDD Studio im eigenen Projekt einsetzen

Diese Anleitung führt Schritt für Schritt durch die Einführung von `sdd` in **(A) ein neues Projekt** und **(B) ein bestehendes Projekt**. Sie beschreibt den Ablauf, nicht jede Regel; Einzelheiten stehen im [Handbuch](../handbuch/03-mechanismen.md), jede Regel erklärt `sdd explain <regel-id>`.

## Inhalt

- [Grundbegriffe in Kürze](#grundbegriffe-in-kürze)
- [Installation](#installation)
- [A) Neues Projekt](#a-neues-projekt)
- [B) Bestehendes Projekt](#b-bestehendes-projekt)
- [Der Alltag: ein Feature von der Idee bis zum Release](#der-alltag-ein-feature-von-der-idee-bis-zum-release)
- [Spickzettel](#spickzettel)
- [Häufige Stolpersteine](#häufige-stolpersteine)
- [Optional: Oberfläche und KI-Desktop](#optional-oberfläche-und-ki-desktop)

---

## Grundbegriffe in Kürze

| Begriff | Bedeutung |
|---|---|
| **Verfassung** | `constitution.md` — dauerhafte, systemweite Regeln (Architektur, Security, strikte Regeln). Ranghöchstes Dokument. Änderungen nur mit ADR. |
| **ADR** | Architekturentscheidung unter `docs/adr/`, nach der Entscheidung unveränderlich. |
| **PRD** | Was und warum eines Features: Problem, Ziel, Nutzer, **Nicht-Ziele**, **Akzeptanzkriterien** `AC-1`, `AC-2`, … |
| **Tech Spec** | Wie: Architektur, Schnittstellen, welche Komponente welches Kriterium umsetzt. |
| **Plan** | Optional: Tasks, jede mit Bezug auf ein Akzeptanzkriterium. |
| **Feature-Slug** | Kurzname eines Features aus `a-z`, `0-9`, `-`, z. B. `csv-export`. Steht als `feature:` im Frontmatter und im Branch `feature/<slug>`. |
| **Größenschwelle** | Ab welcher Änderung PRD und Tech Spec Pflicht sind — konfiguriert in `sdd.config.yaml`. Kleine Änderungen brauchen keine Artefakte. |
| **Gates** | `pre-implementation` (vor dem Coden), `pre-merge` (im Pull Request), `pre-release` (vor dem Tag). |
| **Baseline** | `.sdd/baseline.json` — hält Altbefunde eines Bestandsprojekts fest, damit sie das Gate nicht blockieren. Darf nur schrumpfen. |

Exit-Codes aller Prüfläufe: `0` sauber, `1` mindestens ein neuer `error`, `2` Ausführungsfehler (z. B. Konfiguration fehlt).

---

## Installation

Voraussetzung: Go 1.27 und Git.

```text
git clone https://github.com/Zheng-Bote/sdd_studio.git
cd sdd_studio
go build -o bin/sdd ./cmd/sdd
```

`bin/sdd` in den `PATH` legen (z. B. nach `~/.local/bin/`). Prüfen:

```text
sdd --help
```

In CI baut die GitHub Action `sdd` selbst; dort ist keine Installation nötig.

---

## A) Neues Projekt

### 1. Projekt anlegen

Im (leeren oder frischen) Git-Repository:

```text
git init -b main            # falls noch kein Repository
sdd init --dry-run          # zeigt, was angelegt würde
sdd init                    # legt an
```

`sdd init` schreibt nur Dateien, die es noch nicht gibt:

```text
sdd.config.yaml             Konfiguration (Profil, Pfade, Schwellen)
constitution.md             Verfassung mit den Pflichtabschnitten
docs/adr/  docs/prd/  docs/design/  docs/plans/  docs/archive/  docs/notes/
.github/workflows/sdd.yml   CI: pre-merge im Pull Request, pre-release auf release/**
.sdd/baseline.json          leere Baseline — ein neues Projekt hat keine Altlasten
```

Optionen: `--profile minimal|standard|regulated` (Standard: `standard`), `--no-ci` ohne Workflow.

### 2. Verfassung ausfüllen

`constitution.md` enthält drei Pflichtabschnitte mit Platzhaltern. Ersetzen Sie sie durch die dauerhaften Regeln Ihres Projekts — keine User Stories, keine Tasks, keine Planung (das prüft `SDD-003`):

```markdown
## Architektur/Layering
1. Die Oberfläche greift nur über den Dienst `store` auf Daten zu.

## Security
1. Keine Secrets im Repository; Passwörter nur als Argon2-Hash.

## Strikte Regeln
1. Jede Änderung braucht einen Test.
```

### 3. Konfiguration anpassen

Nach `sdd init` sind die Schwellen leer — dann verlangt `sdd` **nie** ein PRD. Tragen Sie in `sdd.config.yaml` ein, was bei Ihnen „groß“ ist:

```yaml
tests:
  include:                     # wo Tests liegen (leer = Standardliste, z. B. **/*_test.go)
    - "**/*_test.go"
    - "web/src/**/*.spec.ts"

thresholds:
  layers:                      # die Schichten Ihres Projekts
    - internal/api/**
    - internal/storage/**
    - web/**
  feature_artifacts_required_when:
    min_layers_touched: 2      # ab 2 berührten Schichten: PRD + Tech Spec
    touched_paths:             # oder sobald einer dieser Pfade berührt wird
      - cmd/**
      - openapi.yaml
```

Prüfen:

```text
sdd lint                       # erwartet: 0 error
sdd next                       # zeigt den ersten Prozessschritt
```

### 4. Einchecken und CI aktivieren

```text
git add -A
git commit -m "chore: SDD Studio einrichten"
git push -u origin main
```

Der Workflow `.github/workflows/sdd.yml` prüft ab jetzt jeden Pull Request mit `pre-merge` und jeden Push auf `release/**` mit `pre-release`; der Bericht erscheint in der Job-Zusammenfassung. Er braucht `fetch-depth: 0` (ist im Workflow gesetzt), weil `sdd` den Diff gegen den Basis-Branch auswertet.

Weiter mit [Der Alltag](#der-alltag-ein-feature-von-der-idee-bis-zum-release).

---

## B) Bestehendes Projekt

Bei einem gewachsenen Projekt soll `sdd` nichts umbauen und den Altbestand nicht als Fehler melden. Dafür gibt es `sdd adopt`: erst ein Probelauf, der nur liest, dann die Übernahme auf einem eigenen Branch.

### 1. Probelauf

Im Repository, auf einem sauberen Stand:

```text
sdd adopt
```

Es wird nichts geschrieben. Der Bericht zeigt:

- **Reifegrad L0–L4** und den einen nächsten Schritt:
  - L1 Verfassung ohne Befunde
  - L2 PRD und Tech Spec getrennt, Kennungen `AC-n`
  - L3 CI ruft `sdd gate` auf, Baseline vorhanden
  - L4 Feature-Artefakte mit `feature:` und `status`, Archiv gepflegt
- **Inventar** — jede Markdown-Datei mit vorgeschlagenem Typ (Verfassung, ADR, PRD, Tech Spec, Plan, Doku). Erkannt werden feste Muster, z. B. GitHub Spec Kit (`specs/features/*/spec.md`). „kein Muster“ bedeutet Doku/Exploration, nicht geraten.
- **Schritte**, die `--apply` ausführen würde, und **Konflikte** (Dateien, die es nicht anfasst).

Lesen Sie das Inventar genau: Liegen Ihre ADRs oder Specs an anderer Stelle als erwartet, erscheinen sie als Doku. Das korrigieren Sie nach der Übernahme in `sdd.config.yaml` (Schritt 3).

### 2. Übernahme anwenden

Voraussetzungen: sauberer Arbeitsbaum, kein losgelöster `HEAD`.

```text
sdd adopt --apply                      # Branch sdd/adopt
sdd adopt --apply --branch chore/sdd   # anderer Branchname
```

`sdd adopt --apply` legt einen **neuen Branch** an und macht je Schritt einen Commit:

| Commit | Inhalt |
|---|---|
| Konfiguration | `sdd.config.yaml` mit den erkannten Pfaden (falls noch keine da ist) |
| Verfassung | `constitution.md` aus der Vorlage (falls keine da ist) — Inhalt ist auszufüllen |
| Frontmatter | `feature: <slug>` und `status: active` vor vorhandene Feature-Artefakte ohne Frontmatter; sonst wird nichts an ihnen geändert |
| Baseline | `.sdd/baseline.json` mit den Altbefunden aller Gates, `.sdd/adr-lock.json` |
| Bericht | `.sdd/adopt-report.md` |

Es wird nichts gelöscht oder überschrieben, nichts gepusht oder gemergt; Ihr Ausgangsbranch bleibt unverändert. Ansehen und notfalls verwerfen:

```text
git diff main...sdd/adopt
git switch main && git branch -D sdd/adopt    # verwerfen
```

### 3. Nacharbeiten auf dem Übernahme-Branch

1. **Verfassung ausfüllen** — wie in [A) Schritt 2](#2-verfassung-ausfüllen). Gibt es schon eine Architektur- oder Security-Beschreibung, übernehmen Sie deren dauerhafte Regeln.
2. **Pfade prüfen** in `sdd.config.yaml`:
   ```yaml
   artifacts:
     adr: docs/architecture            # wenn Ihre ADRs dort liegen
     prd: specs/features/*/spec.md     # Ordner oder Glob — kein Umzug nötig
     design: specs/features/*/plan.md
     plans: specs/features/*/tasks.md
     archive: specs/archive
     docs:
       - docs                          # alles, was keine Feature-Prüfung auslösen soll
   ```
3. **Schwellen und Tests** eintragen — wie in [A) Schritt 3](#3-konfiguration-anpassen).
4. **CI** ergänzen. `sdd adopt` legt keinen Workflow an. Entweder `sdd init --dry-run` ansehen und mit `sdd init` die fehlenden Teile ergänzen (vorhandene Dateien bleiben unberührt; es legt allerdings auch die Standardordner `docs/…` an), oder den Workflow selbst anlegen:
   ```yaml
   # .github/workflows/sdd.yml
   name: sdd
   on:
     pull_request:
     push:
       branches: ["release/**"]
   permissions:
     contents: read
   jobs:
     pre-merge:
       if: github.event_name == 'pull_request'
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
           with:
             fetch-depth: 0
         - uses: Zheng-Bote/sdd_studio@v0.4.0
           with:
             gate: pre-merge
     pre-release:
       if: github.event_name != 'pull_request'
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
           with:
             fetch-depth: 0
         - uses: Zheng-Bote/sdd_studio@v0.4.0
           with:
             gate: pre-release
   ```
5. **Prüfen:**
   ```text
   sdd lint                 # error-Befunde des Altbestands sind gebaselinet; warn bleiben sichtbar
   sdd report               # Reifegrad, Gates, Schuldenstand
   sdd next                 # nächster Schritt je erkanntem Feature
   ```
   Haben Sie Pfade oder Verfassung geändert, entstehen dadurch eventuell neue Befunde. Beheben Sie sie, statt sie zu baselinen — die Baseline ist für den Altbestand, nicht für Neues.

Dann den Branch pushen, als Pull Request öffnen und mergen.

### 4. Inkrementell heben

- **Die Baseline schrumpft nur.** Ändert sich eine gebaselinete Stelle, gilt der Eintrag als erledigt, und der Befund ist wieder ein Fehler — wer Altbestand anfasst, bringt ihn in Ordnung. Erledigte Einträge entfernt `sdd baseline --prune`.
- **Neue Einträge nur bewusst:** `sdd baseline --accept <regel-id> --gate <gate>`, als eigener Commit mit Begründung im Pull Request. Für `SDD-050`, `SDD-051` und `SDD-070`–`SDD-073` ist das verboten.
- **Längst umgesetzte Features** auf `status: implemented` setzen; vor dem ersten Release archivieren (`SDD-050`, siehe unten).
- **Neue Features** laufen ab jetzt wie im Alltag beschrieben. `sdd report` zeigt den Trend des Schuldenstands.

---

## Der Alltag: ein Feature von der Idee bis zum Release

Am Beispiel des Features `csv-export`. Kleine Änderungen unterhalb der Größenschwelle brauchen keine Artefakte — `sdd gate pre-merge` meldet dann nichts.

### 1. Branch und Artefakte

```text
git switch -c feature/csv-export
sdd new --feature csv-export --title "CSV-Export" --touches "internal/api/**,internal/storage/**"
```

Das legt `docs/prd/csv-export.md` und `docs/design/csv-export.md` an. `--touches` erklärt, welche Pfade das Feature ändern wird; ohne Angabe meldet `pre-implementation` „Absicht nicht erklärt“. Mit `--type prd,design,plan` kommt ein Plan dazu, mit `--type adr --title "…"` ein ADR.

### 2. PRD und Tech Spec schreiben

- **PRD:** Problem, Ziel, Nutzer, Umfang, **Nicht-Ziele** (`SDD-023`) und **Akzeptanzkriterien** mit stabilen Kennungen `AC-1`, `AC-2`, … (`SDD-021`). Kennungen werden nie umnummeriert; ein entfallenes Kriterium wird als `~~AC-2~~` markiert.
- **Tech Spec:** wie die Kriterien umgesetzt werden; die Tabelle „Umsetzung der Akzeptanzkriterien“ nennt jede Kennung (`SDD-031`).
- **Plan (optional):** jede Task nennt ein Kriterium (`SDD-041`); Tasks erst, wenn die Tech Spec steht (`SDD-040`).
- **Verfassung betroffen?** Dann zuerst ein ADR (`sdd new --type adr --title "…"`); erst ein angenommenes ADR (`status: accepted`) erlaubt die Änderung (`SDD-010`).

Der Fortschritt:

```text
sdd next --feature csv-export
```

zeigt jeden Prozessschritt mit ✓, ✗ oder – und den nächsten Schritt mit Befehl.

### 3. Vor dem Coden prüfen

```text
sdd gate pre-implementation --feature csv-export
```

Prüft die erklärte Absicht (`touches`) gegen die Schwelle: Fehlen PRD oder Tech Spec, ist das ein Fehler, bevor eine Zeile Code entsteht. Ohne Feature lässt sich eine geplante Änderung vorab bewerten:

```text
sdd gate pre-implementation --paths "internal/api/**,internal/storage/**"
```

### 4. Umsetzen und testen

Jedes Akzeptanzkriterium braucht mindestens einen Test, der es nennt (`SDD-022`), am besten in der Testdeklaration oder ihrem Rumpf (`SDD-025`). Sobald es mehr als ein Feature gibt, gehört der Slug dazu:

```go
// csv-export AC-1: Zeilen werden als CSV mit Kopfzeile geschrieben
func TestExportWritesHeader(t *testing.T) { … }
```

```ts
// csv-export AC-2: leere Tabelle ergibt nur die Kopfzeile
it('exports only the header for an empty table', () => { … });
```

### 5. Pull Request

```text
sdd gate pre-merge          # lokal, wie später in CI
git push -u origin feature/csv-export
```

`pre-merge` wertet den Diff gegen den Basis-Branch aus: Schwelle überschritten ohne PRD/Tech Spec → `error`; Kriterium ohne Test → `error`; geänderte Dateien außerhalb von `touches` → `warn` (`SDD-024`). Das Feature erkennt `sdd` am Frontmatter im Diff, sonst am Branchnamen `feature/<slug>`, oder ausdrücklich mit `--feature`.

### 6. Nach dem Merge und vor dem Release

1. Nach dem Merge im PRD `status: implemented` setzen.
2. Vor dem Release die Artefakte abgeschlossener Features nach `docs/archive/` verschieben und `status: archived` setzen (`SDD-050`). Archiviert wird verschoben, nicht gelöscht; Links auf die neue Lage anpassen.
3. Auf dem Release-Branch prüfen:
   ```text
   sdd gate pre-release
   ```
4. **Tipp:** Ändert die Release-Vorbereitung Pfade über der Schwelle (etwa eine Versionsdatei unter `cmd/**`), findet `pre-merge` auf dem Release-Branch kein Feature mehr — alle Artefakte liegen dann im Archiv. Solche Änderungen vorher als Commit des zuständigen Features auf `main` bringen.

---

## Spickzettel

| Befehl | Zweck |
|---|---|
| `sdd init [--profile …] [--dry-run] [--no-ci]` | neues Projekt anlegen |
| `sdd adopt [--apply] [--branch …]` | bestehendes Projekt übernehmen |
| `sdd new --feature <slug> [--type prd,design,plan] [--title …] [--touches …]` | Feature-Artefakte anlegen |
| `sdd new --type adr\|doc --title …` | ADR oder Doku anlegen |
| `sdd next [--feature <slug>]` | nächster Prozessschritt |
| `sdd lint [--feature …] [--json]` | alle Regeln ohne Gate-Kontext |
| `sdd gate pre-implementation [--feature …] [--paths …]` | vor dem Coden |
| `sdd gate pre-merge [--feature …]` | im Pull Request |
| `sdd gate pre-release` | vor dem Release-Tag |
| `sdd report [--format markdown\|json]` | Reifegrad, Gates, Schuldenstand |
| `sdd baseline --prune` | erledigte Baseline-Einträge entfernen |
| `sdd baseline --accept <regel> --gate <gate>` | Befunde bewusst in die Baseline aufnehmen |
| `sdd explain <regel-id>` | Regel, Begründung, Prüfvertrag |
| `sdd ui [--open]` | lokale Oberfläche |

---

## Häufige Stolpersteine

| Beobachtung | Ursache und Abhilfe |
|---|---|
| `sdd` verlangt nie ein PRD | Die Schwellen in `sdd.config.yaml` sind leer — `thresholds.layers` bzw. `touched_paths` eintragen. |
| `SDD-022`: „Keine Testdatei gefunden“ | `tests.include` passt nicht zu Ihren Testdateien. |
| „kein Feature auflösbar“ | Branch heißt nicht `feature/<slug>`, und kein Feature-Artefakt ist im Diff — Branch umbenennen oder `--feature <slug>` angeben. |
| Gate in CI scheitert mit Exit-Code `2` (Basis nicht gefunden) | Der Checkout braucht `fetch-depth: 0`. |
| Lokal 0 Befunde, in CI Fehler | Lokal wird gegen einen anderen Stand verglichen. Mit `GITHUB_BASE_REF=main sdd gate pre-merge` lokal wie in CI prüfen; `origin/main` vorher aktualisieren (`git fetch`). |
| `SDD-024`-Warnung zu einer Datei | Die Datei liegt außerhalb von `touches` — im Frontmatter nachtragen oder die Änderung abtrennen. |
| Ein Befund soll „einfach weg“ | Erst `sdd explain <regel-id>` lesen. Beheben ist der Normalfall; die Baseline ist nur für Altbestand. |

---

## Optional: Oberfläche und KI-Desktop

- **`sdd ui --open`** — lokale Oberfläche auf `127.0.0.1`: Übersicht, Artefakte, Befunde (live), neues Artefakt, Übernahme, Handbuch. Schreibt nur über Vorschau und Bestätigung.
- **`sdd mcp`** — Schnittstelle für KI-Agenten wie Claude Code; Einbindung im [Handbuch, Teil 4](../handbuch/04-agenten.md).
- **`sdd-desk`** — optionaler, getrennter KI-Desktop: führt im Gespräch mit einem wählbaren Sprachmodell (lokal oder Cloud) von der Idee über PRD und Tech Spec bis zum Gate, gleicht die Tech Spec mit der Verfassung ab und führt `sdd`-Werkzeuge nur mit Ihrer Freigabe aus. Einrichtung in der [README](../README.md#ki-desktop-sdd-desk-optional). `sdd` selbst bleibt ohne KI und offline.
