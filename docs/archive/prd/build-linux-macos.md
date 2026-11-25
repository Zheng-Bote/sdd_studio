---
feature: build-linux-macos
status: archived
archive-after:
touches:
  - Makefile
  - README.md
  - handbuch/**
  - CHANGELOG.md
  - apps/desk/**
release: v0.11.0
---

# PRD — Bauen und Installieren auf Linux und macOS (`build-linux-macos`)

Issue: [#31](https://github.com/Zheng-Bote/sdd_studio/issues/31)

## Problem

`make` und `make install` sind auf Linux zugeschnitten. Auf macOS — dem zweiten Zielsystem — entstehen zwei Lücken, beide beim Beleg am 2026-11-25 (v0.10.0, macOS arm64) von Hand umgangen:

- **`make` liefert keine startbare Anwendung.** `desk-app` ruft `tauri build --no-bundle`. Auf Linux genügt das, weil `install-desk` das nackte Binary nach `$(BINDIR)` legt. macOS startet Anwendungen dagegen aus einem `.app`-Bundle: ohne Info.plist, Icons und Sidecars in `Contents/MacOS` erscheint die App weder im Finder noch im Launchpad. Es fehlt also nicht die Funktion, sondern die Hülle, die das System erwartet.
- **`install-desk` legt wirkungslose Dateien an.** Der Schritt schreibt `share/applications/*.desktop` und ein `share/icons/hicolor/**`-Thema. macOS kennt weder das eine noch das andere. Nach einem `make install` liegen die Dateien im Home und tun nichts — schlimmer als nichts, weil sie den Eindruck erwecken, die App sei registriert.

Wer heute auf macOS installieren will, muss wissen, dass er zwei Befehle von Hand braucht:

```text
cd apps/desk && npm run tauri -- build --bundles app
ditto "SDD Studio Desktop.app" ~/Applications/"SDD Studio Desktop.app"
```

Dieses Wissen steht in keiner Datei des Projekts. Es ist am 2026-11-25 durch Ausprobieren entstanden.

## Ziel

Ein `make` und ein `make install` je System, ohne Handarbeit und ohne Wissen, das nur im Kopf existiert:

- **Linux:** unverändert — Oberfläche, `sdd`, `sdd-desk`, App-Binary, `.desktop`-Datei, Icons.
- **macOS:** Oberfläche, `sdd`, `sdd-desk` und ein `.app`-Bundle unter `$(APPDIR)`; keine `.desktop`-Datei, kein hicolor-Thema.

Das System wird **erkannt**, nicht geraten; die Pfade bleiben überschreibbar.

## Nutzer

- **Entwickler auf macOS** — wollen nach `git clone` mit einem Befehl bauen und mit einem zweiten installieren, wie auf Linux.
- **Betreiber beider Systeme** — brauchen eine Doku, die je System sagt, was entsteht und wie man es wieder entfernt.
- **Beitragende** — sollen nicht versehentlich die Linux-Installation beschädigen, während sie macOS unterstützen.

## Umfang

- `Makefile`: Systemerkennung, OS-abhängiger App-Bau (`--bundles app` vs. `--no-bundle`), OS-abhängige Installation und Deinstallation, neues `APPDIR`.
- `README.md` und `handbuch/`: Installationsziele je System.
- `CHANGELOG.md` unter `[Unreleased]`.
- Test, der die Zielpfade je System prüft, damit die Linux-Seite nicht still zurückfällt.

## Nicht-Ziele

- **Windows.** Die Sidecar-Tabelle kennt Windows-Triples, das Makefile deckt es nicht ab; eigener Schritt.
- **Signierung und Notarisierung** für macOS. Das Bundle bleibt ad-hoc signiert, wie Tauri es erzeugt; Gatekeeper-Warnungen bleiben möglich.
- **Universal-Binary** (x86_64 + arm64). Gebaut wird für die Host-Architektur.
- **Paketformate** (`.dmg`, `.deb`, `.rpm`, AppImage). Das `.app`-Bundle bzw. das Binary genügen.
- **Schlüssel installieren.** `llm.env` bleibt Sache des Menschen (ADR 0002).
- **Änderungen an `sdd` selbst.** Betroffen sind Bau und Installation, nicht der Prüfkern.

## Akzeptanzkriterien

- **AC-1** `make` baut auf Linux und macOS Oberfläche, `sdd`, `sdd-desk` und die Desktop-Anwendung; auf macOS entsteht dabei ein `.app`-Bundle, auf Linux ein App-Binary ohne Bundle.
- **AC-2** `make install` auf Linux legt `sdd`, `sdd-desk`, das App-Binary, die `.desktop`-Datei und die Icons in hicolor an — wie bisher.
- **AC-3** `make install` auf macOS legt `sdd` und `sdd-desk` in `$(BINDIR)` sowie das `.app`-Bundle in `$(APPDIR)` an und schreibt **keine** `.desktop`-Datei und kein hicolor-Icon-Thema.
- **AC-4** Das erzeugte Bundle heißt wie `productName` aus `tauri.conf.json` und enthält `Contents/MacOS/sdd-desk-app`, `Contents/MacOS/sdd-desk` und `Contents/MacOS/sdd-studio-sdd`; seine Version entspricht der Projektversion.
- **AC-5** `make uninstall` entfernt je System genau das, was `make install` angelegt hat; auf macOS auch ein früher dort abgelegtes `sdd-desk-app`-Binary.
- **AC-6** Das System wird über `uname -s` erkannt und lässt sich mit `OS=<system>` überschreiben; die Pfade `PREFIX`, `BINDIR`, `DATADIR` und `APPDIR` bleiben überschreibbar.
- **AC-7** Ein Test prüft die Zielpfade je System über eine Probelauf-Ausgabe von `make` (`-n`), sodass eine Änderung, die Linux oder macOS bricht, auffällt.
- **AC-8** `README.md` und Handbuch nennen je System, was `make` und `make install` anlegen und wie man es entfernt.
