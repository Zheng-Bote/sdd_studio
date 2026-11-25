---
feature: build-linux-macos
status: draft
archive-after:
touches:
  - Makefile
  - README.md
  - handbuch/**
  - CHANGELOG.md
---

# Tech Spec — Bauen und Installieren auf Linux und macOS (`build-linux-macos`)

Setzt das PRD [`docs/prd/build-linux-macos.md`](../prd/build-linux-macos.md) um. Die Verfassung hat Vorrang; diese Tech Spec darf sie präzisieren, nie lockern.

## Architektur

Betroffen ist allein der Bau- und Installationsweg. `internal/`, `cmd/` und der Prüfkern bleiben unberührt; `sdd` ändert sich nicht.

```text
Makefile                      Systemerkennung, OS-abhängige Ziele für App-Bau, Installation, Deinstallation
apps/desk/scripts/build-sidecars.mjs   unverändert: bildet Target-Triples auf GOOS/GOARCH ab (Linux, macOS, Windows)
apps/desk/src-tauri/tauri.conf.json    unverändert: productName, mainBinaryName, externalBin
makefile_install_test.go      Probelauf der Ziele je System (Wurzelpaket)
README.md, handbuch/04-agenten.md      Installationsziele je System
```

Zwei Systeme, zwei Formen der Anwendung:

- **Linux** startet das Programm aus `$(BINDIR)`; die Verknüpfung im Anwendungsmenü entsteht über eine `.desktop`-Datei, das Icon über das hicolor-Thema.
- **macOS** startet ein `.app`-Bundle. Tauri packt es aus `tauri.conf.json`: `productName` bestimmt den Verzeichnisnamen, `mainBinaryName` das Programm in `Contents/MacOS`, `externalBin` die beiden Sidecars daneben.

## Schnittstellen

### Ziele des Makefile

| Ziel | Linux | macOS |
|---|---|---|
| `make` | Oberfläche, `bin/sdd`, `bin/sdd-desk`, App-Binary | dasselbe, zusätzlich das `.app`-Bundle |
| `make install` | `install-cli` + `install-desktop-linux` | `install-cli` + `install-bundle-macos` |
| `make uninstall` | `uninstall-desktop-linux` + `uninstall-cli` | `uninstall-bundle-macos` + `uninstall-cli` |
| `make print-os` | `linux` | `darwin` |

### Variablen

| Variable | Standard | Bedeutung |
|---|---|---|
| `OS` | `$(shell uname -s)` | erkanntes System; `OS=Darwin` erzwingt eines |
| `PREFIX` | `$(HOME)/.local` | Wurzel der Benutzerinstallation |
| `BINDIR` | `$(PREFIX)/bin` | `sdd`, `sdd-desk` |
| `DATADIR` | `$(PREFIX)/share` | `.desktop`-Datei und Icons (nur Linux) |
| `APPDIR` | `$(HOME)/Applications` | Zielordner des `.app`-Bundles (nur macOS) |
| `DESTDIR` | leer | Wurzel für Paketbauten |

`BUNDLE_ARG` ist `--bundles app` bei `OS=Darwin`, sonst `--no-bundle` — das entscheidet, ob Tauri ein Bundle packt.

## Datenmodell

Kein Datenmodell. Neu ist eine Konstante im Makefile, `APP_BUNDLE_NAME := SDD Studio Desktop.app`, die dem `productName` aus `tauri.conf.json` folgen muss. Eine Prüfung hält beides zusammen, weil ein Auseinanderlaufen sonst erst beim Kopieren auffällt („Bundle fehlt“).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `desk-app` baut mit `$(BUNDLE_ARG)` | `TestMakeAppBuildsBundleOnlyOnDarwin` |
| AC-2 | `install-desktop-linux` (unverändert aus `install-desk` übernommen) | `TestMakeInstallLinux` |
| AC-3 | `install-bundle-macos`: `ditto` nach `$(APPDIR)`, kein `.desktop`, kein hicolor | `TestMakeInstallMacOS` |
| AC-4 | `APP_BUNDLE_NAME` = `productName` + `.app`; `Contents/MacOS` = `mainBinaryName` + `externalBin` | `TestMacOSBundleMatchesTauriConfig` |
| AC-5 | `uninstall-*` je System, auf macOS zusätzlich `rm -f $(BINDIR)/sdd-desk-app` | `TestMakeUninstallPerSystem` |
| AC-6 | `OS ?= $(shell uname -s)`, `print-os`, überschreibbare Pfade | `TestMakeSystemIsDetectedAndOverridable` |
| AC-7 | Probelauf `make -n` mit `--old-file` für die Dateiziele | alle vier `TestMake*` |
| AC-8 | README-Abschnitt „Alles bauen und lokal installieren (Linux und macOS)“, Handbuchabschnitt | `TestInstallDocsCoverBothSystems` |

**Warum der Probelauf mit `--old-file`:** `make -n` führt Befehlszeilen aus, die `$(MAKE)` enthalten. Ohne die Dateiziele als „aktuell“ zu deklarieren, würde der Test einen vollständigen Bau auslösen, statt die Installationsbefehle zu zeigen.

## Sicherheit

- **Keine Schlüssel.** `make install` legt weiterhin kein `llm.env` an (ADR 0002); die Meldung nennt nur den Ort.
- **Löschbefehle bleiben eng.** `install-bundle-macos` und `uninstall-bundle-macos` löschen ausschließlich `$(APPDIR)/$(APP_BUNDLE_NAME)` — ein konstanter Name, der auf `.app` endet — und `$(BINDIR)/sdd-desk-app`. Kein rekursives Löschen eines überschreibbaren Pfades ohne diesen Namensbestandteil.
- **Signatur unverändert.** Das Bundle bleibt ad-hoc signiert; dieses Feature signiert und notarisiert nicht (PRD, Nicht-Ziele).

## Offene Punkte

- **Windows** bleibt offen; die Sidecar-Tabelle kennt die Triples, das Makefile nicht.
- **Ältere Linux-Reste auf macOS.** Wer vor diesem Feature `make install` auf macOS ausgeführt hat, behält `~/.local/share/applications/*.desktop` und `~/.local/share/icons/hicolor/` zurück. `make uninstall` entfernt sie auf macOS nicht, weil sie zu keiner Installation dieses Features gehören. Von Hand löschen, falls sie stören.
- **Gatekeeper.** Ohne Signatur kann macOS beim ersten Start warnen; ob das Projekt später signiert, ist eine eigene Entscheidung.
