---
type: Note
title: Kombination von SpecDD, Spec Kit, OpenSpec und BMAD
description: Vier Spec-Frameworks, ihre Ebenen — und was die drei Einstiegspunkte Greenfield, Brownfield und BMAD-Vorgeschichte jeweils brauchen
status: ""
notebook: ""
tags:
  - BMAD
  - SpecKit
  - OpenSpec
  - SDD
  - SpecDD
uses: []
depends_on: []
contains: []
references:
adr:
  - https://github.com/github/spec-kit
  - https://github.com/bmad-code-org/BMAD-METHOD
  - https://github.com/Fission-AI/OpenSpec
relation: []
api: ""
creation date: 2026-11-18 02:03
modification date: 2026-11-22
---
# Kombination von SpecDD, Spec Kit, OpenSpec und BMAD

Die vier Ansätze greifen auf **verschiedenen Ebenen**, deshalb schließen sie sich nicht aus: **BMAD** liefert die vorgelagerte Strategie, Konzeption und Rollen-Simulation (Makro-Ebene), **Spec Kit** die strukturierte, verifizierbare Feature-Ausführung (Mikro-Ebene), **OpenSpec** die versionierbare Änderungsspezifikation, die auch einen gewachsenen Bestand erfasst, und **SpecDD** das Paradigma, unter dem alle drei arbeiten: erst die Spezifikation, dann der Code.

Während BMAD ein komplettes agiles Team (Analyst, PM, Architect) simuliert, um tiefe Planungsartefakte zu erzeugen, sorgt Spec Kit mit einer projektweiten Verfassung (`constitution.md`) und klaren Befehlen (`/specify`, `/plan`, `/tasks`, `/implement`) dafür, dass der Coding-Agent bei der Umsetzung nicht vom Kurs abkommt ([speckit.org](https://speckit.org/), [reenbit.com](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/)).

## Die vier Bausteine und die Ebene, auf der sie greifen

| Ebene | Werkzeug | Aufgabe |
|---|---|---|
| Methodik | **SpecDD** | Grundprinzip: Spezifikation vor Implementierung |
| Governance | **Spec Kit** | Strukturierter Workflow mit Constitution, Spec, Plan, Tasks |
| Spezifikationsmodell | **OpenSpec** | Änderungs- und Feature-Spezifikationen als versionierbare Artefakte |
| Organisation | **BMAD** | Multi-Agent-Prozess mit Analyst, Architect, PM, Developer, QA |

Die Ebenen sind keine Leiter, die man immer von oben beginnt. **Was zuerst nötig ist, hängt davon ab, was beim Start schon existiert** — und genau darin unterscheiden sich die drei Einstiegspunkte.

## Einstiegspunkte: Greenfield, Brownfield, BMAD-Vorgeschichte

| Einstieg | Was beim Start existiert | Was zuerst kommt | Werkzeug, das trägt |
|---|---|---|---|
| **Greenfield** | leeres Repository, eine Idee | Idee schärfen, Architektur entwerfen, Regeln setzen — dann Features | BMAD (Makro), dann Spec Kit oder `sdd` (Feature-Strecke) |
| **Brownfield** | laufender Code, unklare oder fehlende Spezifikationen | inventarisieren, Altlasten einfrieren, Regeln nachziehen — dann Features | OpenSpec (Änderungen), `sdd adopt` (Bestand), Spec Kit optional |
| **BMAD-Vorgeschichte** | Initiative `R1…`, Epics mit „Done when“, Architektur-Spine | nichts neu erfinden: wörtlich übernehmen, dann verfeinern | `sdd adopt` mit `pipeline: bmad` |

### Greenfield — das leere Repository

Hier lohnt die volle Makro-Ebene, weil es noch nichts gibt, woran man sich halten könnte: Analyst und PM schärfen Anforderungen und schneiden Epics, der Architect legt Tech-Stack, Schnittstellen und Datenmodell fest, und diese Leitplanken wandern in die Verfassung. **Die Verfassung ist das Ergebnis der Architektur, nicht ihr Ersatz** — Regeln über ein System, das noch nicht entworfen ist, wären Behauptungen.

Erst danach beginnt die Feature-Strecke: ein Epic nach dem anderen durch `/specify` → `/plan` → `/tasks` → `/implement`. Jede kleine Story durch den vollen BMAD-Zyklus zu schleusen, ist der teure Weg; dafür gibt es die schlanke Strecke.

Was `sdd` an dieser Stelle tut: `sdd init` legt Konfiguration, Verfassungsgerüst und CI an (`--pipeline bmad`, wenn die Makro-Ebene von Anfang an mitlaufen soll), `sdd new --type initiative|epic` legt die Makro-Artefakte an, `sdd next` nennt danach genau einen nächsten Schritt.

### Brownfield — die gewachsene Codebasis

**Hier kippt die Reihenfolge.** Es gibt Code, aber keine verlässliche Spezifikation; eine Makro-Planung von vorn würde den Bestand ignorieren statt ihn zu heben. Was zuerst kommt, ist Bestandsaufnahme: Was ist dauerhafte Regel, was ist Planung, was ist veraltet?

Beide betroffenen Frameworks sagen das selbst:

- **BMAD** hat für den Bestand einen eigenen Pfad ([`help/existing-codebase.md`](https://github.com/bmad-code-org/BMAD-METHOD)). Er arbeitet **ohne einen vorgelagerten Dokumentationsdurchgang**: ein `AGENTS.md` mit wenigen geprüften Regeln (und übernimmt eine vorhandene `AGENTS.md`/`CLAUDE.md`, statt neu anzufangen), ein geführter Durchgang durch den Code, wenn ihn niemand kennt, Architektur **nur wenn nötig** und erst wenn ein Ticket existiert, das sie lesen kann, danach ein kleiner erster `build`. Was eine Änderung eine Stufe höher hebt, ist nicht ihre Größe, sondern ob sie eine Entscheidung erzwingt, die die bestehenden Muster nicht decken.
- **OpenSpec** beschreibt sich selbst als „built for brownfield not just greenfield" ([README](https://github.com/Fission-AI/OpenSpec)). Es arbeitet pro Änderung: ein Change Proposal beschreibt, was sich ändert, die Delta-Specs beschreiben die Anforderung, und nach der Umsetzung wird die Änderung archiviert, während die Specs fortgeschrieben werden.

Was `sdd` an dieser Stelle tut: `sdd adopt` ist ein **Probelauf** — Inventar, Klassifikation, Reifegrad, nichts geschrieben; `--apply` legt einen eigenen Branch an und committet Schritt für Schritt; `sdd baseline` friert die Altbefunde gate-gebunden ein, damit der erste Lauf nicht an Schulden scheitert, die niemand an einem Tag abträgt. Ohne Baseline wird ein Gate auf einem gewachsenen Projekt nach dem ersten Lauf abgeschaltet — und ist damit tot, bevor es wirkt.

Der Satz aus der ersten Fassung dieser Notiz bleibt richtig und wird durch OpenSpec nur präziser: für kleine, isolierte Änderungen in einem bestehenden Repo braucht es kein Makro-Framework. Eine Änderungsspezifikation und eine Prüfung genügen.

### BMAD-Vorgeschichte — Bestand mit Makro-Ebene

Der dritte Fall ist der interessanteste, weil er Arbeit schon enthält, die man nur falsch aufheben kann: eine Initiative mit nummerierten Anforderungen, Epics mit prüfbaren „Done when"-Punkten, eine Architektur-Spine, oft eine geschärfte Idee („forge") und Review-Berichte.

Hier ist **nichts neu zu erfinden**. Anforderungen werden **wörtlich** übernommen — nicht umnummeriert, nicht umformuliert —, und aus den Done-when-Punkten werden Akzeptanzkriterien. Die Reihenfolge ist: übernehmen → Inhalte sichern → fehlende Architektur ergänzen → Verfassung aus Constraints **und** Spine → Epics verfeinern → Gates.

Was `sdd` an dieser Stelle tut: `sdd adopt` erkennt `_bmad-output/` und schaltet `pipeline: bmad` ein; `sdd-ingest-bmad`, `sdd-architecture` und `sdd-epic-refine` führen durch die Schritte, und `SDD-080`–`SDD-082` prüfen maschinell, dass Anforderung → Epic → Index zusammenpassen. Der Ledger unter `_bmad-output/` bleibt unverändert und wird nur gelesen.

**Ein Befund aus der Praxis (2026-11-22):** BMAD v6 installiert sein Werkzeug nicht mehr zwingend unter `_bmad/`, sondern als Skills unter `.agents/skills/…`; im Referenzbestand `GeoGallery` ist `_bmad/` verschwunden, während `_bmad-output/` blieb. Die Erkennung hängt deshalb zu Recht am **Ergebnis** (`_bmad-output/`), nicht am Werkzeugordner. Ein Probelauf über diesen Bestand erkennt 8 Epics und die Initiative; Architektur-Spine, Forge-Brief, Review-Berichte und die Stories unter `tickets/epic-*/tickets/` werden als Doku klassifiziert, nicht als Feature-Artefakte.

## Rollenverteilung im hybriden Workflow (Greenfield)

| Phase | Zuständiges Framework | Kernaufgabe & Output |
|---|---|---|
| **1. Discovery & Produktvision** | **BMAD-Method** | Analyst- und PM-Agenten erarbeiten Produktziele, User-Flows und ein grobes PRD. |
| **2. Systemarchitektur & Regeln** | **BMAD + Spec Kit** | Der BMAD-Architect definiert Tech-Stack und Datenmodelle. Diese Leitplanken wandern in die `constitution.md` von Spec Kit. |
| **3. Feature-Spezifikation** | **GitHub Spec Kit** | Einzelne Epics aus BMAD werden per `/specify` und `/clarify` in eindeutige, testbare Feature-Specs übersetzt. |
| **4. Technische Planung & Tasks** | **GitHub Spec Kit** | `/plan` und `/tasks` zerlegen die Spec in kleinteilige, maschinenlesbare Arbeitspakete (`tasks.md`). |
| **5. Implementierung & Review** | **Spec Kit + BMAD QA** | `/implement` schreibt den Code; optional übernimmt der BMAD-QA-Agent abschließend Code-Review und Testing. |

## Schritt für Schritt (Greenfield)

1. **Projekt-Setup im Repository.** Beide Werkzeuge im selben Projektverzeichnis initialisieren, sodass sie auf dieselbe Codebasis zugreifen ([Speckit-Deep-Dive](https://blog.lpains.net/posts/2025-12-07-deep-dive-into-speckit/)): BMAD über `npx bmad-method install`, Spec Kit über `specify init`. Auf GitHub existiert mit [BMAD-SPEC-KIT](https://github.com/oimiragieo/BMAD-SPEC-KIT) ein Community-Fork, der beide Ansätze vorintegriert.
2. **Makro-Planung mit BMAD** (einmalig zum Projektstart): Analyst und Product Manager schärfen die Anforderungen und schneiden Epics; der Architect entwirft Systemarchitektur, API-Schnittstellen und Datenbankschemas.
3. **Architektur in der Verfassung verankern.** Das größte Risiko bei reinem BMAD ist der „Context Drift" über viele Agenten-Übergaben hinweg. Die wichtigsten Architekturentscheidungen (Tech-Stack, Coding-Standards, Sicherheitsvorgaben) wandern in die `constitution.md`; jede spätere Spec erbt sie automatisch.
4. **Feature-Umsetzung mit Spec Kit** (iterativ pro Epic): `/specify` mit dem Epic als Eingabe, `/clarify` für offene Randfälle, `/plan` und `/tasks`, dann `/implement`.
5. **Qualitätssicherung.** Der BMAD-QA-Agent prüft den erzeugten Code gegen die Spec: Edge-Cases, nicht-funktionale Anforderungen, Testabdeckung.

## OpenSpec: die Änderungsspezifikation als versioniertes Artefakt

OpenSpec setzt eine Ebene unter BMAD und neben Spec Kit an: Es speichert nicht den Plan, sondern **was entschieden wurde** — als lesbare Markdown-Artefakte im Repository ([README](https://github.com/Fission-AI/OpenSpec)).

```text
openspec/
├── specs/                     # der geltende Stand, je Fähigkeit eine Spec
│   └── <capability>/spec.md
└── changes/                   # offene Änderungen
    ├── <change-id>/
    │   ├── proposal.md        # warum, und was sich ändert
    │   ├── specs/             # Delta: ADDED / MODIFIED / REMOVED Requirements
    │   ├── design.md          # technischer Ansatz
    │   └── tasks.md           # Umsetzungscheckliste
    └── archive/               # abgeschlossene Änderungen, mit Datum
        └── 2025-01-23-<change-id>/
```

Der Ablauf ist ein Kreislauf aus vier Schritten: `/openspec-explore` (erst denken, den Code lesen, Optionen abwägen), `/openspec-propose <change>` (Vorschlag anlegen), `/openspec-apply-change` (Tasks abarbeiten) und `/openspec-archive-change` (archivieren, Specs fortschreiben). Requirements stehen in schlichtem Markdown mit konkreten Szenarien — „keine Spezialsyntax zu lernen".

Zwei Eigenschaften machen OpenSpec für den Brownfield-Fall wertvoll:

- **Die Änderung ist die Einheit, nicht das Feature.** Ein Change Proposal lässt sich auf einen Bestand anwenden, der nie als Feature-Backlog beschrieben wurde.
- **Der Verlauf bleibt sichtbar.** `changes/archive/` ist ein nachvollziehbares Protokoll: Wer wissen will, warum eine Anforderung heute so lautet, findet die Änderung, die sie eingeführt hat.

Grenze: OpenSpec ersetzt weder Rollen noch Gates. Es sagt, *was* gilt, nicht *wer* es entscheidet und *wann* es blockiert.

## SpecDD als Klammer

SpecDD ist das Paradigma über allen drei Werkzeugen:

```text
Spec → Plan → Code → Test
```

statt

```text
Prompt → Code → Hoffen
```

Alle Artefakte werden erzeugt, bevor implementiert wird. Die Aufteilung, die sich daraus ergibt:

- **SpecDD** verantwortet **System-Intent**, **Architektur**, **Governance/Security**.
- **Spec Kit** und **OpenSpec** verantworten **Feature-Intent**, **Feature-Planung** und **Feature-Implementierung**.
- Für **Drift Control** sind beide zuständig — und in dieser Rolle sind sie nur so gut wie die Prüfung, die sie erzwingt.

Diese Notiz beschreibt den Entwurf. Was davon in `sdd` tatsächlich mechanisch durchgesetzt wird — und was bewusst Prompt-Text in den Abläufen bleibt —, steht in [ADR 0009](../adr/0009-bmad-import-statt-zweitem-prozess.md) und im Abschnitt „Diskussion" der [README](../../README.md).

## Wann sich welche Kombination lohnt

- **Voller Aufbau (BMAD + Spec Kit/`sdd`):** komplexe Greenfield-Projekte, Multi-Service-Architekturen, Teams, die am Anfang tiefe fachliche und architektonische Klarheit brauchen, im Alltag aber schnelle, token-effiziente Iterationen bevorzugen.
- **Bestand (OpenSpec + `sdd adopt`):** gewachsene Repositories ohne verlässliche Spezifikation. Änderung für Änderung, mit Baseline und schrumpfendem Schuldenstand.
- **Nur die Feature-Strecke:** kleine, isolierte Änderungen — hier ist ein Makro-Framework Overkill; eine Spec, ein Gate und ein Test genügen.
