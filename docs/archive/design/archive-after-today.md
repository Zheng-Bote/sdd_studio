---
feature: archive-after-today
status: archived
archive-after:
touches:
  - internal/desk/**
  - internal/sddcore/scaffold/**
  - handbuch/**
release: v0.5.0
---

# Tech Spec — Heutiges Datum für das Modell, kein Stichtag in der Vergangenheit

Setzt das PRD `docs/archive/prd/archive-after-today.md` um. Grundlage sind die Tech Specs `docs/archive/design/sdd-desk.md` (Agent, Gesprächszug) und `docs/archive/design/sdd-init-new.md` (`sdd new`).

## Architektur

```text
internal/desk/agent.go           Agent.Now; systemFor() hängt das Datum an den Systemtext
internal/sddcore/scaffold/new.go featurePlan: archive-after vor opts.Today ablehnen
handbuch/03-mechanismen.md       sdd new: Stichtag nicht in der Vergangenheit
```

`Agent` erhält `Now func() time.Time` (nil = `time.Now`, damit `cmd/sdd-desk` unverändert bleibt). Zu Beginn eines Zuges bildet `systemFor()` einmal den Systemtext `a.System + "\n\nHeutiges Datum: YYYY-MM-DD."`. Alle Modellaufrufe dieses Zuges verwenden ihn, `Complete` wie `Stream`. Der Modelltest („LLM testen“) behält seinen eigenen, festen Systemtext.

`featurePlan` prüft nach dem Parsen von `archive-after`: Ist `opts.Today` gesetzt und liegt das Datum vor dem Kalendertag von `opts.Today` (lokal), folgt `ErrInvalidInput` mit `archive-after <wert> liegt vor dem heutigen Datum <heute>`. Der Vergleich ist derselbe wie in SDD-050 (`date.Before(today)`), deshalb ist der heutige Tag erlaubt. CLI (`time.Now()`), `sdd ui` und `sdd mcp` (`s.Now()`) übergeben `Today` bereits und brauchen keine Änderung.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `agent.go` (`Now`, `systemFor`) | `TestTurnSystemNamesToday` |
| AC-2 | `scaffold/new.go` (`featurePlan`) | `TestNewRejectsPastArchiveAfter` |
