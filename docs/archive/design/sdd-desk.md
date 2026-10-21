---
feature: sdd-desk
status: archived
release: v0.2.0
archive-after:
touches:
  - cmd/sdd-desk/**
  - internal/desk/**
  - apps/desk/**
  - .gitignore
  - handbuch/**
  - sdd.config.yaml
---

# Tech Spec — KI-Desktop (`sdd-desk`)

Setzt das PRD `docs/archive/prd/sdd-desk.md` um. Rahmen: [ADR 0002](../../adr/0002-optionaler-ki-desktop-getrennt-vom-ki-freien-sdd.md). Vorlage für Aufbau und Werkzeuge ist `md_indexer/apps/desktop` (Angular 22, Spartan-NG, Tailwind 4, Tauri v2 mit Go-Sidecar).

## Architektur

```text
apps/desk/                Angular 22 + Spartan-NG (Helm, Style "nova") + Tailwind 4; später src-tauri/
cmd/sdd-desk/             Einstieg: --project, --config, --sdd, --port, --open
internal/desk/
  ├── config.go           llm.env laden, Rechte prüfen, Prozessvariablen vorziehen
  ├── provider*.go        Ollama (lokal/Cloud), OpenAI-kompatibel (OpenRouter, NVIDIA, Gemini), Claude — reines HTTP
  ├── mcpclient.go        startet `sdd mcp --allow-write`, JSON-RPC über stdio
  ├── agent.go            Gesprächsschleife mit Erlaubnis je Werkzeugaufruf
  ├── server.go           lokale HTTP-API + ausgelieferte Oberfläche (embed)
  └── webdist/            gebaute Oberfläche (Build-Ausgabe, nicht eingecheckt)
```

`sdd-desk` importiert keine `sdd`-Pakete außer in Tests: Werkzeuge erreicht es nur über den Prozess `sdd mcp`. Später startet Tauri das Go-Backend als Sidecar; die Oberfläche spricht es dann wie heute über `http://127.0.0.1:<port>` mit Token an.

## Schnittstellen

**Konfiguration** `config/llm.env` (Format wie `md_indexer`):

| Präfix | Anbieter | Protokoll | Standard-Endpunkt |
|---|---|---|---|
| `OLLAMA_LOCAL_` | Ollama lokal (ohne Schlüssel; Modelle zusätzlich aus `/api/tags`) | Ollama | `http://localhost:11434` |
| `OLLAMA_` | Ollama Cloud | Ollama | `https://ollama.com` |
| `OPENROUTER_` | OpenRouter | OpenAI | `https://openrouter.ai/api/v1` |
| `NVIDIA_` | NVIDIA NIM | OpenAI | `https://integrate.api.nvidia.com/v1` |
| `CLAUDE_` | Claude | Anthropic Messages | `https://api.anthropic.com` |
| `GEMINI_` | Gemini | OpenAI | `https://generativelanguage.googleapis.com/v1beta/openai` |

Je Präfix `_API_KEY` (leer, fehlend oder `none` = aus), `_ENDPOINT`, `_LLM_01…`. Ollama lokal ist eingeschaltet, sobald `OLLAMA_LOCAL_ENDPOINT` gesetzt ist oder `/api/tags` antwortet.

**HTTP-API** (nur `127.0.0.1`, `Host` geprüft, Header `X-SDD-Token` für jeden `/api/`-Aufruf):

| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/api/meta` | Projekt, Anbieter mit Modellen und `cloud`-Kennzeichen, Konfigurationswarnungen |
| GET | `/api/tools` | Werkzeuge von `sdd mcp` |
| GET | `/api/prompts`, POST `/api/prompts/get` | geführte Abläufe von `sdd mcp` |
| GET | `/api/next` | `sdd_next` für die Seitenleiste |
| POST | `/api/turn` | ein Gesprächszug (siehe unten) |

**Gesprächszug:** Die Oberfläche hält den Verlauf und schickt ihn mit `provider`, `model`, `allowed_tools` (für die Sitzung freigegeben), `approved_calls`, `denied_calls` und `disclosure_ack`. Das Backend arbeitet offene Werkzeugaufrufe ab — freigegebene ausführen, abgelehnte als Ablehnung beantworten — und fragt dann das Modell, bis es antwortet oder ein Aufruf eine Erlaubnis braucht. Antwort: neuer Verlauf und `pending` (Aufrufe, die eine Erlaubnis brauchen; für `sdd_new` mit `apply` samt Vorschau). Höchstens acht Modellaufrufe je Zug.

## Datenmodell

Verlauf anbieterneutral: `{role: user|assistant|tool, content, tool_calls[{id, name, arguments}], tool_call_id}`. Die Anbieter übersetzen: Ollama (`tool_calls[].function`, ohne IDs — das Backend vergibt sie), OpenAI (`tool_calls[].function.arguments` als JSON-Text, `role: tool` mit `tool_call_id`), Claude (`tool_use`-/`tool_result`-Blöcke, `system` getrennt).

## Umsetzung der Akzeptanzkriterien

| AC | Umsetzung | Test |
|---|---|---|
| AC-1 | `config.go` | `TestLoadConfig` |
| AC-2 | `provider*.go`, Modellauswahl | `TestProviders` |
| AC-3 | `mcpclient.go`, `agent.go` | `TestAgentUsesOnlySddTools` |
| AC-4 | `agent.go`, Freigabe-Dialog | `TestApprovalFlow`, `approval-dialog.spec.ts` |
| AC-5 | `agent.go`, Hinweis-Dialog | `TestCloudNeedsDisclosure` |
| AC-6 | `/api/prompts`, `/api/next`, Seitenleiste | `TestPromptsAndNext` |
| AC-7 | `server.go` | `TestServerSecurity` |
| AC-8 | Importprüfung | `TestSddDoesNotImportDesk` |

## Sicherheit

- Schlüssel nur in `config/llm.env` beim Programm, Rechte höchstens `0600`; nie im Projekt, nie in Antworten der API.
- Werkzeuge nur über `sdd mcp`; das Backend ruft kein Werkzeug auf, das `sdd mcp` nicht listet, und keines ohne Erlaubnis.
- Cloud-Aufrufe nur nach bestätigtem Hinweis; keine Telemetrie.

## Offene Punkte

- Streaming der Modellantworten folgt mit der Tauri-Hülle.
