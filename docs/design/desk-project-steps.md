---
feature: desk-project-steps
status: active
archive-after:
touches:
  - apps/desk/src/**
  - internal/desk/**
  - handbuch/**
  - CHANGELOG.md
---

# Tech Spec — Projektschritte in der Seitenleiste des KI-Desktops

Setzt das PRD `docs/prd/desk-project-steps.md` um. Das Backend bleibt unverändert: `/api/next` liefert `constitution`, `adoption` und `architecture` bereits im JSON von `sdd_next`.

## Architektur

```text
apps/desk/src/app/core/types.ts              Guide: constitution?, adoption?, architecture?
apps/desk/src/app/process/process-panel.ts   computed projectSteps; Block vor Einstieg und Features
handbuch/04-agenten.md                       Absatz „Im Gespräch“
```

`projectSteps` ist `[guide.constitution, guide.adoption, guide.architecture]` ohne leere Einträge. Je Schritt zeigt die Seitenleiste Badge (Zustand), Titel, Erläuterung und den Befehl als `code`.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `process-panel.ts`, `types.ts` | `process-panel.spec.ts` |
| AC-2 | `process-panel.ts` | `process-panel.spec.ts` |
| AC-3 | `handbuch/04-agenten.md` | `TestHandbookProjectSteps` |
