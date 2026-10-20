# sdd-desk — Oberfläche

Angular 22, Spartan-NG (Helm, Style „nova“) und Tailwind 4. Die Oberfläche des optionalen KI-Desktops `sdd-desk` ([ADR 0002](../../docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)); das Go-Backend liegt in `cmd/sdd-desk` und `internal/desk`. Später wird daraus eine Tauri-v2-Anwendung (`src-tauri/`), das Go-Backend läuft dann als Sidecar.

| Befehl | Zweck |
|---|---|
| `npm ci` | Abhängigkeiten installieren |
| `npm run build:desk` | bauen und nach `internal/desk/webdist` kopieren (wird in `sdd-desk` eingebettet) |
| `npm test` | Unit-Tests (Vitest) |
| `npm start` | Entwicklungsserver auf `localhost:4200`; `/api` geht an `sdd-desk --port 47900` |

Für `npm start` das Backend mit `--port 47900` starten und die ausgegebene `?token=…` an `http://localhost:4200/` anhängen.

Bedienung, Freigaben und Konfiguration: [Handbuch, Teil 4](../../handbuch/04-agenten.md#sdd-desk).
