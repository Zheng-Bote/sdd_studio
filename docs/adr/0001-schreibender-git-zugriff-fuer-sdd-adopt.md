---
status: accepted
date: 2026-10-20
supersedes: []
references:
  - constitution.md
---

# ADR 0001 — Schreibender Git-Zugriff für sdd adopt

## Kontext

Bis Version 1 greift `sdd` ausschließlich lesend auf Git zu (`rev-parse`, `diff`); geschrieben wird nur über den Plan/Apply-Pfad in den Arbeitsbaum. Der Adopt-Ablauf des Konzepts (Abschnitt 6) verlangt jedoch: alles auf einem Branch, ein Commit pro Schritt, jeder Schritt einzeln rücknehmbar, der Hauptzweig bleibt unberührt. Überlässt man Branch und Commits dem Nutzer, landen die Schritte erfahrungsgemäß in einer Sammeländerung — genau das, was der Ablauf verhindern soll.

## Entscheidung

`sdd adopt --apply` darf schreibend auf Git zugreifen, und zwar ausschließlich so:

1. Nur in einem Git-Repository mit sauberem Arbeitsbaum (keine geänderten, gestagten oder unversionierten Dateien außerhalb von `exclude`).
2. Zuerst wird ein **neuer** Branch angelegt und ausgecheckt; existiert er bereits, bricht der Lauf ab, ohne etwas zu schreiben.
3. Jeder Schritt wird als eigener Commit festgehalten, der genau die Dateien dieses Schritts enthält. Commit-Autor ist die Git-Identität des Nutzers; Hooks laufen normal.
4. Kein `push`, kein `merge`, kein `rebase`, kein `reset`, kein Löschen von Branches. Der Ausgangsbranch wird nie verändert.

Alle übrigen Befehle bleiben Git-lesend.

## Konsequenzen

- Die Verfassung erhält eine Security-Regel, die diese Grenzen festschreibt.
- Ein abgebrochener Lauf hinterlässt höchstens einen Branch mit den bis dahin fertigen Commits; der Bericht nennt den letzten erfolgreichen Schritt.
- Tests für den Schreibpfad laufen gegen echte temporäre Git-Repositories, nicht gegen Fakes, weil die Zusicherungen (sauberer Baum, ein Commit pro Schritt, Ausgangsbranch unverändert) nur dort beweisbar sind.
