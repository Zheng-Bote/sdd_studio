# 4. Für Agenten

Dieser Teil richtet sich an LLM-Agenten (Claude Code, Claude Desktop, der KI-Desktop `sdd-desk`) und an Menschen, die sie einrichten. Er beschreibt, wie ein Agent die Verfassung liest, Präzedenz auflöst und Gates prüft — **identisch mit dem, was `sdd` tut**. Für Agenten zählen Regel-ID und Prüfvertrag; Titel und Begründungen sind Erläuterung.

---

## Einrichten {#agenten-einrichten}

`sdd mcp` ist ein MCP-Server über stdio: ohne Port, ohne Netzwerk, ohne LLM (ADR 0002, abgelöst durch ADR 0003). Der Client startet ihn im Projektverzeichnis.

```json
{
  "mcpServers": {
    "sdd": { "command": "sdd", "args": ["mcp"] }
  }
}
```

Mit `--allow-write` dürfen `sdd_new`, `sdd_write`, `sdd_init` und `sdd_archive` schreiben; ohne liefern sie nur die Vorschau. `--allow-critical` bietet zusätzlich die kritischen Werkzeuge an (siehe [Grenzen](#agenten-grenzen)) und schließt `--allow-write` ein.

- **`sdd_new`** legt neue Artefakte an, aus dem Template oder mit `content` (genau ein Typ) aus dem geprüften Entwurf. Bestehende Dateien überspringt es.
- **`sdd_write`** legt an oder ersetzt, aber nur die Verfassung, Markdown-Artefakte (`adr`, `prd`, `design`, `plans`, `archive`, `docs`) und Testdateien (`tests.include`, leer = Fallback-Liste). `.git/`, `.sdd/`, `exclude`-Treffer und symbolische Links nach außen sind ausgeschlossen. Eine bestehende Datei ersetzt es nur mit `base_sha256` aus `sdd_read`, atomar und mit gleichen Dateirechten.
- **Prüfung vor dem Schreiben:** Artefakte prüfen beide Werkzeuge vorher wie `sdd_validate_draft`; ein `error`-Befund, ein `archive-after` vor heute oder ein ADR mit `status: accepted` verhindern das Schreiben. Ein ADR entscheidet der Mensch ([ADR 0003](../docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md)). `sdd.config.yaml` ändert nur das kritische Werkzeug `sdd_config` ([ADR 0004](../docs/adr/0004-konfiguration-aendern-ist-kritisch.md)); der neue Inhalt muss sich laden lassen.
- **Ergebnis:** Die Vorschau zeigt Inhalt oder Unified Diff, das Ergebnis Status, SHA-256, Zeilenzahl und Diff dessen, was wirklich geschrieben wurde.

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
| zum Anlegen | `sdd_new` | Vorschau, dann Anlegen nach Bestätigung, mit `content` aus dem geprüften Entwurf (nur mit `--allow-write`) |
| vor touches und Tests | `sdd_stack` | Sprachen des Projekts aus Dateiendungen und Marker-Dateien, je Sprache Testdatei-Muster und Kommentarsyntax; liest keine Inhalte |
| zum Lesen | `sdd_read` | Artefakt, `sdd.config.yaml`, Testdatei oder Kontextdatei (`README.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.gemini/GEMINI.md` …, nur lesbar) mit Zeilennummern und SHA-256 |
| zum Projektstart | `sdd_init` | wie `sdd init [--git]`: Vorschau, dann Anlegen nach Bestätigung (nur mit `--allow-write`) |
| nach dem Release | `sdd_archive` | wie `sdd archive`: Feature ins Archiv verschieben, nur wenn kein PRD mehr `status: draft` trägt und jedes Kriterium getestet ist (nur mit `--allow-write`) |
| zum Ändern | `sdd_write` | Artefakt oder Testdatei ändern bzw. anlegen; Diff vorab (nur mit `--allow-write`) |
| Konfiguration ändern | `sdd_config` | kritisch: Diff und je Prüflauf wegfallende und hinzukommende Befunde vorab, einzeln bestätigt (nur mit `--allow-critical`) |
| nach `sdd-conflicts` | `sdd_conflicts_check` | Zitate gegen Datei und Zeile prüfen, fehlende Regeln der Verfassung nennen; bewertet nichts |
| nach der Änderung | `sdd_lint`, `sdd_gate` | Befunde, Gate bestanden? |
| bei einem Befund | `sdd_explain` | Regel, Begründung, Prüfvertrag |
| zum Abschluss | `sdd_trace` | Hat jedes Kriterium Tech-Spec-Stelle, Test und Task? |

Geführte Abläufe stehen als Prompts bereit: `sdd-idea` (Idee → PRD), `sdd-techspec`, `sdd-adr`, `sdd-plan`, `sdd-adopt` und `sdd-start` (Einstieg in ein bestehendes bzw. neues Projekt), `sdd-constitution` (Verfassung aus README und Agenten-Dateien erarbeiten, Regel für Regel mit Fundstelle), `sdd-tests` (je Kriterium ein Test in der Projektsprache), `sdd-conflicts`, `sdd-review`. Sie fragen, statt zu erfinden, enden mit einer Zusammenfassung in Worten (gespeicherte Dateien, Prüfergebnis, offene Punkte, nächster Schritt), nennen die geltenden Regeln aus dem aktuellen Katalog und die Reihenfolge des Prozesses: Idee → PRD → Tech Spec (wenn nötig) → Plan → Tests je Kriterium → Umsetzung → Gate → Merge → Release → Archiv. `sdd-idea` klärt mit `sdd_stack` die Technik, bevor es nach Pfaden (touches) fragt; bei einem neuen Projekt schlägt das Modell eine Sprache mit Begründung vor, der Mensch entscheidet.

Ressourcen: `sdd://constitution`, `sdd://catalog`, `sdd://handbook` und `sdd://artifact/<pfad>` für jedes Artefakt.

---

## Grenzen {#agenten-grenzen}

- **Kein Agent entscheidet über die Baseline**, quittiert Befunde (`sdd baseline --accept`), schreibt das ADR-Lockfile, führt `sdd adopt --apply` aus oder setzt ein ADR auf `accepted`. Der Mensch entscheidet (Konzept §9.6, [ADR 0003](../docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md)).
  - Die kritischen Werkzeuge `sdd_baseline`, `sdd_adr_accept` und `sdd_adopt_apply` gibt es nur mit `sdd mcp --allow-critical`, also nur für Clients, die jede Ausführung einzeln bestätigen lassen (wie `sdd-desk`).
  - Ein Agent ruft sie nur auf ausdrücklichen Wunsch des Menschen auf und fragt nach einer Ablehnung nicht erneut.
  - `sdd_baseline` arbeitet wie `sdd baseline`; Regeln mit `baseline: forbidden` bleiben gesperrt. `sdd_adr_accept` entscheidet nur ein ADR mit `status: proposed`.
- **Ein Agent formuliert, der Mensch nimmt ab.** Akzeptanzkriterien, Nicht-Ziele und Entscheidungen entstehen im Gespräch, nicht aus Platzhaltertext.
- **Testverweise nennen den Feature-Slug:** `// <slug> AC-<n>: …` direkt über dem Test (`SDD-022`, `SDD-025`).

---

## KI-Desktop `sdd-desk` {#sdd-desk}

`sdd-desk` ist ein **getrenntes, optionales Programm** ([ADR 0003](../docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md), löst ADR 0002 ab): Es führt im Gespräch mit einem frei wählbaren Sprachmodell durch den Prozess und nutzt dafür `sdd mcp`. `sdd` selbst bleibt KI-frei und offline.

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

**Modelle:** `config/llm.env` neben der ausführbaren Datei (Vorlage: `config/llm.env.example`), Rechte `0600`. Die Desktop-Anwendung nimmt stattdessen `llm.env` im Konfigurationsordner der Anwendung, falls vorhanden (Linux `~/.config/io.github.zhengbote.sdddesk/`, macOS `~/Library/Application Support/io.github.zhengbote.sdddesk/`, Windows `%APPDATA%\io.github.zhengbote.sdddesk\`) — bei installierten Bündeln ist das der richtige Ort, und fehlt die Datei, nennt die Warnung in der Anwendung genau diesen Pfad. Lokales Ollama, Ollama Cloud, OpenRouter, NVIDIA NIM, Claude und Gemini; Prozessvariablen haben Vorrang. Ein Gesprächszug endet nach 30 Minuten mit lokalem Ollama und nach 10 Minuten mit Cloud-Anbietern; `<ANBIETER>_TURN_TIMEOUT` (z. B. `OLLAMA_LOCAL_TURN_TIMEOUT=45m`, 1m bis 4h) ändert das je Anbieter. Mehrstufige Abläufe wie der Abgleich mit der Verfassung brauchen mit lokalen Modellen oft mehr als 10 Minuten. Ohne Datei steht nur lokales Ollama zur Verfügung.

**Modell testen:** „LLM testen“ neben der Modellauswahl schickt eine feste Testnachricht samt den Werkzeugbeschreibungen von `sdd mcp` an das gewählte Modell — keine Gesprächs- oder Projektinhalte, bei Cloud-Anbietern erst nach dem Hinweis. Das Ergebnis sagt, ob das Modell antwortet und ob es Werkzeuge annimmt; ein Modell ohne Werkzeuge ist im Gespräch nicht nutzbar. Ein unbekannter Modellname zeigt sich hier als Meldung des Anbieters, z. B. `HTTP 404`.

**Abgleich mit der Verfassung (`SDD-060`, assistiert):** Der geführte Ablauf `sdd-conflicts` lässt das Modell die Tech Spec eines Features gegen die Verfassung halten. Es liest beide und verlinkte ADRs mit `sdd_read` und meldet nur Widersprüche, die es auf beiden Seiten wörtlich mit Datei und Zeile zitiert — als Widerspruch, Spannung oder unklar, mit dem Vorschlag „Tech Spec anpassen“ oder „ADR für eine Verfassungsänderung“. Bevor es antwortet, prüft das lesende Werkzeug `sdd_conflicts_check` sein Ergebnis maschinell: Steht jedes Zitat wörtlich in der genannten Zeile (Leerraum und Hervorhebungen zählen nicht; in genau einer anderen Zeile gilt es mit korrigierter Zeile), stammt es aus Verfassung, Tech Spec des Features oder einem ADR, und fehlt eine nummerierte Regel der Verfassung in der Regelliste? Die Antwort zeigt nur belegte Meldungen; nicht belegte stehen getrennt als verworfen mit Grund. Das Werkzeug vergleicht nur Zeichenketten — ob ein Widerspruch inhaltlich besteht, prüft es nicht. Das Ergebnis ist kein Befund: Es steht in keinem Bericht und keiner Baseline, `SDD-060` ist keine Katalogregel. Ob ein Zitat stimmt und wie aufgelöst wird, entscheidet der Mensch.

**Im Gespräch:** Antworten erscheinen, während das Modell schreibt; liefert das Modell seinen Gedankengang gesondert (z. B. `qwen3` über Ollama), ist er während des Zuges aufklappbar zu sehen, wird aber nicht Teil des Gesprächs. **Abbrechen** beendet den Zug und die Anfrage an den Anbieter; fertige Nachrichten bleiben, ein Werkzeugaufruf ohne Ergebnis wird verworfen.

**Was der Desktop darf:**

- Werkzeuge ausschließlich über einen selbst gestarteten `sdd mcp` — genau dessen zwanzig Werkzeuge (mit `--allow-critical`), keine eigenen. `sdd_read` liest Artefakte, `sdd.config.yaml`, Testdateien und Kontextdateien; `sdd_write` schreibt nur Artefakte und Testdateien, keinen übrigen Code.
- **Nicht erfasstes Projekt:** Fehlt `sdd.config.yaml`, zeigt der Desktop beim Öffnen einen Hinweis: bei einem Ordner mit Dateien „Bestehendes Projekt übernehmen“ (Ablauf `sdd-adopt`: Bericht, Übernahme auf eigenem Branch, Verfassung vor dem Merge), bei einem leeren Ordner „Neues Projekt anlegen“ (Ablauf `sdd-start`: Init mit Git, Technik, Verfassung, erstes Feature). Die Schaltfläche legt den Ablauf nur ins Eingabefeld; jeder Werkzeugaufruf braucht wie immer Ihre Freigabe.
- **Kritische Aktionen** (Baseline, Lockfile, `adopt --apply`, ADR entscheiden, Konfiguration ändern) brauchen jedes Mal eine einzelne Bestätigung; eine Freigabe für die Sitzung gibt es dafür nicht. Die Karte ist rot abgesetzt, zeigt die ganze Wirkung (Einträge, gelockte ADRs, Diff oder Bericht) und nennt die Aktion auf der Schaltfläche. Nach einer Ablehnung lehnt der Desktop weitere Aufrufe desselben Werkzeugs ab, bis Sie wieder schreiben. `sdd-desk` startet dafür `sdd mcp --allow-write --allow-critical`.
- **Konfiguration ändern** (`sdd.config.yaml`) geht nur über `sdd_config` und ist kritisch ([ADR 0004](../docs/adr/0004-konfiguration-aendern-ist-kritisch.md)): Eine Zeile kann Regeln abschalten, die keine Baseline unterdrücken darf (`profile: minimal`), oder ganze Pfade aus der Prüfung nehmen (`exclude`). Die Karte zeigt den Diff und je Prüflauf (Lint und die drei Gates) die Befunde, die wegfallen oder hinzukommen; die Schaltfläche nennt die geänderten Schlüssel, einen Profilwechsel mit altem und neuem Wert und jede Änderung an `override`. Dass keine Befunde wegfallen, heißt nicht, dass nichts abgeschwächt wird: Ein `exclude` wirkt auch auf künftige Dateien. Das Modell darf eine Änderung vorschlagen, wenn ein Ablauf sie braucht (z. B. `tests.include` in `sdd-tests`). Andere MCP-Clients ohne `--allow-critical` können die Konfiguration nicht ändern; `sdd_write` lehnt sie ab.
- Jeder Aufruf braucht eine Freigabe: **einmal**, **für die Sitzung** (je Werkzeug) oder **Ablehnen**. Ein schreibender Aufruf (`sdd_new` oder `sdd_write` mit `apply`) zeigt vorher die Vorschau mit Inhalt bzw. Diff. Ist das Werkzeug auf einer solchen Karte für die Sitzung freigegeben, schreibt es ohne weitere Rückfrage; was es geschrieben hat (mit Diff), steht im Ergebnis im Gespräch ([ADR 0003](../docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md), Stufe „schreibend“). Die Freigabe für die Sitzung gilt je Werkzeug **und je Stufe**: Ruft das Modell ein schreibendes Werkzeug ohne `apply` auf, um zu sehen, was geschrieben würde, trägt die Karte „nur Vorschau“. Für die Sitzung freigegeben, laufen dann nur weitere Vorschauen ohne Rückfrage; Schreiben wird erneut gefragt, mit Inhalt bzw. Diff auf der Karte.
- Vor dem ersten Aufruf eines Cloud-Anbieters muss der Hinweis „Inhalte verlassen dieses Gerät“ bestätigt sein; ohne Bestätigung lehnt das Backend ab. Inhalte werden nicht geschwärzt.
- Lokaler Server nur auf `127.0.0.1`, `Host`-Prüfung, Sitzungstoken für jeden API-Aufruf, Content-Security-Policy ohne Inline-Skripte; keine Telemetrie.
- In der Desktop-Anwendung zeigt das Fenster nur die eigene Startseite und den laufenden Sidecar; andere Web-Adressen öffnet der Standardbrowser. Keine Seite erhält Tauri-Rechte, das Token erscheint weder im Protokoll noch auf der Fehlerseite, und stirbt die Anwendung, enden `sdd-desk` und `sdd mcp` mit ihr.
