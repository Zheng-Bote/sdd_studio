---
feature: next-draft-hint
status: active
archive-after:
touches:
  - internal/sddcore/guide/**
  - handbuch/**
  - CHANGELOG.md
---

# PRD — `sdd next` nennt ein PRD im Entwurf vor dem Archivieren

## Problem

Das PRD-Template beginnt mit `status: draft`. Seit v0.6.0 lehnt `sdd archive` ein Feature ab, dessen PRD noch ein Entwurf ist. `sdd next` sagt das nicht: Der Schritt „Abschließen und archivieren“ sieht bei einem Entwurf genauso aus wie bei einem fertigen PRD. Wer den Status vergisst, erfährt es erst beim Archivieren.

## Ziel

`sdd next` nennt im Abschlussschritt das PRD, das noch `status: draft` trägt, und was zu tun ist.

## Nutzer

- **Alle, die mit `sdd next` durch den Prozess gehen**, im Terminal und im Desktop.

## Umfang

- Text des Abschlussschritts in `sdd next`.
- Handbuch und CHANGELOG.

## Nicht-Ziele

- Änderungen an `sdd archive` oder an den Statuswerten.
- Eine Regel im Katalog oder ein neuer offener Schritt: Während der Arbeit ist `draft` zulässig.

## Akzeptanzkriterien

- **AC-1** Trägt ein PRD des Features `status: draft`, nennt die Erläuterung des Abschlussschritts dessen Pfad, dass `sdd archive` Entwürfe ablehnt und dass der Status zu setzen ist. Der Zustand des Schritts und der nächste Schritt ändern sich dadurch nicht.
- **AC-2** Ohne PRD im Entwurf bleibt die Erläuterung unverändert.
- **AC-3** Das Handbuch beschreibt den Hinweis.
