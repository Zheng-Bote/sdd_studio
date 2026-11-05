---
feature: requirements-depth
status: active
archive-after:
touches:
  - internal/mcp/**
  - handbuch/**
  - README.md
  - CHANGELOG.md
---

# Tech Spec — Erweitertes Anforderungsmanagement von der Idee zum PRD

Setzt das PRD `docs/prd/requirements-depth.md` um, auf Grundlage von ADR 0007. Reine Prompt- und Doku-Änderung; der Prüfkern bleibt unverändert.

## Architektur

```text
internal/mcp/prompts.go   neuer Prompt sdd-requirements; Hinweise in sdd-idea, sdd-start, sdd-arc42
handbuch/03-mechanismen.md  optionale PRD-Abschnitte und QA-<n> beim PRD-Template
handbuch/04-agenten.md      Liste der geführten Abläufe
README.md, CHANGELOG.md
```

### Prompt `sdd-requirements` (AC-1 – AC-5)

Argument `feature` (Pflicht). Regeln im Kopf: SDD-021, SDD-023. Schritte:

1. `sdd_read` des PRDs (`sdd_read` ohne path, falls der Pfad unbekannt ist). Kurz zusammenfassen, was schon da ist.
2. **Qualitätsanforderungen:** die sieben Merkmale einzeln (Leistung und Effizienz, Zuverlässigkeit und Verfügbarkeit, Sicherheit, Wartbarkeit, Benutzbarkeit, Kompatibilität, Übertragbarkeit und Betrieb). Je Merkmal: „Zählt es hier?“; wenn ja, Maß, Zielwert und Bedingung erfragen und als `- **QA-<n>** <Merkmal>: <Zielwert> <Maß> unter <Bedingung>` formulieren. Nächste freie Nummer, nie umnummerieren.
3. **Randbedingungen** (Technik, Recht, Organisation, Betrieb) und **Annahmen und Abhängigkeiten** (was als gegeben gilt, wovon das Feature abhängt).
4. **Grenzfälle und Fehlerverhalten:** leere und ungültige Eingabe, nicht erreichbar, fehlende Rechte, Zeitüberschreitung, gleichzeitiger Zugriff, Wiederholung nach Abbruch. Testbare Antworten werden zu neuen `AC-<n>` im Abschnitt Akzeptanzkriterien, auf Wunsch als Gegeben/Wenn/Dann.
5. **Vage Wörter:** Kriterien und Qualitätsanforderungen auf „schnell“, „einfach“, „benutzerfreundlich“, „sollte“, „möglichst“, „usw.“, „ggf.“, „etc.“, „flexibel“, „intuitiv“ durchgehen; je Fund eine messbare Fassung vorschlagen, nur nach Zustimmung übernehmen.
6. Entwurf mit `sdd_validate_draft` prüfen, mit `sdd_write` speichern (base_sha256), mit `sdd_read` gegenlesen.

Abschnittsreihenfolge im PRD: nach `## Umfang` bzw. `## Nicht-Ziele` kommen `## Qualitätsanforderungen`, `## Randbedingungen`, `## Annahmen und Abhängigkeiten`, danach `## Akzeptanzkriterien`. Die Abschnitte entstehen nur, wenn es Inhalt gibt.

Der Prompt nennt ausdrücklich: keine Prioritäten oder Schätzungen; eine Frage nach der anderen; Merkmale, die nicht zählen, werden übersprungen, nicht erfunden.

### Hinweise in anderen Abläufen (AC-6)

- `sdd-idea`, letzter Schritt vor der Zusammenfassung: „Ist das Vorhaben mittelgroß oder größer (mehrere Komponenten, Qualitätsanforderungen, mehrere Beteiligte), biete den Ablauf sdd-requirements an.“
- `sdd-start`, Schritt 4: dasselbe Angebot nach dem ersten PRD.
- `sdd-arc42`, Schritt 4: „Qualitätsziele (Abschnitt 1) und Qualitätsszenarien (Abschnitt 10) aus den QA-<n> der PRDs, mit Kennung als Quelle.“

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `prompts.go` | `TestRequirementsPrompt` |
| AC-2 | `prompts.go` | `TestRequirementsPrompt` |
| AC-3 | `prompts.go` | `TestRequirementsPrompt` |
| AC-4 | `prompts.go` | `TestRequirementsPrompt` |
| AC-5 | `prompts.go` | `TestRequirementsPrompt` |
| AC-6 | `prompts.go` | `TestRequirementsOffered` |
| AC-7 | Handbuch, README, CHANGELOG | `TestRequirementsDocumented` |
