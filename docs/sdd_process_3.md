---
type: Note
title: "sdd_process_3"
# Kurzbeschreibung in einem Satz; md-ai zeigt sie als Entitätsbeschreibung.
description: "Überarbeiteter SDD-Prozess: Trennung von Verfassung, ADR, PRD, Tech Spec und Tasks mit Präzedenzordnung, Traceability und durchgesetzten Mechanismen statt eines neunphasigen Pflichtverfahrens."
# z. B. active, deprecated; bei ADR: proposed, accepted, superseded
status: ""
notebook: "Snippets"
tags:
  - SpecDD
  - Prozess
# Relationen für den Wissensgraphen: exakte Dateinamen anderer Notizen,
# ohne .md und ohne [[ ]], z. B. uses: [Encryption Service]
uses: []
depends_on: []
contains: []
references: []
adr: []
relation:
  - sdd_prozess
  - sdd_process_2
  - SDD-Howto_de
  - SpecDD_SpecKit_Integration_DE
# Endpunkt oder Dateiname der API-Doku (leer = keine API dokumentiert)
api: ""
creation date: 2026-10-18 13:33
modification date: Sonntag 18. Oktober 2026 13:33:39
---

# sdd_process_3

----

## Überblick

Diese Fassung übernimmt die Diagnose aus `sdd_process_2` — nicht das Tool ist das Problem, sondern die Vermischung von Lebenszyklen — und zieht eine andere Konsequenz. Statt eines neunphasigen Pflichtverfahrens mit fünf Artefakten pro Feature werden die Artefakttypen getrennt, mit einer Präzedenzordnung versehen und durch Mechanismen durchgesetzt. Der Phasenapparat wird auf die Stellen reduziert, an denen ein Gate tatsächlich Schaden verhindert.

Zwei Aussagen sind neu gegenüber der Vorgängerversion:

> Die Reduktion von SpecDD ist nicht die Antwort. Die Durchsetzung der bestehenden Regeln ist es.

> Dauerhafte Architekturentscheidungen dürfen nicht mit temporären Featureentscheidungen vermischt werden — aber ein Prozess, der jede Änderung durch neun Tore schickt, erzeugt genau die Dokumentationsleichen, die er verhindern soll.

---

## 1. Ausgangspunkt und Kernbefund

Übernommen aus `sdd_process_2`:

- **SpecDD (`.sdd`) = Verfassung des Systems:** Single Source of Truth für Architektur, Security, Datenhaltung, Layering und technische Leitplanken.
- **Feature-Spezifikationen = Arbeitsaufträge:** beschreiben konkrete Vorhaben, haben einen begrenzten Lebenszyklus.

Der Schaden entsteht nicht durch schlechte Inhalte, sondern durch Vermischung:

- Produktanforderungen
- Architekturentscheidungen
- Technisches Design
- Tasklisten
- Testpläne
- Code-Verträge

Diese Typen haben unterschiedliche Zielgruppen, Verantwortliche und **vor allem unterschiedliche Lebenszyklen**.

### Der eigentliche Befund

SpecDD besitzt für dieses Problem bereits eine Regel. `.specdd/bootstrap.md` sagt ausdrücklich, eine Spec sei *kein* Work Order, keine Migration Note, kein Adoption Plan, kein Ticket, kein Task Contract. Der Bootstrap verlangt außerdem, dass `Tasks` nur den lokalen Gegenstand der Spec betreffen.

Wenn eine Spec in der Praxis trotzdem PRD, Tech Spec und Taskliste gleichzeitig ist, dann liegt das nicht an der Frameworksprache. Es liegt daran, dass **niemand ihre Einhaltung prüft**.

Daraus folgt die Umkehrung gegenüber `sdd_process_2`:

- Die Diagnose ist richtig.
- Die vorgeschlagene Therapie — ein neunphasiges Pflichtverfahren mit fünf Artefakten — behandelt ein Durchsetzungsproblem mit mehr Zeremonie.
- Gebraucht werden **Mechanismen**, nicht mehr Phasen.

Ein Mechanismus ist hier alles, was ohne menschliche Disziplin wirkt: ein Lint, ein CI-Gate, ein Template-Feld, eine Präzedenzregel, ein Archivdatum.

---

## 2. Was aus `sdd_process_2` übernommen wird

- **Die Lebenszyklus-These.** Sie erklärt konkrete Schäden, etwa zwei Dokumente, die dieselbe Architektur in verschiedenen Fassungen beschreiben, weil einmal „Dauerzustand" und einmal „Arbeitsstand" im selben Format liegen.
- **Die Ownership- und Lebensdauer-Tabelle.** SpecDD beantwortet, *wer* was ändern darf, aber nicht, *wie alt* ein Dokument werden darf. „PRD bis Release, Taskliste bis Merge, ADR permanent" ist prüfbar, „gehört halt dazu" nicht.
- **Die Trennung von Umsetzungsreihenfolge (Plan) und Arbeitspaketen (Tickets).** Das wird regelmäßig falsch gemacht: erst Tickets, dann fällt die Reihenfolge auf.
- **Die Archivierungsstrategie.** Sie ist der Grund, warum das Modell überhaupt funktionieren kann. Ohne Archivierung erzeugt jeder Prozess Dokumentationsleichen.
- **Die Leitfrage:** *Unterstützt das Tool den Prozess, oder zwingt der Prozess sich dem Tool unterzuordnen?*

---

## 3. Korrekturen an `sdd_process_2`

### 3.1 Phase 0 wird entzerrt

Ein Pflicht-ADR **vor** der Idee ist realitätsfremd. Ob eine Architekturentscheidung nötig ist, weiß man oft erst beim technischen Design. Ein Gate vor Phase 1 erzeugt ADRs pro forma — genau der Vorwurf, den das Dokument Spec Kit macht.

**Neu:** Ein ADR darf in jeder Phase eröffnet werden. Es muss nur **vor Beginn der Implementierung entschieden** sein (`status: accepted`). Der Auslöser ist nicht die Phase, sondern die Frage: *Ändert dieses Vorhaben eine systemweite Regel?*

### 3.2 Die Phasen werden zu einer Denkreihenfolge, nicht zu Toren

`SpecDD_SpecKit_Integration_DE` und der Bootstrap behandeln Specs als Vertrag, nicht als Arbeitsauftrag. Ein neunstufiger Wasserfall mit Exit-Kriterium pro Stufe hebt diese Trennung wieder auf und macht aus jedem Ein-Zeilen-Bugfix ein Verfahren.

**Neu:** Phasen beschreiben die Reihenfolge der Klärung. Verbindliche Gates gibt es nur dort, wo ein Fehler teuer ist (siehe Abschnitt 7).

### 3.3 Es fehlt eine Größenschwelle

Ohne Schwellwert gilt der Prozess entweder für alles oder für nichts. Beides ist schlecht.

**Vorschlag für die Schwelle** — PRD und Tech Spec sind erforderlich, wenn mindestens eines zutrifft:

- eine neue öffentliche Schnittstelle (HTTP-Endpoint, CLI-Flag, Dateiformat)
- ein neues Persistenzfeld, eine neue Tabelle oder eine Migration
- ein neuer externer Abhängigkeitstyp (Provider, Dienst, Bibliothek mit Netzwerkzugriff)
- eine Änderung an Security-, Datenschutz- oder Aufbewahrungsregeln
- eine Änderung, die mehr als einen Layer betrifft

Darunter: Issue plus Feature-Spec. Kein PRD, keine separate Tech Spec, kein Execution Plan.

### 3.4 Traceability fehlt — der einzige echte Verlust

Phase 6 sagt „Tests basierend auf Akzeptanzkriterien aus PRD und Tech Spec". Ohne stabile Bezeichner ist das nicht prüfbar, sondern behauptet. In SpecDD war `Scenario` genau dieses Bindeglied; wer es entfernt, muss es ersetzen.

**Neu: Akzeptanzkriterien erhalten IDs.**

```text
PRD:   AC-1  Ein neuer Vault wird ohne manuellen Eingriff indiziert.
       AC-2  Ein Modellwechsel erzeugt eine sichtbare Warnung.

Task:  GH-143  AC-1, AC-2
Test:  TestVaultIndex (AC-1), TestModelMismatch (AC-2)
```

Regel: Jedes Akzeptanzkriterium wird von mindestens einem Test referenziert. Ein Kriterium ohne Test ist entweder nicht umgesetzt oder nicht geprüft — beides muss sichtbar sein.

### 3.5 Die ADR-Ablage war doppelt

ADRs standen gleichzeitig unter „Gehört in die Verfassung" und in `docs/architecture/adr/`. Damit ist offen, wo die Wahrheit liegt — und es entsteht derselbe Drift wie zwischen zwei Spec-Fassungen, nur mit zwei Ordentypen.

**Neu: klare Rollenteilung.**

- Die **Verfassung** hält den *aktuell gültigen Zustand* und verlinkt die ADR-ID.
- Das **ADR** hält die *unveränderliche Begründung* und wird nie umgeschrieben. Ablösung erfolgt durch ein neues ADR mit `supersedes`.
- Wer den Zustand ändern will, ändert die Verfassung und verweist auf das neue ADR.

### 3.6 Es fehlt eine Präzedenzordnung

SpecDD ist vor allem deshalb agententauglich, weil es einen expliziten Auflösungsalgorithmus hat: Pfadvererbung, nächster Kontext gewinnt, bei Konflikt gewinnt die strengere Regel. Das vorgeschlagene Modell hatte fünf Artefakttypen ohne Ordnung.

Für einen Menschen ist ein Widerspruch verhandelbar. Für einen Agenten ist er Zufall. Deshalb verbindlich:

1. **Verfassung** — systemweite Regeln. Gewinnt gegen alles.
2. **ADR** — begründet eine Regel der Verfassung, ersetzt sie aber nicht.
3. **Tech Spec** — darf die Verfassung präzisieren, niemals lockern.
4. **PRD** — fachliche Anforderungen. Darf die Verfassung nicht überschreiben; ein Konflikt ist ein Blocker, keine Auslegungsfrage.
5. **Plan und Tasks** — Umsetzung. Dürfen keine der obigen Regeln brechen; ein Verstoß ist ein Prozessfehler, kein Freibrief.

Bei Konflikt zwischen gleichermaßen bindenden Regeln gilt: **nicht entscheiden, sondern eskalieren.** Ein stiller Kompromiss ist der teuerste Fehler.

### 3.7 Ownership-Tabelle war unvollständig

Es fehlten: die Verfassung selbst, Testfälle und Testplan, Runbooks, `archive/` sowie der wichtigste Rückfluss — **ein Feature ändert die Architektur**.

Dieser Fall ist in `.specdd/bootstrap.project.md` bereits geregelt und besser als alles, was `sdd_process_2` dazu anbietet:

> If a feature changes the architecture, `md-indexer.sdd` MUST be updated in the same change.

Das ist ein Mechanismus, kein Ratschlag. Er gehört übernommen.

---

## 4. Das Artefaktmodell

Fünf Typen, jeder mit genau einer Rolle:

- **Verfassung** (`.sdd`): systemweite Regeln. Ändert sich selten, nur durch ADR begründet.
- **ADR**: eine Entscheidung mit Kontext, Optionen, Begründung, Folgen. Unveränderlich.
- **PRD**: fachliche Anforderungen mit IDs und Nicht-Zielen. Bis Release gültig.
- **Tech Spec**: technische Umsetzung, Schnittstellen, Datenmodell, Sicherheitsaspekte. Bis Release gültig.
- **Plan und Tasks**: Reihenfolge und Arbeitspakete. Bis Merge gültig.

Nicht mehr Teil der Verfassung: User Stories, fachliche Szenarien, Feature-Beschreibungen, Tasklisten, Sprint- und Releaseplanung. Diese gehören in PRD, Tech Spec und Projektmanagement.

Nicht mehr Teil einer Feature-Spec: systemweite Regeln. Wer eine Architekturregel ändern will, ändert die Verfassung — im selben Change.

### Die Verfassung enthält

- **Architektur:** System Context, Container, Layering, Modulgrenzen, Prinzipien
- **Security:** Verschlüsselung, PII-Regeln, Key Management, Logging-Richtlinien
- **Datenhaltung:** Datenbankstrategie, Naming, Migrationsregeln, Klassifikation
- **Engineering Standards:** API-Konventionen, Error Handling, Observability
- **Verweise** auf die ADRs, die diese Regeln begründen

### Die Verfassung enthält nicht

```yaml
Feature:
  Customer Export
```

```yaml
Collector:
  SAPCollector
```

```yaml
Story:
  User can import CSV files
```

---

## 5. Prozess: Denkreihenfolge mit drei Gates

Die Phasen bleiben als Reihenfolge erhalten, damit niemand vor der Klärung implementiert. Verbindlich sind nur drei Punkte.

```text
0.  Architecture Decision      (nur wenn eine systemweite Regel betroffen ist)
1.  Idea / Issue               Problem, Motivation, Value, grobe Schätzung
2.  PRD                        Was, Warum, Wer, Nicht-Ziele, AC-IDs
3.  Tech Spec                  Wie, Schnittstellen, Daten, Security
4.  Plan                       Reihenfolge entlang technischer Abhängigkeiten
5.  Tasks                      Arbeitspakete mit AC-Referenz
6.  Implementation             Code auf Feature-Branch
7.  Test                       AC -> Test-Mapping vollständig
8.  Review                     Architektur-Konformität, Security, Verfassung
9.  Release & Doku             Changelog, API-Doku, Betriebsdoku, Archivierung
```

**Gate 1 — vor der Implementierung:** ADR entschieden, falls eine systemweite Regel betroffen ist. Tech Spec abgenommen, falls die Größenschwelle greift.

**Gate 2 — vor dem Merge:** Akzeptanzkriterien durch Tests referenziert; keine Verfassungsverletzung; bei Architekturänderung ist die Verfassung im selben Change aktualisiert.

**Gate 3 — nach dem Release:** PRD, Tech Spec und Plan sind archiviert, nicht gelöscht. Die Verfassung beschreibt den neuen Zustand.

Alles andere ist Reihenfolge, kein Tor.

---

## 6. Ownership und Lebenszyklus

| Dokument | Owner | Lebensdauer |
|-----------|---------|-------------|
| Verfassung (`.sdd`) | Architektur | Permanent, versioniert |
| ADR | Architektur | Permanent, unveränderlich |
| Architektur-Doku | Architektur | Permanent |
| Security-Doku | Architektur | Permanent |
| API-Dokumentation | Architektur/Entwicklung | Permanent |
| Runbook | Betrieb | Permanent |
| Changelog | Produkt/Entwicklung | Permanent |
| PRD | Product | Bis Release, dann Archiv |
| Tech Spec | Architektur | Bis Release, dann Archiv |
| Execution Plan | Entwicklung | Bis Release, dann Archiv |
| Taskliste | Entwicklung | Bis Merge |
| Pull Request | Entwicklung | Historisch |
| Testfälle | Entwicklung | Leben mit dem Code |

Zwei Ergänzungen, die den Unterschied machen:

- **`archive/` ist kein Mülleimer, sondern Beweis.** Was archiviert ist, ist nicht mehr gültig, aber nachvollziehbar. Wer ein archiviertes Dokument zitiert, zitiert Historie.
- **Die Verfassung ist versioniert.** Ohne Versionshistorie ist „dauerhaft" nur ein Wort.

---

## 7. Mechanismen statt Disziplin

Der Kern dieser Fassung. Alles hier wirkt ohne Zutun.

1. **Syntax-Lint für `.sdd`.** Prüft Section-Labels, Einrückung, Pflichtfelder, erlaubte Task-Marker. Ohne Lint ist „SpecDD ist bindend" eine Absichtserklärung. Ein Projekt, dessen Verfassung die eigene Grammatik nicht erfüllt, hat keinen Vertrag, sondern Prosa.
2. **CI-Gate „Architektur berührt?"** Wenn ein PR Layer, Datenmodell, Security oder öffentliche Schnittstelle berührt, muss die Verfassung im selben PR geändert sein. Als Checkliste existiert das bereits — als Gate wirkt es.
3. **AC-ID-Check.** Jedes `AC-n` im PRD erscheint in mindestens einem Testnamen oder Testkommentar. Fehlt eine Referenz, schlägt der Check fehl.
4. **ADR-Pflicht bei Regeländerung.** Eine Verfassungsänderung ohne ADR-Referenz wird abgelehnt. Ausnahme: Tippfehler und Formulierungen ohne Bedeutungsänderung.
5. **Archivdatum.** Jede Feature-Spec trägt `status` und ein geplantes Archivdatum. Nach dem Release wird sie verschoben, nicht vergessen.
6. **Template-Felder statt Prosa-Bitten.** Ein Feld, das ausgefüllt werden muss, ist stärker als ein Satz, der beachtet werden soll.

Mechanismen sind billiger als Phasen: Ein Lint kostet einen Nachmittag und wirkt dauerhaft. Ein neues Pflichtdokument kostet pro Feature und wirkt nur, solange jemand nachhält.

---

## 8. Bewertung GitHub Spec Kit

**Vorteile:** strukturierter Workflow, vorgegebene Templates, gute Einstiegshilfe.

**Risiken:** erzwingt bestimmte Denkweisen, kann vom eigentlichen Arbeitsprozess abweichen, führt häufig zu Dokumenten, die nur pro forma entstehen, und begünstigt Dokumentationsleichen.

Für das Risiko „pro forma" ist die Ursache nicht das Framework, sondern das Fehlen einer Größenschwelle und eines Archivs. Beides ist unabhängig vom Tool nachrüstbar.

**Alternative:** GitHub Issues, GitHub Projects, ADRs und Markdown-Dateien — kombiniert mit den Mechanismen aus Abschnitt 7. Oft langlebiger und wartungsärmer als ein Framework, das selbst gepflegt werden muss.

Die entscheidende Frage bleibt: *Unterstützt das Tool den Prozess, oder zwingt der Prozess sich dem Tool unterzuordnen?*

---

## 9. Empirischer Befund (md-indexer)

Der Vorschlag ist zur Hälfte bereits umgesetzt — ohne den Phasenapparat:

- `md-indexer.sdd` übernimmt die Rolle der Verfassung.
- `.github/ISSUE_TEMPLATE/feature_spec.md` übernimmt die Rolle des PRD: EARS-Requirements, Scope pro Layer, offene Entscheidungen, Architektur-Abgleich als Checkliste, Akzeptanzkriterien.
- `specs/features/<name>/` enthält Feature-Artefakte, `docs/` die dauerhafte Dokumentation.
- `docs/SpecDD_SpecKit_Integration_EN.md` regelt das Zusammenspiel beider Ebenen.

Aufschlussreich ist die Verteilung: Von 22 Feature-Verzeichnissen haben 11 `spec + plan + tasks` und 11 nur `spec.md`. Der Apparat wird also **selektiv nach Aufwand** angewendet, nicht nach Vorschrift. Das ist gesund — und es widerlegt das „zwingend" eines starren Phasenmodells.

Was fehlt, ist nicht ein neuer Prozess, sondern:

- ein Lint, der die `.sdd`-Syntax prüft (die vorhandene Verfassung erfüllt die Grammar des Bootstraps nicht: sie beginnt mit `# SpecDD: …` und `## 1. System Intent`, verlangt sind `Spec: Name` und bekannte Section-Labels),
- das CI-Gate statt der Checkliste,
- AC-IDs im Feature-Template,
- ein Archivdatum und ein `archive/`.

---

## 10. Zukunft von SpecDD

Die Ausgangsfrage von `sdd_process_2` lautete: *Ist SpecDD weiterhin das richtige Fundament?* Sie wurde gestellt und nicht beantwortet. Die ehrliche Antwort:

**Ja für die Rolle, nein für den Umfang.**

Wenn User Stories, Szenarien und Tasklisten herausfallen, bleibt eine Verfassung aus etwa fünf Section-Typen: Architektur, Security, Datenhaltung, Standards, Verweise. Das ist Markdown mit Section-Konventionen.

Ob dafür die `.sdd`-Grammatik nötig ist, sollte man offen prüfen dürfen. Ihr eigentlicher Wert liegt nicht in der Syntax, sondern in drei Eigenschaften:

- **Pfadvererbung** — Regeln gelten dort, wo sie stehen, und vererben sich nach unten.
- **Präzedenz** — bei Konflikt gewinnt die spezifischere, bei gleicher Spezifität die strengere Regel.
- **Eigentum** — `Owns` und `Can modify` beantworten, wer was ändern darf.

Das sind Eigenschaften, die ein Agent zuverlässig auflösen kann und ein Mensch in Prosa nicht. Wenn man sie behält, ist die Syntax zweitrangig. Wenn man sie aufgibt, verliert man mehr als eine Dateiendung.

---

## 11. Empfohlene Dokumentstruktur

```text
/docs
│
├── architecture/
│   ├── system-context.md
│   ├── security.md
│   ├── database.md
│   ├── layering.md
│   └── adr/
│       ├── ADR-012-plugin-framework.md
│       └── ADR-013-envelope-encryption.md
│
├── prd/
│   └── feature-xyz.md
│
├── design/
│   └── feature-xyz-tech-spec.md
│
├── plans/
│   └── feature-xyz-implementation.md
│
├── api/
├── runbooks/
├── changelog/
└── archive/
    ├── prd/
    ├── design/
    └── plans/
```

Der einzige Unterschied zu `sdd_process_2` ist `archive/` als **eigener Baum** statt als Fußnote. Archivierte Dokumente bleiben auffindbar, sind aber eindeutig als Historie markiert.

---

## 12. Archivierungsstrategie

Nach erfolgreichem Release werden `prd/`, `design/` und `plans/` des jeweiligen Features nach `archive/` verschoben — mit gesetztem `status: archived` und dem Release-Tag im Frontmatter.

Nicht archiviert werden: Verfassung, ADRs, Architektur-, Security- und API-Dokumentation, Runbooks, Changelog. Diese beschreiben den aktuellen Zustand oder eine unveränderliche Entscheidung.

Der Nutzen ist nicht Ordnungsliebe. Er ist Beantwortbarkeit der drei Fragen, die `sdd_process_2` zu Recht als Kernproblem benennt:

- Was gilt dauerhaft? → Verfassung, ADRs
- Was ist nur Planung? → `plans/`, bis zum Release lebend
- Was ist umgesetzt und was veraltet? → `archive/` plus Changelog

---

## Offene Punkte

- Ist die Größenschwelle aus 3.3 richtig geschnitten, oder erzeugt sie für ein Ein-Personen-Projekt zu viel Dokumentation?
- Soll die Verfassung weiter `.sdd` bleiben, wenn ihr Umfang auf fünf Section-Typen schrumpft? (Abschnitt 10)
- Wer besitzt die Verfassung, wenn es nur eine Person gibt — und wer prüft dann das CI-Gate?
- Wie wird `archive/` durchsuchbar gehalten, ohne dass es die Suche im aktuellen Bestand verwässert?
- Braucht es für ADRs ein eigenes Ablöse-Verfahren (`supersedes` / `superseded-by`), oder reicht ein Statusfeld?
