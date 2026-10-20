package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
)

var stateMarks = map[string]string{
	guide.StateDone:      "✓",
	guide.StateOpen:      "✗",
	guide.StateNotNeeded: "–",
	guide.StateManual:    "?",
	guide.StateLater:     "·",
}

func runNext(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("next", flag.ContinueOnError)
	fs.SetOutput(stderr)
	feature := fs.String("feature", "", "Feature-Slug (Standard: alle aktiven Features)")
	jsonOut := fs.Bool("json", false, "Ausgabe als JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	g, err := guide.Next(repoRoot, *feature)
	if err != nil {
		fmt.Fprintf(stderr, "sdd next: %v\n", err)
		return 2
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(g); err != nil {
			fmt.Fprintf(stderr, "sdd next: %v\n", err)
			return 2
		}
		return 0
	}
	if g.Start != nil {
		fmt.Fprintf(stdout, "Nächster Schritt: %s\n  %s\n  → %s\n", g.Start.Title, g.Start.Detail, g.Start.Command)
		return 0
	}
	for i, f := range g.Features {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "Feature %s\n", f.Feature)
		if *feature != "" {
			for _, s := range f.Steps {
				fmt.Fprintf(stdout, "  %s %-34s %s\n", stateMarks[s.State], s.Title, s.Detail)
			}
		}
		if f.Next == nil {
			fmt.Fprintln(stdout, "  Nächster Schritt: keiner offen")
			continue
		}
		fmt.Fprintf(stdout, "  Nächster Schritt: %s — %s\n", f.Next.Title, f.Next.Detail)
		if f.Next.Command != "" {
			fmt.Fprintf(stdout, "  → %s\n", f.Next.Command)
		}
	}
	return 0
}
