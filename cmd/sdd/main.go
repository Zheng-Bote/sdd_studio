package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

func run(args []string, repoRoot string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "lint":
		return runLint(args[1:], repoRoot, stdout, stderr)
	case "gate":
		return runGate(args[1:], repoRoot, stdout, stderr)
	case "baseline":
		return runBaseline(args[1:], repoRoot, stdout, stderr)
	case "init":
		return runInit(args[1:], repoRoot, stdout, stderr)
	case "new":
		return runNew(args[1:], repoRoot, stdout, stderr)
	case "adopt":
		return runAdopt(args[1:], repoRoot, stdout, stderr)
	case "report":
		return runReport(args[1:], repoRoot, stdout, stderr)
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "sdd: unbekannter Befehl %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runLint(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "Befunde im JSON-Format ausgeben")
	gate := fs.String("gate", "", "Optionales Gate (pre-implementation, pre-merge, pre-release)")
	feature := fs.String("feature", "", "Feature-Slug(s), kommagetrennt (überschreibt die Feature-Auflösung)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *gate != "" && !isValidGate(*gate) {
		fmt.Fprintf(stderr, "sdd lint: ungültiges Gate %q (erlaubt: pre-implementation, pre-merge, pre-release)\n", *gate)
		return 2
	}

	rep, err := sddcore.RunCheck(sddcore.RunOptions{
		RepoRoot: repoRoot,
		Gate:     *gate,
		Feature:  *feature,
	})
	if err != nil {
		fmt.Fprintf(stderr, "sdd lint: %v\n", err)
		return 2
	}

	if *jsonOut {
		if err := rep.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "sdd lint: %v\n", err)
			return 2
		}
	} else {
		if err := rep.WriteText(stdout); err != nil {
			fmt.Fprintf(stderr, "sdd lint: %v\n", err)
			return 2
		}
	}
	return rep.ExitCode()
}

func runGate(args []string, repoRoot string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "sdd gate: Gate-Name erforderlich (pre-implementation | pre-merge | pre-release)")
		return 2
	}
	gateName := args[0]
	if !isValidGate(gateName) {
		fmt.Fprintf(stderr, "sdd gate: ungültiges Gate %q (erlaubt: pre-implementation, pre-merge, pre-release)\n", gateName)
		return 2
	}

	fs := flag.NewFlagSet("gate "+gateName, flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "Befunde im JSON-Format ausgeben")
	feature := fs.String("feature", "", "Feature-Slug(s), kommagetrennt (überschreibt die Feature-Auflösung)")
	pathsRaw := fs.String("paths", "", "Erklärte Änderungsabsicht als kommagetrennte Pfad-Globs")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	paths := splitList(*pathsRaw)

	rep, err := sddcore.RunCheck(sddcore.RunOptions{
		RepoRoot: repoRoot,
		Gate:     gateName,
		Feature:  *feature,
		Paths:    paths,
	})
	if err != nil {
		fmt.Fprintf(stderr, "sdd gate: %v\n", err)
		return 2
	}

	if *jsonOut {
		if err := rep.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "sdd gate: %v\n", err)
			return 2
		}
	} else {
		if err := rep.WriteText(stdout); err != nil {
			fmt.Fprintf(stderr, "sdd gate: %v\n", err)
			return 2
		}
	}
	return rep.ExitCode()
}

func runBaseline(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	gate := fs.String("gate", "", "Nur für ein bestimmtes Gate aufnehmen")
	prune := fs.Bool("prune", false, "Erledigte Einträge aus der Baseline entfernen")
	acceptRule := fs.String("accept", "", "Neue Befunde einer bestimmten Regel in die Baseline aufnehmen")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *gate != "" && !isValidGate(*gate) {
		fmt.Fprintf(stderr, "sdd baseline: ungültiges Gate %q\n", *gate)
		return 2
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "sdd baseline: %v\n", err)
		return 2
	}
	cat, _, err := catalog.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "sdd baseline: %v\n", err)
		return 2
	}

	if *acceptRule != "" {
		var found *catalog.Rule
		for i := range cat.Rules {
			if cat.Rules[i].ID == *acceptRule {
				found = &cat.Rules[i]
				break
			}
		}
		if found == nil {
			fmt.Fprintf(stderr, "sdd baseline: unbekannte Regel %q\n", *acceptRule)
			return 2
		}
		if found.Baseline == "forbidden" {
			fmt.Fprintf(stderr, "sdd baseline: Regel %s trägt baseline: forbidden und darf nicht unterdrückt werden\n", *acceptRule)
			return 2
		}
	}

	gatesToRun := catalog.AllGates
	if *gate != "" {
		gatesToRun = []string{*gate}
	}

	currentEntries, err := sddcore.BaselineEntries(repoRoot, cfg, gatesToRun)
	if err != nil {
		fmt.Fprintf(stderr, "sdd baseline: %v\n", err)
		return 2
	}
	if *acceptRule != "" {
		var filtered []baseline.Entry
		for _, e := range currentEntries {
			if e.RuleID == *acceptRule {
				filtered = append(filtered, e)
			}
		}
		currentEntries = filtered
	}

	existing, hasExisting, err := baseline.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "sdd baseline: %v\n", err)
		return 2
	}

	var resultEntries []baseline.Entry
	switch {
	case *prune:
		if !hasExisting {
			resultEntries = []baseline.Entry{}
		} else {
			activeMap := make(map[string]bool)
			for _, e := range currentEntries {
				key := fmt.Sprintf("%s|%s|%s|%s|%d", e.Gate, e.RuleID, e.Path, e.ContentHash, e.Ordinal)
				activeMap[key] = true
			}
			for _, e := range existing.Entries {
				if *gate != "" && e.Gate != *gate {
					resultEntries = append(resultEntries, e)
					continue
				}
				key := fmt.Sprintf("%s|%s|%s|%s|%d", e.Gate, e.RuleID, e.Path, e.ContentHash, e.Ordinal)
				if activeMap[key] {
					resultEntries = append(resultEntries, e)
				}
			}
		}
	case *acceptRule != "" && hasExisting:
		resultEntries = baseline.MergeEntries(existing.Entries, currentEntries)
	case *gate != "" && hasExisting:
		var kept []baseline.Entry
		for _, e := range existing.Entries {
			if e.Gate != *gate {
				kept = append(kept, e)
			}
		}
		resultEntries = baseline.MergeEntries(kept, currentEntries)
	default:
		resultEntries = baseline.MergeEntries(nil, currentEntries)
	}

	bf := &baseline.File{
		Version: 1,
		Entries: resultEntries,
	}
	if err := baseline.Save(repoRoot, bf); err != nil {
		fmt.Fprintf(stderr, "sdd baseline: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Baseline geschrieben (%s): %d Einträge\n", baseline.DefaultBaselineRelPath, len(bf.Entries))
	return 0
}

func isValidGate(g string) bool {
	for _, valid := range catalog.AllGates {
		if g == valid {
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Verwendung: sdd <Befehl> [Optionen]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Befehle:")
	fmt.Fprintln(w, "  lint       Prüft Verfassung, Katalog und Artefakte (--json, --gate, --feature)")
	fmt.Fprintln(w, "  gate       Führt ein Gate aus: pre-implementation | pre-merge | pre-release")
	fmt.Fprintln(w, "  baseline   Verwaltet .sdd/baseline.json (--gate, --prune, --accept)")
	fmt.Fprintln(w, "  init       Legt ein neues Projekt an (--profile, --dry-run, --no-ci)")
	fmt.Fprintln(w, "  report     Reifegrad, Befunde, Gates und Schuldenstand mit Trend (--format markdown|json, --since <ref>)")
	fmt.Fprintln(w, "  adopt      Übernimmt ein bestehendes Projekt: Bericht, mit --apply auf neuem Branch (--profile, --branch)")
	fmt.Fprintln(w, "  new        Legt Artefakte aus Templates an (--feature, --type, --title, --touches, --archive-after, --dry-run)")
}
