---
feature: sdd-report
status: active
archive-after:
touches:
  - cmd/**
  - internal/sddcore/**
  - action.yml
  - handbuch/**
---

# Tech Spec — Projektbericht und Schuldenstand (`sdd-report`)

Setzt das PRD `docs/prd/sdd-report.md` um. Die Verfassung hat Vorrang; `sdd report` ist Git-lesend (Security 4).

## Architektur

```text
internal/sddcore/overview
  ├── overview.go     Build: Reifegrad (adopt.Take/Assess), Lint, drei Gates, Schuldenstand, Trend
  └── render.go       Markdown und JSON
internal/sddcore/scan/git.go   CLIGit.ShowFile(ref, path) — `git show <ref>:<path>`, lesend
cmd/sdd                        report [--format markdown|json] [--since <ref>]
action.yml                     zusätzlicher Schritt „sdd report“ mit `if: always()`
```

## Schnittstellen

- `sdd report [--format markdown|json] [--since <ref>]` — Exit `0`, sobald ein Bericht entsteht; Exit `2` nur bei fehlender oder ungültiger Konfiguration, ungültigem Format oder nicht lesbarer aktueller Baseline.
- **Vergleichsstand:** `--since`, sonst die Merge-Basis nach `base_ref` (wie `pre-merge`). Fehlt die Baseline am Vergleichsstand, gilt sie als leer; ist Git nicht verfügbar oder der Ref unbekannt, entfällt der Trend mit Grund im Bericht (AC-4).
- **Action:** neuer Schritt nach dem Gate mit `if: always()` und `continue-on-error: true`, der `sdd report` an `$GITHUB_STEP_SUMMARY` anhängt. Der Gate-Schritt bleibt unverändert der Schritt, dessen Exit-Code zählt (AC-5).

## Datenmodell

```go
type Overview struct {
    Profile, Since, SinceError string
    Maturity   adopt.Maturity
    Lint       Counts                 // error, warn, info, baselined von `sdd lint`
    Findings   []report.Finding       // offene error/warn von `sdd lint`
    Gates      []GateStatus           // je Gate Counts oder Error
    Debt       []DebtRow              // je (Regel, Gate): Now, Before
    DebtNow, DebtBefore int
    Added      []baseline.Entry       // in der aktuellen, nicht in der früheren Baseline
}
```

Ein Eintrag ist „neu“ (AC-3), wenn seine Identität `gate + rule_id + path + content_hash + ordinal` in der Baseline am Vergleichsstand fehlt — dieselbe Identität wie beim Unterdrücken. Wird ein Befund durch Textänderung neu gehasht und wieder aufgenommen, zählt er damit ebenfalls als neu; genau das soll im Review auffallen.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `overview.Build`: Assess, Lint, drei Gates | `TestBuildShowsMaturityLintAndGates` |
| AC-2 | `overview.Build`: Debt-Tabelle, `ShowFile` | `TestDebtTrendAgainstSince` |
| AC-3 | `Added`, Hervorhebung in `RenderMarkdown` | `TestDebtTrendAgainstSince` |
| AC-4 | `RenderJSON`, Fehler je Gate/Vergleichsstand | `TestReportDegradesWithoutGit` |
| AC-5 | `action.yml` | `TestActionPassesInputsViaEnv` |

## Sicherheit

- Nur lesende Git-Aufrufe (`rev-parse`, `diff`, `show`); `ref` und Pfad werden als ein Argument `<ref>:<path>` übergeben, nie über eine Shell.
- Der Action-Schritt erhält keine Eingaben über Ausdrücke im Skripttext.

## Offene Punkte

- Ob die Job-Zusammenfassung als Ersatz für den PR-Kommentar genügt, zeigt der Alltag; ein optionaler Kommentar mit `pull-requests: write` wäre ein eigenes Feature.
