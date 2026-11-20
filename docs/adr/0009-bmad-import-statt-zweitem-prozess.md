---
status: accepted
date: 2026-11-20
supersedes: []
references:
  - constitution.md
  - docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md
  - docs/adr/0004-konfiguration-aendern-ist-kritisch.md
  - docs/prd/bmad-import.md
  - docs/design/bmad-import.md
---

# ADR 0009 — BMAD-Bestand übernehmen statt einen zweiten Prozess führen

## Kontext

Die Kombination aus BMAD, GitHub Spec Kit und SpecDD verteilt die Arbeit auf drei Ebenen: BMAD liefert Makro-Planung und Rollen (Vision, Initiative, Epics, Architektur), Spec Kit die Feature-Strecke (Spec, Plan, Tasks), SpecDD die systemweiten Regeln, die alle späteren Specs erben. Der Wert der Kombination liegt in der Rückverfolgung über diese Ebenen: Eine Anforderung der Initiative muss in einem Epic auftauchen, ein Epic in einer Spec, ein Akzeptanzkriterium in einem Test.

`sdd` deckt heute nur die Feature-Ebene und die Regeln. Ein Projekt, das mit BMAD v6 beginnt, hat `_bmad-output/` mit einer Initiative (nummerierte Anforderungen `R-<n>`), acht Epics und einer Architektur-Spine. `sdd adopt` erkennt davon nichts: `Classify` prüft ausschließlich die konfigurierten Artefaktpfade (`docs/prd`, `docs/design`, `docs/plans`, `docs/archiv`, `docs/adr`), alles andere wird „Doku/Exploration“ mit der Begründung „kein Muster“. Damit beginnt die Übernahme mit einem leeren Verfassungsgerüst und lässt den einzigen vorhandenen Kontext liegen.

Erwogen wurden:

1. **Sich auf Spec Kit beschränken** und BMAD als Vorgeschichte behandeln — verworfen. Genau die Makro-Ebene ist der Teil, der später driftet: Ohne Bindung zwischen Anforderung, Epic und Spec prüft niemand mehr, ob ein Epic noch etwas deckt.
2. **BMAD-Werkzeuge mitinstallieren und den Prozess an BMAD delegieren** — verworfen. `sdd` ist KI-frei und offline (ADR 0003); BMAD ist ein Agenten-Rahmen mit Rollenpersonas und eigenem Installer. Ein zweiter Prozess neben dem Regelkatalog hätte zwei Wahrheiten, und die Präzedenzordnung wäre nicht mehr eindeutig.
3. **Eigene Artefakttypen und Regeln für den BMAD-Bestand, ohne zweite Engine** — gewählt.

## Entscheidung

1. **Ein Streckenprofil statt einer zweiten Engine.** `sdd.config.yaml` erhält `pipeline` (`""`, `sdd` oder `bmad`). Das Profil schaltet ausschließlich Prüfungen und Vorgabepfade; der Prüfkern bleibt derselbe. Ohne Feld ändert sich nichts.
2. **Zwei Makro-Artefakttypen: Initiative und Epic.** Die Initiative trägt die Anforderungen `R-<n>` samt Constraints und Nicht-Zielen; ein Epic trägt `covers` auf diese Anforderungen und wird beim Verfeinern zum Feature mit `AC-<n>`. Beide sind für den Prüfkern **keine** Feature-Artefakte: sie bekommen kein `feature:`-Frontmatter, und SDD-020 bis SDD-041 greifen für sie nicht.
3. **BMAD v6 wird erkannt, v4 nicht.** Merkmal sind die Verzeichnisse `_bmad/` und `_bmad-output/`. `.bmad-core/` (v4) bleibt unberücksichtigt; es gibt keinen Bestand, der es verlangt, und zwei Erkennungen wären zwei Pfade zum Pflegen.
4. **Die Übernahme kopiert, sie aliast nicht.** `sdd adopt` schreibt die BMAD-Dateien nicht um; `_bmad-output/` bleibt BMADs Ledger. Feature-Artefakte entstehen dort, wo die Konfiguration sie erwartet. Technischer Grund: `scan.TargetPath` ersetzt genau ein ganzes `*`-Segment; ein Glob mit zwei Segmenten (`_bmad-output/*/tickets/*/epic-*.md`) wäre für `sdd new` nicht beschreibbar.
5. **Eine Verfassung, die von `sdd`.** Die Verfassung entsteht aus den Constraints des Bestands und aus der Architektur-Spine, Regel für Regel mit Fundstelle, entschieden vom Menschen. Spec Kits eigener `/constitution`-Befehl wird nicht benutzt; zwei Verfassungen würden die Präzedenzordnung brechen. Bei der Erstanlage auf dem Übernahmebranch ist kein ADR nötig (`SDD-010`); danach ist jede Änderung eine Verfassungsänderung.
6. **Rollen bleiben Prompt-Text.** Analyst, Product Manager, Architect und QA sind Abläufe in `sdd mcp`, keine Personas im Code und keine zusätzliche Installation. Ein Prompt ist prüfbar, ein Persona-Rahmen nicht.
7. **Drei Regeln, nicht mehr.** `SDD-080` (Epic nennt eine existierende Anforderung), `SDD-081` (jede Anforderung ist gedeckt oder ausdrücklich nicht geplant), `SDD-082` (Index und Epic-Dateien nennen dieselben Slugs). Keine Regel „Feature nennt sein Epic“ — die braucht die Story-Ebene, die es noch nicht gibt. Keine Regel für Architektur-Abdeckung — sie ist nicht deterministisch prüfbar und bleibt ADR-Pflicht aus `SDD-010`.

## Konsequenzen

- **Verfassung**, Architektur/Layering: Die Makro-Ebene ist Teil des Artefaktmodells, nicht der Verfassung; Feature-Inhalte bleiben draußen (`SDD-003`).
- **Konzept §4.3** (Artefakttypen) und §8.3 (Projektkonfiguration) werden um Initiative und `pipeline` ergänzt; `docs/sdd_process_3.md` §11 (Dokumentstruktur) bleibt gültig, weil BMADs Ablage eine Quelle ist und kein Ziel.
- **Bestehende Projekte** sind nicht betroffen: ohne `pipeline: bmad` und ohne `_bmad/` bleibt jedes Verhalten wie bisher; ein Test hält das fest.
- **Risiko:** Ein BMAD-Projekt kann die Regeln umgehen, indem es `pipeline` entfernt. Das ist keine Lücke, sondern die bewusste Entscheidung, dass die Makro-Prüfung nur dort gilt, wo die Makro-Ebene erklärt ist — und eine Abschwächung der Konfiguration braucht nach `SDD-012` ohnehin ein ADR.
- **Risiko:** Zwei Ablagen (`_bmad-output/` und `docs/`) können auseinanderlaufen. Gegenmaßnahme ist `SDD-082` für den Index und der Grundsatz, dass nach der Übernahme `docs/` die Wahrheit ist.
