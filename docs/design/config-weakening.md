---
feature: config-weakening
status: active
archive-after:
touches:
  - catalog/**
  - internal/sddcore/catalog/**
  - internal/sddcore/rules/**
  - handbuch/**
  - docs/sdd_konzept.md
  - README.md
  - CHANGELOG.md
---

# Tech Spec — Abschwächung der Konfiguration braucht ein ADR (SDD-012)

Setzt das PRD `docs/prd/config-weakening.md` um, auf Grundlage von ADR 0005. Eine Regel je Commit: zuerst `pinned`, dann SDD-012, jeweils mit Handbuch, Vertrag und Tests.

## Architektur

```text
internal/sddcore/catalog/catalog.go   Rule.Pinned; Parse prüft pinned ⇒ profile minimal und baseline forbidden
internal/sddcore/rules/registry.go    EffectiveRulesForGate: pinned-Regeln ihrer gate_default stets hinzufügen
internal/sddcore/rules/config.go      checkSDD012: Vergleich Merge-Basis ↔ HEAD, ADR im Diff
internal/sddcore/rules/selfcheck.go   Registrierung config_weakening_has_adr
catalog/catalog.yaml                  SDD-012, version 1.2.0
handbuch/03-mechanismen.md            SDD-012 mit Vertrag; pinned bei SDD-072 und beim Katalog
```

### `pinned` (AC-1, AC-2)

`Rule` erhält `Pinned bool` (`yaml:"pinned"`). Beim Parsen gilt: `pinned: true` verlangt `profile: minimal` und `baseline: forbidden`, sonst kommt ein Fehler mit der Regel-ID.

`EffectiveRulesForGate`: Mit nicht leerem Override ist die Menge `override(g)` ∪ {Regeln mit `pinned` und `g ∈ gate_default`}. Ohne Override bleibt sie unverändert, denn eine `minimal`-Regel ist dort ohnehin aktiv. SDD-072 nutzt dieselbe Funktion und sieht `pinned` damit automatisch.

### SDD-012 (AC-3, AC-4)

Das Gerüst folgt `checkSDD010`:

1. Nur in `pre-implementation` und `pre-merge`. `sdd.config.yaml` muss in `ctx.changedFiles()` vorkommen.
2. `before := gitSource().ShowFile(baseRef, "sdd.config.yaml")`. Fehlt die Datei an der Basis, ist das ein erstmaliges Anlegen, und die Regel meldet nichts. `after` ist die Datei im Arbeitsbaum. Lässt sich eine der beiden nicht parsen, meldet die Regel nichts; die aktuelle Datei prüft schon das Laden der Konfiguration.
3. `weakenings(before, after)` liefert eine sortierte Liste von Texten:
   - `profile standard → minimal`, wenn `rank(after) < rank(before)`;
   - `gates.<gate>.override geändert`, wenn sich die Liste als Menge unterscheidet;
   - `exclude: +<muster>` je neuem Muster;
   - `thresholds.layers: −<eintrag>` bzw. `touched_paths: −<eintrag>` je fehlendem Eintrag;
   - `min_layers_touched <a> → <b>`, wenn `b > a`.
4. Ist die Liste leer, meldet die Regel nichts. Sonst prüft sie wie SDD-010, ob ein ADR im Diff `sdd.config.yaml` nennt (`readADRLinks(...).names(f, "sdd.config.yaml")`). Fehlt es, entsteht ein `error` an `sdd.config.yaml:1` mit den Abschwächungen und dem Hinweis auf `references: [sdd.config.yaml]`. `ContentHash` wird aus Regel und Liste gebildet.

### Katalog (AC-5)

```yaml
- id: SDD-012
  title: Abschwächende Änderung der Konfiguration verweist auf ein ADR
  severity: {by_gate: {pre-implementation: error, pre-merge: error}}
  profile: minimal
  pinned: true
  check: config_weakening_has_adr
  gate_default: [pre-implementation, pre-merge]
  baseline: forbidden
```

`version: "1.2.0"`.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `registry.go` | `TestPinnedSurvivesOverride` |
| AC-2 | `catalog.go` | `TestPinnedNeedsMinimalAndForbidden` |
| AC-3 | `rules/config.go` | `TestSDD012Weakenings` |
| AC-4 | `rules/config.go` | `TestSDD012NeedsADR`, `TestSDD012IgnoresHarmless` |
| AC-5 | `catalog.yaml` | `TestSDD012InCatalog` |
| AC-6 | Handbuch, Konzept, README, CHANGELOG | `TestSDD012Documented` |

## Risiken

- **Ein falsches „abschwächend“** blockiert einen PR, der nur aufräumt, etwa ein `exclude` für einen gelöschten Ordner. Ein kurzes ADR löst das auf, und das ist gewollt: Auch Aufräumen an der Prüfung ist eine Entscheidung.
- **Der eigene Repo-Lauf** (Dogfooding): Die Umsetzung ändert `sdd.config.yaml` nicht, also löst sie SDD-012 nicht selbst aus.
