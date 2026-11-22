---
feature: web-api-hardening
status: active
archive-after:
touches:
  - internal/ui/**
  - internal/desk/**
  - CHANGELOG.md
---

# PRD — Security Hardening für lokale Web-APIs (`web-api-hardening`)

## Problem

Die Code-Review- und Security-Analyse ([Issue #22](https://github.com/Zheng-Bote/sdd_studio/issues/22)) ergab zwei Härtungspunkte in den lokalen Web-Oberflächen `sdd ui` und `sdd-desk`:

1. **Symlink-Auflösung beim Öffnen im Editor (`SEC-01`):** Der Endpunkt `POST /api/open` in `internal/ui/api.go` (`handleOpen`) prüft den übergebenen Pfad zwar syntaktisch über `scan.CleanRel`, verwendet zur Existenzprüfung jedoch `os.Stat`. Da `os.Stat` symbolischen Links folgt, könnte ein innerhalb des Repositories abgelegter Symlink auf eine Datei außerhalb des Repository-Wurzelverzeichnisses verweisen und diese im konfigurierten `$EDITOR` öffnen. Dies verletzt die Pfad-Sicherheitsregel der Verfassung („Dateioperationen dürfen niemals außerhalb des konfigurierten Repository-Wurzelverzeichnisses lesen oder schreiben").
2. **Fehlende explizite Origin-Validierung in `sdd-desk` (`SEC-02`):** Während `sdd ui` (`internal/ui/server.go`) bei allen schreibenden Anfragen neben Sitzungstoken und `Host`-Header auch den `Origin`-Header strikt gegen `"http://" + s.Addr` prüft, prüft `sdd-desk` (`internal/desk/server.go`) bei `POST`-Anfragen auf `/api/*` bisher nur `Host` und `X-SDD-Token`. Im Sinne von Defense-in-Depth und einer einheitlichen Sicherheitsarchitektur beider lokaler Web-Server soll auch `sdd-desk` den `Origin`-Header für mutierende `POST`-Anfragen validieren.

## Ziel

Beide lokalen Web-Schnittstellen setzen die Sicherheitsregeln der Verfassung konsistent und ohne Lücken durch:

- `POST /api/open` in `sdd ui` verweigert symbolische Links sowie Pfade, deren Elternverzeichnis durch symbolische Links außerhalb des Repository-Wurzelverzeichnisses auflöst.
- `sdd-desk` lehnt alle `POST`-Anfragen unter `/api/` ab, deren `Origin`-Header nicht exakt `"http://" + s.Addr` entspricht (sofern ein `Origin`-Header gesetzt ist bzw. verpflichtend analog zur lokalen Browser-/Webview-Oberfläche).

## Nutzer

- **Entwickler mit `sdd ui`** — öffnen Artefakte aus der lokalen Oberfläche in `$EDITOR`, ohne dass präparierte Symlinks im Repository Dateien außerhalb der Projektgrenze öffnen können.
- **Nutzer von `sdd-desk` (Browser & Tauri-Desktop)** — profitieren von zusätzlichem Defense-in-Depth-Schutz gegen Cross-Origin-Anfragen auf mutierenden Endpunkten.

## Umfang

- Härtung von `handleOpen` in `internal/ui/api.go` mittels `os.Lstat` und Prüfung der aufgelösten Pfadgrenze innerhalb von `s.Root`.
- Ergänzung der `Origin`-Prüfung für `POST /api/*` in `internal/desk/server.go`.
- Anpassung und Ergänzung der Unit-Tests in `internal/ui/ui_test.go` und `internal/desk/desk_test.go`.
- Dokumentation der Sicherheitsverbesserung in `CHANGELOG.md`.

## Nicht-Ziele

- **Änderung der Token-Authentifizierung oder Bindungsadresse:** Beide Server binden weiterhin ausschließlich an `127.0.0.1` und verlangen `X-SDD-Token`.
- **Freigabe von Symlinks innerhalb des Repositories für `/api/open`:** Wie in `internal/mcp/write.go` (`readExisting`) werden in `/api/open` ausschließlich reguläre Dateien (keine Symlinks, Verzeichnisse oder Devices) zugelassen, deren aufgelöster Pfad innerhalb des Repository-Roots liegt.
- **Änderungen an `internal/sddcore` oder dem Regelkatalog `catalog/catalog.yaml`:** Es werden keine neuen SDD-Lint-Regeln eingeführt.

## Akzeptanzkriterien

- **AC-1** Wenn `POST /api/open` in `sdd ui` für eine reguläre Datei innerhalb des Repository-Wurzelverzeichnisses aufgerufen wird, startet der Server wie bisher `$EDITOR` und antwortet mit HTTP `204 No Content`.
- **AC-2** Wenn der Zielpfad in `POST /api/open` ein symbolischer Link ist oder über ein symbolisch verlinktes Elternverzeichnis außerhalb von `s.Root` auflöst, lehnt `sdd ui` die Anfrage mit HTTP `400 Bad Request` oder `404 Not Found` ab und führt `$EDITOR` nicht aus.
- **AC-3** Wenn `sdd-desk` eine `POST`-Anfrage auf `/api/*` empfängt, deren `Origin`-Header fehlt oder nicht `"http://" + s.Addr` lautet, lehnt der Server die Anfrage mit HTTP `403 Forbidden` ab.
- **AC-4** Wenn `sdd-desk` eine `POST`-Anfrage auf `/api/*` mit gültigem `X-SDD-Token`, passendem `Host` und `Origin: http://<s.Addr>` empfängt, wird die Anfrage regulär verarbeitet.
