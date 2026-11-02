package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

func runArchive(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	fs.SetOutput(stderr)
	feature := fs.String("feature", "", "Feature-Slug")
	release := fs.String("release", "", "Version, die als release: ins Frontmatter kommt (optional)")
	dryRun := fs.Bool("dry-run", false, "Nur anzeigen, was verschoben würde")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	plan, err := guide.Archive(repoRoot, strings.TrimSpace(*feature), *release)
	if err != nil {
		fmt.Fprintf(stderr, "sdd archive: %v\n", err)
		return 2
	}
	verb := "verschoben     "
	if *dryRun {
		verb = "würde verschieben"
	} else if _, err := plan.Apply(repoRoot); err != nil {
		fmt.Fprintf(stderr, "sdd archive: %v\n", err)
		return 2
	}
	for _, op := range plan.Ops {
		fmt.Fprintf(stdout, "%s %s -> %s\n", verb, op.MoveFrom, op.Path)
		for _, line := range changedLines(op.Expect, op.Content) {
			fmt.Fprintf(stdout, "    %s\n", line)
		}
	}
	return 0
}

// changedLines are the added lines of the archived frontmatter, e.g. "+status: archived".
func changedLines(old, new []byte) []string {
	var out []string
	for _, l := range strings.Split(scaffold.UnifiedDiff("x", old, new), "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			out = append(out, l)
		}
	}
	return out
}
