package adopt

import (
	"fmt"
	"sort"
	"strings"
)

// RenderReport renders the take-over report as Markdown. res is nil for the dry run.
func RenderReport(p *Proposal, res *Result) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	inv := p.Inventory

	w("---")
	w("status: active")
	w("---")
	w("")
	w("# Adopt-Bericht — %s", p.Options.Project)
	w("")
	if res == nil {
		w("**Probelauf** vom %s, Profil `%s`. Es wurde nichts geschrieben; `sdd adopt --apply` legt Branch `%s` an.", p.Options.Today.Format("2006-01-02"), inv.Config.Profile, p.Options.Branch)
	} else {
		w("Angewendet am %s auf Branch `%s` (ausgehend von `%s`), Profil `%s`.", p.Options.Today.Format("2006-01-02"), res.Branch, res.FromBranch, inv.Config.Profile)
	}
	w("")

	w("## Reifegrad: L%d", p.Maturity.Level)
	w("")
	w("| Stufe | Kriterium | Erfüllt | Befund |")
	w("|---|---|---|---|")
	for _, c := range p.Maturity.Criteria {
		mark := "nein"
		if c.Met {
			mark = "ja"
		}
		w("| L%d | %s | %s | %s |", c.Level, c.Name, mark, c.Detail)
	}
	w("")
	if p.Maturity.Next != nil {
		w("**Nächster Schritt (L%d):** %s.", p.Maturity.Next.Level, p.Maturity.Next.Advice)
	} else {
		w("**Nächster Schritt:** keiner — alle Kriterien bis L4 sind erfüllt.")
	}
	w("")
	w("Der Reifegrad beschreibt den Stand vor der Übernahme.")
	w("")

	w("## Inventar")
	w("")
	w("- Artefaktpfade: %s", inv.Layout)
	counts := map[Kind]int{}
	for _, f := range inv.Files {
		counts[f.Kind]++
	}
	for _, k := range []Kind{KindConstitution, KindADR, KindPRD, KindDesign, KindPlan, KindDoc} {
		w("- %s: %d", k, counts[k])
	}
	w("- CI-Workflows: %d (ruft `sdd gate` auf: %s)", len(inv.Workflows), yesNo(inv.CallsGate))
	w("")
	w("| Datei | Typ | Grund |")
	w("|---|---|---|")
	for _, f := range inv.Files {
		kind := string(f.Kind)
		if f.Archived {
			kind += " (Archiv)"
		}
		w("| `%s` | %s | %s |", f.Path, kind, f.Reason)
	}
	w("")
	w("Die Zuordnung ist ein Vorschlag nach festen Mustern; „kein Muster“ heißt Doku/Exploration, nicht „geraten“.")
	w("")

	w("## Schritte")
	w("")
	if res == nil {
		n := 0
		for _, s := range p.Steps {
			n++
			w("%d. **%s** — ein Commit", n, s.Title)
			for _, d := range s.Detail {
				w("   - %s", d)
			}
		}
		if !inv.BaselineExists {
			n++
			w("%d. **Baseline** — `.sdd/baseline.json` für alle Gates, nach den vorherigen Schritten und vor dem ersten Gate-Lauf", n)
		}
		w("%d. **Bericht** — `%s`", n+1, ReportPath)
	} else {
		for i, c := range res.Commits {
			w("%d. **%s** — Commit `%s`: %s", i+1, c.Step, c.Hash, strings.Join(c.Paths, ", "))
		}
		w("%d. **Bericht** — diese Datei", len(res.Commits)+1)
		if !inv.BaselineExists {
			w("")
			w("Baseline: %d Einträge.", res.BaselineEntries)
		}
		if len(res.Skipped) > 0 {
			w("")
			w("Übersprungen:")
			for _, s := range res.Skipped {
				w("- `%s` (%s)", s.Path, s.Reason)
			}
		}
	}
	w("")
	if inv.BaselineExists {
		w("Eine Baseline ist bereits vorhanden und wird nicht verändert.")
		w("")
	}

	w("## Konflikte")
	w("")
	if len(p.Conflicts) == 0 {
		w("Keine.")
	} else {
		for _, c := range p.Conflicts {
			w("- %s", c)
		}
	}
	w("")

	w("## Archivierung vor dem ersten Release")
	w("")
	if len(p.Archive) == 0 {
		w("Kein Feature-Artefakt ist abgeschlossen oder abgelaufen.")
	} else {
		w("SDD-050 ist nicht baselinefähig. Vor dem ersten Release nach der Übernahme müssen %d Dateien nach `%s` verschoben werden und `status: archived` tragen. sdd adopt führt das nicht aus.", len(p.Archive), inv.Config.Artifacts.Archive)
		w("")
		paths := make([]string, 0, len(p.Archive))
		for _, f := range p.Archive {
			paths = append(paths, f.Path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			w("- `%s`", path)
		}
	}
	w("")

	w("## Hinweise")
	w("")
	w("- Ergänzte Feature-Artefakte tragen `status: active`. Längst umgesetzte Features auf `status: implemented` setzen — dann weist sie SDD-050 für die Archivierung aus.")
	if len(inv.Config.Thresholds.Layers) == 0 && len(inv.Config.Thresholds.FeatureArtifactsRequiredWhen.TouchedPaths) == 0 {
		w("- Die Größenschwelle ist leer: SDD-020/030 verlangen erst dann PRD und Tech Spec, wenn `thresholds.layers` oder `touched_paths` gesetzt sind.")
	}
	return b.String()
}

func yesNo(b bool) string {
	if b {
		return "ja"
	}
	return "nein"
}
