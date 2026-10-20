---
status: accepted
date: 2026-10-20
supersedes: []
references:
  - constitution.md
  - docs/sdd_konzept.md
---

# ADR 0002 — Optionaler KI-Desktop getrennt vom KI-freien sdd

## Kontext

`sdd` prüft Spec-Driven Development mechanisch: deterministisch, offline, ohne Telemetrie (Verfassung, Security 1). Der Prozess selbst — von der Produkt-Idee über PRD, Tech Spec, ADR und Plan bis zu Tests, Gate und Archiv — ist für Einsteiger schwer zu überblicken. Eine KI, die durch diese Strecke führt, Fragen stellt und Entwürfe formuliert, senkt diese Hürde erheblich.

Das Konzept (§9.6) sah KI bisher nur lokal (Ollama) und nur als Klassifikationshilfe für `adopt` vor. Eine Führung durch den Prozess braucht wählbare, auch cloudbasierte Modelle (wie in `md_indexer`: Ollama lokal und Cloud, OpenRouter, NVIDIA NIM, Claude, Gemini). Cloud-Aufrufe widersprechen dem Offline-Grundsatz, wenn sie in `sdd` selbst stattfinden.

Erwogen wurden:

1. **KI in `sdd`** — eine Ansicht „Assistent“ in `sdd ui` mit Anbieterwahl. Verworfen: Der Prüfkern bekäme Netzwerkzugriff und Schlüsselverwaltung; die Zusage „`sdd` ist offline“ wäre nur noch bedingt wahr.
2. **Nur externe Agenten über MCP** — `sdd` bleibt KI-frei, die Führung übernimmt ein fremder Client. Verworfen als alleinige Lösung: kein eigener, auf den Prozess zugeschnittener Ablauf, keine eigene Modellwahl.
3. **Getrennter, optionaler KI-Desktop** — gewählt.

## Entscheidung

1. **`sdd` bleibt KI-frei und offline.** Kern (`internal/sddcore`), CLI, `sdd ui` und der MCP-Server `sdd mcp` bauen keine ausgehenden Verbindungen auf und enthalten keinen LLM-Code. Ein Test stellt sicher, dass diese Teile keines der KI-Pakete importieren.
2. **Der KI-Desktop ist ein eigenes, optionales Programm** (Arbeitsname `sdd-desk`, eigenes `cmd/` und eigene Pakete unter `internal/desk/`). Er ist nicht Teil von `sdd`, wird getrennt gebaut und ist für die Nutzung von `sdd` nicht nötig.
3. **Modellwahl wie in `md_indexer`:** lokales Ollama sowie die Cloud-Anbieter Ollama Cloud, OpenRouter, NVIDIA NIM, Claude und Gemini, konfiguriert über `<ANBIETER>_API_KEY`, `<ANBIETER>_ENDPOINT` und `<ANBIETER>_LLM_01…` in **`config/llm.env` im Programmverzeichnis des Desktops** (neben der ausführbaren Datei `sdd-desk`), unabhängig vom geöffneten Projekt. Ein Anbieter ohne Schlüssel ist abgeschaltet; Prozessvariablen haben Vorrang. Die Anbieter werden über ihre HTTP-APIs angesprochen, ohne Hersteller-SDK.
4. **Werkzeuge nur mit Erlaubnis:** Der Desktop darf `sdd`-Werkzeuge für den Nutzer ausführen — über `sdd mcp` (oder die CLI mit `--json`). Jede Ausführung ist erlaubnispflichtig: einzeln bestätigen oder für die Sitzung je Werkzeug freigeben. Schreibende Werkzeuge zeigen vorher ihre Vorschau.
5. **Nie für die KI erreichbar:** Baseline, `sdd baseline --accept`, ADR-Lockfile, `sdd adopt --apply` und das Setzen eines ADR auf `accepted`. Dafür gibt es weder ein MCP-Werkzeug noch einen Befehlsweg im Desktop (Konzept §9.6: Eine KI darf keine Baseline-Einträge erzeugen, entfernen oder begründen).
6. **Keine Schwärzung:** Inhalte gehen unverändert an den gewählten Anbieter. Der Desktop zeigt vor dem ersten Aufruf einer Sitzung an, welcher Anbieter gewählt ist und dass Inhalte das Gerät verlassen.
7. **Schutz der Schlüssel:** `config/llm.env` liegt außerhalb jedes Projekts und wird nie eingecheckt. Der Desktop verweigert die Nutzung, wenn die Datei für Gruppe oder andere lesbar ist (`chmod 600`). Im Repository von `sdd` steht `config/llm.env` in `.gitignore` (für Entwicklungs-Builds im Repository); eingecheckt wird nur eine Vorlage `config/llm.env.example` ohne Schlüssel.
8. **Keine Telemetrie**, auch nicht im Desktop.

## Konsequenzen

- Die Verfassung erhält in Security 1 die Abgrenzung: offline gilt für `sdd`; ausgehende Verbindungen gibt es nur im optionalen KI-Desktop und nur zum vom Nutzer gewählten Anbieter.
- Konzept §9.6 wird angepasst: KI-Assistenz ist ein getrenntes, optionales Programm, keine Funktion von `sdd`.
- Reihenfolge der Umsetzung: `sdd next` (nächster Prozessschritt, KI-frei) → `sdd mcp` (Werkzeuge, KI-frei, eigenes schmales JSON-RPC-Protokoll über stdio) → `sdd-desk` (Modellwahl, Gesprächsführung, erlaubnispflichtige Werkzeugausführung).
- Ohne Schwärzung trägt der Nutzer die Verantwortung dafür, welche Inhalte er welchem Anbieter gibt; mit lokalem Ollama verlassen Inhalte das Gerät nicht.
- Schlüssel und Modellwahl gelten für alle Projekte, die der Desktop öffnet; sie liegen beim Programm, nicht bei den Daten.
