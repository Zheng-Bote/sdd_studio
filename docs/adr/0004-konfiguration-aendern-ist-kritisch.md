---
status: accepted
date: 2026-11-05
supersedes: []
references:
  - docs/adr/0003-ki-desktop-darf-alle-sdd-aktionen-mit-freigabe.md
  - docs/sdd_konzept.md
---

# ADR 0004 — Konfiguration ändern ist im KI-Desktop eine kritische Aktion

## Kontext

ADR 0003 ordnet „Konfiguration lesen und ändern“ der Stufe **schreibend** zu. Diese Stufe zeigt vorher einen Diff und lässt sich für die Sitzung je Werkzeug freigeben. Die Tech Spec `desk-write-content` hat dabei als Risiko festgehalten, dass eine Änderung an `sdd.config.yaml` Prüfungen aushebeln kann, und die Entscheidung an ein späteres ADR verwiesen.

Im Live-Test vom 2026-11-05 (Claude Sonnet 5.5, Szenario cert-expiry-watcher) ist genau das sichtbar geworden. Der Nutzer hatte `sdd_write` für die Sitzung freigegeben, um ein PRD zu ändern. Im Ablauf `sdd-tests` hat das Modell danach `tests.include` in `sdd.config.yaml` gesetzt, ohne Karte. Die Änderung war richtig und erwünscht. Aber dasselbe Werkzeug mit derselben Freigabe hätte auch `exclude: ["**"]` schreiben können.

Die Konfiguration ist das Gesetz über die Prüfungen. Fast jedes Feld kann eine Prüfung abschwächen:

| Feld | Abschwächung, z. B. |
|---|---|
| `profile` | `standard` → `minimal` schaltet Regeln ab |
| `exclude` | ein Muster nimmt Dateien aus jeder Prüfung |
| `gates.<gate>.override` | ersetzt die Regeln eines Gates durch eine kürzere Liste |
| `thresholds` | leere Listen: PRD und Tech Spec werden nie verlangt (SDD-020, SDD-030) |
| `tests.include` | ein zu weites Muster lässt beliebige Dateien als Testverweis gelten (SDD-022) |
| `artifacts.*` | verschobene Pfade lassen Artefakte aus dem Blick fallen |
| `base_ref` | eine andere Basis verkleinert den Diff für SDD-024 |
| `constitution_sections` | weniger Pflichtabschnitte (SDD-002a) |

Eine Abschwächung über die Konfiguration wirkt wie ein Eintrag in der Baseline, nur breiter: Sie unterdrückt nicht einen Befund, sondern ganze Regeln oder Pfade. Und sie gilt auch in CI, weil CI dieselbe Datei liest. Konzept §9.6 will gerade verhindern, dass ein Modell die einzige Bremse verhandeln kann.

Erwogen wurden:

1. **Bleibt schreibend** (Stand ADR 0003). Der Diff in Vorschau und Ergebnis und das Review des Pull Requests müssen reichen. Verworfen: Bei einer Freigabe für die Sitzung sieht der Mensch keine Vorschau mehr. Das Ergebnis mit Diff steht im Gespräch, also erst nachdem die Änderung geschehen ist. Die Konfiguration hat damit einen schwächeren Schutz als die Baseline, obwohl sie mehr Wirkung hat.
2. **Nur abschwächende Änderungen sind kritisch.** `sdd` erkennt je Feld, ob eine Änderung lockert oder verschärft. Verworfen: Für fast jedes Feld ist die Richtung nicht eindeutig. Ein weiteres `tests.include` kann richtig sein oder Testverweise erschleichen, neue `artifacts`-Pfade können Ordnung oder Versteck sein. Eine solche Erkennung wäre fehleranfällig und müsste mit jedem neuen Feld nachgezogen werden. Eine Bremse, die sich durch eine Lücke in der Erkennung umgehen lässt, ist keine.
3. **Jede Änderung an `sdd.config.yaml` ist kritisch** — vorgeschlagen. Die Bestätigung zeigt den Diff und die Wirkung auf die Befunde. Kosten: In einem neuen Projekt kommen einige zusätzliche Einzelbestätigungen dazu (im Live-Test eine: `tests.include`).

## Entscheidung

1. **Konfiguration ändern ist kritisch.** Im KI-Desktop ist jede Änderung an `sdd.config.yaml` eine kritische Aktion im Sinne von ADR 0003, Entscheidung 5: einzeln bestätigt, nie für die Sitzung, mit der vollständigen Wirkung in der Bestätigung. Das Anlegen der Konfiguration bei `sdd init` bzw. `sdd_init` bleibt schreibend. Dort gibt es noch nichts abzuschwächen.
2. **Eigenes Werkzeug.** Die Stufe gehört zum Werkzeug (ADR 0003, Entscheidung 5). Deshalb ändert ein eigenes kritisches Werkzeug `sdd_config` die Konfiguration, nur mit `--allow-critical`. `sdd_write` lehnt `sdd.config.yaml` künftig ab und verweist auf `sdd_config`. `sdd_read` liest die Konfiguration unverändert.
3. **Wirkung in der Bestätigung.** Die Vorschau von `sdd_config` zeigt:
   - den Diff;
   - je Gate (Lint, `pre-implementation`, `pre-merge`, `pre-release`) die Befunde, die mit der neuen Konfiguration wegfallen und die hinzukommen;
   - eine Zusammenfassung für die Schaltfläche, z. B. „sdd.config.yaml ändern: tests.include (+0 / −1 Befund)“.

   Dass keine Befunde wegfallen, beweist nicht, dass nichts abgeschwächt wird: Ein `exclude` für einen Pfad ohne heutige Befunde wirkt erst später. Deshalb entscheidet der Mensch anhand des Diffs. Die Befunde sind eine Hilfe, keine Freigabe.
4. **Mechanische Grenzen bleiben, wie sie sind.** Die Konfiguration muss sich laden und validieren lassen (wie heute in `sdd_write`), sonst wird nichts geschrieben. Dieses ADR fügt keine neuen inhaltlichen Sperren hinzu. Für die Entscheidung ist aber wichtig, was heute möglich ist:
   - **`profile`** kann Regeln mit `baseline: forbidden` abschalten. SDD-011, SDD-050 und SDD-051 gelten ab `standard`, ein Wechsel auf `minimal` nimmt sie aus der Prüfung. Die Konfiguration erlaubt also, was die Baseline ausdrücklich verbietet.
   - **`gates.<gate>.override`** kann eine Regel nicht aus allen Gates nehmen (SDD-072, selbst `forbidden`), wohl aber aus einem einzelnen Gate, z. B. aus `pre-merge`.

   Ob `sdd` solche Änderungen künftig selbst verweigern soll (etwa: kein Profilwechsel nach unten ohne ADR), ist eine eigene Frage für ein späteres ADR. Bis dahin schützt die kritische Stufe davor, dass ein Modell das ohne bewusste Bestätigung tut. Die Vorschau nennt einen Wechsel von `profile` und jede Änderung an `override` ausdrücklich.
5. **Terminal unverändert.** Im Terminal ändert der Mensch die Datei selbst. Ein CLI-Befehl zum Ändern der Konfiguration ist nicht vorgesehen.

## Konsequenzen

- **ADR 0003** bleibt gültig und unverändert (SDD-011). Dieses ADR verschiebt nur „Konfiguration ändern“ von der Stufe „schreibend“ in die Stufe „kritisch“. Es löst ADR 0003 nicht ab.
- **Konzept §9.6:** In der Aufzählung der Stufen wandert „Konfiguration“ von „schreibend“ zu „kritisch“, mit Verweis auf dieses ADR.
- **Verfassung:** unverändert. Sie nennt die Freigabestufen nicht.
- **Neues Feature** mit PRD und Tech Spec:
  - `sdd_config` in `sdd mcp` (kritisch, `--allow-critical`, Vorschau mit Diff und Befund-Differenz je Gate);
  - `sdd_write` lehnt `sdd.config.yaml` ab;
  - `sdd-desk` stuft `sdd_config` als kritisch ein;
  - Prompts (`common`, `sdd-tests`) nennen `sdd_config` statt `sdd_write` für die Konfiguration;
  - Handbuch.
- **Risiko:** Mehr Einzelbestätigungen bei der Einrichtung eines Projekts. Wer sie ungeprüft bestätigt, schwächt die Prüfungen ebenso wie mit einem Editor. Das ADR macht die Wirkung sichtbar, die Verantwortung bleibt beim Menschen.
- **Risiko:** Der Vergleich der Befunde braucht je Vorschau mehrere Prüfläufe. Für Projekte üblicher Größe ist das unkritisch. Wird es zu langsam, kann die Vorschau sich auf Lint und `pre-merge` beschränken, ohne dass sich an der Entscheidung etwas ändert.
