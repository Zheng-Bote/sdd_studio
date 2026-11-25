---
feature: agenten-cli-proxy
status: archived
archive-after:
touches:
  - internal/desk/**
  - apps/desk/src/**
  - cmd/sdd-desk/**
  - README.md
  - CHANGELOG.md
  - docs/arc42.md
release: v0.10.0
---

# PRD — Lokale Agenten-CLIs im Desktop, über einen MCP-Proxy (`agenten-cli-proxy`)

Setzt [ADR 0010](../adr/0010-lokale-agenten-clis-ueber-mcp-proxy.md) um (Phase 7 Teil B). Belege: [`docs/notes/headless-vertraege-agenten-clis.md`](../notes/headless-vertraege-agenten-clis.md).

## Problem

Der KI-Desktop führt den Werkzeugkreislauf heute selbst: Er fragt ein Sprachmodell, stuft jeden Werkzeugwunsch ein (`readTools`, `writeTools`, `criticalTools` in `internal/desk/agent.go`), zeigt die Freigabekarte und reicht erst nach der Entscheidung an `sdd mcp` weiter. Genau darauf beruht die Zusage aus ADR 0003 — jede kritische Aktion wird einzeln bestätigt, nie für die Sitzung.

Eine installierte Agenten-CLI (`claude`, `codewhale`) ist dagegen **selbst ein Agent**: Sie plant, ruft Werkzeuge auf und schreibt Dateien in ihrer eigenen Schleife. Wer sie einfach neben den Desktop stellt, verliert die Freigabehoheit — die Karten des Desktops sehen nichts, und die Zusage aus ADR 0003 stimmt stillschweigend nicht mehr. Wer sie gar nicht anbietet, lässt den Nutzen liegen: Beide CLIs sind auf diesem Rechner installiert, beide Verträge sind belegt, und der Desktop könnte sie führen, ohne `sdd` selbst mit KI zu belasten.

Dazu kommen drei Ehrlichkeitsprobleme, die mit der Umsetzung fällig werden:

- **Die README-Zusage würde falsch.** Dort steht „Nur die Werkzeuge von `sdd mcp`, keine eigenen, kein Dateizugriff daran vorbei" (README, „Was der Desktop darf"). Für ein gehostetes CLI gilt das nur noch für Aktionen **über** `sdd`; was die CLI mit ihren eigenen Werkzeugen schreibt, sieht keine Karte, keine Regelprüfung und kein Gate.
- **`claude` und `codewhale` sind Cloud.** Der Prozess läuft lokal, das Modell nicht. Ohne Hinweis vor dem ersten Zug wäre der Desktop stillschweigend zum Cloud-Client geworden.
- **„Läuft" ist nicht „belegt".** Ob eine CLI die Abläufe wirklich trägt, ist nach dem Vertragsbeleg genauso offen wie bei den lokalen Modellen. Eine Auskunft ohne Test wäre eine Behauptung.

## Ziel

Ein Nutzer kann im Desktop eine belegte Agenten-CLI als Antrieb wählen. Jeder Werkzeugaufruf dieser CLI erscheint als Freigabekarte, bevor er läuft; ohne Entscheidung fließt kein Inhalt. `sdd` bleibt KI-frei und offline, `--allow-critical` erreicht die CLI nie, und alles, was der Desktop über eine CLI behauptet, ist durch einen Test gedeckt oder als „nicht belegt" benannt.

## Nutzer

- **Entwickler, die Claude Code oder codewhale ohnehin nutzen** — wollen den SDD-Prozess mit ihrer vertrauten CLI fahren, ohne die Freigabehoheit an sie abzugeben.
- **Reviewer und Betreiber** — müssen sehen, welcher Agent was vorgeschlagen hat, und im Nachhinein belegen können, dass nichts ohne Entscheidung lief.
- **Nutzer, die `agy` erwarten** — sollen erfahren, dass es zurückgestellt ist und warum, statt es in einer Auswahl zu finden, deren Sperre nicht belegt ist.

## Umfang

- Ein MCP-Endpunkt in `sdd-desk` auf `127.0.0.1`, den eine CLI als Server anspricht; `tools/list` wird aus dem Kindprozess `sdd mcp` durchgereicht.
- Start der CLI als Kindprozess von `sdd-desk` mit dem je CLI belegten Headless-Vertrag.
- Jeder `tools/call` wird zur Freigabekarte; die Entscheidung wird über eine Kennung je Aufruf korreliert, nicht über die Reihenfolge.
- Frist je Aufruf, Rücknahme offener Karten beim Ende der CLI-Sitzung.
- Auswahl der CLI in der Oberfläche, Cloud-Hinweis vor dem ersten Zug, sichtbare Kennzeichnung des CLI-Modus.
- Belegstand je CLI in `internal/desk/evidence.go`, analog zu den Modellen.
- Einschränkung der README-Zusage, Nachzug in `docs/arc42.md` und `CHANGELOG.md`.

## Nicht-Ziele

- **`agy` anbieten.** Die wirksame Schreibsperre ist nicht belegt; ohne Beleg kein Angebot.
- **Die eigenen Werkzeuge der CLI proxyen.** Der Proxy reicht ausschließlich die Werkzeuge von `sdd mcp` durch; was die CLI darüber hinaus kann, bleibt ihre Sache und wird nicht zum Werkzeug des Desktops.
- **`sdd` ändern.** `internal/mcp` bekommt keinen zweiten Transport, `cmd/sdd` keinen KI-Bezug; der Proxy lebt in `sdd-desk` (ADR 0010, Entscheidung 2).
- **Ein zweiter Werkzeugkatalog.** Die CLI sieht genau die Werkzeuge, die `sdd mcp` anbietet.
- **Pauschale Freigaben.** Keine Sitzungsfreigabe für kritische Aktionen, kein Freigabeschalter „alle Karten dieses Laufs".
- **CLIs installieren, anmelden oder konfigurieren.** Vorhandensein und Anmeldung setzt der Desktop voraus und meldet sie als Befund.
- **Lokale Modelle über die CLI routen.** Das Routing einer CLI ist nicht belegbar lokal; beide Kandidaten bleiben Cloud (ADR 0010, Entscheidung 6).

## Akzeptanzkriterien

- **AC-1** Der Desktop bietet genau die Agenten-CLIs an, für die beide Bedingungen aus ADR 0010 belegt sind (`claude`, `codewhale`). Fehlt für eine CLI der Beleg oder ist sie nicht installiert, erscheint sie nicht als wählbar; `agy` erscheint nie als wählbar, solange seine Schreibsperre nicht belegt ist.
- **AC-2** Jeder `tools/call` einer CLI erzeugt eine Freigabekarte, bevor er ausgeführt wird. Ohne Entscheidung wird kein Werkzeugergebnis an die CLI geliefert. Eine Sitzungsfreigabe gilt nur für die Stufe der Karte, auf der sie erteilt wurde; eine kritische Aktion läuft nur, wenn genau dieser Aufruf bestätigt wurde.
- **AC-3** Aufrufe werden über eine Kennung je Aufruf korreliert. Laufen zwei Aufrufe gleichzeitig, führt die Bestätigung von Karte X genau Aufruf X aus und niemals Aufruf Y; die Karte zeigt Kennung, CLI und Sitzung.
- **AC-4** Jeder Aufruf hat eine Frist. Eine nicht entschiedene Karte verfällt nach Ablauf, der Aufruf endet mit einem Fehlerergebnis statt einer Ausführung, und eine nach Ablauf eintreffende Bestätigung führt nichts mehr aus.
- **AC-5** Endet die CLI-Sitzung oder ihr Prozess, werden offene Karten zurückgenommen. Kein Aufruf läuft nach dem Ende der Sitzung noch aus, und die Kindprozesse (`sdd mcp`, CLI) enden mit dem Desktop.
- **AC-6** Vor dem ersten Zug einer CLI zeigt der Desktop den Cloud-Hinweis („Inhalte verlassen das Gerät"); ohne Bestätigung startet kein Lauf. Der CLI-Modus ist in der Oberfläche dauerhaft sichtbar gekennzeichnet.
- **AC-7** Die eigenen Schreibzugriffe der CLI sind durch den in der Tech Spec genannten Mechanismus verhindert: Der Desktop übergibt die Flags, die die Sperre herstellen, und übergibt keine Freigabe-umgehenden Flags (`--dangerously-skip-permissions` und Entsprechungen). Die CLI erreicht `sdd mcp` nur über den Proxy und erhält nie selbst `--allow-critical`.
- **AC-8** Der Belegstand je CLI steht in einer versionierten Tabelle in `internal/desk/evidence.go`, je Eintrag mit Quelle. Ohne Testeintrag nennt der Desktop die CLI „nicht belegt", nie „belegt"; die Karte des Laufs zeigt diesen Stand.
- **AC-9** Die Zusage im README ist eingeschränkt: Sie gilt für Aktionen über `sdd` („kein Dateizugriff an `sdd` vorbei") und nennt den CLI-Modus als Betriebsart mit eigenen Werkzeugen der CLI.
- **AC-10** Laufen zwei CLIs gleichzeitig, bleibt die Zuordnung eindeutig: Jede Karte nennt ihre CLI und ihre Sitzung, und die Entscheidung einer Karte wirkt nur in ihrer Sitzung.
