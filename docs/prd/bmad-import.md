---
feature: bmad-import
status: active
archive-after:
touches:
  - internal/sddcore/config/**
  - internal/sddcore/scan/**
  - internal/sddcore/adopt/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/sddcore/rules/**
  - internal/sddcore/report/**
  - internal/mcp/**
  - internal/desk/**
  - cmd/sdd/**
  - apps/desk/src/**
  - apps/desk/src-tauri/**
  - apps/desk/package.json
  - apps/desk/package-lock.json
  - catalog/**
  - templates/**
  - docs/**
  - handbuch/**
  - README.md
  - CHANGELOG.md
  - .gitignore
---

# PRD — BMAD-Bestand übernehmen und als Strecke führen

## Problem

Ein Projekt, das mit BMAD beginnt, hat nach wenigen Tagen eine belastbare Makro-Ebene: eine Initiative mit nummerierten Anforderungen (`R<n>`), Constraints und acht Epics mit prüfbaren „Done when“-Punkten. `sdd` erkennt davon nichts. `sdd adopt` klassifiziert jede dieser Dateien als „Doku/Exploration“ — „kein Muster“ —, legt ein leeres Verfassungsgerüst an und lässt den inhaltlichen Bestand liegen.

Damit fehlt genau der Teil, den die Kombination aus BMAD, Spec Kit und SpecDD verspricht: Die Makro-Ebene bleibt unverbunden mit der Verfassung, und nichts prüft, ob ein Epic noch eine Anforderung deckt. Der Drift, den die Kombination verhindern soll, entsteht an der Naht zwischen zwei Werkzeugen.

Zusätzlich ist die Oberfläche zu abstrakt: `sdd next` und die Seitenleiste des KI-Desktops kennen nur Features. Ein Vorhaben, das aus Initiative, Epics und Architektur besteht, hat keinen sichtbaren Ort.

## Ziel

`sdd` erkennt einen BMAD-v6-Bestand, übernimmt ihn verlustfrei in seine Artefakttypen und führt ihn als Strecke: Makro-Ebene vor Feature-Ebene, mit maschinell geprüfter Rückverfolgung von Anforderung zu Epic.

## Nutzer

- **Einzelentwickler mit BMAD-Vorgeschichte**: hat Initiative und Epics, braucht Verfassung, Gates und Drift-Kontrolle, ohne den Bestand neu zu schreiben.
- **Nutzer des KI-Desktops**: will sehen, wo im Vorhaben er steht, nicht nur in welchem Feature.

## Umfang

- Profile `pipeline: bmad` in `sdd.config.yaml`, rückwärtskompatibel (`""` = bisheriges Verhalten).
- Artefakttypen **Initiative** und **Epic** in Konfiguration, Erkennung, Frontmatter und Gerüst.
- Erkennung in `sdd adopt`: `_bmad/` und `_bmad-output/` (BMAD v6), Ausschluss der Werkzeugbäume.
- `sdd init --pipeline bmad` legt Initiative und Epic-Ordner an.
- `sdd next` und die Seitenleiste des Desktops zeigen zwei Etagen: Vorhaben und Features.
- Drei Katalogregeln `SDD-080` bis `SDD-082`.
- Geführte Abläufe `sdd-ingest-bmad`, `sdd-architecture`, `sdd-epic-refine`.

## Nicht-Ziele

- BMAD installieren, ausführen oder ersetzen; `.bmad-core/` (v4) wird nicht unterstützt.
- Stories, Sprints oder Personas als eigene Artefakttypen; Rollen bleiben Prompt-Text.
- Eine zweite Verfassung; es gilt allein die von `sdd`.
- Die `_bmad-output/`-Dateien umschreiben; der Import kopiert und lässt BMADs Ledger unangetastet.
- Eine Regel „Feature nennt sein Epic“ — die braucht erst die Story-Ebene.
- Eine Regel für Architektur-Abdeckung; sie ist nicht deterministisch prüfbar und bleibt ADR-Pflicht aus `SDD-010`.
- LLM-Code im Kern: die inhaltliche Übernahme läuft über `sdd mcp`/Desktop (ADR 0003).

## Akzeptanzkriterien

- **AC-1** `sdd.config.yaml` kennt `pipeline` mit den Werten `""` (Vorgabe, bisheriges Verhalten), `sdd` und `bmad`; jeder andere Wert bricht die Prüfung mit einer Meldung ab, die die erlaubten Werte nennt. `artifacts.initiative` ist konfigurierbar; fehlt es, gilt `docs/initiative.md`.
- **AC-2** Das Frontmatter eines Artefakts trägt `type`, `parent`, `covers` und `refined`; sie werden gelesen, ohne die bestehenden Felder zu verändern.
- **AC-3** `sdd adopt` erkennt in einem BMAD-v6-Bestand die Initiative als Typ Initiative und jede `epic-*.md` als Typ Epic, auch wenn die Dateien unterhalb der Standardpfade für PRD, Tech Spec oder Plan liegen. `_bmad/**` und `**/.memlog.md` erscheinen nicht im Inventar. Ohne `_bmad/` bleibt das Verhalten unverändert.
- **AC-4** `sdd adopt` vergibt an Makro-Artefakte kein `feature:`-Frontmatter und meldet keinen Konflikt dafür; vorhandenes Frontmatter bleibt unverändert.
- **AC-5** `sdd init --pipeline bmad` schreibt `pipeline: bmad`, legt die Initiative aus einem Template und den Epic-Ordner an. `sdd new --type initiative|epic` legt beide an, der Epic-Pfad wird aus `artifacts.prd` abgeleitet.
- **AC-6** `sdd next` meldet die Makrostationen vor den Features; ein Epic mit `refined: false` erscheint als eigener Zustand („noch nicht verfeinert“) und löst nicht die Meldung „PRD fehlt“ aus. Ohne BMAD-Bestand bleibt die Ausgabe unverändert.
- **AC-7** Die Seitenleiste des Desktops trennt „Vorhaben“ und „Features“ und zeigt je Etage den nächsten Schritt.
- **AC-8** `SDD-080` meldet ein Epic, dessen `covers` keine existierende Anforderung `R<n>` der Initiative nennt. `SDD-081` meldet eine Anforderung, die kein Epic deckt und die nicht als nicht geplant markiert ist. `SDD-082` meldet eine Abweichung zwischen `tickets.toml` und den Epic-Dateien. Alle drei sind im Handbuch mit Prüfvertrag dokumentiert, im Prüfkern registriert und in mindestens einem Gate aktiv.
- **AC-9** Die geführten Abläufe `sdd-ingest-bmad`, `sdd-architecture` und `sdd-epic-refine` stehen in `sdd mcp` bereit; keiner von ihnen ändert ohne Bestätigung etwas.
