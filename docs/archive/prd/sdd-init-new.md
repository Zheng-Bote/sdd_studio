---
feature: sdd-init-new
status: archived
release: v0.1.0
archive-after:
touches:
  - cmd/**
  - internal/sddcore/scaffold/**
  - templates/**
  - embed.go
  - action.yml
  - .github/workflows/**
  - handbuch/**
---

# PRD — Projektanlage, Artefakt-Templates und CI-Baustein (`sdd-init-new`)

## Problem

Mit dem Prüfkern (`sdd-core-v1`) kann `sdd` ein Repository prüfen — aber nur, wenn Konfiguration, Verfassung und Artefaktverzeichnisse bereits existieren und die Feature-Artefakte genau die Frontmatter-Felder tragen, die die Regeln verlangen. Ein Prüfwerkzeug, das Felder verlangt, die es selbst nicht erzeugt, ist inkonsistent. Ohne CI-Baustein läuft außerdem kein Gate dort, wo es Schaden verhindert.

## Ziel

Ein neues Projekt ist mit einem Befehl prüfbar und ab dem ersten Pull Request in CI abgesichert. Jedes Artefakt, das `sdd` verlangt, lässt sich mit `sdd new` so anlegen, dass es die Regeln des Katalogs strukturell erfüllt.

## Nutzer

- **Entwicklerinnen und Entwickler**, die ein Projekt aufsetzen oder ein Feature beginnen und dem Gate-Hinweis `sdd new --feature <slug>` folgen.
- **CI-Pipelines**, die `sdd gate` ohne eigene Installationslogik aufrufen.

## Umfang

`sdd init`, `sdd new`, eingebettete Templates und eine GitHub Action (Composite Action) als dünner Wrapper um `sdd gate`.

## Nicht-Ziele

- `sdd adopt` (Bestandsprojekte inventarisieren und klassifizieren) — Roadmap-Schritt 3.
- Interaktive Assistenten, Editor-Integration und `sdd lint --watch`.
- Wrapper für andere CI-Systeme als GitHub Actions; diese rufen `sdd gate` direkt auf.
- Erkennen der Projektsprache oder der Layer-Struktur; `thresholds.layers` bleibt Aufgabe des Projekts.

## Akzeptanzkriterien

- **AC-1** `sdd init` legt `sdd.config.yaml`, die Verfassung, die konfigurierten Artefaktverzeichnisse, einen CI-Workflow und eine leere Baseline an; ein anschließendes `sdd lint` meldet keinen `error`.
- **AC-2** `sdd init` und `sdd new` überschreiben keine bestehende Datei; vorhandene Dateien werden als übersprungen gemeldet, und `--dry-run` schreibt nichts.
- **AC-3** `sdd init --profile` akzeptiert ausschließlich `minimal`, `standard` und `regulated` und schreibt das gewählte Profil in die Konfiguration.
- **AC-4** `sdd new --feature <slug>` legt fehlendes PRD und fehlende Tech Spec mit `feature`, `status: draft`, `archive-after` und `touches` an; existiert für den Slug bereits ein Artefakt des Typs, bleibt es unverändert und wird nicht dupliziert.
- **AC-5** `sdd new --type plan|adr|doc` legt einen Plan mit `feature`-Feld, ein ADR mit `status: proposed` und fortlaufender vierstelliger Nummer bzw. ein Dokument mit `status: active` an.
- **AC-6** Ein mit `sdd new` erzeugtes Artefakt erfüllt `SDD-021` und `SDD-050` ohne Nacharbeit; ungültige Slugs, ungültige Daten und Zielpfade außerhalb des Repositorys werden mit Exit-Code `2` abgelehnt.
- **AC-7** Die GitHub Action baut `sdd` aus dem eigenen Quellstand, führt `sdd gate <gate>` im Arbeitsverzeichnis aus und übernimmt dessen Exit-Code; Eingaben gelangen nur über Umgebungsvariablen in das Skript.
