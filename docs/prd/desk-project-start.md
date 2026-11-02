---
feature: desk-project-start
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
---

# PRD — Nicht erfasste Projekte im KI-Desktop erkennen und starten

## Problem

Öffnet man im KI-Desktop ein Projekt ohne `sdd.config.yaml`, merkt das Modell es erst an einem Werkzeugfehler („keine Projektkonfiguration gefunden“). Der Nutzer muss selbst wissen, dass er ein bestehendes Projekt mit `sdd adopt` übernimmt oder ein neues mit `sdd init` anlegt, und in welcher Reihenfolge Übernahme, Agenten-Dateien, Verfassung und erstes PRD folgen.

## Ziel

Der Desktop erkennt beim Öffnen, ob das Projekt mit sdd erfasst ist. Wenn nicht, schlägt er den passenden Einstieg vor: einen Ordner mit Inhalt übernehmen oder einen leeren Ordner neu anlegen. Er führt in der richtigen Reihenfolge bis zur Verfassung.

## Nutzer

- **Nutzer des KI-Desktops**, die ein Projekt zum ersten Mal mit sdd öffnen.

## Umfang

- `sdd_stack` zählt alle Dateien.
- Der Projektstatus in den Metadaten des Desktops.
- Ein Hinweis mit Schaltfläche im Gesprächsbereich.
- Die Prompts `sdd-adopt` und `sdd-start`.
- Die Server-Anweisungen.
- Handbuch und README.

## Nicht-Ziele

- Etwas automatisch ausführen. Der Hinweis legt nur den geführten Ablauf ins Eingabefeld; jeder Werkzeugaufruf braucht wie bisher eine Freigabe.
- Die Erkennung in `sdd ui`.
- Branches mergen. Das macht der Mensch nach der Übernahme.

## Akzeptanzkriterien

- **AC-1** `sdd_stack` nennt zusätzlich die Zahl aller gezählten Dateien (ohne `.git/`, `.sdd/` und `exclude`).
- **AC-2** Die Metadaten des Desktops (`/api/meta`) nennen den Projektstatus: `configured` mit `sdd.config.yaml`, sonst `existing`, wenn es Dateien gibt, sonst `empty`. Der Desktop ermittelt ihn selbst über einen lesenden Aufruf von `sdd_stack`, ohne das Modell. Scheitert der Aufruf, fehlt der Status.
- **AC-3** Ist das Projekt nicht erfasst, zeigt der Gesprächsbereich einen Hinweis. Bei `existing` steht darin die Schaltfläche „Bestehendes Projekt übernehmen“, bei `empty` „Neues Projekt anlegen“. Die Schaltfläche legt den Prompt `sdd-adopt` bzw. `sdd-start` ins Eingabefeld, ohne ihn zu senden.
- **AC-4** `sdd-adopt` führt so: `sdd_stack`, dann der Bericht von `sdd_adopt_apply` ohne `apply` mit Erklärung (Reifegrad, Schritte, Konflikte). Nach Zustimmung `sdd_adopt_apply` mit `apply` (kritisch, einzeln bestätigt). Danach `sdd-constitution` auf dem Branch der Übernahme, vor dem Merge, und der Hinweis, den Branch zu committen und zu mergen. `sdd-start` führt so: `sdd_init` mit `git: true`, dann die Technik klären (`sdd_stack`, Vorschlag mit Begründung, ADR anbieten), dann `sdd-constitution`, dann `sdd-idea`. Die Server-Anweisungen nennen beide Abläufe für ein Projekt ohne Konfiguration.
