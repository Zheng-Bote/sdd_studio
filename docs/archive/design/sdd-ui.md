---
feature: sdd-ui
status: archived
release: v0.2.0
archive-after:
touches:
  - cmd/**
  - internal/ui/**
  - internal/sddcore/**
  - handbuch/**
  - sdd.config.yaml
---

# Tech Spec — Lokale Oberfläche (`sdd-ui`)

Setzt das PRD `docs/archive/prd/sdd-ui.md` um. Maßgeblich sind Verfassung (Security 2: Bindung, `Host`/`Origin`, Token, Plan/Apply) und Konzept §8.6/§10.

## Architektur

```text
cmd/sdd            ui [--port N] [--open]
internal/ui
  ├── server.go    Listener auf 127.0.0.1, Middleware (Host, Header), Routen
  ├── api.go       JSON-Endpunkte über sddcore, overview, adopt, scaffold
  ├── markdown.go  minimale Markdown-Darstellung für das eingebettete Handbuch
  └── web/         index.html, app.js, app.css (per embed.FS, ohne Build-Kette)
```

Die Oberfläche ist eine Ansicht auf dieselbe Bibliothek: Jeder Endpunkt ruft genau die Funktion auf, die auch der entsprechende CLI-Befehl nutzt. Kein Handler greift selbst auf Dateien schreibend zu.

## Schnittstellen

| Methode | Pfad | Zweck | Token |
|---|---|---|---|
| GET | `/`, `/app.js`, `/app.css` | Oberfläche | nein |
| GET | `/api/overview` | `overview.Build` (wie `sdd report --format json`) | nein |
| GET | `/api/findings?gate=` | `sddcore.RunCheck` (wie `sdd lint`/`gate --json`) | nein |
| GET | `/api/artifacts` | Inventar mit Typ, Status, Alter, `archive-after` | nein |
| GET | `/api/adopt` | `adopt.Propose` + Bericht (wie `sdd adopt`) | nein |
| GET | `/handbook` | eingebettetes Handbuch als HTML | nein |
| POST | `/api/new` | `scaffold.NewPlan`; `apply:false` = Vorschau mit Inhalt, `apply:true` = `Plan.Apply` | ja |
| POST | `/api/adopt/apply` | `adopt.Apply` auf neuem Branch (wie `sdd adopt --apply`) | ja |
| POST | `/api/open` | `$EDITOR <pfad>` bzw. `+<zeile>`, nur Pfade im Repository | ja |

**Middleware für jede Anfrage:** `Host` muss exakt `127.0.0.1:<port>` sein, sonst `403`. Antwort-Header: `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `Cache-Control: no-store`; nie `Access-Control-Allow-Origin`.

**Schreibende Anfragen** (nur `POST`): Header `X-SDD-Token` gleich dem Sitzungstoken (Vergleich in konstanter Zeit) und `Origin` gleich `http://127.0.0.1:<port>`, sonst `403`. Der Body ist JSON mit `Content-Type: application/json` (verhindert einfache Formular-Posts), höchstens 64 KiB.

**Token:** 32 Byte aus `crypto/rand`, hexadezimal. Die ausgegebene URL trägt es als `?token=`; die Oberfläche übernimmt es in den Sitzungsspeicher des Tabs und entfernt es aus der Adresszeile.

## Datenmodell

- **Artefakte:** `adopt.Take` liefert Typ und Frontmatter; das Alter ist die Zeit seit der letzten Änderung der Datei (`mtime`), die Archivfrist das Feld `archive-after`.
- **Befunde:** Status `new` bzw. `baselined` aus dem Bericht; „Altbestand“ heißt `baselined`. Export als JSON (Berichtsformat v1) oder CSV (`rule_id,severity,status,path,line,message`).
- **Live-Prüfung:** Die Befundliste fragt `/api/findings` alle drei Sekunden ab, solange der Tab sichtbar ist; die Antwort wird nie zwischengespeichert.
- **Deep-Links:** `vscode://file/<absoluter pfad>:<zeile>` als Link; `$EDITOR` über `/api/open` (Argumente als Liste, keine Shell; ist `$EDITOR` leer, antwortet der Server mit `400`).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `ui.Listen` | `TestListenBindsLoopbackWithToken` |
| AC-2 | Middleware, statische Dateien | `TestRejectsForeignHost`, `TestSecurityHeadersAndNoExternalResources` |
| AC-3 | `requireWrite` | `TestWritesNeedTokenAndOrigin` |
| AC-4 | `/api/overview`, `/api/artifacts`, `/api/findings`, Oberfläche | `TestReadEndpoints` |
| AC-5 | `/api/new`, `/api/adopt`, `/api/adopt/apply` | `TestNewPreviewThenApply`, `TestAdoptPreviewThenApply`, `TestNoBaselineEndpoint` |
| AC-6 | ungespeicherte `/api/findings`, Polling in `app.js` | `TestFindingsReflectDiskChanges` |
| AC-7 | Befehlspalette in `app.js`, `/api/open` | `TestPaletteReachesAllViews`, `TestOpenStaysInRepository` |

## Sicherheit

- Bindung an `127.0.0.1`, nicht an `localhost` (Dual-Stack könnte auf `::1` oder `0.0.0.0` auflösen).
- Die `Host`-Prüfung schützt vor DNS-Rebinding; `Origin` und Token vor Cross-Site-Anfragen aus anderen Tabs.
- Alle Ausgaben in der Oberfläche werden als Text gesetzt (`textContent`), nie als HTML; das Handbuch-HTML erzeugt der Server aus dem eingebetteten Markdown mit Escaping.
- Kein Netzwerkzugriff nach außen; die Oberfläche lädt nur eigene Dateien.

## Offene Punkte

- Eine Desktop-Hülle bleibt optional (Konzept §8.4).
