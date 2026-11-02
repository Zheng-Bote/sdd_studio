package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Zheng-Bote/sdd_studio/internal/mcp"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// runMCP serves the Model Context Protocol on stdin/stdout; diagnostics go to stderr.
func runMCP(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	allowWrite := fs.Bool("allow-write", false, "sdd_new, sdd_write, sdd_init und sdd_archive dürfen schreiben (sonst nur Vorschau)")
	allowCritical := fs.Bool("allow-critical", false, "zusätzlich die kritischen Werkzeuge sdd_baseline, sdd_adr_accept und sdd_adopt_apply anbieten (schließt --allow-write ein; nur für Clients, die jede Ausführung einzeln bestätigen lassen, ADR 0003)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	srv := mcp.New(repoRoot, *allowWrite || *allowCritical, report.ToolVersion, stderr)
	srv.AllowCritical = *allowCritical
	if err := srv.Serve(os.Stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "sdd mcp: %v\n", err)
		return 2
	}
	return 0
}
