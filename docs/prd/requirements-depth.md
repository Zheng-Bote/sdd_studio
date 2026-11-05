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

# PRD — Erweitertes Anforderungsmanagement von der Idee zum PRD

## Problem

`sdd-idea` führt schlank zum PRD: Problem, Ziel, Nutzer, Umfang, Nicht-Ziele, Akzeptanzkriterien, Technik. Wer ein mittleres oder größeres Projekt beginnt, braucht mehr. Messbare Qualitätsanforderungen, Randbedingungen, Annahmen und das Verhalten im Fehlerfall werden sonst erst während der Umsetzung entdeckt, wenn sie teuer sind. Im Live-Test wuchs ein kleines PRD erst nach mehreren Rückfragen von vier auf zehn Kriterien.

## Ziel

Ein optionaler, geführter Ablauf vertieft ein PRD systematisch. Er erfragt Qualitätsanforderungen je Merkmal messbar, hält Randbedingungen und Annahmen fest, leitet aus Grenzfällen prüfbare Akzeptanzkriterien ab und weist auf vage Formulierungen hin. Der schlanke Weg bleibt unverändert.

## Nutzer

- **Teams**, die ein mittleres oder größeres Vorhaben beginnen.
- **Nutzer des KI-Desktops** und von Agenten über `sdd mcp`.

## Umfang

- Neuer Prompt `sdd-requirements` in `sdd mcp`.
- Hinweis in `sdd-idea` und `sdd-start` auf den vertieften Ablauf.
- `sdd-arc42` nutzt die Qualitätsanforderungen.
- Handbuch, README, CHANGELOG.

## Nicht-Ziele

- Prioritäten, Schätzungen, Releaseplanung (Konzept: kein Projektmanagement-Werkzeug).
- Eine neue Pflichtregel oder ein Testverweis für `QA-<n>` (ADR 0007).
- Eine Regel für vage Formulierungen. Der Ablauf weist nur darauf hin.
- Änderungen am PRD-Template. Die zusätzlichen Abschnitte entstehen nur, wenn der Ablauf sie füllt.

## Akzeptanzkriterien

- **AC-1** `sdd mcp` bietet den Prompt `sdd-requirements` mit Pflichtargument `feature` an. Er liest das PRD des Features mit `sdd_read` und ändert es nur mit `sdd_write` nach `sdd_validate_draft`.
- **AC-2** Der Ablauf geht die Qualitätsmerkmale einzeln durch (Leistung und Effizienz, Zuverlässigkeit und Verfügbarkeit, Sicherheit, Wartbarkeit, Benutzbarkeit, Kompatibilität, Übertragbarkeit und Betrieb). Er fragt je Merkmal, ob es für das Vorhaben zählt, und hält zählende als `QA-<n>` mit Maß, Zielwert und Bedingung im Abschnitt `## Qualitätsanforderungen` fest.
- **AC-3** Der Ablauf erfragt Randbedingungen sowie Annahmen und Abhängigkeiten und hält sie in `## Randbedingungen` und `## Annahmen und Abhängigkeiten` fest.
- **AC-4** Der Ablauf erfragt Grenzfälle und Fehlerverhalten systematisch (leere oder ungültige Eingabe, nicht erreichbar, fehlende Rechte, Zeitüberschreitung, gleichzeitiger Zugriff) und ergänzt testbare Fälle als neue `AC-<n>`. Auf Wunsch formuliert er sie im Schema Gegeben/Wenn/Dann. Bestehende Kennungen nummeriert er nie um.
- **AC-5** Der Ablauf prüft die Formulierungen der Kriterien und Qualitätsanforderungen auf vage Wörter (z. B. „schnell“, „benutzerfreundlich“, „sollte“, „usw.“, „ggf.“) und schlägt eine messbare Fassung vor. Er übernimmt sie erst nach Zustimmung.
- **AC-6** `sdd-idea` und `sdd-start` bieten nach dem PRD für mittlere und größere Vorhaben den Ablauf `sdd-requirements` an. `sdd-arc42` füllt Qualitätsziele und -szenarien aus den `QA-<n>` der PRDs.
- **AC-7** Handbuch, README und CHANGELOG beschreiben den Ablauf, die optionalen PRD-Abschnitte und `QA-<n>`.
