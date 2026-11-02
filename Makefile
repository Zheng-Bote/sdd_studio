# SDD Studio — baut alle Programme und installiert sie lokal.
#
#   make                 sdd, sdd-desk (Web-Oberfläche eingebettet) und die Desktop-Anwendung sdd-desk-app bauen
#   make install         nach $(PREFIX) installieren, inkl. Desktop-Verknüpfung und Icons für sdd-desk-app
#   make uninstall       alles wieder entfernen
#   make clean           Build-Ausgaben löschen
#
# Standard ist eine Installation für den eigenen Benutzer (~/.local, ohne root).
# Systemweit:  sudo make install PREFIX=/usr/local
# Für Pakete:  make install DESTDIR=/tmp/pkgroot PREFIX=/usr
#
# sdd-desk-app, sdd-desk und sdd liegen im selben Verzeichnis: Tauri sucht den Sidecar sdd-desk neben
# sdd-desk-app, und sdd-desk sucht sdd neben sich. Die Schlüssel (llm.env) installiert make nie (ADR 0002);
# die Desktop-Anwendung liest sie aus ~/.config/io.github.zhengbote.sdddesk/llm.env (Rechte 0600).

PREFIX   ?= $(HOME)/.local
BINDIR   ?= $(PREFIX)/bin
DATADIR  ?= $(PREFIX)/share
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

# Icons aus apps/desk/src-tauri/icons: Größe im hicolor-Thema -> Datei
ICONS := 32x32:32x32.png 64x64:64x64.png 128x128:128x128.png 256x256:128x128@2x.png 512x512:icon.png

GOFLAGS_BUILD := -trimpath

.PHONY: all cli desk-app ui install install-cli install-desk uninstall clean help
.NOTPARALLEL:

# Reihenfolge: Oberfläche zuerst, weil sdd-desk sie einbettet und die Desktop-Anwendung sie mitbringt.
all: ui cli desk-app

help:
	@sed -n '2,15p' Makefile | sed 's/^# \{0,1\}//'

# --- Oberfläche -------------------------------------------------------------------------------------------------

$(DESK_DIR)/node_modules/.package-lock.json: $(DESK_DIR)/package-lock.json
	cd $(DESK_DIR) && $(NPM) ci

# Baut die Angular-Oberfläche und kopiert sie nach internal/desk/webdist, wo sdd-desk sie einbettet.
ui: $(DESK_DIR)/node_modules/.package-lock.json
	cd $(DESK_DIR) && $(NPM) run build:desk

# --- Go-Programme -----------------------------------------------------------------------------------------------

cli:
	@test -f $(WEBDIST) || { echo "Oberfläche fehlt: zuerst 'make ui' oder 'make'"; exit 1; }
	$(GO) build $(GOFLAGS_BUILD) -o bin/sdd ./cmd/sdd
	$(GO) build $(GOFLAGS_BUILD) -o bin/sdd-desk ./cmd/sdd-desk

# --- Desktop-Anwendung (Tauri) ----------------------------------------------------------------------------------

# Die Oberfläche ist schon gebaut; als beforeBuildCommand laufen daher nur die Sidecars (statt desk:sidecars).
# --no-bundle erzeugt nur das Programm, keine deb/rpm/AppImage.
desk-app: $(DESK_DIR)/node_modules/.package-lock.json
	@test -f $(WEBDIST) || { echo "Oberfläche fehlt: zuerst 'make ui' oder 'make'"; exit 1; }
	cd $(DESK_DIR) && $(NPM) run tauri -- build --no-bundle \
		--config '{"build":{"beforeBuildCommand":"node scripts/build-sidecars.mjs"}}'

# make install ohne vorheriges make baut einmal alles.
$(GO_BINS) $(APP_OUT):
	$(MAKE) all

# --- Installation -----------------------------------------------------------------------------------------------

install: install-cli install-desk

install-cli: $(GO_BINS)
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 0755 bin/sdd "$(DESTDIR)$(BINDIR)/sdd"
	$(INSTALL) -m 0755 bin/sdd-desk "$(DESTDIR)$(BINDIR)/sdd-desk"

install-desk: install-cli $(APP_OUT)
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

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/sdd" "$(DESTDIR)$(BINDIR)/sdd-desk" "$(DESTDIR)$(BINDIR)/$(APP_BIN)"
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
	find internal/desk/webdist -mindepth 1 ! -name README.md -delete
