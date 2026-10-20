# sdd-desk — Oberfläche

Angular 22, Spartan-NG (Helm, Style „nova“) und Tailwind 4. Die Oberfläche des optionalen KI-Desktops `sdd-desk` ([ADR 0002](../../docs/adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md)); das Go-Backend liegt in `cmd/sdd-desk` und `internal/desk`. Die Tauri-v2-Hülle liegt in `src-tauri/`; sie startet `sdd-desk` und `sdd` als Sidecars und zeigt die Oberfläche direkt vom Sidecar.

| Befehl | Zweck |
|---|---|
| `npm ci` | Abhängigkeiten installieren |
| `npm run build:desk` | bauen und nach `internal/desk/webdist` kopieren (wird in `sdd-desk` eingebettet) |
| `npm test` | Unit-Tests (Vitest) |
| `npm start` | Entwicklungsserver auf `localhost:4200`; `/api` geht an `sdd-desk --port 47900` |
| `npm run desk:sidecars` | Oberfläche einbetten, `sdd` und `sdd-desk` nach `src-tauri/binaries/` bauen |
| `npm run tauri:dev` | Desktop-Anwendung im Entwicklungsmodus |
| `npm run tauri:build` | Bündel unter `src-tauri/target/release/bundle/` |
| `cd src-tauri && cargo test` | Tests der Hülle |

Für `npm start` das Backend mit `--port 47900` starten und die ausgegebene `?token=…` an `http://localhost:4200/` anhängen.

Bedienung, Freigaben und Konfiguration: [Handbuch, Teil 4](../../handbuch/04-agenten.md#sdd-desk).
