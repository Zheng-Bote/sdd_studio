---
feature: desk-write-content
status: archived
archive-after:
touches:
  - internal/mcp/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/config/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# Tech Spec — Arbeitsergebnisse aus dem KI-Desktop mit Inhalt speichern

Setzt das PRD `docs/archive/prd/desk-write-content.md` um, auf Grundlage von ADR 0003 (Stufe „schreibend“). Grundlagen sind die Tech Specs `docs/archive/design/sdd-init-new.md` (Plan/Apply) und `docs/archive/design/sdd-desk.md` (Freigaben).

## Architektur

```text
internal/sddcore/config/config.go     Parse(data) — Load nutzt es; sdd_write prüft sdd.config.yaml damit
internal/sddcore/scaffold/plan.go      Op.Replace + Expect: atomares Ersetzen, nur wenn die Datei noch Expect entspricht
internal/sddcore/scaffold/new.go       CheckArchiveAfter (gemeinsam für die Option und das Frontmatter eines Entwurfs)
internal/sddcore/scaffold/diff.go      UnifiedDiff(path, old, new) — gemeinsamer Präfix/Suffix, ein Hunk
internal/mcp/write.go                  sdd_write: Pfadklassen, Prüfung, Vorschau, Anwenden
internal/mcp/tools.go                  sdd_new: content; Ergebnis mit sha256/lines; sdd_read: Konfiguration, Tests, sha256
internal/mcp/prompts.go, server.go     Grundsätze und Anweisungen (AC-8)
internal/desk/agent.go                 writes(): sdd_new/sdd_write mit apply; Sitzungsfreigabe auch für Schreiben
apps/desk/src/app/…                    Freigabekarte: Diff, „ändern“, Sitzungsfreigabe für Schreiben
handbuch/04-agenten.md                 Werkzeugtabelle, Einrichten, Grenzen
```

### `sdd_new` mit Inhalt (AC-1, AC-5)

`content` ist nur mit genau einem Typ erlaubt, sonst `ErrInvalidInput`. `NewPlan` erzeugt den Plan wie bisher, danach ersetzt `content` den Inhalt der einen Operation. Ist die Operation übersprungen (Datei existiert), bleibt sie übersprungen. Dann prüft `sdd_new`:

1. Bei `prd`, `design` und `plan` muss das Frontmatter `feature: <slug>` tragen.
2. `archive-after` im Frontmatter wird mit `scaffold.CheckArchiveAfter` geprüft, wie die Option.
3. `guide.ValidateDraft(root, path, content)`. Bei `Errors > 0` endet der Aufruf mit einem Werkzeugfehler, der die Befunde mit Regel und Meldung aufzählt.

Die Prüfung läuft in Vorschau und `apply` gleich. Das Ergebnis von `apply` enthält je Datei `path`, `status`, `sha256` und `lines`.

### `sdd_write` (AC-2 bis AC-5)

Argumente: `path` (Pflicht), `content` (Pflicht, höchstens 1 MiB), `base_sha256`, `apply`.

**Pfadklassen** (`classify`), nach `scan.CleanRel`:

| Klasse | Pfade | Prüfung vor dem Schreiben |
|---|---|---|
| `artifact` | `artifacts.constitution`; `*.md`, die `artifacts.adr/prd/design/plans/archive/docs` auswählen | `ValidateDraft`, `error` blockiert; ADR-Regel (AC-4) |
| `config` | `sdd.config.yaml` | `config.Parse`, Fehler blockiert |
| `test` | Treffer von `tests.include`, leer = `rules.FallbackTestGlobs` | keine |

Alles andere wird abgelehnt (`nicht beschreibbar: <pfad> ist weder Artefakt, Konfiguration noch Testdatei`). Vor der Klassifizierung werden ein erstes Pfadsegment `.git` oder `.sdd` und `exclude`-Treffer abgelehnt. Existiert das Ziel und ist es keine reguläre Datei (`Lstat`), wird es abgelehnt.

**Ersetzen:** Existiert die Datei, muss `base_sha256` gleich `sha256(aktueller Inhalt)` sein. Sonst kommt der Fehler `base_sha256 fehlt oder ist veraltet; aktueller Stand: <hash> — erst mit sdd_read lesen`. Der Plan besteht aus einer `Op{Replace: true, Expect: aktueller Inhalt}`. `Apply` ersetzt über eine temporäre Datei und `rename` (wie `prependFile`), mit denselben Dateirechten, und überspringt, wenn sich die Datei seit der Planung geändert hat. Ist der neue Inhalt gleich dem alten, ist die Aktion `unchanged`, und es wird nichts geschrieben. Eine neue Datei wird wie bisher mit `O_EXCL` angelegt.

**ADR-Schutz (AC-4):** Liegt der Pfad unter `artifacts.adr` und trägt der neue Inhalt `status: accepted`, während die bisherige Datei fehlt oder einen anderen Status trägt, kommt der Fehler `ein ADR entscheidet der Mensch (ADR 0003, kritische Aktion)`.

**Antwort:** Die Vorschau liefert `{applied: false, ops: [{path, action, diff|content}], findings}`. Nach `apply` kommt `{applied: true, path, status, sha256, lines, diff, findings}`. Die Vorschau hat dieselbe Form wie die von `sdd_new` (`ops`), damit die Freigabekarte beide darstellt. `UnifiedDiff` schneidet gemeinsame Anfangs- und Endzeilen ab und zeigt den Rest als einen Hunk mit drei Zeilen Kontext. Das ist nicht minimal, aber vollständig und bei großen Dateien linear.

### `sdd_read` (AC-6)

Neben Artefakten liest `sdd_read` auch `sdd.config.yaml` und die Klasse `test` aus `classify`. Das Ergebnis erhält `sha256` über den vollständigen Inhalt, auch wenn der Text gekürzt ist. Die Liste ohne `path` nennt zusätzlich `sdd.config.yaml`, aber keine Testdateien.

### Desktop (AC-7)

`writes(tc)` ist wahr für `sdd_new` und `sdd_write` mit `apply: true`. `decide` erlaubt einen Aufruf, wenn seine Kennung einmalig freigegeben wurde oder das Werkzeug für die Sitzung freigegeben ist. Lesend und schreibend werden dabei gleich behandelt. Die Vorschau für die Karte entsteht nur, wenn der Aufruf noch eine Entscheidung braucht. Im Frontend entfällt die Umstufung `session → once` für Schreibaufrufe. Die Karte bietet „Für diese Sitzung erlauben“ auch dort an und kennzeichnet sie mit „schreibt Dateien ohne weitere Rückfrage“. Für `update` zeigt die Karte das Badge „ändern“ und den Diff, für `unchanged` „unverändert“.

### Prompts (AC-8)

`common` in `prompts.go` erhält statt „bestehende Dateien änderst du mit deinen eigenen Werkzeugen“:

- Neue Artefakte legst du mit `sdd_new` an und übergibst den geprüften Entwurf als `content`, nie das leere Template.
- Bestehende Dateien (Artefakte, Testdateien, `sdd.config.yaml`) liest du mit `sdd_read` und änderst sie mit `sdd_write` (mit `base_sha256` aus `sdd_read`).
- Melde nur, was ein Werkzeugergebnis belegt. Lies nach dem Speichern mit `sdd_read` gegen.

`sdd-idea`, Schritt 6, lautet dann: „Prüfe den ausgefüllten Entwurf mit sdd_validate_draft und lege das PRD mit sdd_new (types: [prd], content: der Entwurf) an.“ Die Server-Anweisungen nennen `sdd_write`. Der Satz zu Baseline, Lockfile, `adopt --apply` und ADR-Entscheid bleibt, bis die kritischen Werkzeuge existieren.

## Risiken

- **Konfiguration schwächt Prüfungen:** Über `sdd_write` auf `sdd.config.yaml` kann eine Änderung Prüfungen aushebeln, z. B. `exclude: ["**"]`. ADR 0003 ordnet Konfiguration der Stufe „schreibend“ zu, die sich für die Sitzung freigeben lässt. Der Diff ist in Vorschau und Ergebnis sichtbar. Ob Änderungen an `exclude`, `profile` und `gates` kritisch werden sollen, entscheidet ein späteres ADR.
- **Diff nicht minimal:** Bei verstreuten Änderungen zeigt der eine Hunk auch unveränderte Zeilen dazwischen. Der Inhalt bleibt dabei korrekt.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `tools.go` (`toolNew`), `scaffold/new.go` | `TestNewWithContent` |
| AC-2 | `write.go` (`classify`) | `TestWritePathClasses` |
| AC-3 | `write.go`, `scaffold/plan.go` (`Replace`), `config.Parse` | `TestWriteReplace`, `TestApplyReplace` |
| AC-4 | `write.go` (ADR-Schutz) | `TestWriteNeverAcceptsADR` |
| AC-5 | `write.go`, `scaffold/diff.go`, `tools.go` | `TestWritePreviewAndResult`, `TestUnifiedDiff` |
| AC-6 | `tools.go`, `resources.go` (`sdd_read`) | `TestReadConfigAndTests` |
| AC-7 | `desk/agent.go`, `approval-card.ts`, `chat-session.ts` | `TestSessionAllowsWrites`, `approval-card.spec.ts` |
| AC-8 | `prompts.go`, `server.go` | `TestPromptsDemandContent` |
