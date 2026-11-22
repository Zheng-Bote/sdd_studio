---
feature: mcpclient-lifecycle
status: active
archive-after:
touches:
  - internal/desk/**
  - CHANGELOG.md
---

# Tech Spec — Robustes Ressourcenmanagement und idempotenter Lifecycle im MCPClient (`mcpclient-lifecycle`)

Setzt das PRD `docs/prd/mcpclient-lifecycle.md` ([Issue #23](https://github.com/Zheng-Bote/sdd_studio/issues/23)) um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

Alle Änderungen liegen in `internal/desk/mcpclient.go` und den zugehörigen Tests in `internal/desk/desk_test.go`. `internal/sddcore`, `internal/mcp` und `cmd/sdd` werden nicht berührt.

```text
internal/desk/mcpclient.go   MCPClient.Call (defer delete + time.NewTimer),
                             MCPClient.Close (sync.Once + <-done nach Kill)
internal/desk/desk_test.go   Unit-Tests für AC-1, AC-2 und AC-3
```

### 1. Bereinigung von `c.pending` und Timer-Ressourcen in `MCPClient.Call`

Nach der Registrierung `c.pending[id] = ch` stellt ein `defer`-Block sicher, dass `id` beim Verlassen von `Call` immer aus `c.pending` entfernt wird:

```go
defer func() {
    c.mu.Lock()
    delete(c.pending, id)
    c.mu.Unlock()
}()
```

Zusätzlich wird `time.After(c.Timeout)` durch `timer := time.NewTimer(c.Timeout)` mit `defer timer.Stop()` ersetzt, damit der Timer bei sofortiger Antwort nicht bis zum Ablauf von `c.Timeout` im Go-Runtime-Heap verbleibt.

### 2. Idempotentes `MCPClient.Close()` mit Prozess-Reaping

`MCPClient` erhält ein Feld `closeOnce sync.Once` (sowie eine konfigurierbare Schließfrist `closeTimeout`, Standard `3 * time.Second`, damit Tests den Kill-Pfad schnell prüfen können):

```go
func (c *MCPClient) Close() error {
    c.closeOnce.Do(func() {
        if closer, ok := c.w.(io.Closer); ok {
            _ = closer.Close()
        }
        if c.cmd != nil {
            done := make(chan error, 1)
            go func() { done <- c.cmd.Wait() }()
            timeout := c.closeTimeout
            if timeout <= 0 {
                timeout = 3 * time.Second
            }
            timer := time.NewTimer(timeout)
            defer timer.Stop()
            select {
            case <-done:
            case <-timer.C:
                if c.cmd.Process != nil {
                    _ = c.cmd.Process.Kill()
                }
                <-done
            }
        }
    })
    return nil
}
```

## Schnittstellen

Die öffentliche Signatur von `MCPClient` (`StartMCP`, `NewMCPClient`, `Call`, `Initialize`, `Tools`, `CallTool`, `Close`) bleibt unverändert.

## Datenmodell

Erweiterung der internen Struktur `MCPClient` in `internal/desk/mcpclient.go` um:
- `closeOnce sync.Once`
- `closeTimeout time.Duration` (optional für Tests; `0` nutzt die Vorgabe `3 * time.Second`)

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `MCPClient.Call` entfernt `id` per `defer` aus `c.pending` und stoppt den Timeout-Timer per `defer timer.Stop()`. | `internal/desk/desk_test.go` `TestMCPClientCallTimeoutCleansPending` |
| AC-2 | `MCPClient.Close` kapselt das Schließen von `c.w` und das Warten auf `c.cmd` in `c.closeOnce.Do(...)`. | `internal/desk/desk_test.go` `TestMCPClientCloseIdempotentAndReapsProcess` |
| AC-3 | `MCPClient.Close` wartet nach `c.cmd.Process.Kill()` im Timeout-Zweig auf `<-done` (`cmd.Wait()`). | `internal/desk/desk_test.go` `TestMCPClientCloseIdempotentAndReapsProcess` |

## Sicherheit

- **Verfassung Security #1 (Offline & Prozessisolation):** Verbessert die Zuverlässigkeit der Prozessisolation zwischen `sdd-desk` und `sdd mcp` (ADR 0002), verhindert Speicher-Leaks bei blockiertem Kindprozess und vermeidet verwaiste Kindprozesse.

## Offene Punkte

- Keine offenen Punkte.
