---
feature: sdd-conflicts
status: archived
release: v0.4.0
archive-after:
touches:
  - internal/mcp/**
  - internal/desk/**
  - docs/sdd_konzept.md
  - handbuch/**
  - README.md
---

# PRD — Assistierter Abgleich von Tech Spec und Verfassung (`SDD-060`)

## Problem

Eine Tech Spec kann einer Regel der Verfassung widersprechen — etwa eine ausgehende Verbindung vorsehen, wo die Verfassung „offline“ verlangt, oder eine Schicht umgehen. Deterministisch prüfen lässt sich das nicht: Ob zwei Texte einander widersprechen, ist eine Frage der Bedeutung. Das Konzept führt dafür `SDD-060` als „assistiert, nicht automatisch entscheidbar“; umgesetzt ist es nicht. Ein Sprachmodell in `sdd-desk` könnte helfen, kann die Inhalte von Verfassung und Tech Spec aber heute nicht lesen: `sdd mcp` liefert Befunde, Prozessstand und Rückverfolgung, keine Artefakttexte als Werkzeug.

## Ziel

In `sdd-desk` lässt sich ein Feature gegen die Verfassung abgleichen: Das Modell liest Verfassung, Tech Spec und verlinkte ADRs und nennt mögliche Widersprüche mit wörtlichen Zitaten beider Seiten. Der Mensch entscheidet, ob die Tech Spec angepasst oder die Verfassung über ein ADR geändert wird. `sdd` bleibt KI-frei; das Ergebnis ist kein Befund, steht in keinem Bericht und berührt keine Baseline.

## Nutzer

- **Autoren von Tech Specs**, die vor dem Review prüfen wollen, ob ihr Entwurf der Verfassung folgt.
- **Reviewer**, die einen Hinweis auf Stellen suchen, die sie genauer lesen sollten.

## Umfang

Ein lesendes Werkzeug für Artefakttexte in `sdd mcp`, ein geführter Ablauf „Abgleich mit der Verfassung“ und die Einordnung von `SDD-060` in Konzept und Handbuch als assistierter Abgleich statt Katalogregel.

## Nicht-Ziele

- `SDD-060` als Regel im Katalog, als Befund im Bericht oder in der Baseline.
- Automatisches Ändern von Tech Spec, Verfassung oder ADR.
- Maschinelles Prüfen der Zitate gegen die Dateien (möglicher späterer Schritt).
- Abgleich anderer Artefaktpaare (PRD ↔ Tech Spec, ADR ↔ Verfassung).
- Lesen von Dateien, die keine Artefakte sind (Code, Konfiguration, Geheimnisse).

## Akzeptanzkriterien

- **AC-1** Das Werkzeug `sdd_read` liefert ohne Pfad die Liste der lesbaren Artefakte (Verfassung, ADRs, PRDs, Tech Specs, Pläne, Archiv) und mit Pfad den Text eines dieser Artefakte mit Zeilennummern; es schreibt nichts.
- **AC-2** `sdd_read` lehnt jeden Pfad ab, der kein lesbares Artefakt ist — Dateien außerhalb der Artefaktpfade, Pfade außerhalb des Repositorys, `..`, absolute Pfade, andere Endungen als `.md` und `.sdd` — und kürzt Texte über 256 KiB mit einem Hinweis.
- **AC-3** Der geführte Ablauf `sdd-conflicts` mit Argument `feature` weist das Modell an, Verfassung, Tech Spec und verlinkte ADRs mit `sdd_read` zu lesen, nur Widersprüche mit wörtlichem Zitat und Zeile beider Seiten zu melden, jeden als Widerspruch, Spannung oder unklar einzuordnen, „Tech Spec anpassen“ oder „ADR für eine Verfassungsänderung“ vorzuschlagen und nichts zu ändern.
- **AC-4** `SDD-060` steht nicht im Katalog; Konzept und Handbuch beschreiben es als assistierten Abgleich in `sdd-desk`, dessen Ergebnis kein Befund ist.
