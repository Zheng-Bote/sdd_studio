---
status: accepted
date: 2026-11-05
supersedes: []
references:
  - docs/sdd_konzept.md
  - NOTICE
---

# ADR 0006 — arc42-Template als eigenständige Datei unter CC BY-SA 4.0 einbetten

## Kontext

Projekte, die mit `sdd` arbeiten, sollen bei jedem fertigen Stand eine Architekturdokumentation `docs/arc42.md` nach arc42 erhalten bzw. nachführen. Für sdd_studio selbst gibt es sie seit 2026-11-05. Dafür muss `sdd` die Gliederung von arc42 mitbringen.

Lizenzen:

- **sdd_studio** steht unter der Apache License 2.0 mit `NOTICE`.
- **Das arc42-Template** (Version 9.1, © Dr. Peter Hruschka, Dr. Gernot Starke und Mitwirkende, <https://arc42.org>) und seine Dokumentation stehen unter der Creative Commons Attribution-ShareAlike 4.0 International License (CC BY-SA 4.0). Verlangt sind Namensnennung, ein Hinweis auf die Lizenz, die Kennzeichnung von Änderungen, und Bearbeitungen stehen wieder unter CC BY-SA 4.0.

Eine deutsche Fassung der Gliederung (übersetzte Überschriften und Kurzhinweise) ist eine Bearbeitung des Templates.

Erwogen wurden:

1. **Nicht einbetten, nur verweisen.** `sdd` legt eine leere Datei mit einem Link auf arc42.org an. Verworfen: Dann hat der Mensch beim Anlegen keine Gliederung, und `sdd` kann nicht prüfen, ob die Abschnitte da sind.
2. **Eigene Gliederung schreiben, die arc42 nur ähnelt.** Verworfen: Das wäre eine Bearbeitung ohne Namensnennung oder eine Nachahmung, die den Vorteil des bekannten Standards verliert.
3. **Template als eigenständige Datei unter CC BY-SA 4.0 einbetten** — vorgeschlagen. Der Go-Code bleibt Apache 2.0. Das Template ist ein getrennter Bestandteil unter eigener Lizenz, wie eine mitgelieferte Schrift oder ein Icon-Satz.
4. **Pfad zu einem Template des Nutzers konfigurieren.** Als Ergänzung möglich, nicht als Ersatz: Ohne mitgeliefertes Template funktioniert `sdd new` nicht ohne Vorarbeit.

## Entscheidung

1. Die deutsche Fassung der arc42-Gliederung liegt als eigene Datei `templates/arc42.md.tmpl` im Repository und wird mit den übrigen Templates eingebettet. Sie steht unter **CC BY-SA 4.0**. Ihr Kopf nennt Urheber, Quelle, Version, Lizenz mit Link und die Änderung („deutsche Übersetzung der Gliederung, Hilfetexte gekürzt“).
2. **`NOTICE`** erhält einen Abschnitt: „`templates/arc42.md.tmpl` — arc42-Template 9.1, © Hruschka, Starke und Mitwirkende, arc42.org, CC BY-SA 4.0, bearbeitet (deutsch)“. `LICENSE` bleibt Apache 2.0 für alles Übrige.
3. **Jede erzeugte `docs/arc42.md`** trägt im Kopf eine Zeile mit Namensnennung und Lizenzhinweis zur Gliederung, wie die Datei von sdd_studio. Der vom Projekt geschriebene Inhalt gehört dem Projekt. Ob die gefüllte Datei als Bearbeitung des Templates gilt, entscheidet dieses ADR nicht. Die Namensnennung erfüllt die Pflicht in jedem Fall, und die ShareAlike-Frage liegt beim Projekt.
4. **Kein weiterer fremder Inhalt** unter einer anderen Lizenz als Apache 2.0 ohne eigenes ADR.

## Konsequenzen

- **Feature „arc42 für Projekte“** mit PRD und Tech Spec: Template, `sdd new --type arc42`, Angebot bei `sdd init` und `sdd adopt`, ein Schritt in `sdd next` zum Release, ein geführter Ablauf im KI-Desktop, der die Doku aus Verfassung, ADRs und Tech Specs füllt.
- **sdd_studio** steht weiterhin unter Apache 2.0. Wer `sdd` weitergibt, gibt das Template mit seinem Lizenzhinweis weiter. `NOTICE` deckt das ab.
- **Risiko:** Dieses ADR ist keine Rechtsberatung. Die Einordnung als getrennter Bestandteil entspricht der üblichen Praxis bei CC-lizenzierten Inhalten in Software. Bei Zweifeln am Weg 3 bleibt Weg 4 (Template des Nutzers) ohne eingebettete Datei.
