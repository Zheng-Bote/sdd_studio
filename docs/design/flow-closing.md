---
feature: flow-closing
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - internal/sddcore/scan/**
  - internal/mcp/**
  - cmd/sdd/**
  - apps/desk/src/**
  - handbuch/**
---

# Tech Spec — Klarer Abschluss geführter Abläufe

Setzt das PRD `docs/prd/flow-closing.md` um. Grundlagen sind `docs/design/constitution-guide.md` (`sdd-constitution`, Schritt „Verfassung ausfüllen“) und `docs/archive/design/sdd-adopt.md` (Bericht, Branch).

## Architektur

```text
internal/mcp/prompts.go               common: Grundsatz zum Abschluss; sdd-constitution: Commit/Merge
apps/desk/src/app/chat/message-view.ts  Hinweis bei einer Antwort ohne Text und ohne Werkzeugaufruf
internal/sddcore/scan/git.go          CLIGit.IsAncestor(a, b) (lesend: git merge-base --is-ancestor)
internal/sddcore/guide/next.go        Guide.Adoption; adoptionPending(root)
cmd/sdd/next.go                       Schritt nach „Verfassung ausfüllen“ ausgeben
```

### Prompts (AC-1)

`common` erhält als letzten Grundsatz statt „Zum Schluss: sdd_next, um den nächsten Prozessschritt zu nennen.“:

> Zum Schluss: sdd_next aufrufen und dann in eigenen Worten zusammenfassen — welche Dateien gespeichert wurden (Pfad), was die Prüfung ergeben hat (in Klartext, nicht nur Regelkennungen), was offen ist und welcher Schritt als Nächstes kommt. Ende nie mit einem bloßen Werkzeugaufruf.

`sdd-constitution`, Schritt 6: „Sage, dass die Verfassung committet werden muss. Kam sie aus einer Übernahme (sdd_next meldet ‚Übernahme abschließen‘), nenne auch: Branch prüfen und mergen, erst danach das erste PRD.“

### Desktop (AC-2)

In `MessageView`, Fall `assistant`: Ist `content` leer (nach `trim`) und gibt es keine `tool_calls`, erscheint ein gedämpfter, kursiver Hinweis mit `role="note"`. Gestreamte Teilantworten sind davon nicht betroffen, sie laufen über `streamingMessage`.

### `sdd next` (AC-3)

`CLIGit.IsAncestor(a, b)`: `git merge-base --is-ancestor a b` gelingt bedeutet wahr; jeder Fehler bedeutet falsch.

`adoptionPending(root)`:
1. `.sdd/adopt-report.md` lesen. Fehlt die Datei, entfällt der Schritt.
2. Mit `Angewendet am … auf Branch `<b>` (ausgehend von `<from>`)` Branch und Ausgangsbranch auslesen. Ohne Treffer entfällt der Schritt.
3. `CurrentBranch()` muss gleich `<b>` sein, und `IsAncestor("HEAD", <from>)` muss falsch sein. Jeder Git-Fehler lässt den Schritt entfallen.

`Guide.Adoption` hat ID `adoption`, Titel „Übernahme abschließen“ und Zustand offen.
- **Detail:** „Die Übernahme auf <b> ist noch nicht in <from>. Änderungen (Verfassung, Agenten-Dateien) auf <b> committen, prüfen (git diff <from>...<b>) und in <from> mergen — erst danach das erste PRD.“
- **Befehl:** „git diff <from>...<b>“

`sdd next` gibt den Schritt nach „Verfassung ausfüllen“ aus („Dann: …“).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `prompts.go` | `TestPromptsDemandClosing` |
| AC-2 | `message-view.ts` | `message-view.spec.ts` |
| AC-3 | `git.go`, `next.go`, `cmd/sdd/next.go` | `TestNextAdoptionStep` |
