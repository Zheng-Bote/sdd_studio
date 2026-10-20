package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

func runInit(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "standard", "Profil: minimal, standard oder regulated")
	dryRun := fs.Bool("dry-run", false, "Nur anzeigen, was angelegt würde")
	noCI := fs.Bool("no-ci", false, "Keinen GitHub-Actions-Workflow anlegen")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	project := "Projekt"
	if abs, err := filepath.Abs(repoRoot); err == nil {
		project = filepath.Base(abs)
	}
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: *profile, Project: project, CI: !*noCI})
	if err != nil {
		fmt.Fprintf(stderr, "sdd init: %v\n", err)
		return 2
	}
	if code := applyPlan("sdd init", plan, repoRoot, *dryRun, stdout, stderr); code != 0 {
		return code
	}
	if !*dryRun {
		fmt.Fprintln(stdout, "Nächster Schritt: thresholds.layers und thresholds.feature_artifacts_required_when.touched_paths in sdd.config.yaml an das Projekt anpassen — solange beide leer sind, verlangt SDD-020/030 nie ein PRD.")
	}
	return 0
}

func runNew(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	feature := fs.String("feature", "", "Feature-Slug für prd, design und plan")
	types := fs.String("type", "", "Artefakttypen, kommagetrennt: prd, design, plan, adr, doc (Standard mit --feature: prd,design)")
	title := fs.String("title", "", "Titel des Artefakts")
	slug := fs.String("slug", "", "Slug für adr und doc (Standard: aus dem Titel abgeleitet)")
	touches := fs.String("touches", "", "Erklärte Änderungsabsicht als kommagetrennte Pfad-Globs")
	archiveAfter := fs.String("archive-after", "", "Archivierungsstichtag YYYY-MM-DD")
	dryRun := fs.Bool("dry-run", false, "Nur anzeigen, was angelegt würde")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(repoRoot)
	if errors.Is(err, config.ErrConfigNotFound) {
		fmt.Fprintln(stderr, "sdd new: keine sdd.config.yaml gefunden; verwende die Standardpfade von `sdd init`")
		cfg, err = scaffold.DefaultConfig("standard"), nil
	}
	if err != nil {
		fmt.Fprintf(stderr, "sdd new: %v\n", err)
		return 2
	}

	plan, err := scaffold.NewPlan(repoRoot, cfg, scaffold.NewOptions{
		Types:        splitList(*types),
		Feature:      strings.TrimSpace(*feature),
		Title:        *title,
		Slug:         strings.TrimSpace(*slug),
		ArchiveAfter: strings.TrimSpace(*archiveAfter),
		Touches:      splitList(*touches),
		Today:        time.Now(),
	})
	if err != nil {
		fmt.Fprintf(stderr, "sdd new: %v\n", err)
		return 2
	}
	return applyPlan("sdd new", plan, repoRoot, *dryRun, stdout, stderr)
}

// applyPlan previews or applies a scaffold plan and prints one line per operation.
func applyPlan(cmd string, plan scaffold.Plan, repoRoot string, dryRun bool, stdout, stderr io.Writer) int {
	var results []scaffold.Result
	var err error
	if dryRun {
		results, err = plan.Preview(repoRoot)
	} else {
		results, err = plan.Apply(repoRoot)
	}
	for _, r := range results {
		switch {
		case r.Status == scaffold.StatusSkipped:
			fmt.Fprintf(stdout, "übersprungen  %s (%s)\n", r.Path, r.Reason)
		case dryRun:
			fmt.Fprintf(stdout, "würde anlegen %s\n", r.Path)
		default:
			fmt.Fprintf(stdout, "angelegt      %s\n", r.Path)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", cmd, err)
		return 2
	}
	return 0
}

func splitList(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
