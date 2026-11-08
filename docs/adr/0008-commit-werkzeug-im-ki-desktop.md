---
status: accepted
date: 2026-11-08
supersedes: []
references:
  - constitution.md
  - docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md
  - docs/sdd_konzept.md
---

# ADR 0008 — Commit-Werkzeug `sdd_commit` im KI-Desktop

## Kontext

Der KI-Desktop soll den ganzen Prozess tragen (ADR 0003). Im Live-Test vom 2026-11-05 endete aber jeder Ablauf mit dem Hinweis, dass der Mensch die gespeicherten Artefakte selbst committen muss. Das zwingt zum Wechsel ins Terminal und lässt Ergebnisse uncommittet liegen.

Die Verfassung, Security 4, erlaubt schreibenden Git-Zugriff bisher nur in zwei Ausnahmen: `sdd adopt --apply` und `sdd init --git`. Ein Commit-Werkzeug ist eine dritte.

Erwogen wurden:

1. **Beim Terminal bleiben.** Verworfen: Die Lücke im Ablauf bleibt.
2. **Stufe „schreibend“, alle Änderungen committen (`git add -A`).** Verworfen: Eine Freigabe für die Sitzung ließe das Modell ungesehen committen, und `-A` nähme fremden Code und versehentlich Geheimnisse mit.
3. **Kritisches Werkzeug mit ausdrücklichen Pfaden** — vorgeschlagen.

## Entscheidung

1. `sdd mcp` erhält das Werkzeug `sdd_commit` (`paths`, `message`, `apply`), nur mit `--allow-critical`. Es ist **kritisch** im Sinne von ADR 0003: jede Ausführung einzeln bestätigt, nie für die Sitzung.
2. **Nur ausdrücklich genannte Pfade**, kein Platzhalter und kein `git add -A`. Erlaubt sind Dateien, die `sdd_write` schreiben darf, `sdd.config.yaml`, `.sdd/baseline.json` und `.sdd/adr-lock.json`. Alles andere, etwa Quellcode, `.git/`, `config/llm.env` oder ein Pfad außerhalb des Repositorys, wird abgelehnt. Committet wird nur der Inhalt dieser Pfade; anderes, bereits Vorgemerktes bleibt draußen.
3. **Nicht auf dem Standard-Branch.** Auf `main`, `master` oder dem Branch, auf den `base_ref` zeigt, sowie ohne Branch (detached HEAD) lehnt das Werkzeug ab. Features entstehen auf Branches, und der Merge braucht das Gate. Das Werkzeug legt keine Branches an.
4. **Vorschau:** Dateien, Diff und Commit-Nachricht auf der Bestätigungskarte; die Schaltfläche nennt die Zahl der Dateien und den Branch.
5. Git-Hooks werden nicht übersprungen. Weiterhin kein `push`, `merge`, `rebase` oder `reset`.

## Konsequenzen

- **Verfassung**, Security 4: dritte Ausnahme für `sdd_commit`, mit Verweis auf dieses ADR.
- **Konzept §9.6:** `sdd_commit` in der Stufe „kritisch“.
- **Neues Feature** mit PRD und Tech Spec: Werkzeug, Einstufung im Desktop, Prompts, Handbuch. Eine CLI-Variante ist nicht vorgesehen; im Terminal gibt es `git`.
- **Einschränkung:** In einem frisch angelegten Projekt steht man auf `main`. Vor dem ersten `sdd_commit` muss der Mensch außerhalb des Desktops einen Branch anlegen; das Werkzeug nennt das in seiner Ablehnung.
- **Risiko:** Größere Angriffsfläche für schreibenden Git-Zugriff. Gegenmaßnahmen sind die Pfadliste, die Sperre des Standard-Branches und die einzelne Bestätigung.
