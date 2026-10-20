# 4. Für Agenten

Dieser Teil richtet sich an LLM-Agenten (Claude Code, Claude Desktop, der KI-Desktop `sdd-desk`) und an Menschen, die sie einrichten. Er beschreibt, wie ein Agent die Verfassung liest, Präzedenz auflöst und Gates prüft — **identisch mit dem, was `sdd` tut**. Für Agenten zählen Regel-ID und Prüfvertrag; Titel und Begründungen sind Erläuterung.

---

## Einrichten {#agenten-einrichten}

`sdd mcp` ist ein MCP-Server über stdio: ohne Port, ohne Netzwerk, ohne LLM (ADR 0002). Der Client startet ihn im Projektverzeichnis.

```json
{
  "mcpServers": {
    "sdd": { "command": "sdd", "args": ["mcp"] }
  }
}
```

Mit `--allow-write` darf das Werkzeug `sdd_new` neue Artefakte anlegen; ohne liefert es nur die Vorschau. Bestehende Dateien ändert `sdd` nie — das tut der Agent mit seinen eigenen Werkzeugen und prüft danach.

---

## Präzedenz auflösen {#agenten-praezedenz}

Bei einem Widerspruch gilt (sdd_process_3 §3.6):

1. **Verfassung** — systemweite Regeln, gewinnt gegen alles.
2. **ADR** — begründet eine Regel der Verfassung, ersetzt sie aber nicht.
3. **Tech Spec** — darf die Verfassung präzisieren, nie lockern.
4. **PRD** — fachliche Anforderungen; ein Konflikt mit der Verfassung ist ein Blocker.
5. **Plan und Tasks** — Umsetzung; darf keine der obigen Regeln brechen.

Bei einem Konflikt zwischen gleich bindenden Regeln: **nicht entscheiden, sondern eskalieren.** `sdd_context <pfad>` liefert die für eine Datei geltenden Artefakte bereits in dieser Reihenfolge: die Verfassung, ADRs, deren `references` den Pfad betreffen, und die aktiven Feature-Artefakte, deren `touches` ihn überschneiden.

---

## Ablauf einer Änderung {#agenten-ablauf}

| Wann | Werkzeug | Zweck |
|---|---|---|
| vor der Änderung | `sdd_check_intent` | Greift die Größenschwelle, ist die Verfassung betroffen, welche Artefakte sind nötig? |
| vor der Änderung | `sdd_context` | Was gilt für die Dateien, in welcher Rangfolge? |
| jederzeit | `sdd_next` | Wo steht das Feature im Prozess, was ist der nächste Schritt? |
| beim Schreiben eines Artefakts | `sdd_validate_draft` | Regeln auf den Entwurf anwenden, bevor er gespeichert wird |
| zum Anlegen | `sdd_new` | Vorschau, dann Anlegen nach Bestätigung (nur mit `--allow-write`) |
| nach der Änderung | `sdd_lint`, `sdd_gate` | Befunde, Gate bestanden? |
| bei einem Befund | `sdd_explain` | Regel, Begründung, Prüfvertrag |
| zum Abschluss | `sdd_trace` | Hat jedes Kriterium Tech-Spec-Stelle, Test und Task? |

Geführte Abläufe stehen als Prompts bereit: `sdd-idea` (Idee → PRD), `sdd-techspec`, `sdd-adr`, `sdd-plan`, `sdd-review`. Sie fragen, statt zu erfinden, und nennen die geltenden Regeln aus dem aktuellen Katalog.

Ressourcen: `sdd://constitution`, `sdd://catalog`, `sdd://handbook` und `sdd://artifact/<pfad>` für jedes Artefakt.

---

## Grenzen {#agenten-grenzen}

- **Kein Agent verändert die Baseline**, quittiert Befunde (`sdd baseline --accept`), schreibt das ADR-Lockfile, führt `sdd adopt --apply` aus oder setzt ein ADR auf `accepted`. Dafür gibt es kein Werkzeug; der Mensch entscheidet (Konzept §9.6).
- **Ein Agent formuliert, der Mensch nimmt ab.** Akzeptanzkriterien, Nicht-Ziele und Entscheidungen entstehen im Gespräch, nicht aus Platzhaltertext.
- **Testverweise nennen den Feature-Slug:** `// <slug> AC-<n>: …` direkt über dem Test (`SDD-022`, `SDD-025`).
