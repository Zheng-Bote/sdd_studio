---
feature: clean-code-refactoring
status: active
archive-after:
---

# Plan — Clean Code: Toter Code, Defensives Dispatching und Deduplizierung (`clean-code-refactoring`)

Reihenfolge der Umsetzung für [Issue #24](https://github.com/Zheng-Bote/sdd_studio/issues/24). Jede Task nennt das Akzeptanzkriterium, auf das sie einzahlt.

## Tasks

- [x] AC-1 Unerreichbaren `case classConfig:`-Zweig in `internal/mcp/write.go` entfernen und Unit-Test ergänzen.
- [x] AC-2 In `internal/mcp/critical.go` (`runCritical`) `case "sdd_adopt_apply":` explizit aufführen und im `default:`-Zweig `errUnknownTool` zurückgeben; Unit-Test in `internal/mcp/critical_test.go` ergänzen.
- [x] AC-3 In `internal/ui/markdown.go` (`inline`) numerische Platzhalter für Inline-Code-Spans verwenden und Unit-Test mit >26 Code-Spans in `internal/ui/ui_test.go` ergänzen.
- [x] AC-4 `OpenBrowser` in `internal/ui/server.go` zentralisieren, in `cmd/sdd/ui.go` und `cmd/sdd-desk/main.go` wiederverwenden und Unit-Test in `internal/ui/ui_test.go` ergänzen.
- [x] AC-1, AC-2, AC-3, AC-4 `CHANGELOG.md` aktualisieren sowie `sdd gate pre-merge --feature clean-code-refactoring` und `go test ./...` ausführen.
