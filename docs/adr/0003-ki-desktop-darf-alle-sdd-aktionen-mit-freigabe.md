---
status: accepted
date: 2026-11-02
supersedes:
  - docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md
references:
  - constitution.md
  - docs/sdd_konzept.md
  - docs/adr/0001-schreibender-git-zugriff-fuer-sdd-adopt.md
---

# ADR 0003 — KI-Desktop darf alle sdd-Aktionen mit Freigabe

## Kontext

ADR 0002 hat den KI-Desktop `sdd-desk` als getrenntes, optionales Programm eingeführt. Er darf `sdd`-Werkzeuge nur mit Erlaubnis ausführen. Einige Aktionen sind für ihn ganz gesperrt (Entscheidung 5): Baseline, `sdd baseline --accept`, ADR-Lockfile, `sdd adopt --apply` und das Setzen eines ADR auf `accepted`.

Ein Praxistest hat gezeigt, dass der Desktop so nicht zu Ergebnissen führt. Der Dialog liefert ein gutes PRD, gespeichert wird aber nur das leere Template, denn `sdd_new` kennt keinen Inhalt, und andere Werkzeuge zum Schreiben hat der Desktop nicht. Projekt-Init, Konfiguration (`tests.include`) und Archivierung muss der Nutzer außerhalb des Desktops erledigen. Das Modell weicht auf ungeeignete Werkzeuge aus (`sdd_new` mit Typ `doc` als Ersatz für Konfiguration, Test und Archiv) und meldet trotzdem Erfolg. Wer den Desktop für den ganzen Prozess nutzen will, muss ständig zwischen Desktop und Terminal wechseln.

Der Nutzer hat entschieden, dass alles über den KI-Desktop möglich sein soll, auch das, was Entscheidung 5 sperrt.

Dagegen steht Konzept §9.6: „Eine KI darf niemals Baseline-Einträge erzeugen, entfernen oder begründen. Sonst wäre die einzige Bremse im System maschinell verhandelbar.“ Die Sorge dahinter ist berechtigt. Ein Modell, das Befunde selbst wegschalten kann, macht das Ratchet wertlos. Die Frage ist also nicht, _ob_ der Desktop diese Aktionen ausführt, sondern _wer entscheidet_.

Erwogen wurden:

1. **Sperre beibehalten** (ADR 0002). Verworfen: Der Nutzer muss für jede dieser Aktionen das Terminal öffnen. Das Ziel „ganzer Prozess im Desktop“ ist so nicht erreichbar.
2. **Alles erlauben, mit derselben Freigabe wie heute.** Verworfen: Eine Freigabe für die ganze Sitzung würde das Modell die Baseline selbst erweitern lassen, und genau davor warnt §9.6.
3. **Alles erlauben, gestuft nach Tragweite** — gewählt. Das Modell darf jede Aktion _vorschlagen_. Ausgeführt wird eine kritische Aktion aber nur durch die einzelne, bewusste Bestätigung des Menschen. Entscheidend bleibt der Mensch, der Desktop ist nur der Weg.

## Entscheidung

ADR 0003 löst ADR 0002 ab. Die Entscheidungen 1–3 und 6–8 aus ADR 0002 werden hier unverändert übernommen, die Entscheidungen 4 und 5 ersetzt.

1. **`sdd` bleibt KI-frei und offline** (ADR 0002, Entscheidung 1). Kern, CLI, `sdd ui` und `sdd mcp` bauen keine ausgehenden Verbindungen auf und enthalten keinen LLM-Code. Ein Test sichert das ab.
2. **Der KI-Desktop ist ein eigenes, optionales Programm** (ADR 0002, Entscheidung 2): `sdd-desk`, mit eigenem `cmd/` und eigenen Paketen unter `internal/desk/`.
3. **Modellwahl** wie in ADR 0002, Entscheidung 3: lokales Ollama und Cloud-Anbieter, konfiguriert in `config/llm.env` im Programmverzeichnis des Desktops.
4. **Der Desktop darf alles, was `sdd` kann.** Dazu gehören Artefakte mit Inhalt anlegen und ändern, Testdateien anlegen, die Projektkonfiguration lesen und ändern, ein Projekt initialisieren, Features archivieren, die Baseline pflegen (`--accept`, `--prune`), das ADR-Lockfile aktualisieren, `sdd adopt --apply` und ein ADR auf `accepted` setzen. Jede Aktion läuft über ein eigenes `sdd`-Werkzeug (`sdd mcp`) mit festem Zweck. Eine allgemeine Shell oder ein beliebiger Befehl ist nicht vorgesehen.
5. **Freigabe in drei Stufen.** Die Stufe gehört zum Werkzeug und ist in `sdd mcp` festgelegt, nicht im Modell:
   - **Lesend** (z. B. `sdd_next`, `sdd_lint`, `sdd_read`): einzeln erlauben oder für die Sitzung je Werkzeug.
   - **Schreibend** (Artefakte, Tests, Konfiguration, Init, Archiv): Vorher zeigt der Desktop die Vorschau bzw. einen Diff. Erlauben lässt sich das einzeln oder für die Sitzung je Werkzeug. Auch bei einer Freigabe für die Sitzung zeigt der Desktop nach jeder Ausführung im Gespräch, was tatsächlich geschrieben wurde: Pfad und Diff.
   - **Kritisch** (Baseline, ADR-Lockfile, `adopt --apply`, ADR auf `accepted`): Nur einzeln. Eine Freigabe für die Sitzung gibt es nicht. Die Bestätigung zeigt die vollständige Wirkung, also die betroffenen Befunde, Dateien, ADRs und Hashes. Sie ist sichtbar abgesetzt und trägt die Aktion im Text der Schaltfläche, z. B. „3 Befunde SDD-022 in die Baseline aufnehmen“. Eine abgelehnte kritische Aktion darf das Modell im selben Zug nicht erneut anfordern.
6. **Kritische Werkzeuge in `sdd mcp` nur auf ausdrücklichen Wunsch.** `sdd mcp` bietet sie nur mit dem neuen Schalter `--allow-critical` an (der `--allow-write` einschließt). `sdd-desk` startet `sdd mcp` damit. Andere MCP-Clients bekommen diese Werkzeuge nur, wenn der Nutzer sie ausdrücklich so startet. Die Werkzeuge tragen `destructiveHint: true`.
7. **Mechanische Grenzen bleiben bestehen**, egal wer die Aktion auslöst:
   - Regeln mit `baseline: forbidden` lassen sich nicht unterdrücken (Verfassung, Strikte Regel 4).
   - Dateioperationen nur innerhalb des Repositorys (Security 3), nie in `.git/` und nie in `config/llm.env`.
   - Bestehende Dateien überschreibt nur ein Werkzeug, das ausdrücklich ändert, und nur nach Diff.
   - Für `adopt --apply` gilt ADR 0001 unverändert.
8. **Git beim Projekt-Init:** `sdd init` erhält eine Option, die in einem Ordner ohne Git-Repository `git init` ausführt und einen ersten Commit mit den angelegten Dateien erstellt. Das ist die zweite Ausnahme vom lesenden Git-Zugriff (Security 4), neben `adopt --apply`. Sie gilt nur, wenn noch kein Repository existiert. Es gibt weiterhin kein `push`, `merge`, `rebase` oder `reset`. Archivieren verschiebt Dateien im Arbeitsbaum und committet nicht.
9. **Keine Schwärzung** (ADR 0002, Entscheidung 6). Inhalte gehen unverändert an den gewählten Anbieter, und der Desktop weist vor dem ersten Aufruf darauf hin.
10. **Schutz der Schlüssel** (ADR 0002, Entscheidung 7): `config/llm.env` liegt außerhalb jedes Projekts, mit `chmod 600`, und wird nie eingecheckt.
11. **Keine Telemetrie** (ADR 0002, Entscheidung 8), auch nicht im Desktop.

## Konsequenzen

- **Verfassung**, Security 1: Der Verweis wechselt von ADR 0002 auf ADR 0003.
- **Verfassung**, Security 4: Die Ausnahme für `sdd init` mit Git (Entscheidung 8) kommt neben `adopt --apply` hinzu.
- **Konzept §9.6:**
  - Der Satz „Eine KI darf niemals Baseline-Einträge erzeugen, entfernen oder begründen“ wird zu: „Eine KI entscheidet nie über Baseline-Einträge. Sie kann sie vorschlagen; ausgeführt werden sie nur durch die einzelne Bestätigung des Menschen (ADR 0003).“
  - „Werkzeuge nur mit Erlaubnis“ beschreibt künftig die drei Stufen.
- **ADR 0002** erhält `status: superseded`. Der Text bleibt unverändert (SDD-011).
- **Neue Werkzeuge** in `sdd mcp`, jeweils mit PRD und Tech Spec:
  - `sdd_new` mit Inhalt; zuerst `sdd_validate_draft`, bei Fehlern wird nichts geschrieben
  - ändern bestehender Artefakte und Testdateien
  - Konfiguration lesen und ändern
  - Init (optional mit Git)
  - Archivieren
  - kritische Werkzeuge für Baseline, Lockfile, `adopt --apply` und ADR-Entscheid
  - `sdd archive` als neuer CLI-Befehl, damit die Aktion auch ohne Desktop existiert
- **Desktop:** Die Freigabekarte kennt drei Stufen. Für kritische Aktionen gibt es eine eigene, abgesetzte Darstellung ohne „Für diese Sitzung erlauben“.
- **Risiko:** Ein Nutzer, der kritische Aktionen ungeprüft bestätigt, kann die Baseline über den Desktop ebenso aufweichen wie über `sdd baseline --accept` im Terminal. Das ADR verschiebt die Verantwortung nicht. Es macht sie im Desktop so sichtbar wie im Terminal und verhindert, dass das Modell ohne Bestätigung handelt.
- **Risiko:** Mehr schreibende Werkzeuge erhöhen die Angriffsfläche von `sdd mcp`. Gegenmaßnahmen sind die festen Werkzeugzwecke, die Pfadprüfung, der Diff vor dem Ändern und `--allow-critical` als ausdrückliche Freischaltung.
