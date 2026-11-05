---
feature: desk-config-critical
status: active
archive-after:
touches:
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
---

# Tech Spec — Konfiguration im KI-Desktop nur mit einzelner Bestätigung ändern

Setzt das PRD `docs/prd/desk-config-critical.md` um, auf Grundlage von ADR 0004 und ADR 0003 (Entscheidungen 5 und 6).

## Architektur

```text
internal/mcp/config.go       sdd_config: Prüfung, Diff, Befund-Differenz je Prüflauf, Zusammenfassung, Apply
internal/mcp/critical.go     criticalTools und runCritical um sdd_config ergänzt
internal/mcp/write.go        classify: classConfig wird für sdd_write abgelehnt (Hinweis auf sdd_config)
internal/mcp/prompts.go      common, sdd-tests: sdd_config für die Konfiguration
internal/mcp/server.go       Server-Anweisungen: dito
internal/desk/agent.go       criticalTools um sdd_config ergänzt
apps/desk/src/app/…          Freigabekarte zeigt dropped/added je Prüflauf; types.ts
handbuch/04-agenten.md       kritische Aktionen, Zahl der Werkzeuge (neunzehn)
```

### `sdd_config` (AC-1, AC-4)

Argumente: `content` (Pflicht), `base_sha256` (Pflicht), `apply`.

1. `readExisting(root, "sdd.config.yaml")`. Fehlt die Datei, kommt der Fehler „keine Konfiguration — Projekt erst mit sdd_init anlegen“. Weicht `base_sha256` vom aktuellen Inhalt ab, kommt derselbe Fehler wie bei `sdd_write`, mit dem aktuellen Hash.
2. `config.Parse(content)`. Bei einem Fehler wird nichts geschrieben, und der Fehler wird gemeldet.
3. Ist der Inhalt unverändert, liefert die Vorschau `action: unchanged`, und `apply` schreibt nichts.
4. `apply`: `scaffold.Plan{Replace, Expect: alt}.Apply` (atomar, Rechte bleiben). Das Ergebnis enthält `applied`, `critical`, `summary`, `path`, `sha256`, `lines` und `diff`.

### Befund-Differenz (AC-2)

Für jeden Prüflauf `g ∈ {"" (Lint), pre-implementation, pre-merge, pre-release}` laufen:

- `alt := sddcore.RunCheckWithConfig({RepoRoot, Gate: g}, cfgAlt)`
- `neu := sddcore.RunCheckWithConfig({RepoRoot, Gate: g}, cfgNeu)`

Beide Läufe wenden die Baseline an, so wie CI sie sieht. Ein Befund wird über `(rule_id, path, line, ordinal)` identifiziert; die Konfiguration ändert keine Dateien, die Zeilen bleiben also gleich. `dropped` sind die Befunde in `alt` ohne Gegenstück in `neu`, `added` umgekehrt. Ergebnis je Lauf: `{gate, dropped: [Finding], added: [Finding]}`. Läufe ohne Differenz entfallen aus der Liste. Scheitert ein Lauf mit der neuen Konfiguration (z. B. ein ungültiges Gate in `override`), gilt die Vorschau als fehlgeschlagen, und `apply` schreibt nichts.

### Zusammenfassung (AC-2, AC-3)

`changedKeys(alt, neu)` vergleicht die beiden YAML-Dokumente auf oberster Ebene, unter `gates` eine Ebene tiefer. Heraus kommt eine sortierte Liste wie `tests.include`, `exclude` oder `gates.pre_merge.override`. Die Zusammenfassung lautet:

`sdd.config.yaml ändern: <Schlüssel, kommagetrennt> (−<dropped gesamt> / +<added gesamt> Befunde)`

Bei einem Profilwechsel steht statt `profile` der Ausdruck `profile <alt> → <neu>`. Änderungen an `override` stehen immer mit ihrem vollen Schlüssel da.

### `sdd_write` (AC-5)

`classify` meldet `classConfig` weiter. `toolWrite` lehnt diese Klasse vor jeder Vorschau ab: „sdd.config.yaml ändert nur sdd_config (kritisch, ADR 0004)“. Damit verliert ein MCP-Client ohne `--allow-critical` die Möglichkeit, die Konfiguration zu ändern. Das ist gewollt, denn eine Stufe, die man über ein anderes Werkzeug umgehen kann, gibt es nicht. `sdd_read` behält `classConfig`.

### Desktop (AC-6)

`criticalTools` erhält `sdd_config`. `TestToolTiersCoverAllTools` sichert die Einstufung. Damit gilt die bestehende Logik:
- Ein Aufruf mit `apply` wartet immer auf die einzelne Bestätigung.
- Ohne `apply` ist er eine Vorschau mit `preview_only` (desk-approval-scope).
- Nach einer Ablehnung gilt die Sperre bis zur nächsten Nachricht des Nutzers.

Die Freigabekarte erhält einen Block „Wirkung auf die Befunde“. Er zeigt je Lauf die wegfallenden (`− SDD-0xx Pfad:Zeile`) und die hinzukommenden Befunde. In `types.ts` erhält `preview` das Feld `findings_delta?: {gate: string; dropped: Finding[]; added: Finding[]}[]`.

### Prompts (AC-7)

- **`common`:** „Bestehende Dateien (Artefakte, Testdateien) … mit sdd_write“; die Konfiguration fällt dort heraus. Neu kommt dazu: „Die Konfiguration (sdd.config.yaml) änderst du nur mit sdd_config. Das ist kritisch: Erkläre die Änderung, zeige die Vorschau, ausgeführt wird sie nur nach der einzelnen Bestätigung des Menschen; lehnt er ab, frage im selben Zug nicht erneut.“
- **`sdd-tests`, Schritt 3:** „… ergänze tests.include mit sdd_config um das passende Muster“.
- **Server-Anweisungen:** dieselbe Regel in Kurzform.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.go`, `critical.go` | `TestConfigToolNeedsFlag`, `TestConfigToolRefusesInvalid` |
| AC-2 | `config.go` (Befund-Differenz) | `TestConfigToolPreview` |
| AC-3 | `config.go` (`changedKeys`) | `TestConfigToolSummary` |
| AC-4 | `config.go` | `TestConfigToolApply` |
| AC-5 | `write.go` | `TestWriteRefusesConfig` |
| AC-6 | `agent.go`, Freigabekarte | `TestToolTiersCoverAllTools`, `TestConfigIsCritical`, `approval-card.spec.ts` |
| AC-7 | `prompts.go`, `server.go` | `TestPromptsConfigTool` |
| AC-8 | `handbuch/04-agenten.md` | `TestHandbookConfigTool` |

## Risiken

- **Laufzeit der Vorschau:** acht Prüfläufe je Vorschau. Für Projekte üblicher Größe ist das unkritisch. Sonst nur Lint und `pre-merge`, siehe ADR 0004.
- **Andere MCP-Clients** können die Konfiguration ohne `--allow-critical` nicht mehr ändern (AC-5). Im Handbuch steht das.
