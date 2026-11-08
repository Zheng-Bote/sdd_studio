---
feature: qa-trace
status: active
archive-after:
touches:
  - internal/sddcore/rules/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - handbuch/**
  - CHANGELOG.md
---

# PRD — `sdd_trace` verfolgt Qualitätsanforderungen

## Problem

Seit ADR 0007 kann ein PRD messbare Qualitätsanforderungen `QA-<n>` enthalten. `sdd_trace` kennt aber nur Akzeptanzkriterien. Ob eine Qualitätsanforderung in der Tech Spec aufgegriffen oder durch einen Test belegt ist, muss man von Hand suchen, und doppelt vergebene Kennungen fallen niemandem auf.

## Ziel

`sdd_trace` zeigt je `QA-<n>` eines Features, wo sie aufgegriffen wird, und macht doppelte Kennungen sichtbar.

## Nutzer

- **Nutzer des KI-Desktops und Agenten**, die den Stand eines Features prüfen.
- **Reviewer**, die eine Qualitätsanforderung zurückverfolgen wollen.

## Umfang

- Ein zusätzliches Feld im Ergebnis von `sdd_trace`.
- Handbuch und CHANGELOG.

## Nicht-Ziele

- Eine Katalogregel oder eine Pflicht für Tech-Spec- oder Testverweise. ADR 0007 bleibt gültig: `QA-<n>` braucht keinen Testverweis.
- Auswertung in `sdd next` oder in den Gates.

## Akzeptanzkriterien

- **AC-1** Das Ergebnis von `sdd_trace` enthält neben `criteria` eine Liste `quality`, immer als Liste. Sie nennt je Kennung im Abschnitt „Qualitätsanforderungen“ des PRDs Kennung, PRD, Zeile und Text. Als ~~QA-n~~ gestrichene Kennungen und Nennungen außerhalb des Abschnitts zählen nicht.
- **AC-2** Je Kennung stehen die Tech Specs des Features, die sie nennen, und die Testzeilen (`Datei:Zeile`), die Kennung und Feature-Slug zusammen nennen.
- **AC-3** Kommt eine Kennung im Abschnitt mehrfach vor, trägt ihr Eintrag `duplicate: true`.
- **AC-4** `criteria` bleibt unverändert, und das Handbuch beschreibt `quality`.
