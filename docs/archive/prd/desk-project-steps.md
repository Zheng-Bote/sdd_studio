---
feature: desk-project-steps
status: archived
archive-after:
touches:
  - apps/desk/src/**
  - internal/desk/**
  - handbuch/**
  - CHANGELOG.md
release: v0.7.0
---

# PRD — Projektschritte in der Seitenleiste des KI-Desktops

## Problem

`sdd next` kennt neben den Schritten je Feature drei Projektschritte: „Verfassung ausfüllen“, „Übernahme abschließen“ und „Architekturdoku anlegen/nachführen“. Die API des Desktops liefert sie, die Seitenleiste zeigt sie aber nicht. Wer nur im Desktop arbeitet, erfährt deshalb nicht, dass die Verfassung noch Platzhalter hat, eine Übernahme nicht gemergt ist oder die Architekturdoku hinter dem letzten Release steht.

## Ziel

Die Seitenleiste zeigt offene Projektschritte im Prozessstand, in derselben Reihenfolge wie `sdd next`.

## Nutzer

- **Nutzer des KI-Desktops**, die nicht zusätzlich `sdd next` im Terminal aufrufen.

## Umfang

- Die Anzeige der drei Projektschritte in der Seitenleiste.
- Handbuch und CHANGELOG.

## Nicht-Ziele

- Neue Projektschritte oder Änderungen an `sdd next`.
- Eine Schaltfläche, die aus dem Schritt einen Ablauf startet.

## Akzeptanzkriterien

- **AC-1** Der Prozessstand zeigt vorhandene Projektschritte vor dem Einstieg und den Features, in der Reihenfolge Verfassung, Übernahme, Architekturdoku, je mit Zustand, Titel, Erläuterung und Befehl.
- **AC-2** Liefert `sdd next` einen Projektschritt nicht, erscheint für ihn nichts.
- **AC-3** Das Handbuch nennt die Projektschritte in der Seitenleiste.
