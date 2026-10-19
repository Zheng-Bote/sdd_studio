---
status: active
---

# Verfassung — SDD Studio (`sdd`)

Diese Verfassung definiert die systemweiten, dauerhaften Architektur-, Sicherheits- und Prozessregeln für `sdd_studio` (`sdd` CLI und `internal/sddcore`).

## Architektur/Layering

1. **Headless zuerst:** Alle fünf Betriebsarten (`init`, `adopt`, `lint`, `new`, `gate`) sind vollständig in `internal/sddcore` gekapselt und über das CLI `cmd/sdd` ohne grafische Oberfläche ausführbar.
2. **Schichtentrennung:**
   - `cmd/sdd`: Ausschließlich CLI-Parsing, Ein-/Ausgabe und Exit-Code-Steuerung (`0`, `1`, `2`).
   - `internal/sddcore`: Domänenlogik (`catalog`, `config`, `scan`, `rules`, `baseline`, `report`).
3. **Dateien sind der einzige Zustand:** Es existiert keine versteckte Datenbank als Wahrheit. Konfigurationszustand liegt in `sdd.config.yaml`, maschinell verwalteter Prüfzustand in `.sdd/` (`baseline.json`, `adr-lock.json`).

## Security

1. **Vollständig offline und ohne Telemetrie:** Das Werkzeug baut keine ausgehenden Netzwerkverbindungen zu Cloud-Diensten auf.
2. **Lokale Oberfläche (`sdd ui`):** Bindung ausschließlich an `127.0.0.1`, strikte `Host`/`Origin`-Prüfung, Sitzungstoken für alle Schreiboperationen und Schreibzugriffe nur über den Plan/Apply-Pfad.
3. **Pfad-Sicherheit:** Dateioperationen dürfen niemals außerhalb des konfigurierten Repository-Wurzelverzeichnisses lesen oder schreiben.

## Strikte Regeln

1. **Keine Regel ohne Prüfvertrag und Handbuchabschnitt:** Jede Regel im Katalog (`catalog/catalog.yaml`) muss im selben Commit ihre Implementierung, ihre Gate-Zuordnung, ihren Handbuchabschnitt (`handbuch/`) und — sofern `error` — ihren deterministischen Prüfvertrag erhalten (`SDD-070` bis `SDD-073`).
2. **Keine Heuristik als `error`:** Blockierende Befunde (`error`) müssen deterministisch prüfbar sein. Heuristische Prüfungen sind maximal `warn`.
3. **Keine stillen Defaults:** Fehlt `sdd.config.yaml`, bricht jeder Prüflauf mit Exit-Code `2` ab.
4. **Baseline-Disziplin:** Regeln mit `baseline: forbidden` (`SDD-050`, `SDD-051`, `SDD-070`–`SDD-073`) dürfen niemals in `.sdd/baseline.json` unterdrückt werden.
