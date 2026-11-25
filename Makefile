# SDD Studio — baut alle Programme und installiert sie lokal.
#
#   make                 sdd, sdd-desk (Web-Oberfläche eingebettet) und die Desktop-Anwendung bauen
#                        (macOS: als .app-Bundle, Linux: als Programm ohne Bundle)
#   make install         nach $(PREFIX) installieren; auf Linux zusätzlich Desktop-Verknüpfung und Icons,
#                        auf macOS das .app-Bundle nach $(APPDIR)
#   make uninstall       alles wieder entfernen
#   make clean           Build-Ausgaben löschen
#
# Das System wird über `uname -s` erkannt (Linux, Darwin) und lässt sich mit OS=<system> überschreiben.
# Standard ist eine Installation für den eigenen Benutzer (~/.local bzw. ~/Applications, ohne root).
# Systemweit:  sudo make install PREFIX=/usr/local APPDIR=/Applications
# Für Pakete:  make install DESTDIR=/tmp/pkgroot PREFIX=/usr
#
# sdd-desk-app, sdd-desk und sdd liegen im selben Verzeichnis: Tauri sucht den Sidecar sdd-desk neben
# sdd-desk-app, und sdd-desk sucht sdd neben sich. Die Schlüssel (llm.env) installiert make nie (ADR 0002);
# die Desktop-Anwendung liest sie aus ~/.config/io.github.zhengbote.sdddesk/llm.env (Rechte 0600).

# uname -s liefert "Linux" oder "Darwin"; ein gesetztes OS= hat Vorrang (Tests, Probeläufe).
OS       ?= $(shell uname -s)

PREFIX   ?= $(HOME)/.local
BINDIR   ?= $(PREFIX)/bin
DATADIR  ?= $(PREFIX)/share
# Nur auf macOS benutzt: Zielordner für das .app-Bundle (im Finder und Launchpad sichtbar).
APPDIR   ?= $(HOME)/Applications
DESTDIR  ?=

GO       ?= go
NPM      ?= npm
INSTALL  ?= install

DESK_DIR  := apps/desk
TAURI_DIR := $(DESK_DIR)/src-tauri
APP_ID    := io.github.zhengbote.sdddesk
APP_BIN   := sdd-desk-app
ICON_NAME := sdd-desk-app

GO_BINS  := bin/sdd bin/sdd-desk
APP_OUT  := $(TAURI_DIR)/target/release/$(APP_BIN)
WEBDIST  := internal/desk/webdist/index.html

# Quellen, von denen die Ziele abhängen (Issue #35). Ohne sie hält make eine vorhandene Zieldatei für aktuell,
# egal wie alt die Quellen sind — und `make install` kopiert dann Veraltetes.
# Die Go-Programme sind schnell gebaut; die Dateiliste erspart trotzdem unnötige Läufe und macht `make -q` ehrlich.
GO_SOURCES := go.mod $(shell find cmd internal -name '*.go' 2>/dev/null) $(shell test -f embed.go && echo embed.go)
# In die Programme eingebettet (embed.go): Katalog, Handbuch und Templates.
EMBEDDED   := $(shell find catalog handbuch templates -type f 2>/dev/null)
# Die Desktop-Anwendung baut Tauri aus diesen Quellen.
TAURI_SRC  := $(shell find $(TAURI_DIR)/src $(TAURI_DIR)/tauri.conf.json $(TAURI_DIR)/Cargo.toml -type f 2>/dev/null)

# Das Bundle, das Tauri auf macOS packt: Verzeichnisname ist productName + ".app" aus tauri.conf.json
# (eine Prüfung hält beides zusammen). Auf Linux bleibt es ungenutzt.
#
# Achtung: make trennt Ziele und Voraussetzungen an Leerzeichen. Als Make-Ziel muss das Leerzeichen im
# Produktnamen maskiert werden ($(APP_BUNDLE_GOAL)), sonst sieht make drei Ziele — "…/macos/SDD", "Studio"
# und "Desktop.app" — und führt das Rezept dreimal aus. Für die Shell bleibt der unmaskierte Name richtig.
APP_BUNDLE_NAME := SDD Studio Desktop.app
APP_BUNDLE      := $(TAURI_DIR)/target/release/bundle/macos/$(APP_BUNDLE_NAME)
APP_BUNDLE_GOAL := $(TAURI_DIR)/target/release/bundle/macos/SDD\ Studio\ Desktop.app

# macOS startet Anwendungen aus einem .app-Bundle; Linux installiert das Programm direkt.
ifeq ($(OS),Darwin)
BUNDLE_ARG := --bundles app
else
BUNDLE_ARG := --no-bundle
endif

# Icons aus apps/desk/src-tauri/icons: Größe im hicolor-Thema -> Datei
ICONS := 32x32:32x32.png 64x64:64x64.png 128x128:128x128.png 256x256:128x128@2x.png 512x512:icon.png

GOFLAGS_BUILD := -trimpath

.PHONY: all cli desk-app ui install install-cli install-bundle-macos install-desktop-linux \
        uninstall uninstall-cli uninstall-bundle-macos uninstall-desktop-linux clean help print-os
.NOTPARALLEL:

# Reihenfolge: Oberfläche zuerst, weil sdd-desk sie einbettet und die Desktop-Anwendung sie mitbringt.
all: ui cli desk-app

help:
	@sed -n '2,17p' Makefile | sed 's/^# \{0,1\}//'

# Zeigt das erkannte System; ein gesetztes OS= gewinnt. Nützlich, wenn ein Ziel das falsche tut.
print-os:
	@printf '%s\n' '$(OS)' | tr '[:upper:]' '[:lower:]'

# --- Oberfläche -------------------------------------------------------------------------------------------------

$(DESK_DIR)/node_modules/.package-lock.json: $(DESK_DIR)/package-lock.json
	cd $(DESK_DIR) && $(NPM) ci

# Baut die Angular-Oberfläche und kopiert sie nach internal/desk/webdist, wo sdd-desk sie einbettet.
ui: $(DESK_DIR)/node_modules/.package-lock.json
	cd $(DESK_DIR) && $(NPM) run build:desk

# --- Go-Programme -----------------------------------------------------------------------------------------------

cli: bin/sdd bin/sdd-desk

bin/sdd: $(GO_SOURCES) $(EMBEDDED)
	@test -f $(WEBDIST) || { echo "Oberfläche fehlt: zuerst 'make ui' oder 'make'"; exit 1; }
	$(GO) build $(GOFLAGS_BUILD) -o bin/sdd ./cmd/sdd

# sdd-desk bettet die Oberfläche ein, deshalb hängt es zusätzlich an webdist.
bin/sdd-desk: $(GO_SOURCES) $(EMBEDDED) $(WEBDIST)
	@test -f $(WEBDIST) || { echo "Oberfläche fehlt: zuerst 'make ui' oder 'make'"; exit 1; }
	$(GO) build $(GOFLAGS_BUILD) -o bin/sdd-desk ./cmd/sdd-desk

# --- Desktop-Anwendung (Tauri) ----------------------------------------------------------------------------------

# Die Oberfläche ist schon gebaut; als beforeBuildCommand laufen daher nur die Sidecars (statt desk:sidecars).
# Auf macOS entsteht hier das .app-Bundle, auf Linux nur das Programm — deshalb $(BUNDLE_ARG).
# desk-app bleibt ohne Dateiziel: Tauri entscheidet selbst, was neu kompiliert wird; ein Bundle-Ziel würde
# `make` bei jedem Aufruf ausbremsen (Issue #35).
#
# Der Befehl steht als Variable, weil ihn zwei Stellen brauchen: desk-app und die Dateiziele darunter. Er darf kein
# $(MAKE) enthalten — GNU make führt solche Zeilen auch unter -q und -n aus (Issue #39).
define TAURI_BUILD
	@test -f $(WEBDIST) || { echo "Oberfläche fehlt: zuerst 'make ui' oder 'make'"; exit 1; }
	cd $(DESK_DIR) && $(NPM) run tauri -- build $(BUNDLE_ARG) \
		--config '{"build":{"beforeBuildCommand":"node scripts/build-sidecars.mjs"}}'
endef

desk-app: $(DESK_DIR)/node_modules/.package-lock.json
	$(TAURI_BUILD)

# Wer ein Dateiziel direkt anspricht (install-cli, install-desktop-linux, install-bundle-macos), bekommt es aktuell:
# die Programme hängen an ihren Quellen, das Linux-Programm und das Bundle zusätzlich an allem, was Tauri einpackt.
# Das Rezept baut selbst, statt einen Sub-Make zu rufen — sonst würde `make -q` auf ein veraltetes Ziel einen vollen
# Bau starten und einen Fehlercode liefern, statt 0 oder 1 zu antworten (Issue #39).
$(APP_OUT): $(GO_BINS) $(EMBEDDED) $(TAURI_SRC) $(WEBDIST)
	$(TAURI_BUILD)

ifeq ($(OS),Darwin)
$(APP_BUNDLE_GOAL): $(GO_BINS) $(EMBEDDED) $(TAURI_SRC) $(WEBDIST)
	$(TAURI_BUILD)
endif

# --- Installation -----------------------------------------------------------------------------------------------

# Je System andere Ziele: Linux legt Programm, Verknüpfung und Icons ab, macOS kopiert das .app-Bundle.
# Gemeinsam ist sdd und sdd-desk auf dem PATH.
# `all` steht bei jedem install-Ziel: sonst kopierte make eine vorhandene, aber veraltete Zieldatei (Issue #35).
ifeq ($(OS),Darwin)
install: all install-cli install-bundle-macos
uninstall: uninstall-bundle-macos uninstall-cli
else
install: all install-cli install-desktop-linux
uninstall: uninstall-desktop-linux uninstall-cli
endif

install-cli: $(GO_BINS)
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 0755 bin/sdd "$(DESTDIR)$(BINDIR)/sdd"
	$(INSTALL) -m 0755 bin/sdd-desk "$(DESTDIR)$(BINDIR)/sdd-desk"

uninstall-cli:
	rm -f "$(DESTDIR)$(BINDIR)/sdd" "$(DESTDIR)$(BINDIR)/sdd-desk"

# macOS: das Bundle ist die Anwendung — es trägt Info.plist, Icons und die drei Binaries in Contents/MacOS.
# sdd-desk liegt deshalb nicht zusätzlich in $(BINDIR); Tauri sucht die Sidecars neben dem Programm.
install-bundle-macos: $(APP_BUNDLE_GOAL)
	@test -d "$(APP_BUNDLE)" || { echo "Bundle fehlt: zuerst 'make' — es baut mit $(BUNDLE_ARG)"; exit 1; }
	$(INSTALL) -d "$(DESTDIR)$(APPDIR)"
	# Ein früheres 'make install' (vor der Bundle-Unterstützung) legte das Programm nach $(BINDIR); sonst stünde
	# eine zweite, ältere Kopie der Anwendung im PATH.
	rm -f "$(DESTDIR)$(BINDIR)/$(APP_BIN)"
	rm -rf "$(DESTDIR)$(APPDIR)/$(APP_BUNDLE_NAME)"
	ditto "$(APP_BUNDLE)" "$(DESTDIR)$(APPDIR)/$(APP_BUNDLE_NAME)"
	@if [ -z "$(DESTDIR)" ]; then \
		echo "Installiert: $(APPDIR)/$(APP_BUNDLE_NAME)"; \
		echo "Schlüssel: ~/Library/Application Support/$(APP_ID)/llm.env anlegen (Vorlage config/llm.env.example, chmod 600)."; \
	fi

uninstall-bundle-macos:
	rm -rf "$(DESTDIR)$(APPDIR)/$(APP_BUNDLE_NAME)"
	# Ein früheres 'make install' (vor der Bundle-Unterstützung) legte das Programm nach $(BINDIR).
	rm -f "$(DESTDIR)$(BINDIR)/$(APP_BIN)"

# Linux: Programm nach $(BINDIR), dazu Desktop-Verknüpfung und hicolor-Icon-Thema.
install-desktop-linux: $(APP_OUT)
	$(INSTALL) -m 0755 $(APP_OUT) "$(DESTDIR)$(BINDIR)/$(APP_BIN)"
	@set -e; for entry in $(ICONS); do \
		size=$${entry%%:*}; file=$${entry#*:}; \
		$(INSTALL) -d "$(DESTDIR)$(DATADIR)/icons/hicolor/$$size/apps"; \
		$(INSTALL) -m 0644 "$(TAURI_DIR)/icons/$$file" "$(DESTDIR)$(DATADIR)/icons/hicolor/$$size/apps/$(ICON_NAME).png"; \
	done
	$(INSTALL) -d "$(DESTDIR)$(DATADIR)/icons/hicolor/scalable/apps"
	$(INSTALL) -m 0644 $(TAURI_DIR)/app-icon.svg "$(DESTDIR)$(DATADIR)/icons/hicolor/scalable/apps/$(ICON_NAME).svg"
	$(INSTALL) -d "$(DESTDIR)$(DATADIR)/applications"
	printf '%s\n' \
		'[Desktop Entry]' \
		'Type=Application' \
		'Name=SDD Studio Desktop' \
		'Comment=Spec-Driven Development im Gespräch mit einem Sprachmodell' \
		'Exec=$(BINDIR)/$(APP_BIN) %f' \
		'TryExec=$(BINDIR)/$(APP_BIN)' \
		'Icon=$(ICON_NAME)' \
		'StartupWMClass=$(APP_BIN)' \
		'Terminal=false' \
		'Categories=Development;' \
		'Keywords=SDD;Spec;PRD;LLM;KI;' \
		> "$(DESTDIR)$(DATADIR)/applications/$(APP_ID).desktop"
	chmod 0644 "$(DESTDIR)$(DATADIR)/applications/$(APP_ID).desktop"
	@if [ -z "$(DESTDIR)" ]; then \
		command -v desktop-file-validate >/dev/null && desktop-file-validate "$(DATADIR)/applications/$(APP_ID).desktop" || true; \
		command -v update-desktop-database >/dev/null && update-desktop-database -q "$(DATADIR)/applications" || true; \
		command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q -t -f "$(DATADIR)/icons/hicolor" || true; \
		echo "Installiert nach $(BINDIR); Verknüpfung: $(DATADIR)/applications/$(APP_ID).desktop"; \
		echo "Schlüssel: ~/.config/$(APP_ID)/llm.env anlegen (Vorlage config/llm.env.example, chmod 600)."; \
	fi

uninstall-desktop-linux:
	rm -f "$(DESTDIR)$(BINDIR)/$(APP_BIN)"
	rm -f "$(DESTDIR)$(DATADIR)/applications/$(APP_ID).desktop"
	@for entry in $(ICONS); do rm -f "$(DESTDIR)$(DATADIR)/icons/hicolor/$${entry%%:*}/apps/$(ICON_NAME).png"; done
	rm -f "$(DESTDIR)$(DATADIR)/icons/hicolor/scalable/apps/$(ICON_NAME).svg"
	@if [ -z "$(DESTDIR)" ]; then \
		command -v update-desktop-database >/dev/null && update-desktop-database -q "$(DATADIR)/applications" || true; \
		command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -q -t -f "$(DATADIR)/icons/hicolor" || true; \
	fi

clean:
	rm -f $(GO_BINS)
	rm -rf $(TAURI_DIR)/binaries $(TAURI_DIR)/target/release/$(APP_BIN)
	rm -rf $(TAURI_DIR)/target/release/bundle
	find internal/desk/webdist -mindepth 1 ! -name README.md -delete
