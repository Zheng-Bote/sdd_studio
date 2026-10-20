---
status: active
---

# Adopt-Bericht — sdd_studio

Angewendet am 2026-10-20 auf Branch `sdd/adopt` (ausgehend von `main`), Profil `standard`.

## Reifegrad: L2

| Stufe | Kriterium | Erfüllt | Befund |
|---|---|---|---|
| L1 | Verfassung vorhanden | ja | constitution.md |
| L1 | Verfassung ohne Befunde (SDD-001, SDD-002a) | ja | 0 Befunde |
| L2 | PRD und Tech Spec getrennt vorhanden | ja | 3 PRD, 3 Tech Spec |
| L2 | Akzeptanzkriterien mit Kennung | ja | 24 AC-Kennungen, 0 Befunde SDD-021 |
| L3 | CI ruft sdd gate auf | ja | 1 Workflow-Datei(en) |
| L3 | Baseline vorhanden | nein | .sdd/baseline.json |
| L4 | Feature-Artefakte tragen feature: und status | ja | 0 von 6 ohne beide Felder |
| L4 | Archiv konfiguriert | ja | docs/archive |
| L4 | Archivierung aktuell (SDD-050) | ja | 0 Befunde |

**Nächster Schritt (L3):** Baseline aufnehmen (Schritt „Baseline“ von sdd adopt --apply oder `sdd baseline`).

Der Reifegrad beschreibt den Stand vor der Übernahme.

## Inventar

- Artefaktpfade: vorhandene sdd.config.yaml
- Verfassung: 1
- ADR: 1
- PRD: 3
- Tech Spec: 3
- Plan: 0
- Doku/Exploration: 4
- CI-Workflows: 1 (ruft `sdd gate` auf: ja)

| Datei | Typ | Grund |
|---|---|---|
| `README.md` | Doku/Exploration | kein Muster |
| `constitution.md` | Verfassung | artifacts.constitution |
| `docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md` | ADR | NNNN-*.md unter docs/adr |
| `docs/design/sdd-adopt.md` | Tech Spec | artifacts.design (docs/design) |
| `docs/design/sdd-core-v1.md` | Tech Spec | artifacts.design (docs/design) |
| `docs/design/sdd-init-new.md` | Tech Spec | artifacts.design (docs/design) |
| `docs/prd/sdd-adopt.md` | PRD | artifacts.prd (docs/prd) |
| `docs/prd/sdd-core-v1.md` | PRD | artifacts.prd (docs/prd) |
| `docs/prd/sdd-init-new.md` | PRD | artifacts.prd (docs/prd) |
| `docs/sdd_konzept.md` | Doku/Exploration | kein Muster |
| `docs/sdd_process_3.md` | Doku/Exploration | kein Muster |
| `handbuch/03-mechanismen.md` | Doku/Exploration | kein Muster |

Die Zuordnung ist ein Vorschlag nach festen Mustern; „kein Muster“ heißt Doku/Exploration, nicht „geraten“.

## Schritte

1. **Baseline** — Commit `e532bc0`: .sdd/baseline.json
2. **Bericht** — diese Datei

Baseline: 0 Einträge.

## Konflikte

Keine.

## Archivierung vor dem ersten Release

Kein Feature-Artefakt ist abgeschlossen oder abgelaufen.

## Hinweise

- Ergänzte Feature-Artefakte tragen `status: active`. Längst umgesetzte Features auf `status: implemented` setzen — dann weist sie SDD-050 für die Archivierung aus.
