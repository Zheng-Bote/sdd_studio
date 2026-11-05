package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

var arc42Anchors = []string{
	"{#section-introduction-and-goals}", "{#section-architecture-constraints}", "{#section-context-and-scope}",
	"{#section-solution-strategy}", "{#section-building-block-view}", "{#section-runtime-view}",
	"{#section-deployment-view}", "{#section-concepts}", "{#section-design-decisions}",
	"{#section-quality-scenarios}", "{#section-technical-risks}", "{#section-glossary}",
}

// arc42-docs AC-1: the template has the twelve arc42 sections with the original anchors, names authors, source,
// version, license and the change; NOTICE names the file with its license.
func TestArc42Template(t *testing.T) {
	data, err := os.ReadFile("../../../templates/arc42.md.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, a := range arc42Anchors {
		if !strings.Contains(text, "# ") || !strings.Contains(text, a) {
			t.Errorf("template lacks section %s", a)
		}
	}
	for _, want := range []string{"Hruschka", "Starke", "https://arc42.org", "9.1", "CC BY-SA 4.0", "creativecommons.org/licenses/by-sa/4.0", "deutsche Übersetzung"} {
		if !strings.Contains(text, want) {
			t.Errorf("template header lacks %q", want)
		}
	}
	notice, err := os.ReadFile("../../../NOTICE")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notice), "templates/arc42.md.tmpl") || !strings.Contains(string(notice), "Attribution-ShareAlike 4.0") {
		t.Errorf("NOTICE does not name the arc42 template and its license")
	}
}

// arc42-docs AC-2: sdd new --type arc42 creates artifacts.architecture (default docs/arc42.md) without a title, with
// status, empty release and the attribution line; it never overwrites, and a configured path is used.
func TestNewArc42(t *testing.T) {
	root := initProject(t, "standard")
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Types: []string{scaffold.TypeArc42}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ops) != 1 || plan.Ops[0].Path != "docs/arc42.md" {
		t.Fatalf("plan: %+v", plan.Ops)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "docs/arc42.md"))
	text := string(data)
	for _, want := range []string{"---\nstatus: active\nrelease:\n---\n", "# Architekturdokumentation — " + filepath.Base(root), "CC BY-SA 4.0", "{#section-glossary}"} {
		if !strings.Contains(text, want) {
			t.Errorf("docs/arc42.md lacks %q", want)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "docs/arc42.md"), []byte("eigener Inhalt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Types: []string{scaffold.TypeArc42}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := again.Apply(root); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "docs/arc42.md")); string(data) != "eigener Inhalt\n" {
		t.Error("existing architecture documentation overwritten")
	}
	cfg.Artifacts.Architecture = "architektur/arc42.md"
	custom, err := scaffold.NewPlan(root, cfg, scaffold.NewOptions{Types: []string{scaffold.TypeArc42}})
	if err != nil || custom.Ops[0].Path != "architektur/arc42.md" {
		t.Errorf("configured path: %+v %v", custom.Ops, err)
	}
}

// arc42-docs AC-6: handbook, README and CHANGELOG describe template, command, step and guided flow.
func TestArc42Documented(t *testing.T) {
	for f, wants := range map[string][]string{
		"../../../handbuch/03-mechanismen.md": {"sdd new --type arc42", "artifacts.architecture", "Architekturdoku nachführen", "CC BY-SA 4.0"},
		"../../../handbuch/04-agenten.md":     {"sdd-arc42"},
		"../../../README.md":                  {"new --type arc42", "docs/arc42.md"},
		"../../../CHANGELOG.md":               {"sdd new --type arc42", "sdd-arc42", "artifacts.architecture"},
	} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range wants {
			if !strings.Contains(string(data), w) {
				t.Errorf("%s lacks %q", f, w)
			}
		}
	}
}

// arc42-docs AC-7: context, building block, runtime and deployment view each hold a Mermaid skeleton of the expected
// diagram type; the prompt sdd-arc42 is tested in internal/mcp.
func TestArc42Mermaid(t *testing.T) {
	data, err := os.ReadFile("../../../templates/arc42.md.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for anchor, kind := range map[string]string{
		"{#_business_context}":        "flowchart",
		"{#_whitebox_overall_system}": "flowchart",
		"{#section-runtime-view}":     "sequenceDiagram",
		"{#_infrastructure_level_1}":  "flowchart",
	} {
		i := strings.Index(text, anchor)
		if i < 0 {
			t.Fatalf("anchor %s missing", anchor)
		}
		rest := text[i:]
		next := strings.Index(rest[len(anchor):], "\n#")
		if next < 0 {
			next = len(rest) - len(anchor)
		}
		section := rest[:len(anchor)+next]
		if !strings.Contains(section, "```mermaid\n"+kind) {
			t.Errorf("section %s lacks a %s skeleton", anchor, kind)
		}
	}
}
