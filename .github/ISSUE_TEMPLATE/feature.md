---
name: Feature (SDD)
about: Ein Vorhaben beschreiben, das als SDD-Feature mit PRD und Plan umgesetzt wird.
title: ''
labels: 'feature'
assignees: ''
---

<!--
Diese Vorlage ist der Startpunkt, nicht das Artefakt. Das PRD (docs/prd/<slug>.md) ist
die geprüfte Fassung; dieses Issue hält Problem, Umfang und Kriterien fest, damit der
Feature-Slug von Anfang an denselben Namen trägt.
-->

## Feature-Slug

`<slug>` — derselbe Name für den Branch (`feature/<slug>`), das PRD (`docs/prd/<slug>.md`) und die Tests (`// <slug> AC-1: …`).

## Problem

Welches Problem wird gelöst, und für wen? Was ist heute nicht möglich, falsch oder unklar? Beobachtungen statt Vermutungen; wenn etwas gemessen wurde, die Messung nennen.

## Ziel

Was ist nach der Umsetzung anders? Ein Satz, der ohne die Lösung zu nennen prüfbar bleibt.

## Umfang

- Was gehört dazu?

## Nicht-Ziele

- Was ausdrücklich **nicht** dazugehört, und warum. (Pflicht: `SDD-023` prüft, dass das PRD Nicht-Ziele nennt — hier beginnt die Liste.)

## Akzeptanzkriterien

Prüfbare Sätze, im PRD als `AC-1`, `AC-2`, … mit stabilen Kennungen (nie umnummerieren, entfallene als ~~AC-n~~ markieren). Jedes Kriterium bekommt später eine Testzeile `// <slug> AC-<n>: …` (`SDD-021`, `SDD-022`).

- [ ] **AC-1** …
- [ ] **AC-2** …

## Betroffene Schichten

Pfade, die sich ändern sollen (`touches` im PRD). Bei `cmd/**`, `constitution.md` oder mindestens zwei Layern sind PRD **und** Tech Spec Pflicht (`SDD-020`/`SDD-030`).

- …

## Verfassung

Die Regeln aus [`constitution.md`](../../constitution.md) gelten ohne Ausnahme ([`AGENTS.md`](../../AGENTS.md)). Eine Änderung der Verfassung braucht ein ADR, das sie in `references` nennt (`SDD-010`).

## Arbeitsweise

Issue → Feature-Branch → Pull Request. Die PR-Beschreibung nennt `Closes #<n>` (englisch; GitHub erkennt deutsche Formulierungen nicht). Gemergt wird vom Menschen.
