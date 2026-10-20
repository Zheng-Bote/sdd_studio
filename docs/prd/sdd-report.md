---
feature: sdd-report
status: active
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - action.yml
  - handbuch/**
---

# PRD — Projektbericht und Schuldenstand (`sdd-report`)

## Problem

Die Baseline macht Bestandsprojekte prüfbar, verdeckt aber den Altbestand: Ein Gate ist grün, obwohl hundert Befunde unterdrückt sind. Ohne sichtbaren Schuldenstand und Trend wird die Baseline zur Decke statt zur Ratsche. Außerdem ist der Reifegrad bisher nur beim einmaligen `sdd adopt` sichtbar, nicht im laufenden Betrieb.

## Ziel

`sdd report` zeigt jederzeit, wo ein Projekt steht: Reifegrad mit nächstem Schritt, offene Befunde, Status der drei Gates und den Schuldenstand mit Veränderung gegenüber dem Vergleichsstand — lokal und in jedem CI-Lauf.

## Nutzer

- **Teams**, die ihre Baseline abbauen und den Fortschritt sehen wollen.
- **Reviewer**, die in einem Pull Request erkennen müssen, ob die Baseline gewachsen ist.

## Umfang

Befehl `sdd report` mit Markdown- und JSON-Ausgabe sowie die Ausgabe des Berichts in die Job-Zusammenfassung der GitHub Action.

## Nicht-Ziele

- Kommentare im Pull Request über die GitHub-API (bräuchte Schreibrechte und ein Token).
- Eine Historie über mehr als einen Vergleichsstand oder Diagramme.
- Neue Regeln: Ein Wachstum der Baseline wird hervorgehoben, aber nicht als Regelbefund gemeldet.

## Akzeptanzkriterien

- **AC-1** `sdd report` zeigt den Reifegrad L0–L4 mit nächstem Schritt, die offenen Befunde von `sdd lint` nach Schweregrad und für jedes der drei Gates die Anzahl der `error`-, `warn`- und baselineten Befunde.
- **AC-2** Der Schuldenstand nennt die Baseline-Einträge je Regel und Gate sowie die Veränderung gegenüber der Baseline am Vergleichsstand (`--since <ref>`, sonst die Merge-Basis); der Vergleichsstand wird ausschließlich lesend aus Git gelesen.
- **AC-3** Enthält die aktuelle Baseline einen Eintrag, den der Vergleichsstand nicht enthielt, hebt der Bericht das als Verstoß gegen die Regel „die Baseline darf nur schrumpfen“ hervor und nennt die neuen Einträge.
- **AC-4** `sdd report --format json` liefert dieselben Inhalte maschinenlesbar; ein nicht ausführbares Gate oder ein nicht lesbarer Vergleichsstand erscheint im Bericht mit Grund, statt ihn abzubrechen.
- **AC-5** Die GitHub Action hängt den Bericht an die Job-Zusammenfassung an, auch wenn das Gate fehlschlägt, und verändert dessen Exit-Code nicht.
