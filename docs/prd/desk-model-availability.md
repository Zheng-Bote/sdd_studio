---
feature: desk-model-availability
status: active
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - README.md
  - CHANGELOG.md
  - docs/arc42.md
---

# PRD — Modelle im Desktop, die ehrlich verfügbar sind (`desk-model-availability`)

## Problem

Der KI-Desktop bietet lokale Ollama-Modelle an, ohne zu sagen, was sie können und was mit ihnen belegt ist. Drei Folgen, alle im Betrieb beobachtet:

- **Die Liste ist eingefroren.** Sie entsteht einmal beim Laden von `/api/meta`. Ein Modell, das gerade mit `ollama pull` geholt wurde, fehlt bis zum Neustart des Desktops.
- **Ungeeignete Modelle sehen aus wie geeignete.** Einbettungsmodelle wie `nomic-embed-text` stehen in der Auswahl, lassen sich wählen und fallen erst im Modelltest durch, weil sie keine Werkzeuge aufrufen. Im Gespräch sind sie unbrauchbar — der Werkzeugkreislauf ist der Kern des Desktops.
- **Ein Ablauf verspricht mehr, als belegt ist.** `sdd-conflicts` wird mit jedem Modell angeboten. Im Gegentest erfand `qwen3:14b` Pfade, Zeilen und Zitate und fand die eingebauten Widersprüche nicht; `qwen3:32b` rief die Prüfung auf, ohne Verfassung und Tech Spec gelesen zu haben, und fand nach rund 47 Minuten nichts. Belegt ist allein Claude Sonnet 5.5. Der Desktop sagt davon nichts.

Der letzte Punkt ist der teuerste: Wer mit einem lokalen Modell 47 Minuten auf ein Ergebnis wartet, das nichts belegt, hat die Zeit nicht dem Werkzeug zu verdanken, sondern einer Auskunft, die gefehlt hat.

## Ziel

Jedes im Desktop angebotene Modell ist entweder benutzbar oder als ungeeignet gekennzeichnet, und jede Aussage über ein Modell ist durch eine Fundstelle gedeckt. Nach `ollama pull` genügt ein Knopf statt eines Neustarts.

Der Desktop bleibt dabei, was er ist: eine Hülle um `sdd mcp`, die nichts ohne Freigabe ändert und nichts behauptet, was nicht belegt ist.

## Nutzer

- **Entwickler mit lokalem Modell** — wollen offline und ohne Datenabfluss arbeiten und müssen vor dem ersten Zug wissen, ob das gewählte Modell Werkzeuge aufrufen kann.
- **Nutzer von Cloud-Anbietern** — sollen sehen, welcher Ablauf mit ihrem Modell belegt ist und welcher nicht, statt es im Ergebnis zu merken.
- **Betreiber des Desktops** — wollen ein gerade geholtes Modell ohne Neustart ausprobieren.

## Umfang

- Auffrischen der Anbieter- und Modellliste im Kopfbereich, ohne Neustart.
- Fähigkeiten je lokalem Modell (Werkzeugaufruf) über `POST /api/show`, sichtbar in der Auswahl.
- Vorauswahl, die ein Modell mit Werkzeugaufruf bevorzugt.
- Belegstand je Ablauf für die Abläufe, deren Ergebnis ein Urteil des Modells ist (`sdd-conflicts`, `sdd-review`).
- Nur lesend: keine neuen schreibenden Werkzeuge, keine Änderung an den Freigabestufen.

## Nicht-Ziele

- **Modelle holen oder löschen.** `ollama pull` und `ollama rm` bleiben beim Menschen; der Desktop liest nur.
- **Neue Anbieter oder Protokolle.** Lokale Agenten-CLIs (`claude`, `agy`, `codewhale`) sind Teil B der Backlog-Notiz und brauchen zuerst ein ADR zur Freigabehoheit.
- **Ein `think`-Schalter für Reasoning-Modelle.** `ollamaChat` sendet heute kein `think`; die Stream-Pipeline kann `type: "thinking"` bereits lesen. Das ist ein eigener Schritt.
- **Eine Empfehlung über den belegten Stand hinaus.** Was nicht getestet wurde, wird als „nicht belegt" benannt, nicht geschätzt — auch nicht für große Cloud-Modelle.
- **Änderungen an `sdd` selbst.** Das Werkzeug bleibt KI-frei und offline (ADR 0003); alles hier betrifft `internal/desk` und `apps/desk`.

## Akzeptanzkriterien

- **AC-1** Der Kopfbereich hat einen Knopf „Modelle aktualisieren", der `/api/meta` erneut liest. Ein inzwischen per `ollama pull` geholtes lokales Modell erscheint danach in der Auswahl, ohne Neustart. Gewählter Anbieter, gewähltes Modell und das laufende Gespräch bleiben erhalten; ist das gewählte Modell verschwunden, wählt der Desktop das erste verfügbare.
- **AC-2** `/api/meta` nennt für jedes Modell des lokalen Anbieters `tools: true|false`, ermittelt über `POST /api/show`. Kennt der Ollama-Dienst die Fähigkeitsliste nicht (ältere Version) oder antwortet er nicht, fehlt `tools`: der Wert bleibt unbekannt und wird nicht geraten.
- **AC-3** Ein Modell mit `tools: false` bleibt wählbar und ist in der Auswahl als „ohne Werkzeuge" gekennzeichnet. Vorausgewählt wird zuerst ein Modell mit `tools: true`, dann eines mit unbekanntem Wert, zuletzt eines mit `tools: false`.
- **AC-4** Die Karten der Abläufe `sdd-conflicts` und `sdd-review` nennen den Belegstand zum gewählten Modell: „belegt", „im Test durchgefallen" oder „nicht belegt". Ein Modell, für das nichts belegt ist, wird nie als „belegt" geführt.
- **AC-5** Belegstände stehen in einer versionierten Tabelle im Code, je Eintrag mit Quelle (README „Nächste Schritte"), Protokoll und optionalem Modellpräfix; der spezifischste Treffer gewinnt. Die Tabelle wird nicht aus Modellnamen geschätzt, und ein Eintrag ohne Quelle ist ungültig.
- **AC-6** Ist Ollama nicht erreichbar, bleibt der lokale Anbieter wie bisher gelistet: mit dem Hinweis „nicht erreichbar", und die aus der Konfiguration bekannten Modelle bleiben sichtbar und wählbar — `available` bleibt wahr, solange Modelle konfiguriert sind. Ohne `sdd.config.yaml` steht weiterhin nur lokales Ollama bereit.
