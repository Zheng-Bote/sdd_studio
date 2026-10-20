# Testen des KI-Interfaces

_Zum Ausprobieren:_

```bash
cd apps/desk && npm run build:desk && cd ../..
go build -o bin/sdd ./cmd/sdd && go build -o bin/sdd-desk ./cmd/sdd-desk
bin/sdd-desk --project . --open
```

Ohne bin/config/llm.env steht nur das lokale Ollama zur Verfügung. bin/ ist in .gitignore ausgeschlossen.
