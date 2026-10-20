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
	allowWrite := fs.Bool("allow-write", false, "sdd_new darf Dateien anlegen (sonst nur Vorschau)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	srv := mcp.New(repoRoot, *allowWrite, report.ToolVersion, stderr)
	if err := srv.Serve(os.Stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "sdd mcp: %v\n", err)
		return 2
	}
	return 0
}
