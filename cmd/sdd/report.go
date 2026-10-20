package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/overview"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

func runReport(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "markdown", "Ausgabeformat: markdown oder json")
	since := fs.String("since", "", "Vergleichsstand für den Schuldenstand (Standard: Merge-Basis)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "markdown" && *format != "json" {
		fmt.Fprintf(stderr, "sdd report: ungültiges Format %q (erlaubt: markdown, json)\n", *format)
		return 2
	}

	o, err := overview.Build(repoRoot, *since, scan.CLIGit{Root: repoRoot})
	if err != nil {
		fmt.Fprintf(stderr, "sdd report: %v\n", err)
		return 2
	}
	if *format == "json" {
		err = overview.RenderJSON(stdout, o)
	} else {
		err = overview.RenderMarkdown(stdout, o)
	}
	if err != nil {
		fmt.Fprintf(stderr, "sdd report: %v\n", err)
		return 2
	}
	return 0
}
