---
feature: clean-code-refactoring
status: archived
archive-after:
touches:
  - internal/mcp/**
  - internal/ui/**
  - cmd/**
  - CHANGELOG.md
release: v0.9.1
---

# Tech Spec — Clean Code: Toter Code, Defensives Dispatching und Deduplizierung (`clean-code-refactoring`)

Setzt das PRD `docs/prd/clean-code-refactoring.md` ([Issue #24](https://github.com/Zheng-Bote/sdd_studio/issues/24)) um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

```text
internal/mcp/write.go         Entfernung des toten case classConfig in toolWrite (AC-1)
internal/mcp/critical.go      Expliziter case "sdd_adopt_apply" + default errUnknownTool (AC-2)
internal/ui/markdown.go       Numerische Platzhalter "\x00" + strconv.Itoa(i) + "\x00" (AC-3)
internal/ui/server.go         Exportierte Funktion OpenBrowser(url string) error (AC-4)
cmd/sdd/ui.go                 Aufruf von ui.OpenBrowser(url) statt lokaler Kopie (AC-4)
cmd/sdd-desk/main.go          Aufruf von ui.OpenBrowser(url) statt lokaler Kopie (AC-4)
```

Die Schichtentrennung nach ADR 0002 bleibt gewahrt: `internal/ui` gehört zur KI-freien Standardbibliothek des Repositories; sowohl `cmd/sdd` als auch `cmd/sdd-desk` dürfen `internal/ui` importieren, während `internal/ui`, `internal/sddcore`, `internal/mcp` und `cmd/sdd` weiterhin keinerlei Abhängigkeit zu `internal/desk` besitzen (`TestSddDoesNotImportDesk`).

## Schnittstellen

- **`ui.OpenBrowser(url string) error`** in `internal/ui/server.go`: Startet den Plattform-Standardbrowser (`open` auf `darwin`, `rundll32 url.dll,FileProtocolHandler` auf `windows`, `xdg-open` sonst) ohne Shell-Interpolation.
- Alle externen CLI-, HTTP- und JSON-RPC-Schnittstellen bleiben unverändert.

## Datenmodell

Keine Änderungen am Datenmodell.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | Entfernung des unerreichbaren `case classConfig:` in `internal/mcp/write.go`; `if class == classConfig` weist Schreibversuche auf `sdd.config.yaml` ab. | `internal/mcp/config_test.go` / `internal/mcp/write_test.go` `TestWriteRejectsConfigWithADR0004Hint` |
| AC-2 | `runCritical` in `internal/mcp/critical.go` matcht `"sdd_adopt_apply"` explizit und liefert im `default:`-Zweig `fmt.Errorf("%w %q", errUnknownTool, name)`. | `internal/mcp/critical_test.go` `TestRunCriticalRejectsUnknownTool` |
| AC-3 | `inline` in `internal/ui/markdown.go` nutzt `"\x00" + strconv.Itoa(i) + "\x00"` statt `rune(len(codes)-1+'A')`. | `internal/ui/ui_test.go` `TestRenderMarkdownManyCodeSpans` |
| AC-4 | `ui.OpenBrowser` in `internal/ui/server.go` ersetzt die duplizierten `openBrowser`-Funktionen in `cmd/sdd/ui.go` und `cmd/sdd-desk/main.go`. | `internal/ui/ui_test.go` `TestBrowserCommandForOS` |

## Sicherheit

- **Defensives Dispatching (`AC-2`):** Verhindert, dass durch zukünftige Erweiterungen ein unbekanntes Tool im kritischen Pfad unbeabsichtigt `toolAdoptApply` auslöst.
- **Keine Shell-Ausführung (`AC-4`):** `ui.OpenBrowser` übergibt die URL als einzelnes Argument an `exec.Command`.

## Offene Punkte

- Keine offenen Punkte.
