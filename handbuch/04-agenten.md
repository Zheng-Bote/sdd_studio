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

---

## KI-Desktop `sdd-desk` {#sdd-desk}

`sdd-desk` ist ein **getrenntes, optionales Programm** ([ADR 0002](../docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)): Es führt im Gespräch mit einem frei wählbaren Sprachmodell durch den Prozess und nutzt dafür `sdd mcp`. `sdd` selbst bleibt KI-frei und offline.

**Als Desktop-Anwendung** (Tauri v2, `apps/desk/src-tauri`): Sie bringt `sdd-desk` und `sdd` als Sidecars mit — `sdd` unter dem Namen `sdd-studio-sdd`, damit ein Paket kein zweites `/usr/bin/sdd` installiert; eine eigene Installation von `sdd` bleibt unberührt — und öffnet das Startargument, sonst das zuletzt genutzte Projekt, sonst einen Ordnerdialog. Projektwechsel über **Datei → Projekt öffnen …** (Strg+O).

```text
cd apps/desk && npm ci
npm run tauri:dev        # Entwicklung: baut Oberfläche und Sidecars, startet die Anwendung
npm run tauri:build      # Bündel unter src-tauri/target/release/bundle/
```

**Als lokale Web-Oberfläche** im Browser:

```text
cd apps/desk && npm ci && npm run build:desk     # Oberfläche bauen und einbetten
go build -o bin/sdd ./cmd/sdd
go build -o bin/sdd-desk ./cmd/sdd-desk
bin/sdd-desk --project . --open                   # gibt URL mit Sitzungstoken aus
```

**Modelle:** `config/llm.env` neben der ausführbaren Datei (Vorlage: `config/llm.env.example`), Rechte `0600`. Die Desktop-Anwendung nimmt stattdessen `llm.env` im Konfigurationsordner der Anwendung, falls vorhanden (Linux `~/.config/io.github.zhengbote.sdddesk/`, macOS `~/Library/Application Support/io.github.zhengbote.sdddesk/`, Windows `%APPDATA%\io.github.zhengbote.sdddesk\`) — bei installierten Bündeln ist das der richtige Ort, und fehlt die Datei, nennt die Warnung in der Anwendung genau diesen Pfad. Lokales Ollama, Ollama Cloud, OpenRouter, NVIDIA NIM, Claude und Gemini; Prozessvariablen haben Vorrang. Ohne Datei steht nur lokales Ollama zur Verfügung.

**Modell testen:** „LLM testen“ neben der Modellauswahl schickt eine feste Testnachricht samt den Werkzeugbeschreibungen von `sdd mcp` an das gewählte Modell — keine Gesprächs- oder Projektinhalte, bei Cloud-Anbietern erst nach dem Hinweis. Das Ergebnis sagt, ob das Modell antwortet und ob es Werkzeuge annimmt; ein Modell ohne Werkzeuge ist im Gespräch nicht nutzbar. Ein unbekannter Modellname zeigt sich hier als Meldung des Anbieters, z. B. `HTTP 404`.

**Im Gespräch:** Antworten erscheinen, während das Modell schreibt; liefert das Modell seinen Gedankengang gesondert (z. B. `qwen3` über Ollama), ist er während des Zuges aufklappbar zu sehen, wird aber nicht Teil des Gesprächs. **Abbrechen** beendet den Zug und die Anfrage an den Anbieter; fertige Nachrichten bleiben, ein Werkzeugaufruf ohne Ergebnis wird verworfen.

**Was der Desktop darf:**

- Werkzeuge ausschließlich über einen selbst gestarteten `sdd mcp` — genau dessen zehn Werkzeuge, keine eigenen.
- Jeder Aufruf braucht eine Freigabe: **einmal**, **für die Sitzung** (nur lesende Werkzeuge) oder **Ablehnen**. Ein schreibender Aufruf (`sdd_new` mit `apply`) zeigt vorher die Vorschau mit Dateien und Inhalt und lässt sich nur einzeln freigeben.
- Vor dem ersten Aufruf eines Cloud-Anbieters muss der Hinweis „Inhalte verlassen dieses Gerät“ bestätigt sein; ohne Bestätigung lehnt das Backend ab. Inhalte werden nicht geschwärzt.
- Lokaler Server nur auf `127.0.0.1`, `Host`-Prüfung, Sitzungstoken für jeden API-Aufruf, Content-Security-Policy ohne Inline-Skripte; keine Telemetrie.
- In der Desktop-Anwendung zeigt das Fenster nur die eigene Startseite und den laufenden Sidecar; andere Web-Adressen öffnet der Standardbrowser. Keine Seite erhält Tauri-Rechte, das Token erscheint weder im Protokoll noch auf der Fehlerseite, und stirbt die Anwendung, enden `sdd-desk` und `sdd mcp` mit ihr.
