---
feature: web-api-hardening
status: active
archive-after:
touches:
  - internal/ui/**
  - internal/desk/**
  - CHANGELOG.md
---

# Tech Spec — Security Hardening für lokale Web-APIs (`web-api-hardening`)

Setzt das PRD `docs/prd/web-api-hardening.md` ([Issue #22](https://github.com/Zheng-Bote/sdd_studio/issues/22)) um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

Die Änderungen betreffen ausschließlich die beiden lokalen HTTP-Server-Pakete `internal/ui` und `internal/desk`. Die Domänenlogik in `internal/sddcore` und der MCP-Server in `internal/mcp` bleiben unverändert.

```text
internal/ui/api.go         handleOpen: os.Lstat + Symlink-/Root-Boundary-Check (insideRoot)
internal/ui/ui_test.go     Tests für reguläre Dateien (AC-1) und Symlink-Abweisung (AC-2)
internal/desk/server.go    ServeHTTP: Origin-Header-Prüfung bei POST /api/* (AC-3, AC-4)
internal/desk/desk_test.go Tests für fehlenden/fremden Origin (AC-3) und gültigen Origin (AC-4)
```

### 1. Symlink- und Pfadgrenzen-Prüfung in `internal/ui/api.go` (`handleOpen`)

Analog zu `readExisting` und `insideRoot` in `internal/mcp/write.go` sowie `Plan.Apply` in `internal/sddcore/scaffold/plan.go` wird `handleOpen` zweistufig gehärtet:

1. **`os.Lstat` statt `os.Stat`:** Prüft das Dateisystem-Objekt selbst, ohne einem Symlink am Blatt zu folgen. Ist der Eintrag kein reguläres File (`!info.Mode().IsRegular()`), wird `404 Not Found` (bzw. `400 Bad Request` bei Symlink/Pfadflucht) zurückgegeben.
2. **Auflösung von Elternverzeichnissen (`EvalSymlinks`):** Sowohl `s.Root` als auch der Zielpfad `abs` werden über `filepath.Abs` und `filepath.EvalSymlinks` aufgelöst. Liegt der real aufgelöste Pfad nicht unterhalb des real aufgelösten `s.Root`, wird die Anfrage mit `400 Bad Request` (`scan.ErrUnsafePath`) abgewiesen und `s.Exec` niemals aufgerufen.

### 2. Origin-Validierung in `internal/desk/server.go` (`ServeHTTP`)

In `ServeHTTP` von `internal/desk/server.go` wird im Block `if strings.HasPrefix(r.URL.Path, "/api/")` für alle `POST`-Requests geprüft:

```go
if r.Method == http.MethodPost {
    if r.Header.Get("Origin") != "http://"+s.Addr {
        writeError(w, http.StatusForbidden, errors.New("unerwarteter Origin"))
        return
    }
    if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
        writeError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type application/json erforderlich"))
        return
    }
}
```

Dies entspricht exakt dem Prüfverhalten in `internal/ui/server.go` (`requireWrite`). Browser und die Tauri-Webview senden bei `fetch(..., { method: 'POST' })` automatisch `Origin: http://127.0.0.1:<port>`.

## Schnittstellen

- **`POST /api/open` (`sdd ui`):**
  - Request/Response-Schema bleibt unverändert (`{"path": "...", "line": 1}` → `204 No Content`).
  - Neu: Symlinks oder über verlinkte Verzeichnisse aus `s.Root` ausbrechende Pfade liefern `400 Bad Request` bzw. `404 Not Found` und starten keinen Prozess.
- **`POST /api/*` (`sdd-desk`):**
  - Erfordert neben `Host: <s.Addr>`, `X-SDD-Token: <token>` und `Content-Type: application/json` nun auch `Origin: http://<s.Addr>`.
  - Fehlt `Origin` oder weicht der Wert ab, antwortet der Server mit `403 Forbidden` und `{"error": "unerwarteter Origin"}`.

## Datenmodell

Keine Änderungen am Datenmodell, an Konfigurationsdateien oder an Persistenzformaten.

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `handleOpen` in `internal/ui/api.go` akzeptiert reguläre Dateien innerhalb von `s.Root` und ruft `s.Exec` auf. | `internal/ui/ui_test.go` `TestUIServer` (Untertest `open regular file`) |
| AC-2 | `handleOpen` in `internal/ui/api.go` prüft `os.Lstat` und `filepath.EvalSymlinks` gegen `s.Root`; Symlinks auf externe Dateien oder durch verlinkte Unterordner werden abgewiesen. | `internal/ui/ui_test.go` `TestOpenRejectsSymlinkEscape` |
| AC-3 | `ServeHTTP` in `internal/desk/server.go` weist `POST /api/*` ohne oder mit fremdem `Origin`-Header mit `403 Forbidden` ab. | `internal/desk/desk_test.go` `TestServerSecurity` |
| AC-4 | `ServeHTTP` in `internal/desk/server.go` lässt `POST /api/*` mit `Origin: http://<s.Addr>` und gültigem Token passieren. | `internal/desk/desk_test.go` `TestServerSecurity` |

## Sicherheit

- **Verfassung Security #2 (Lokale Oberfläche):** Vereinheitlicht die strikte `Host`/`Origin`-Prüfung zwischen `sdd ui` und `sdd-desk`.
- **Verfassung Security #3 (Pfad-Sicherheit):** Schließt die Lücke, über symbolische Links im Repository Lese-/Schreibzugriffe in `$EDITOR` auf Dateien außerhalb des Repository-Wurzelverzeichnisses auszulösen.
- **Verfassung Security #1 (Offline):** Keine neuen Abhängigkeiten, keine ausgehenden Verbindungen.

## Offene Punkte

- Keine offenen Punkte.
