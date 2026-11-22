---
feature: web-api-hardening
status: archived
archive-after:
release: v0.9.1
---

# Plan — Security Hardening für lokale Web-APIs (`web-api-hardening`)

Reihenfolge der Umsetzung für [Issue #22](https://github.com/Zheng-Bote/sdd_studio/issues/22). Jede Task nennt das Akzeptanzkriterium, auf das sie einzahlt.

## Tasks

- [x] AC-1, AC-2 `handleOpen` in `internal/ui/api.go` mit `os.Lstat` und `filepath.EvalSymlinks`-Boundary-Check gegen `s.Root` härten, sodass nur reguläre Dateien innerhalb des Repositories im Editor geöffnet werden.
- [x] AC-1, AC-2 Unit-Tests in `internal/ui/ui_test.go` ergänzen (`TestOpenRejectsSymlinkEscape`), die Symlinks auf externe Dateien sowie Symlink-Verzeichnisse prüfen und sicherstellen, dass `s.Exec` nicht aufgerufen wird.
- [x] AC-3, AC-4 In `internal/desk/server.go` (`ServeHTTP`) für alle `POST`-Anfragen unter `/api/` die `Origin`-Prüfung (`r.Header.Get("Origin") == "http://"+s.Addr`) ergänzen.
- [x] AC-3, AC-4 Unit-Tests in `internal/desk/` (`desk_test.go` / `modeltest_test.go`) um den `Origin`-Header ergänzen und explizite Negativtests für fehlenden oder abweichenden `Origin` bei `POST /api/*` hinzufügen.
- [x] AC-1, AC-2, AC-3, AC-4 `CHANGELOG.md` aktualisieren sowie `sdd gate pre-merge --feature web-api-hardening` und `go test ./...` ausführen.
