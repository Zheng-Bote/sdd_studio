---
feature: sdd-desk-turn-timeout
status: archived
release: v0.4.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - config/**
  - handbuch/**
---

# Tech Spec — Einstellbare Frist für einen Gesprächszug in `sdd-desk`

Setzt das PRD `docs/archive/prd/sdd-desk-turn-timeout.md` um. Grundlage ist die Tech Spec `docs/archive/design/sdd-desk.md` (Konfiguration, HTTP-API).

## Architektur

```text
internal/desk/config.go   Provider.TurnTimeout; <PREFIX>_TURN_TIMEOUT lesen, prüfen, Standardwerte
internal/desk/server.go   Server.turnTimeout(providerID) statt der Konstante turnTimeout
cmd/sdd-desk/main.go      HTTP-Client der Anbieter ohne Timeout (die Frist des Zuges gilt)
config/llm.env.example    Eintrag beschrieben
```

`Provider` erhält `TurnTimeout time.Duration` (`json:"-"`). `LoadConfig` setzt ihn für jeden Anbieter: aus `value(prefix + "_TURN_TIMEOUT")` per `time.ParseDuration`, gültig von `1m` bis `4h`; sonst `30m` für `LocalOllamaID`, `10m` für Cloud-Anbieter. Ein ungültiger Wert ergibt die Warnung `<Name>: <PREFIX>_TURN_TIMEOUT „<wert>“ ungültig (1m bis 4h); es gilt <standard>`.

`Server.turnTimeout(id)` liefert die Frist des Anbieters aus `Config.Providers`, für einen unbekannten Anbieter `10m` — `Agent.Check` lehnt ihn ohnehin ab. `handleTurn` und `handleTurnStream` nutzen sie für `context.WithTimeout`.

Der HTTP-Client für die Anbieter in `main.go` verliert `Timeout: 10 * time.Minute`; jede Anfrage läuft im Kontext des Zuges (bzw. des Modelltests mit eigener Frist) und endet mit ihm oder mit „Abbrechen“.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.go` (Standardwerte) | `TestTurnTimeoutConfig` |
| AC-2 | `config.go` (Lesen, Grenzen, Warnung) | `TestTurnTimeoutConfig` |
| AC-3 | `server.go` (`turnTimeout`), `main.go` | `TestTurnUsesProviderTimeout` |
