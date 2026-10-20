---
feature: sdd-ui
status: active
archive-after:
touches:
  - cmd/**
  - internal/ui/**
  - internal/sddcore/**
  - handbuch/**
  - sdd.config.yaml
---

# PRD — Lokale Oberfläche (`sdd-ui`)

## Problem

Die CLI liefert alles, aber verteilt über `lint`, `gate`, `report`, `new` und `adopt`. Wer einen Bestand übernimmt oder Befunde abarbeitet, springt zwischen Terminal, Editor und Handbuch. Eine Übersicht über Artefakte, Status und Archivfristen gibt es nicht.

## Ziel

`sdd ui` öffnet eine lokale, schreibgeschützte Sicht auf dieselbe Bibliothek, mit wenigen, ausdrücklich bestätigten Schreibaktionen — ohne neue Angriffsfläche über den eigenen Rechner hinaus und ohne Funktion, die die CLI nicht auch hat.

## Nutzer

- **Entwicklerinnen und Entwickler**, die Befunde abarbeiten und direkt an die Fundstelle im Editor springen.
- **Verantwortliche einer Übernahme**, die den Adopt-Ablauf mit Probelauf nachvollziehen wollen.

## Umfang

Befehl `sdd ui` mit fünf Ansichten (Übersicht, Artefakte, Befunde, Neues Artefakt, Übernahme), eingebettetem Handbuch und Befehlspalette.

## Nicht-Ziele

- Ein Editor für Artefakte; Dateien öffnet der eigene Editor.
- Bearbeiten der Baseline oder des ADR-Lockfiles; das bleibt `sdd baseline` vorbehalten.
- Zugriff aus dem Netz, Mehrbenutzerbetrieb, Anmeldung.
- Eine Desktop-Hülle (Tauri) und externe Schriften, Skripte oder Telemetrie.

## Akzeptanzkriterien

- **AC-1** `sdd ui` bindet ausschließlich an `127.0.0.1` (Standard: freier Port) und gibt eine URL mit einem beim Start erzeugten Sitzungstoken aus.
- **AC-2** Anfragen mit einem anderen `Host` als der gebundenen Adresse werden mit `403` abgewiesen; keine Antwort trägt `Access-Control-Allow-Origin`; eine Content-Security-Policy erlaubt nur Ressourcen derselben Herkunft, und die ausgelieferten Dateien verweisen auf keine externe Adresse.
- **AC-3** Schreibende Anfragen verlangen das Sitzungstoken und einen `Origin` gleich der gebundenen Adresse; lesende Anfragen kommen ohne Token aus.
- **AC-4** Die Übersicht zeigt Reifegrad, Gates und Schuldenstand; die Artefaktansicht jede Datei mit Typ, Status, Alter und Archivfrist; die Befundliste lässt sich nach Gate und nach „neu“/„Altbestand“ filtern, als JSON oder CSV exportieren und verlinkt Handbuch und Fundstelle.
- **AC-5** „Neues Artefakt“ und „Übernahme“ zeigen vor dem Schreiben den vollständigen Plan mit Dateien und Inhalt; geschrieben wird erst nach Bestätigung und nur über den Plan/Apply-Pfad von `sdd new` bzw. `sdd adopt --apply`; es gibt keinen Endpunkt, der Baseline oder Lockfile verändert.
- **AC-6** Die Befundliste aktualisiert sich ohne Neuladen, wenn sich Dateien im Repository ändern.
- **AC-7** Eine Befehlspalette (`Strg+K`) erreicht jede Ansicht; das Öffnen einer Fundstelle im Editor (`$EDITOR`) ist eine tokengeschützte Aktion und nur für Pfade innerhalb des Repositorys möglich.
