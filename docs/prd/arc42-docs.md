---
feature: arc42-docs
status: active
archive-after:
touches:
  - templates/**
  - internal/sddcore/scaffold/**
  - internal/sddcore/guide/**
  - internal/sddcore/scan/**
  - internal/sddcore/config/**
  - internal/mcp/**
  - cmd/sdd/**
  - handbuch/**
  - README.md
  - docs/sdd_konzept.md
  - NOTICE
  - CHANGELOG.md
---

# PRD — Architekturdokumentation nach arc42 für Projekte

## Problem

Ein mit `sdd` geführtes Projekt hat Verfassung, ADRs und Feature-Artefakte, aber keinen Ort, an dem die Architektur als Ganzes beschrieben ist. Die Tech Specs werden mit dem Release archiviert. Danach muss man das Gesamtbild aus vielen Einzeldokumenten zusammensetzen. Der Nutzer will zu jedem fertigen Stand eine abschließende Architekturdokumentation `docs/arc42.md` nach arc42. ADR 0006 hat entschieden, wie `sdd` das arc42-Template unter CC BY-SA 4.0 mitliefert.

## Ziel

`sdd` legt die Architekturdokumentation aus einem mitgelieferten deutschen arc42-Template an. `sdd next` erinnert daran, sie zu jedem Release nachzuführen. Der KI-Desktop füllt sie in einem geführten Ablauf aus den Quellen des Projekts, ohne etwas zu erfinden.

## Nutzer

- **Teams**, die zu jedem Release einen aktuellen Architekturüberblick wollen.
- **Neue Mitwirkende**, die sich in ein Projekt einlesen.
- **Nutzer des KI-Desktops**, die die Doku nicht von Hand zusammentragen wollen.

## Umfang

- Template `templates/arc42.md.tmpl` unter CC BY-SA 4.0, Eintrag in `NOTICE`.
- `sdd new --type arc42` und `sdd_new` mit Typ `arc42`.
- Ein Projektschritt in `sdd next`.
- Ein geführter Ablauf `sdd-arc42` in `sdd mcp`.
- Handbuch, README, CHANGELOG.

## Nicht-Ziele

- Eine Katalogregel, die die Doku verlangt. Sie bleibt ein Schritt in `sdd next`, kein Befund.
- Die Doku bei `sdd init` oder `sdd adopt` automatisch anlegen. Zu Beginn gibt es noch keinen fertigen Stand.
- Andere Sprachen als Deutsch für das Template.

## Akzeptanzkriterien

- **AC-1** `templates/arc42.md.tmpl` enthält die zwölf Abschnitte von arc42 9.1 mit deutschen Überschriften und den Ankern des Templates. Im Kopf stehen Urheber, Quelle, Version, Lizenz CC BY-SA 4.0 und die Kennzeichnung der Änderung. `NOTICE` nennt die Datei mit ihrer Lizenz.
- **AC-2** `sdd new --type arc42` legt `<artifacts.docs[0]>/arc42.md` an, ohne `--title`. Die Datei hat Frontmatter mit `status: active` und einem leeren `release:` sowie eine Zeile mit Namensnennung und Lizenz der Gliederung. Eine bestehende Datei wird nie überschrieben.
- **AC-3** `sdd_new` legt mit `types: [arc42]` dieselbe Datei an, auch mit `content`.
- **AC-4** `sdd next` zeigt einen Projektschritt „Architekturdoku nachführen“. Er ist offen, sobald mindestens ein Feature-Artefakt mit `release:` im Archiv liegt und `<artifacts.docs[0]>/arc42.md` fehlt oder in `release:` nicht das höchste Release der archivierten Artefakte nennt. Sonst entfällt der Schritt.
- **AC-5** Der geführte Ablauf `sdd-arc42` liest Verfassung, ADRs, aktive und archivierte Tech Specs und Kontextdateien mit `sdd_read`. Er füllt Abschnitt für Abschnitt nur mit Belegtem, mit Verweis auf die Quelle, lässt Unbelegtes als offen stehen, setzt `release:` auf das genannte Release und speichert mit `sdd_new` bzw. `sdd_write`.
- **AC-6** Handbuch, README und CHANGELOG beschreiben Template, Befehl, Schritt und Ablauf.
- **AC-7** Das Template enthält Mermaid-Gerüste in Kontextabgrenzung, Bausteinsicht, Laufzeitsicht und Verteilungssicht, die sich fehlerfrei rendern lassen. `sdd-arc42` zeichnet diese Diagramme in Mermaid (flowchart, sequenceDiagram) nur aus belegten Elementen.
