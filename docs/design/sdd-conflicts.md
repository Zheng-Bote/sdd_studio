---
feature: sdd-conflicts
status: active
archive-after:
touches:
  - internal/mcp/**
  - internal/desk/**
  - docs/sdd_konzept.md
  - handbuch/**
  - README.md
---

# Tech Spec — Assistierter Abgleich von Tech Spec und Verfassung (`SDD-060`)

Setzt das PRD `docs/prd/sdd-conflicts.md` um. Grundlage sind die Tech Specs `docs/archive/design/sdd-guidance.md` (`sdd mcp`) und `docs/archive/design/sdd-desk.md`; Werkzeuge, Freigaben und Grenzen von `sdd-desk` bleiben, wie sie dort beschrieben sind.

## Architektur

```text
internal/mcp/
  ├── resources.go   readArtifact(rel): gemeinsame Prüfung für sdd://artifact/<pfad> und sdd_read
  ├── tools.go       sdd_read (readOnly)
  └── prompts.go     sdd-conflicts
docs/sdd_konzept.md  9.2 ohne SDD-060, 9.6 assistierter Abgleich
handbuch/04-agenten.md  Werkzeug und Ablauf
```

Kein neuer Code in `internal/sddcore` und keine Änderung an `sdd-desk`: Das Werkzeug erscheint dort über `tools/list` und braucht wie jedes Werkzeug die Freigabe des Nutzers; der Ablauf erscheint in der Seitenleiste über `prompts/list`. `sdd` bleibt offline und ohne LLM-Code — `sdd_read` liest nur Dateien, der Prompt ist fester Text.

## Schnittstellen

**`sdd_read`** — Eingabe `{path?: string}`, Anmerkung `readOnlyHint`.

- Ohne `path`: `{artifacts: [pfad, …]}`, die Liste aus `artifactPaths()` (Verfassung, `artifacts.adr`, `prd`, `design`, `plans`, `archive`, ohne `exclude`).
- Mit `path`: `readArtifact` prüft wie die Ressource `sdd://artifact/<pfad>`: `scan.CleanRel` (kein `..`, nicht absolut), Pfad in `artifactPaths()`, Endung `.md` oder `.sdd`. Ergebnis `{path, lines, truncated, text}`; `text` trägt je Zeile die Nummer (`%4d| `). Über 256 KiB wird an einer Zeilengrenze gekürzt und `truncated: true` gesetzt; der Text endet dann mit einem Hinweis.
- Fehler sind Werkzeugergebnisse mit `isError`, wie bei den übrigen Werkzeugen.

Die Ressource `sdd://artifact/<pfad>` nutzt dieselbe Funktion `readArtifact` (ohne Zeilennummern), damit beide Wege dieselbe Grenze haben.

**Prompt `sdd-conflicts`** — Argument `feature` (Pflicht), Regeln `SDD-010`. Text: Vorgehen in Schritten (Pfade über `sdd_read` ohne Pfad bzw. `sdd_context`, Verfassung und Tech Spec lesen, verlinkte ADRs lesen, jede Regel der Verfassung gegen die Tech Spec halten), Meldeformat als Tabelle (Verfassung: Zitat + Datei:Zeile | Tech Spec: Zitat + Datei:Zeile | Einschätzung | Vorschlag), die Bedingung „ohne wörtliches Zitat keine Meldung“, der Hinweis, dass das Ergebnis kein Befund ist, und „nichts ändern“. Wie alle Prompts endet er mit den gemeinsamen Grundsätzen.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `tools.go` (`sdd_read`), `resources.go` (`readArtifact`) | `TestReadTool` |
| AC-2 | `resources.go` (`readArtifact`), `tools.go` (Kürzung) | `TestReadToolRefuses` |
| AC-3 | `prompts.go` (`sdd-conflicts`) | `TestConflictsPrompt` |
| AC-4 | `docs/sdd_konzept.md`, `handbuch/04-agenten.md` | `TestSDD060NotInCatalog` |
