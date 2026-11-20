package rules

import (
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// bmadContext builds a context for the macro rules; pipeline selects whether they apply.
func bmadContext(t *testing.T, pipeline string, files map[string]string) *Context {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		writeFile(t, root, rel, content)
	}
	cfg := featureConfig()
	cfg.Pipeline = pipeline
	cfg.Artifacts.Initiative = "docs/initiative.md"
	return &Context{RepoRoot: root, Config: cfg}
}

func runBMAD(t *testing.T, ctx *Context, fn CheckFunc) []report.Finding {
	t.Helper()
	findings, err := fn(ctx, catalog.Rule{ID: "SDD-08x"})
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

const initiativeBody = "---\ntype: initiative\nunplanned: [\"R3\"]\n---\n\n# I\n\n## Requirements\n\n" +
	"- **R1** eins\n- **R2** zwei\n- **R3** drei\n"

func epicBody(feature string, covers string) string {
	return "---\nfeature: " + feature + "\nstatus: active\ntype: epic\ncovers: " + covers + "\n---\n\n# Epic\n"
}

// bmad-import AC-8: SDD-080 reports an epic that covers no existing requirement, and stays quiet when it covers one.
func TestSDD080(t *testing.T) {
	ctx := bmadContext(t, config.PipelineBMAD, map[string]string{
		"docs/initiative.md":        initiativeBody,
		"docs/prd/epic-empty.md":    epicBody("epic-empty", "[]"),
		"docs/prd/epic-unknown.md":  epicBody("epic-unknown", "[\"R9\"]"),
		"docs/prd/epic-covered.md":  epicBody("epic-covered", "[\"R1\"]"),
		"docs/prd/plain-feature.md": "---\nfeature: plain\nstatus: active\n---\n# PRD\n",
	})
	findings := runBMAD(t, ctx, checkSDD080)
	if len(findings) != 2 {
		t.Fatalf("want 2 findings, got %d: %+v", len(findings), findings)
	}
	byPath := map[string]string{}
	for _, f := range findings {
		byPath[f.Path] = f.Message
	}
	if msg := byPath["docs/prd/epic-empty.md"]; !strings.Contains(msg, "keine Anforderung") {
		t.Errorf("empty covers: %q", msg)
	}
	if msg := byPath["docs/prd/epic-unknown.md"]; !strings.Contains(msg, "R9") {
		t.Errorf("unknown covers: %q", msg)
	}
	if _, ok := byPath["docs/prd/epic-covered.md"]; ok {
		t.Error("a covered epic must not be reported")
	}
	if _, ok := byPath["docs/prd/plain-feature.md"]; ok {
		t.Error("a plain PRD is not an epic")
	}
}

// bmad-import AC-8: SDD-081 reports an uncovered requirement unless the initiative marks it as unplanned.
func TestSDD081(t *testing.T) {
	ctx := bmadContext(t, config.PipelineBMAD, map[string]string{
		"docs/initiative.md":       initiativeBody,
		"docs/prd/epic-covered.md": epicBody("epic-covered", "[\"R1\", \"R2\"]"),
	})
	findings := runBMAD(t, ctx, checkSDD081)
	if len(findings) != 0 {
		t.Fatalf("R1 and R2 are covered, R3 is unplanned: %+v", findings)
	}

	// Without the unplanned entry R3 is reported, in the requirement's own line.
	ctx = bmadContext(t, config.PipelineBMAD, map[string]string{
		"docs/initiative.md":       strings.Replace(initiativeBody, "unplanned: [\"R3\"]\n", "", 1),
		"docs/prd/epic-covered.md": epicBody("epic-covered", "[\"R1\", \"R2\"]"),
	})
	findings = runBMAD(t, ctx, checkSDD081)
	if len(findings) != 1 || !strings.Contains(findings[0].Message, "R3") {
		t.Fatalf("want one finding for R3: %+v", findings)
	}
	if findings[0].Path != "docs/initiative.md" || findings[0].Line != 11 {
		t.Errorf("finding at %s:%d, want docs/initiative.md:11", findings[0].Path, findings[0].Line)
	}

	// Before the first epic nothing is reported: every requirement would look uncovered.
	ctx = bmadContext(t, config.PipelineBMAD, map[string]string{"docs/initiative.md": initiativeBody})
	if findings := runBMAD(t, ctx, checkSDD081); len(findings) != 0 {
		t.Errorf("no epic yet: %+v", findings)
	}
}

// bmad-import AC-8: SDD-082 reports an index entry without file and an epic file without index entry.
func TestSDD082(t *testing.T) {
	ctx := bmadContext(t, config.PipelineBMAD, map[string]string{
		"_bmad-output/x/tickets/tickets.toml": "[[epic]]\nslug = \"epic-a\"\n\n[[epic]]\nslug = \"epic-b\"\n",
		"docs/initiative.md":                  initiativeBody,
		"docs/prd/epic-a.md":                  epicBody("epic-a", "[\"R1\"]"),
		"docs/prd/epic-c.md":                  epicBody("epic-c", "[\"R2\"]"),
	})
	findings := runBMAD(t, ctx, checkSDD082)
	if len(findings) != 2 {
		t.Fatalf("want 2 findings, got %d: %+v", len(findings), findings)
	}
	msgs := []string{findings[0].Message, findings[1].Message}
	joined := strings.Join(msgs, " | ")
	if !strings.Contains(joined, "epic-b") || !strings.Contains(joined, "epic-c") {
		t.Errorf("messages: %s", joined)
	}

	// Before the first epic the index is still BMAD's own list: nothing to compare, no finding.
	ctx = bmadContext(t, config.PipelineBMAD, map[string]string{
		"_bmad-output/x/tickets/tickets.toml": "[[epic]]\nslug = \"epic-a\"\n",
		"docs/initiative.md":                  initiativeBody,
	})
	if findings := runBMAD(t, ctx, checkSDD082); len(findings) != 0 {
		t.Errorf("no epic taken over yet: %+v", findings)
	}
}

// bmad-import AC-8: without `pipeline: bmad` none of the macro rules reports anything.
func TestBMADRulesInactiveWithoutPipeline(t *testing.T) {
	ctx := bmadContext(t, "", map[string]string{
		"_bmad-output/x/tickets/tickets.toml": "[[epic]]\nslug = \"epic-b\"\n",
		"docs/initiative.md":                  initiativeBody,
		"docs/prd/epic-empty.md":              epicBody("epic-empty", "[]"),
	})
	for name, fn := range map[string]CheckFunc{
		"SDD-080": checkSDD080,
		"SDD-081": checkSDD081,
		"SDD-082": checkSDD082,
	} {
		if findings := runBMAD(t, ctx, fn); len(findings) != 0 {
			t.Errorf("%s reported without pipeline bmad: %+v", name, findings)
		}
	}
}
