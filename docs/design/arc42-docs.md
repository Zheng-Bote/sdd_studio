---
feature: arc42-docs
status: active
archive-after:
touches:
  - templates/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/mcp/**
  - cmd/sdd/**
  - handbuch/**
  - README.md
  - NOTICE
  - CHANGELOG.md
---

# Tech Spec — Architekturdokumentation nach arc42 für Projekte

Setzt das PRD `docs/prd/arc42-docs.md` um, auf Grundlage von ADR 0006.

## Architektur

```text
templates/arc42.md.tmpl              deutsche Gliederung, CC-BY-SA-Kopf als HTML-Kommentar, {{.Project}}
NOTICE                               Abschnitt zum Template
internal/sddcore/scaffold/new.go     TypeArc42; otherPlan: Ziel <docs[0]>/arc42.md, kein --title nötig
internal/sddcore/guide/next.go       Guide.Architecture: Projektschritt, latestRelease, semver-Vergleich
internal/mcp/tools.go, prompts.go    Typ arc42 in sdd_new; Prompt sdd-arc42
cmd/sdd/main.go                      Hilfetext von new
```

### Template (AC-1)

Die Datei beginnt mit dem Frontmatter (`status: active`, `release:`), damit sie wie jedes Dokument mit `---` anfängt. Direkt danach folgt der Lizenzkopf als HTML-Kommentar: in der gerenderten Ansicht unsichtbar, in der Datei erhalten. Es folgen der Titel `Architekturdokumentation — {{.Project}}` und eine sichtbare Zeile „Gliederung nach dem arc42-Template 9.1 (© …), lizenziert unter CC BY-SA 4.0; deutsche Fassung der Gliederung“. Danach kommen die zwölf Abschnitte mit den Ankern des Originals (`{#section-introduction-and-goals}` …), je mit einem Platzhalter `<…>` als Kurzhinweis.

### `sdd new --type arc42` (AC-2, AC-3)

`TypeArc42 = "arc42"` zählt zu den Nicht-Feature-Typen. `otherPlan` braucht für `arc42` keinen Titel. Ziel: `path.Join(cfg.Artifacts.Docs[0], "arc42.md")`. `Project` ist `filepath.Base(absRoot)`. Ist `artifacts.docs` leer, kommt `ErrInvalidInput`. Bestehende Dateien überspringt der Plan wie bei jedem `create` (Status `skip`). `sdd_new` mit `content` nutzt den vorhandenen Weg (geprüfter Entwurf statt Template).

### `sdd next` (AC-4)

`Guide.Architecture *Step` (`json:"architecture,omitempty"`). Berechnung:

1. Unter `artifacts.archive` alle Artefakte mit `feature:` und nicht leerem `release:` sammeln, höchstes Release `R` per Versionsvergleich. Führendes `v` wird ignoriert, Bestandteile numerisch; nicht numerische fallen auf String-Vergleich zurück.
2. Kein `R` → kein Schritt.
3. `<docs[0]>/arc42.md` fehlt → offen: „Architekturdoku anlegen (Stand R)“, Befehl `sdd new --type arc42`, dann Ablauf `sdd-arc42`.
4. Frontmatter `release` ≠ `R` → offen: „Architekturdoku auf R nachführen (steht auf …)“, Befehl „Prompt sdd-arc42“.
5. Sonst kein Schritt.

Text- und JSON-Ausgabe von `sdd next` zeigen den Schritt wie `Constitution` und `Adoption`.

### Prompt `sdd-arc42` (AC-5)

Argument `release` (optional). Vorgehen: `sdd_read` ohne path; Verfassung, ADRs, Tech Specs (aktiv und Archiv), `README.md` und `AGENTS.md` lesen; `sdd_stack`. Liegt `docs/arc42.md` noch nicht vor, mit `sdd_new` (`types: [arc42]`) anlegen. Dann Abschnitt für Abschnitt: nur Belegtes, je Aussage die Quelle (Datei:Zeile oder ADR), Unbelegtes als „offen: …“, den Lizenzkopf unverändert lassen; `release:` setzen; mit `sdd_validate_draft` prüfen und mit `sdd_write` speichern; zum Schluss `sdd_next`. Regeln: SDD-001 (Verfassung als Quelle).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `templates/arc42.md.tmpl`, `NOTICE` | `TestArc42Template` |
| AC-2 | `scaffold/new.go` | `TestNewArc42` |
| AC-3 | `mcp/tools.go` | `TestNewToolArc42` |
| AC-4 | `guide/next.go` | `TestNextArchitectureStep` |
| AC-5 | `mcp/prompts.go` | `TestArc42Prompt` |
| AC-6 | Handbuch, README, CHANGELOG | `TestArc42Documented` |
