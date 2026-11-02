---
feature: flow-closing
status: archived
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/sddcore/scan/**
  - internal/mcp/**
  - cmd/sdd/**
  - apps/desk/src/**
  - handbuch/**
release: v0.5.0
---

# PRD — Klarer Abschluss geführter Abläufe

## Problem

Im Projekt Check_Cert hat der Ablauf `sdd-constitution` die Verfassung gespeichert, geprüft und dann `sdd_next` aufgerufen. Danach kam keine Antwort mehr. Sichtbar blieb das rohe Ergebnis von `sdd_next`: „Idee als PRD festhalten“. Unklar blieb, was gespeichert wurde, was das Prüfergebnis bedeutet und was jetzt zu tun ist. Der eigentlich nächste Schritt, die Verfassung auf dem Branch der Übernahme committen und den Branch mergen, wurde nirgends genannt.

Drei Ursachen:
1. Eine Antwort des Modells ohne Text zeigt der Desktop nicht an.
2. Die Prompts verlangen keinen Abschluss in Worten.
3. `sdd next` erkennt eine noch nicht abgeschlossene Übernahme nicht.

## Ziel

Jeder geführte Ablauf endet mit einer verständlichen Zusammenfassung. Ein Gespräch, das ohne Antwort endet, ist als solches erkennbar. Nach einer Übernahme nennt `sdd next` als Nächstes das Abschließen der Übernahme, nicht das erste PRD.

## Nutzer

- **Nutzer des KI-Desktops.**
- **Wer `sdd next` nach `sdd adopt --apply` aufruft.**

## Umfang

- Ein Grundsatz zum Abschluss in den Prompts.
- Der Schluss von `sdd-constitution`.
- Ein Hinweis im Desktop auf eine fehlende Schlussantwort.
- Der Schritt „Übernahme abschließen“ in `sdd next`.
- Das Handbuch.

## Nicht-Ziele

- Das Modell erneut auffordern oder eine Antwort erzwingen.
- Committen oder Mergen durch sdd.
- Eine Übernahme erkennen, deren Branch umbenannt wurde.

## Akzeptanzkriterien

- **AC-1** Jeder geführte Prompt verlangt, den Ablauf mit einer kurzen Zusammenfassung in Worten zu beenden: gespeicherte Dateien mit Pfad, Prüfergebnis in Klartext, offene Punkte und der nächste Schritt. Nie mit einem bloßen Werkzeugaufruf. `sdd-constitution` nennt nach dem Speichern das Committen und, nach einer Übernahme, das Prüfen und Mergen des Branches.
- **AC-2** Endet eine Antwort des Modells ohne Text und ohne Werkzeugaufruf, zeigt der Desktop an ihrer Stelle „Das Modell hat ohne abschließende Antwort aufgehört. Fragen Sie nach, z. B. „Was ist der nächste Schritt?““.
- **AC-3** Ist `.sdd/adopt-report.md` vorhanden, nennt er Branch und Ausgangsbranch der Übernahme, ist der aktuelle Branch der Branch der Übernahme und ist `HEAD` noch nicht im Ausgangsbranch enthalten, dann meldet `sdd next` (CLI und `sdd_next`) den Schritt „Übernahme abschließen“. Er steht nach „Verfassung ausfüllen“ und vor allen anderen Schritten und nennt Commit, `git diff <Ausgang>...<Branch>` und den Merge. Sonst, auch ohne Git, entfällt er.
