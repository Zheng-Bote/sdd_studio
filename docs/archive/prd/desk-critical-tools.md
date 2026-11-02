---
feature: desk-critical-tools
status: archived
archive-after:
touches:
  - cmd/sdd/**
  - internal/sddcore/*.go
  - internal/sddcore/scaffold/**
  - internal/mcp/**
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
  - README.md
release: v0.5.0
---

# PRD — Kritische Aktionen im KI-Desktop mit einzelner Bestätigung

## Problem

Nach ADR 0003 soll im KI-Desktop alles möglich sein, auch die Aktionen, die bisher dem Terminal vorbehalten waren: die Baseline pflegen (`sdd baseline`, `--accept`, `--prune`), das ADR-Lockfile aktualisieren, `sdd adopt --apply` und ein ADR entscheiden (`status: accepted`). Heute fehlen dafür Werkzeuge. Der Nutzer muss für jede dieser Aktionen das Terminal öffnen, wie im Praxistest nach der Entscheidung von ADR 0003. Gleichzeitig darf das Modell diese Aktionen nie selbst entscheiden (Konzept §9.6): Sie bilden die einzige Bremse des Systems.

## Ziel

Jede kritische Aktion lässt sich aus dem Desktop auslösen. Ausgeführt wird sie nur nach einer einzelnen, bewussten Bestätigung des Menschen, die die vollständige Wirkung zeigt. Weder eine Freigabe für die Sitzung noch eine Wiederholung nach einer Ablehnung führen am Menschen vorbei.

## Nutzer

- **Nutzer des KI-Desktops**, die den ganzen Prozess dort durchlaufen.
- **Reviewer**, die nachvollziehen wollen, wie eine Baseline-Änderung entstanden ist.

## Umfang

- Drei Werkzeuge in `sdd mcp` hinter `--allow-critical`.
- Gemeinsamer Code mit `sdd baseline`.
- Die Freigabestufe „kritisch“ in `sdd-desk` (Backend und Freigabekarte).
- Prompts, Handbuch, README.

## Nicht-Ziele

- Ein CLI-Befehl zum Entscheiden eines ADR. Im Terminal ändert der Mensch das Frontmatter selbst.
- Änderungen an `adopt --apply` selbst (ADR 0001 gilt unverändert).
- Kritische Werkzeuge ohne `--allow-critical`, etwa für andere MCP-Clients ohne Freigabekarte.
- Ein Protokoll der Bestätigungen außerhalb des Gesprächs.
- `sdd.config.yaml` als kritisch einstufen (eigenes ADR, siehe Risiko in `desk-write-content`).

## Akzeptanzkriterien

- **AC-1** `sdd mcp` bietet `sdd_baseline`, `sdd_adr_accept` und `sdd_adopt_apply` nur mit `--allow-critical` an. Der Schalter schließt `--allow-write` ein. Ohne ihn bleibt die Werkzeugliste unverändert, und ein Aufruf eines kritischen Werkzeugs scheitert. Die drei Werkzeuge tragen `destructiveHint: true`.
- **AC-2** `sdd_baseline` (`gate`, `prune`, `accept`, `apply`) schreibt dieselbe Baseline und dasselbe Lockfile wie `sdd baseline` mit denselben Optionen, über gemeinsamen Code. Die Ausgabe der CLI bleibt gleich. Die Vorschau nennt hinzukommende und entfallende Einträge, neu gelockte und quittierte ADRs sowie eine Zusammenfassung. Eine Regel mit `baseline: forbidden` wird abgelehnt.
- **AC-3** `sdd_adr_accept` (`path`, `apply`) setzt nur bei einem ADR mit `status: proposed` den Status auf `accepted`. Der übrige Inhalt bleibt byteweise gleich. Die Vorschau zeigt den Diff. Jede andere Datei und jeder andere Status werden abgelehnt.
- **AC-4** `sdd_adopt_apply` (`profile`, `branch`, `apply`) zeigt ohne `apply` den Bericht von `sdd adopt` und eine Zusammenfassung. Mit `apply` führt es `sdd adopt --apply` aus und liefert Branch und Commits.
- **AC-5** Im Desktop wartet ein kritischer Aufruf mit `apply` immer auf eine einzelne Bestätigung mit Vorschau, auch wenn das Werkzeug für die Sitzung freigegeben ist. Die Freigabekarte kennzeichnet ihn als kritisch, bietet keine Freigabe für die Sitzung und nennt die Aktion auf der Schaltfläche. Die Oberfläche nimmt ein kritisches Werkzeug nie in die Sitzungsfreigaben auf.
- **AC-6** Hat der Nutzer einen kritischen Aufruf abgelehnt, lehnt der Desktop jeden weiteren Aufruf desselben Werkzeugs mit `apply` ohne Rückfrage ab, bis eine neue Nachricht des Nutzers kommt. Das Modell erhält dafür einen Hinweis.
- **AC-7** Prompts und Server-Anweisungen sagen: Kritische Aktionen nur, wenn der Mensch sie ausdrücklich will. Jede bestätigt er einzeln, und nach einer Ablehnung wird nicht erneut gefragt.
