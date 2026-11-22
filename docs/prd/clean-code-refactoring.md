---
feature: clean-code-refactoring
status: active
archive-after:
touches:
  - internal/mcp/**
  - internal/ui/**
  - cmd/**
  - CHANGELOG.md
---

# PRD — Clean Code: Toter Code, Defensives Dispatching und Deduplizierung (`clean-code-refactoring`)

## Problem

Im Code-Review ([Issue #24](https://github.com/Zheng-Bote/sdd_studio/issues/24)) wurden vier Clean-Code- und Wartbarkeitspunkte identifiziert:

1. **Unerreichbarer Code in `internal/mcp/write.go` (`CC-01`):** In `toolWrite` wird `class == classConfig` bereits vorab mit einer Fehlermeldung abgewiesen (Konfigurationsänderungen laufen ausschließlich über das kritische Tool `sdd_config`, ADR 0004). Der anschließende `switch`-Zweig `case classConfig:` ist daher toter Code.
2. **Impliziter Fallthrough in `internal/mcp/critical.go` (`CC-03`):** In `runCritical` führt der `default:`-Zweig ohne Namensprüfung `s.toolAdoptApply(raw)` aus. Wird ein unbekannter Tool-Name an `runCritical` übergeben, würde statt eines Fehlers `sdd_adopt_apply` aufgerufen.
3. **ASCII-Zeichenarithmetik für Code-Platzhalter in `internal/ui/markdown.go` (`CC-02`):** `inline` ersetzt Inline-Code-Spans durch `"\x00" + string(rune(len(codes)-1+'A')) + "\x00"`. Ab mehr als 26 Inline-Code-Spans in einem Absatz läuft `rune('A'+i)` aus dem lateinischen Alphabet in ASCII-Sonderzeichen (`[`, `\`, `]`, `*`, `_`), die mit nachfolgenden Markdown-RegEx-Ersetzungen kollidieren können.
4. **Duplizierte `openBrowser`-Implementierung (`CC-04`):** Die plattformspezifische Funktion `openBrowser` (`darwin`, `windows`, `xdg-open`) ist in `cmd/sdd/ui.go` und `cmd/sdd-desk/main.go` identisch dupliziert.

## Ziel

Der Quellcode ist frei von totem Code, nutzt defensives Dispatching mit explizitem Fehlerfall, verarbeitet beliebig viele Inline-Code-Spans im Markdown-Renderer kollisionsfrei und teilt die Browser-Startlogik an einer zentralen Stelle.

## Nutzer

- **Entwickler und Maintainer von SDD Studio** — profitieren von höherer Code-Klarheit, defensiver Fehlerbehandlung bei kritischen MCP-Tools und zentralisierter Hilfslogik ohne Duplikation.

## Umfang

- Entfernen des unerreichbaren `case classConfig:`-Zweigs in `internal/mcp/write.go` (`toolWrite`).
- Explizites Matching von `case "sdd_adopt_apply":` und Rückgabe von `fmt.Errorf("%w %q", errUnknownTool, name)` im `default:`-Zweig von `runCritical` (`internal/mcp/critical.go`).
- Numerische Platzhalter (`"\x00" + strconv.Itoa(i) + "\x00"`) in `internal/ui/markdown.go` (`inline`).
- Bereitstellung von `ui.OpenBrowser` (und `ui.BrowserCommand`) in `internal/ui/server.go` und Nutzung in `cmd/sdd/ui.go` sowie `cmd/sdd-desk/main.go`.
- Unit-Tests für alle vier Akzeptanzkriterien sowie Dokumentation in `CHANGELOG.md`.

## Nicht-Ziele

- **Funktionale Änderung der MCP-Tools oder des Markdown-Umfangs:** Alle bestehenden Tools und Markdown-Elemente verhalten sich nach außen unverändert.
- **Kopplung von `sdd` an `internal/desk`:** Die Schichtentrennung (`sdd` importiert niemals `internal/desk`, ADR 0002) bleibt strikt gewahrt.

## Akzeptanzkriterien

- **AC-1** `toolWrite` in `internal/mcp/write.go` weist `sdd.config.yaml` weiterhin mit Verweis auf `sdd_config` (ADR 0004) ab und enthält keinen unerreichbaren `case classConfig:`-Zweig mehr.
- **AC-2** `runCritical` in `internal/mcp/critical.go` behandelt `"sdd_adopt_apply"` in einem expliziten `case` und gibt im `default:`-Zweig für unbekannte Tool-Namen `errUnknownTool` zurück.
- **AC-3** `RenderMarkdown` in `internal/ui/markdown.go` rendert Absätze mit mehr als 26 Inline-Code-Spans (`...`) vollständig und ohne Kollision mit Links, Fettdruck oder Kursivschrift.
- **AC-4** `cmd/sdd/ui.go` und `cmd/sdd-desk/main.go` nutzen eine gemeinsame `ui.OpenBrowser`-Implementierung aus `internal/ui` ohne Code-Duplikation.
