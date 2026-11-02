---
feature: constitution-guide
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - cmd/sdd/**
  - handbuch/**
  - README.md
---

# Tech Spec — Die Verfassung KI-gestützt erarbeiten

Setzt das PRD `docs/prd/constitution-guide.md` um. Grundlagen sind die Tech Specs `docs/design/desk-write-content.md` (`sdd_read`/`sdd_write`, Pfadklassen) und `docs/design/desk-guided-flow.md` (Prompts, `sdd_stack`).

## Architektur

```text
internal/mcp/write.go          classify: neue Klasse classContext (nur lesen)
internal/mcp/resources.go      readReadable: classContext lesbar
internal/mcp/tools.go          sdd_read-Liste: context
internal/sddcore/guide/next.go Guide.Constitution; constitutionPlaceholders(root, cfg)
cmd/sdd/next.go                Schritt „Verfassung ausfüllen“ zuerst ausgeben
internal/mcp/prompts.go        Prompt sdd-constitution; sdd-idea Schritt 0
```

### Kontextdateien (AC-1)

`contextFiles` ist eine feste Liste: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.gemini/GEMINI.md`, `gemini/GEMINI.md`, `.github/copilot-instructions.md`, `.cursorrules`.

- `classify` prüft sie nach dem Ausschluss von `.git`, `.sdd` und `exclude`, aber vor der Klasse „Artefakt“; ein Treffer ergibt `classContext`.
- `sdd_write` lehnt `classContext` ab: „nicht beschreibbar: <pfad> ist eine Kontextdatei (nur lesbar)“.
- `readReadable` lässt `classContext` zu (nur reguläre Dateien, über `readExisting`).
- Die Liste von `sdd_read` ohne `path` erhält `context`: die vorhandenen Dateien aus `contextFiles`.

### Schritt „Verfassung ausfüllen“ (AC-2)

`constitutionPlaceholders(root, cfg)` liest `artifacts.constitution`. Nur Markdown wird geprüft; eine `.sdd`-Verfassung hat eine eigene Grammatik. Gezählt werden Zeilen außerhalb von Codeblöcken, die `^\s*(?:[-*+]|\d+[.)])\s+<[^<>]+>\s*$` treffen, also Listenzeilen, die nur aus einem Platzhalter bestehen. Fehlt die Datei, zählt sie nicht (das meldet SDD-001).

`Guide` erhält `Constitution *Step` (JSON `constitution`). Er ist gesetzt, wenn mindestens ein Platzhalter gefunden wird:

- **ID:** `constitution`
- **Titel:** „Verfassung ausfüllen“
- **Zustand:** offen
- **Detail:** „<pfad> enthält noch <n> Platzhalter des Templates. Vor dem ersten PRD ausfüllen — nach sdd adopt noch auf dem Branch der Übernahme, dann gilt es als erstmaliges Anlegen (SDD-010).“
- **Befehl:** „Prompt sdd-constitution (sdd-desk oder sdd mcp)“

`sdd next` gibt den Schritt vor allem anderen aus.

### Prompts (AC-3, AC-4)

`sdd-constitution` (ohne Argumente, Regeln SDD-002a, SDD-003, SDD-010):
1. `sdd_read` ohne `path`, dann die Verfassung und jede Datei unter `context` lesen. Dazu `sdd_stack`.
2. Je Pflichtabschnitt der Verfassung (z. B. Architektur/Layering, Security, Strikte Regeln) Regeln vorschlagen. Jede Regel bekommt ihre Fundstelle als Datei:Zeile aus `sdd_read` oder „aus sdd_stack“. Nur dauerhafte, systemweite Regeln; Feature-Inhalte (User Stories, Tasks, Planung) gehören nicht hinein (SDD-003). Ohne Quelle als offene Frage markieren, nicht erfinden.
3. Regel für Regel bestätigen, streichen oder ändern lassen.
4. Die Verfassung ohne Platzhalter formulieren, mit `sdd_validate_draft` prüfen, mit `sdd_write` speichern (`base_sha256` aus `sdd_read`).
5. `sdd_gate` mit `pre-merge`: SDD-010 bedeutet, dass die Verfassung am Vergleichsstand schon existierte; dann mit `sdd-adr` ein ADR anlegen, das sie in `references` nennt. `sdd_next` zum Schluss.

`sdd-idea`, Schritt 0: „Rufe sdd_next auf. Meldet es den Schritt ‚Verfassung ausfüllen‘, schlage vor, zuerst den Ablauf sdd-constitution zu gehen, und fahre erst danach fort.“

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `write.go`, `resources.go`, `tools.go` | `TestReadContextFiles` |
| AC-2 | `guide/next.go`, `cmd/sdd/next.go` | `TestNextConstitutionStep` |
| AC-3 | `prompts.go` (`sdd-constitution`) | `TestConstitutionPrompt` |
| AC-4 | `prompts.go` (`sdd-idea`) | `TestConstitutionPrompt` |
