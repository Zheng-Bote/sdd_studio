---
feature: next-draft-hint
status: archived
archive-after:
touches:
  - internal/sddcore/guide/**
  - handbuch/**
  - CHANGELOG.md
release: v0.7.0
---

# Tech Spec — `sdd next` nennt ein PRD im Entwurf vor dem Archivieren

Setzt das PRD `docs/archive/prd/next-draft-hint.md` um.

## Architektur

```text
internal/sddcore/guide/next.go   featureGuide, Schritt 10: Hinweis aus set.prds mit status: draft
handbuch/03-mechanismen.md       Zeile „Abschluss“ der Schritttabelle
```

Aus `set.prds` entsteht die Liste der Pfade mit `Frontmatter.Status == "draft"`. Ist sie nicht leer, wird der Erläuterung beider Varianten des Abschlussschritts `<pfad> hat noch status: draft; sdd archive lehnt Entwürfe ab, Status nach der Umsetzung setzen. ` vorangestellt. `State` und `Command` bleiben.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `next.go` | `TestNextNamesDraftPRD` |
| AC-2 | `next.go` | `TestNextNamesDraftPRD` |
| AC-3 | `handbuch/03-mechanismen.md` | `TestNextDraftHintDocumented` |
