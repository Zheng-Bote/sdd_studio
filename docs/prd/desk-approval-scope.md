---
feature: desk-approval-scope
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - handbuch/**
---

# PRD — Sitzungsfreigabe gilt nur für die Stufe, die der Mensch gesehen hat

## Problem

Im Live-Test vom 2026-11-05 (Claude Sonnet 5.5, Szenario cert-expiry-watcher) rief das Modell `sdd_new` zuerst ohne `apply` auf, um das Template anzusehen. Die Karte dieser reinen Vorschau war als „nur lesend“ gekennzeichnet und bot „Für diese Sitzung erlauben“ an. Wer das wählte, gab damit auch jedes spätere `sdd_new` **mit** `apply` frei. ADR 0001 und PRD des Testprojekts sind danach ohne Freigabekarte gespeichert worden. Die endgültige Fassung des PRD hat der Mensch nie als Vorschau gesehen.

Ursache: Der Desktop führt eine einzige Liste der für die Sitzung freigegebenen Werkzeuge (`allowed_tools`). Sie gilt für jede Stufe außer „kritisch“, egal auf welcher Karte die Freigabe erteilt wurde. ADR 0003 verlangt für die Stufe „schreibend“ aber, dass der Desktop vorher Vorschau bzw. Diff zeigt. Eine Freigabe für die Sitzung ist dort erst nach einer solchen Karte vorgesehen.

## Ziel

Eine Freigabe für die Sitzung gilt nur für die Stufe der Karte, auf der sie erteilt wurde. Wer eine Vorschau freigibt, gibt kein Schreiben frei. Schreibvorgänge für die Sitzung freigeben kann man nur auf einer Karte, die zeigt, was geschrieben würde.

## Nutzer

- **Nutzer des KI-Desktops**, die lesende Aufrufe und Vorschauen nicht jedes Mal bestätigen wollen, Schreibvorgänge aber erst nach einer gesehenen Vorschau erlauben.

## Umfang

- Zwei getrennte Sitzungsfreigaben im Backend von `sdd-desk`: lesend (einschließlich Vorschauen) und schreibend.
- Freigabekarte und Gesprächszustand der Oberfläche.
- Handbuch.

## Nicht-Ziele

- Änderungen an der Stufe „kritisch“. Sie bleibt ohne Sitzungsfreigabe (desk-critical-tools AC-5).
- Änderungen an `sdd mcp` und seinen Werkzeugen.
- Eine Freigabe je Datei oder Pfad statt je Werkzeug.
- Freigaben über die Sitzung hinaus speichern.
- Einstufung von `sdd.config.yaml` als kritisch (eigenes ADR).

## Akzeptanzkriterien

- **AC-1** Ein Aufruf eines schreibenden Werkzeugs mit `apply` läuft ohne Karte nur, wenn genau dieser Aufruf freigegeben ist oder das Werkzeug in der Sitzungsfreigabe für Schreibvorgänge (`allowed_writes`) steht. Steht es nur in `allowed_tools`, wartet der Aufruf mit seiner Vorschau.
- **AC-2** `allowed_tools` gilt für lesende Aufrufe, auch für Vorschauen (schreibende und kritische Werkzeuge ohne `apply`). Ein Werkzeug in `allowed_writes` darf auch seine Vorschauen ohne Karte ausführen.
- **AC-3** Die Oberfläche nimmt ein Werkzeug bei „Für diese Sitzung erlauben“ in `allowed_writes` auf, wenn die Karte einen Schreibvorgang zeigt, sonst in `allowed_tools`. Ein kritisches Werkzeug kommt in keine der beiden Listen.
- **AC-4** Die Karte einer Vorschau eines schreibenden oder kritischen Werkzeugs ist als „nur Vorschau“ gekennzeichnet. Sie sagt, dass eine Freigabe für die Sitzung nur Vorschauen erlaubt und Schreiben erneut gefragt wird.
- **AC-5** Das Handbuch beschreibt die beiden Sitzungsfreigaben.
