---
feature: qa-trace
status: archived
archive-after:
touches:
  - internal/sddcore/rules/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - handbuch/**
  - CHANGELOG.md
release: v0.7.0
---

# Tech Spec — `sdd_trace` verfolgt Qualitätsanforderungen

Setzt das PRD `docs/archive/prd/qa-trace.md` um, auf Grundlage von ADR 0007.

## Architektur

```text
internal/sddcore/rules/quality.go   QATrace, TraceQuality(ctx, feature)
internal/sddcore/guide/tools.go     TraceQuality(root, feature)
internal/mcp/tools.go               sdd_trace: Feld quality, Beschreibung
```

`TraceQuality` geht die aktiven PRDs des Features durch. Der Abschnitt beginnt bei einer Überschrift mit „Qualitätsanforderungen“ und endet bei der nächsten Überschrift gleicher oder höherer Ebene. Gestrichene Kennungen werden vor der Suche entfernt, die erste Kennung einer Listenzeile definiert den Eintrag. Tech Specs werden wie bei `Trace` je Datei einmal genannt. Testzeilen brauchen immer den Slug, weil `QA-1` in jedem Feature vorkommt.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `quality.go`, `tools.go` | `TestTraceQuality`, `TestTraceToolQuality` |
| AC-2 | `quality.go` | `TestTraceQuality` |
| AC-3 | `quality.go` | `TestTraceQuality` |
| AC-4 | `tools.go`, Handbuch | `TestTraceToolQuality` |
