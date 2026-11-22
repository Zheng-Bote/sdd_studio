---
feature: mcpclient-lifecycle
status: active
archive-after:
---

# Plan — Robustes Ressourcenmanagement und idempotenter Lifecycle im MCPClient (`mcpclient-lifecycle`)

Reihenfolge der Umsetzung für [Issue #23](https://github.com/Zheng-Bote/sdd_studio/issues/23). Jede Task nennt das Akzeptanzkriterium, auf das sie einzahlt.

## Tasks

- [x] AC-1 In `internal/desk/mcpclient.go` (`MCPClient.Call`) `c.pending[id]` per `defer` aufräumen und `time.NewTimer` mit `defer timer.Stop()` einsetzen.
- [x] AC-2, AC-3 In `internal/desk/mcpclient.go` (`MCPClient.Close`) `sync.Once` (`closeOnce`) verwenden und nach `c.cmd.Process.Kill()` auf `<-done` warten.
- [x] AC-1, AC-2, AC-3 Unit-Tests in `internal/desk/desk_test.go` ergänzen (`TestMCPClientCallTimeoutCleansPending`, `TestMCPClientCloseIdempotentAndReapsProcess`).
- [x] AC-1, AC-2, AC-3 `CHANGELOG.md` aktualisieren sowie `sdd gate pre-merge --feature mcpclient-lifecycle` und `go test ./...` ausführen.
