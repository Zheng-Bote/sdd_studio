---
status: accepted
date: 2026-11-05
supersedes: []
references:
  - docs/adr/0004-konfiguration-aendern-ist-kritisch.md
  - docs/sdd_konzept.md
  - catalog/catalog.yaml
---

# ADR 0005 — Eine Abschwächung der Konfiguration braucht ein ADR

## Kontext

ADR 0004 macht jede Änderung an `sdd.config.yaml` im KI-Desktop zu einer kritischen Aktion. Im Terminal, im Editor und in CI gilt das nicht. Dort ändert der Mensch die Datei, und ein Pull Request zeigt den Diff. Ob ein Reviewer darin `profile: minimal` als Abschwächung erkennt, bleibt Disziplin.

Bei der Arbeit an ADR 0004 hat sich gezeigt, wie viel ein Feld bewirkt:

- `profile: minimal` schaltet SDD-011, SDD-050 und SDD-051 ab, obwohl die Baseline sie ausdrücklich nicht unterdrücken darf (`baseline: forbidden`).
- `gates.<gate>.override` ersetzt die Regeln eines Gates. SDD-072 verhindert nur, dass eine Regel aus allen Gates verschwindet. Aus `pre-merge` allein lässt sie sich herausnehmen.
- Ein Muster in `exclude` nimmt Pfade aus jeder Prüfung, auch künftige Dateien.

Für die Verfassung löst SDD-010 dieselbe Frage. Eine Änderung der Verfassung im Diff gegen die Merge-Basis braucht ein ADR im selben Diff, das die Verfassung nennt. Die Begründung steht dann unveränderlich neben der Änderung.

Eine Sperre in der Validierung der Konfiguration ist nicht möglich. Die Datei kennt ihren Vorgänger nicht, und `minimal` ist für ein neues Projekt ein legitimes Profil. Erkennbar ist eine Abschwächung nur im Vergleich mit einem früheren Stand, also über Git, wie bei SDD-010.

Erwogen wurden:

1. **Status quo.** Im Desktop schützt ADR 0004, außerhalb das Review des Pull Requests. Verworfen: Gerade die stärkste Abschwächung, ein Profilwechsel, ist im Diff eine einzige unscheinbare Zeile, und `sdd` selbst sagt nichts dazu.
2. **Jede Änderung an `sdd.config.yaml` braucht ein ADR.** Verworfen: Häufige, harmlose Änderungen wie ein neues Muster in `tests.include` oder eine weitere Schicht in `thresholds.layers` würden zu ADR-Pflichten. Die ADRs verlören ihr Gewicht.
3. **Eine Katalogregel für eindeutig abschwächende Änderungen** — vorgeschlagen, analog zu SDD-010.

ADR 0004 hat eine Erkennung der Richtung als Grundlage für die kritische Stufe verworfen, weil eine Lücke in der Erkennung die Bremse umgehen würde. Hier ist die Lage anders. Die Regel ist ein zusätzliches Netz für Terminal und CI und ersetzt nichts. Sie erfasst nur Felder, deren Richtung eindeutig ist, und benennt die übrigen als bekannte Lücke. Im Desktop bleibt jede Änderung kritisch.

## Entscheidung

1. **Neue Regel SDD-012** „Abschwächende Änderung der Konfiguration verweist auf ein ADR“. Schweregrad `error` in `pre-implementation` und `pre-merge`, Prüfvertrag im Handbuch. Sie vergleicht `sdd.config.yaml` an der Merge-Basis mit `HEAD`, wie SDD-010 die Verfassung.
2. **Als abschwächend gelten** (deterministisch, je Feld):
   - `profile` sinkt in der Ordnung `minimal` < `standard` < `regulated`;
   - `gates.<gate>.override` ändert sich in irgendeiner Weise (Override ersetzt die Regelmenge, die Richtung ist nicht verlässlich bestimmbar);
   - `exclude` erhält ein Muster, das vorher nicht da war;
   - `thresholds.layers` oder `thresholds.feature_artifacts_required_when.touched_paths` verlieren einen Eintrag, oder `min_layers_touched` steigt.
3. **Nicht erfasst** und als Lücke benannt: `tests.include`, `artifacts.*`, `base_ref`, `constitution_sections`. Bei ihnen ist die Richtung nicht eindeutig. Im Desktop deckt sie ADR 0004 ab, außerhalb das Review.
4. **Erfüllt** ist die Regel wie SDD-010: Ein ADR im selben Diff nennt `sdd.config.yaml` in `references`. Ein erstmaliges Anlegen der Konfiguration ist keine Abschwächung.
5. **Nicht abwählbar.** SDD-012 bekommt das Profil `minimal` und `baseline: forbidden`. Damit sie sich nicht über genau die Änderung abschalten lässt, die sie prüfen soll, kommt ein neues Katalogattribut `pinned: true` dazu. Eine so markierte Regel läuft in ihren `gate_default`-Gates unabhängig von `profile` und `gates.<gate>.override`. Vorerst trägt nur SDD-012 dieses Attribut.

## Konsequenzen

- **Katalog** `1.1.0` → `1.2.0`: neue Regel SDD-012, neues Attribut `pinned` mit Schema und Prüfung (SDD-072 berücksichtigt `pinned`).
- **Handbuch:** Abschnitt und Prüfvertrag für SDD-012. Dazu `pinned` in der Beschreibung des Katalogs.
- **Konzept §9.6:** Verweis auf SDD-012 neben ADR 0004.
- **Verfassung:** unverändert.
- **Projekte, die heute schon abschwächen:** Die Regel greift nur auf Änderungen im Diff gegen die Merge-Basis, nicht auf den Bestand. Ein Projekt, das schon auf `minimal` steht, bekommt keinen Befund.
- **`sdd_config` im Desktop:** Die Vorschau zeigt künftig auch, ob die Änderung SDD-012 auslöst. Das folgt ohne eigene Arbeit aus der Befund-Differenz, sobald die Regel im Katalog steht.
- **Risiko:** Ein ADR lässt sich schreiben, ohne dass es eine gute Begründung enthält. Die Regel erzwingt das Warum nicht inhaltlich, sie macht die Abschwächung aber sichtbar und unveränderlich begründet, wie bei der Verfassung.
- **Risiko:** `pinned` ist ein neues Konzept im Katalog. Es darf nicht zur bequemen Art werden, Regeln dem Projekt zu entziehen. Neue `pinned`-Regeln brauchen deshalb jeweils ein eigenes ADR.
