package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
)

func runExplain(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "Ausgabe als JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "sdd explain: Regel-ID erforderlich, z. B. sdd explain SDD-022")
		return 2
	}
	e, err := guide.Explain(repoRoot, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "sdd explain: %v\n", err)
		return 2
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(e)
		return 0
	}
	fmt.Fprintln(stdout, e.Section)
	if e.Contract != "" && !strings.Contains(e.Section, e.Contract) {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, e.Contract)
	}
	return 0
}
