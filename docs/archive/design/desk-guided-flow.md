---
feature: desk-guided-flow
status: archived
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/mcp/**
  - internal/desk/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# Tech Spec — Geführter Ablauf: Technik vor touches, Reihenfolge, Tests je Sprache

Setzt das PRD `docs/archive/prd/desk-guided-flow.md` um. Grundlage sind die Tech Specs `docs/archive/design/sdd-guidance.md` (Prompts) und `docs/archive/design/desk-write-content.md` (`sdd_read`, `sdd_write`).

## Architektur

```text
internal/sddcore/guide/stack.go   Stack(root) → StackInfo; feste Sprachtabelle
internal/mcp/tools.go             Werkzeug sdd_stack (lesend)
internal/mcp/prompts.go           sdd-idea neu, Reihenfolge in common, Prompt sdd-tests
internal/desk/agent.go            readTools: sdd_stack
```

### `guide.Stack` (AC-1)

Eine feste Tabelle `languages` legt je Sprache fest:
- Name,
- Dateiendungen,
- Marker-Dateien (genaue Namen, im Wurzelverzeichnis oder einen Ordner tiefer),
- Testmuster,
- Kommentarpräfix,
- eine Beispielzeile.

| Sprache | Endungen | Marker | Testmuster | Kommentar |
|---|---|---|---|---|
| Go | `.go` | `go.mod` | `**/*_test.go` | `//` |
| Rust | `.rs` | `Cargo.toml` | `tests/**/*.rs`, `src/**/*.rs` | `//` |
| C/C++ | `.c .cc .cpp .cxx .h .hpp` | `CMakeLists.txt`, `meson.build` | `tests/**/*.cpp`, `**/*_test.cpp`, `test/**/*.cpp` | `//` |
| Python | `.py` | `pyproject.toml`, `setup.py`, `requirements.txt` | `**/test_*.py`, `**/*_test.py` | `#` |
| TypeScript/JavaScript | `.ts .tsx .js .jsx .mjs` | `package.json`, `tsconfig.json` | `**/*.spec.ts`, `**/*.test.ts`, `**/*.test.js` | `//` |
| Java/Kotlin | `.java .kt` | `pom.xml`, `build.gradle`, `build.gradle.kts` | `**/src/test/**/*.java`, `**/src/test/**/*.kt` | `//` |
| C# | `.cs` | `*.csproj` (Endung) | `**/*Tests.cs` | `//` |
| Shell | `.sh .bash` | — | `tests/**/*.bats`, `tests/**/*.sh` | `#` |

**Zählung:** `scan.WalkFiles(root, ["**"], exclude)`. Dabei werden `.git` immer und `.sdd` zusätzlich übersprungen. Gezählt werden Dateien je Endung; ein Marker zählt, wenn sein Name vorkommt (im Wurzelverzeichnis oder eine Ebene tiefer). Eine Sprache ist erkannt, wenn sie Dateien oder Marker hat.

**Sortierung:** nach Dateien absteigend, dann nach Name.

**Ergebnis:** `{languages: [{name, files, markers, test_globs, comment, example}], tests_include, configured}`. Das Beispiel lautet `// <slug> AC-1: …` bzw. `# <slug> AC-1: …`. Ohne Konfiguration ist `configured` falsch und `tests_include` leer; die Zählung läuft dann mit `exclude` aus `scaffold.DefaultConfig`.

**Desktop:** `sdd_stack` steht in `readTools`. Der Test zur Stufenzuordnung erzwingt das.

### Prompts (AC-2 bis AC-4)

`common` erhält als ersten Grundsatz:

> Reihenfolge des Prozesses: Idee → PRD → Tech Spec (wenn sdd_check_intent sie verlangt) → Plan → Tests je Kriterium → Umsetzung → Gate pre-merge → Merge → Release → Archiv. Archiviert wird erst nach dem Release. Ein überschrittenes archive-after vor der Umsetzung heißt: Stichtag mit dem Menschen korrigieren, nicht archivieren.

`sdd-idea` bekommt diese Schritte:
1. Problem.
2. Nutzer und Ziel.
3. Umfang und Nicht-Ziele.
4. **Technik:** `sdd_stack`. Erkannte Sprache bestätigen lassen. Ohne Sprache einen Vorschlag mit Begründung machen (Art des Programms, Zielplattform, Umfang) und fragen. Bei einem neuen Projekt anbieten, die Wahl als ADR festzuhalten (Prompt `sdd-adr`).
5. Akzeptanzkriterien.
6. Slug und touches in den Konventionen der Sprache, ohne Platzhalter-Endungen; prüfen mit `sdd_check_intent`.
7. PRD mit `content` anlegen.

`sdd-tests` (Argument `feature`, Regeln SDD-022, SDD-025):
1. `sdd_trace` zeigt die Kriterien ohne Test.
2. `sdd_stack` zeigt Sprache, Testmuster und Kommentar.
3. Erfasst `tests.include` die geplante Testdatei nicht (Muster aus `sdd_stack`), mit `sdd_read` und `sdd_write` ergänzen.
4. Je Kriterium einen Test: einen echten, wenn das Verhalten schon klar ist, sonst ein Gerüst, das ausdrücklich fehlschlägt oder übersprungen wird. Direkt darüber die Zeile `<Kommentar> <slug> AC-<n>: <Kurzbeschreibung>`. Neue Dateien mit `sdd_write`.
5. Mit `sdd_trace` und `sdd_next` prüfen.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `guide/stack.go`, `tools.go`, `agent.go` | `TestStack`, `TestStackTool`, `TestToolTiersCoverAllTools` |
| AC-2 | `prompts.go` (`sdd-idea`) | `TestIdeaAsksTechnologyBeforeTouches` |
| AC-3 | `prompts.go` (`common`) | `TestPromptsNameProcessOrder` |
| AC-4 | `prompts.go` (`sdd-tests`) | `TestTestsPrompt` |
