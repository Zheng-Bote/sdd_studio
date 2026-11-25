---
feature: build-linux-macos
status: draft
archive-after:
---

# Plan — Bauen und Installieren auf Linux und macOS (`build-linux-macos`)

Reihenfolge der Umsetzung. Jede Task nennt das Akzeptanzkriterium, auf das sie einzahlt.

PRD: [`docs/prd/build-linux-macos.md`](../prd/build-linux-macos.md) · Issue: [#31](https://github.com/Zheng-Bote/sdd_studio/issues/31)

## Tasks

- [x] **AC-6** `Makefile`: Systemerkennung `OS ?= $(shell uname -s)`; `APPDIR ?= $(HOME)/Applications`; `BUNDLE_ARG` (`--bundles app` bei Darwin, sonst `--no-bundle`). Alle Pfade bleiben überschreibbar.
- [x] **AC-1** `Makefile`: `desk-app` baut mit `$(BUNDLE_ARG)`, sodass auf macOS das Bundle entsteht und auf Linux das Binary wie bisher.
- [x] **AC-2, AC-3** `Makefile`: `install` ruft `install-cli` und ein OS-abhängiges Ziel — `install-desktop-linux` (.desktop + hicolor + App-Binary) bzw. `install-bundle-macos` (Bundle nach `$(APPDIR)`, kein `.desktop`).
- [x] **AC-5** `Makefile`: `uninstall` entfernt je System dasselbe, OS-abhängig; auf macOS zusätzlich ein altes `sdd-desk-app` in `$(BINDIR)` aus früheren Installationen.
- [x] **AC-4, AC-7** `makefile_install_test.go` (Wurzelpaket): Probelauf `make -n` mit erzwungenem `OS=`; prüft Zielpfade je System, das Fehlen der `.desktop`-Datei auf macOS, das Vorhandensein auf Linux, den Bundlenamen gegen `productName` und die Bundle-Binaries gegen `tauri.conf.json`.
- [x] **AC-8** `README.md` und `handbuch/`: je System sagen, was `make` und `make install` anlegen und wie man es entfernt. `CHANGELOG.md` unter `[Unreleased]`.

## Abschluss (gilt für alle Kriterien)

`make` und `make install` auf diesem Mac wirklich ausführen; `sdd lint` 0/0/0, `go test ./...`, `gate pre-merge`.

## Reihenfolge und Abhängigkeiten

1. `Makefile` (AC-6, dann AC-1, AC-2/3, AC-5) — eine Änderung, kein Zwischenstand, der Linux bricht.
2. Test (AC-4, AC-7) — prüft die Probelauf-Ausgabe, bevor der echte Lauf Zeit kostet.
3. Doku (AC-8).
4. Echter Bau und echte Installation auf macOS; Gegenprobe für Linux über `make -n`, weil hier kein Linux läuft.

## Was nicht in diesen Plan gehört

- Windows, Signierung/Notarisierung, Universal-Binary, Paketformate (siehe PRD, Nicht-Ziele).
- Ein neues Installationsziel für `llm.env`.
