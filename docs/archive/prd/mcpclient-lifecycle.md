---
feature: mcpclient-lifecycle
status: archived
archive-after:
touches:
  - internal/desk/**
  - CHANGELOG.md
release: v0.9.1
---

# PRD — Robustes Ressourcenmanagement und idempotenter Lifecycle im MCPClient (`mcpclient-lifecycle`)

## Problem

Im Rahmen des Code-Reviews ([Issue #23](https://github.com/Zheng-Bote/sdd_studio/issues/23)) wurden zwei Robustheits- und Ressourcenmanagement-Schwachstellen im MCP-Subprozess-Client (`internal/desk/mcpclient.go`) identifiziert:

1. **Memory-Leak in `c.pending` bei RPC-Timeout oder Schreibfehler (`RES-01`):** In `MCPClient.Call` wird für jeden Aufruf ein Rückkanal `ch` unter `c.pending[id]` registriert. Schlägt das anschließende Schreiben auf `c.w` fehl oder läuft der Aufruf in den Timeout (`time.After(c.Timeout)`), kehrt `Call` zurück, ohne `delete(c.pending, id)` auszuführen. Antwortet der Kindprozess auf diese Anfrage nicht mehr, verbleibt der Eintrag dauerhaft in `c.pending`. Zudem nutzt `time.After` einen Timer, der bei vorzeitiger Antwort erst nach Ablauf von `c.Timeout` (Standard 2 Minuten) freigegeben wird.
2. **Fehlende Idempotenz und Prozess-Reaping in `MCPClient.Close` (`RES-02`):** `Close()` schließt `c.w` und startet ohne Synchronisation eine Goroutine `go func() { done <- c.cmd.Wait() }()`. Wird `Close()` mehrfach aufgerufen (z. B. im Fehlerpfad und per `defer`), wird `c.cmd.Wait()` mehrfach auf demselben `*exec.Cmd` ausgeführt, was zu Race-Conditions bzw. Fehlern führt. Darüber hinaus kehrt `Close()` nach `c.cmd.Process.Kill()` im Timeout-Zweig sofort zurück, ohne den Abschluss von `<-done` (Prozess-Reaping) abzuwarten.

## Ziel

`MCPClient` verwaltet ausstehende RPC-Aufrufe und den Lebenszyklus des `sdd mcp`-Kindprozesses leckfrei, deterministisch und idempotent:

- Jeder in `c.pending` eingetragene Aufruf wird beim Verlassen von `Call` (egal ob durch Antwort, Fehler, Stream-Ende oder Timeout) garantiert aus `c.pending` entfernt, und der Timeout-Timer wird sofort gestoppt.
- `MCPClient.Close()` ist idempotent (`sync.Once`), führt die Terminierung des Kindprozesses höchstens einmal aus und wartet nach einem `Kill()` auf das Ende von `cmd.Wait()`, damit kein Zombie-Prozess verbleibt.

## Nutzer

- **Betreiber und Nutzer von `sdd-desk`** — profitieren von stabiler Speichernutzung bei langen Sitzungen und sauberem Beenden des `sdd mcp`-Subprozesses beim Projektwechsel oder Beenden des Desktops.

## Umfang

- Bereinigung von `c.pending[id]` via `defer` sowie Einsatz von `time.NewTimer` mit `defer timer.Stop()` in `MCPClient.Call` (`internal/desk/mcpclient.go`).
- Absicherung von `MCPClient.Close()` mittels `sync.Once` (`closeOnce`) und Abwarten von `<-done` nach `c.cmd.Process.Kill()`.
- Ergänzung von Unit-Tests in `internal/desk/desk_test.go` für Timeout-Cleanup (`AC-1`), idempotentes `Close()` (`AC-2`) und Prozess-Reaping nach `Kill` (`AC-3`).
- Dokumentation in `CHANGELOG.md`.

## Nicht-Ziele

- **Änderung des JSON-RPC-2.0-Protokolls oder der MCP-Methoden:** Das Drahtformat zwischen `sdd-desk` und `sdd mcp` bleibt unverändert.
- **Automatischer Neustart eines abgestürzten `sdd mcp`-Kindprozesses:** Ein beendeter Subprozess wird wie bisher über `c.closed` und `c.err` gemeldet.

## Akzeptanzkriterien

- **AC-1** Wenn ein RPC-Aufruf in `MCPClient.Call` in einen Timeout läuft oder beim Schreiben fehlschlägt, wird die Request-ID aus `c.pending` entfernt, sodass keine verwaisten Einträge in `c.pending` verbleiben.
- **AC-2** Wenn `MCPClient.Close()` mehrfach (auch nebenläufig) aufgerufen wird, wird die Schließ- und `cmd.Wait()`-Logik exakt einmal ausgeführt (`sync.Once`) und jeder Aufruf kehrt fehlerfrei zurück.
- **AC-3** Wenn der Kindprozess in `MCPClient.Close()` nach Ablauf der Schließfrist per `cmd.Process.Kill()` beendet werden muss, wartet `Close()` auf die Rückkehr von `cmd.Wait()`, sodass der Prozess vollständig geerntet (reaped) ist.
