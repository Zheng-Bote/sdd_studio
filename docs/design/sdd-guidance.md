---
feature: sdd-guidance
status: active
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - internal/mcp/**
  - handbuch/**
  - constitution.md
---

# Tech Spec — Prozessführung: sdd next und sdd mcp (`sdd-guidance`)

Setzt das PRD `docs/prd/sdd-guidance.md` um. Rahmen: [ADR 0002](../adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md) — `sdd` bleibt KI-frei und offline.

## Architektur

```text
internal/sddcore/guide     Next (Prozessschritte), Context (Präzedenz), Intent, Trace, Explain, ValidateDraft
internal/mcp               JSON-RPC 2.0 über stdio, Werkzeuge, Ressourcen, Prompts (eigene schmale Umsetzung)
cmd/sdd                    next [--feature] [--json], mcp [--allow-write]
```

`guide` baut ausschließlich auf vorhandenen Bibliotheksfunktionen auf (`sddcore.RunCheck`, `scan`, `rules`, `scaffold`); `mcp` übersetzt nur zwischen Protokoll und `guide`/`sddcore`.

## Schnittstellen

**`sdd next`** — Schritte je Feature, in dieser Reihenfolge; der erste offene ist der nächste:

| Schritt | erledigt, wenn | sonst |
|---|---|---|
| PRD | aktives PRD mit `feature: <slug>` | `sdd new --feature <slug> --type prd` |
| Nicht-Ziele | kein `SDD-023` am PRD | Abschnitt füllen |
| Akzeptanzkriterien | ≥ 1 Kennung, kein `SDD-021` am PRD | Kriterien mit `AC-<n>` |
| Absicht | `touches` in einem Artefakt des Features | `touches` ergänzen |
| Tech Spec | Schwelle nicht überschritten (*nicht nötig*) oder Tech Spec vorhanden | `sdd new --feature <slug> --type design` |
| AC-Bezug | kein `SDD-031` für das Feature | Kriterien in der Tech Spec nennen |
| ADR | `touches` berührt die Verfassung nicht (*nicht nötig*), sonst *manuell* (geprüft von `SDD-010` im Gate) | ADR anlegen und entscheiden |
| Plan | kein Plan (*nicht nötig*) oder kein `SDD-040`/`SDD-041` am Plan | Tasks mit AC, Tech Spec vor Tasks |
| Tests | kein `SDD-022` für die Kennungen des PRDs | `// <slug> AC-<n>` am Test |
| Archiv | Status nicht `implemented` | nach `artifacts.archive`, `status: archived` |

**`sdd mcp`** — JSON-RPC 2.0, eine Nachricht je Zeile auf stdin/stdout, Protokolle auf stderr. Unterstützte Protokollversionen `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`; fordert der Client eine andere an, antwortet der Server mit der neuesten. Fähigkeiten: `tools`, `resources`, `prompts`.

| Werkzeug | Eingabe | Ergebnis |
|---|---|---|
| `sdd_next` | `feature?` | Schritte wie `sdd next --json` |
| `sdd_lint` | `gate?` | Bericht v1 |
| `sdd_gate` | `gate`, `feature?`, `paths?` | Bericht v1 und `passed` |
| `sdd_report` | — | wie `sdd report --format json` |
| `sdd_explain` | `rule_id` | Katalogeintrag, Handbuchabschnitt, Prüfvertrag |
| `sdd_check_intent` | `paths`, `feature?` | Schwelle, nötige und vorhandene Artefakte |
| `sdd_context` | `path` | geltende Artefakte nach Präzedenz |
| `sdd_trace` | `feature` | je AC: Tech Specs, Tests (Datei:Zeile), Tasks |
| `sdd_validate_draft` | `path`, `content` | Befunde, als läge der Entwurf schon im Repository |
| `sdd_new` | wie `sdd new`, `apply?` | Vorschau; Anlegen nur mit `--allow-write` |

Ressourcen: `sdd://constitution`, `sdd://catalog`, `sdd://handbook`, `sdd://artifact/<pfad>` (nur Markdown-Artefakte im Repository). Prompts: `sdd-idea`, `sdd-techspec`, `sdd-adr`, `sdd-plan`, `sdd-review`.

## Datenmodell

- **Präzedenz** (`sdd_context`): Verfassung; ADRs, deren `references` den Pfad überschneiden; aktive Tech Specs, PRDs und Pläne, deren `touches` den Pfad überschneiden — in dieser Reihenfolge (sdd_process_3 §3.6).
- **Entwurfsprüfung:** Konfiguration, Verfassung und alle Artefaktpfade (`prd`, `design`, `plans`, `archive`, `adr`) werden in ein temporäres Verzeichnis kopiert, der Entwurf an seinen Pfad gelegt, `sdd lint` darauf ausgeführt und die Befunde auf den Entwurf gefiltert. `SDD-022`/`SDD-025` entfallen, weil Tests erst nach dem Entwurf entstehen.
- **Werkzeugfehler** (ungültige Eingabe, Pfad außerhalb) sind Ergebnisse mit `isError: true`, keine Protokollfehler.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `guide.Next`, `sdd next` | `TestNextWalksTheProcess`, `TestNextWithoutFeature` |
| AC-2 | `guide.Next` auf `RunCheck` | `TestNextWalksTheProcess` |
| AC-3 | `mcp.Server` | `TestProtocolHandshakeAndErrors` |
| AC-4 | `guide.*`, Werkzeuge | `TestToolsMatchLibrary`, `TestContextPrecedence`, `TestValidateDraft` |
| AC-5 | `sdd_new` | `TestNewNeedsAllowWrite` |
| AC-6 | Werkzeugliste, Pfadprüfung | `TestNoForbiddenCapabilities` |
| AC-7 | Prompts | `TestPrompts` |
| AC-8 | Quelltextprüfung | `TestNoOutboundNetworkCode` |

## Sicherheit

- Kein Netzwerk: stdio ist der einzige Transport; die Quelltextprüfung verbietet `http.Client`, `http.Get/Post`, `net.Dial` und Ähnliches in `sdd`.
- Pfade aus Werkzeugargumenten laufen durch `scan.CleanRel`; Ressourcen lesen nur Markdown-Artefakte im Repository.
- Schreiben nur mit `--allow-write`, nur über `scaffold`.

## Offene Punkte

- Ob der Schritt „ADR“ sich ohne Diff automatisch bewerten lässt (welches ADR gehört zu welchem Feature), bleibt offen; bis dahin ist er *manuell*.
