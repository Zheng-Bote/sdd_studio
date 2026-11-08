package rules

import (
	"os"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

const baseConfig = `version: 1
profile: standard
artifacts:
  constitution: constitution.md
  adr: docs/adr
  prd: docs/prd
  design: docs/design
tests:
  include: ["**/*_test.go"]
thresholds:
  layers: [internal/api/**, internal/store/**]
  feature_artifacts_required_when:
    min_layers_touched: 2
    touched_paths: [cmd/**]
exclude: [vendor/**]
`

// runSDD012 runs SDD-012 with baseConfig at the merge base and current in the working tree.
func runSDD012(t *testing.T, gate, current string, atBase bool, diff []string, adr string) []string {
	t.Helper()
	if _, err := config.Parse([]byte(current)); err != nil {
		t.Fatalf("test setup: current configuration invalid: %v", err)
	}
	root := t.TempDir()
	writeFile(t, root, "sdd.config.yaml", current)
	if adr != "" {
		writeFile(t, root, "docs/adr/0002-x.md", adr)
	}
	files := map[string]string{}
	if atBase {
		files["main:sdd.config.yaml"] = baseConfig
	}
	cfg := featureConfig()
	cfg.Artifacts.ADR = "docs/adr"
	parsed, _ := config.Parse([]byte(current))
	cfg.Exclude = parsed.Exclude // the gate runs with the configuration under review
	ctx := &Context{RepoRoot: root, Config: cfg, Gate: gate, Git: fakeGit{changed: diff, files: files}}
	findings, err := checkSDD012(ctx, catalog.Rule{ID: "SDD-012"})
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, f := range findings {
		if f.Path != "sdd.config.yaml" {
			t.Errorf("finding at %s", f.Path)
		}
		msgs = append(msgs, f.Message)
	}
	return msgs
}

var cfgDiff = []string{"sdd.config.yaml"}

// config-weakening AC-3: every weakening with a clear direction is reported, each named in the message.
func TestSDD012Weakenings(t *testing.T) {
	cases := map[string]struct{ change, want string }{
		"profile lowered": {strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), "profile standard → minimal"},
		"override set":    {baseConfig + "gates:\n  pre_merge:\n    override: [SDD-001]\n", "gates.pre_merge.override geändert"},
		"exclude added":   {strings.Replace(baseConfig, "exclude: [vendor/**]", "exclude: [vendor/**, internal/**]", 1), "exclude: +internal/**"},
		"layer removed":   {strings.Replace(baseConfig, "layers: [internal/api/**, internal/store/**]", "layers: [internal/api/**]", 1), "thresholds.layers: −internal/store/**"},
		"path removed":    {strings.Replace(baseConfig, "touched_paths: [cmd/**]", "touched_paths: []", 1), "touched_paths: −cmd/**"},
		"min layers up":   {strings.Replace(baseConfig, "min_layers_touched: 2", "min_layers_touched: 3", 1), "min_layers_touched 2 → 3"},
	}
	for name, tc := range cases {
		if tc.change == baseConfig {
			t.Fatalf("%s: test setup did not change the configuration", name)
		}
		msgs := runSDD012(t, "pre-merge", tc.change, true, cfgDiff, "")
		if len(msgs) != 1 || !strings.Contains(msgs[0], tc.want) {
			t.Errorf("%s: %v, want %q", name, msgs, tc.want)
		}
	}
	both := strings.Replace(strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), "exclude: [vendor/**]", "exclude: [vendor/**, docs/**]", 1)
	if msgs := runSDD012(t, "pre-implementation", both, true, cfgDiff, ""); len(msgs) != 1 || !strings.Contains(msgs[0], "profile") || !strings.Contains(msgs[0], "exclude: +docs/**") {
		t.Errorf("two weakenings in one finding: %v", msgs)
	}
}

// config-weakening AC-4: an ADR in the same diff naming sdd.config.yaml satisfies the rule; an ADR outside the diff or
// naming something else does not.
func TestSDD012NeedsADR(t *testing.T) {
	weak := strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1)
	adr := "---\nstatus: accepted\nreferences:\n  - sdd.config.yaml\n---\n"
	if msgs := runSDD012(t, "pre-merge", weak, true, []string{"sdd.config.yaml", "docs/adr/0002-x.md"}, adr); len(msgs) != 0 {
		t.Errorf("ADR in diff naming the file: %v", msgs)
	}
	if msgs := runSDD012(t, "pre-merge", weak, true, []string{"sdd.config.yaml", "docs/adr/0002-x.md"}, "---\nadr: ../../sdd.config.yaml\n---\n"); len(msgs) != 0 {
		t.Errorf("ADR naming the file relatively via adr: %v", msgs)
	}
	if msgs := runSDD012(t, "pre-merge", weak, true, cfgDiff, adr); len(msgs) != 1 {
		t.Errorf("ADR not in the diff must not count: %v", msgs)
	}
	if msgs := runSDD012(t, "pre-merge", weak, true, []string{"sdd.config.yaml", "docs/adr/0002-x.md"}, "---\nreferences: [constitution.md]\n---\n"); len(msgs) != 1 {
		t.Errorf("ADR naming something else must not count: %v", msgs)
	}
}

// config-weakening AC-4: first creation, harmless changes, an unchanged file and lint report nothing.
func TestSDD012IgnoresHarmless(t *testing.T) {
	cases := map[string]struct {
		gate    string
		current string
		atBase  bool
		diff    []string
	}{
		"first creation":   {"pre-merge", strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), false, cfgDiff},
		"tests.include":    {"pre-merge", strings.Replace(baseConfig, `include: ["**/*_test.go"]`, `include: ["**/*"]`, 1), true, cfgDiff},
		"layer added":      {"pre-merge", strings.Replace(baseConfig, "layers: [internal/api/**, internal/store/**]", "layers: [internal/api/**, internal/store/**, internal/ui/**]", 1), true, cfgDiff},
		"profile raised":   {"pre-merge", strings.Replace(baseConfig, "profile: standard", "profile: regulated", 1), true, cfgDiff},
		"exclude removed":  {"pre-merge", strings.Replace(baseConfig, "exclude: [vendor/**]", "exclude: []", 1), true, cfgDiff},
		"comment only":     {"pre-merge", "# Kommentar\n" + baseConfig, true, cfgDiff},
		"not in diff":      {"pre-merge", strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), true, []string{"cmd/x.go"}},
		"not run in lint":  {"", strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), true, cfgDiff},
		"not run pre-rel.": {"pre-release", strings.Replace(baseConfig, "profile: standard", "profile: minimal", 1), true, cfgDiff},
	}
	for name, tc := range cases {
		if msgs := runSDD012(t, tc.gate, tc.current, tc.atBase, tc.diff, ""); len(msgs) != 0 {
			t.Errorf("%s: %v", name, msgs)
		}
	}
}

// config-weakening AC-5: SDD-012 is in the catalog with profile minimal, baseline forbidden and pinned; version 1.2.0.
func TestSDD012InCatalog(t *testing.T) {
	data, err := os.ReadFile("../../../catalog/catalog.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if cat.Version != "1.2.0" {
		t.Errorf("catalog version %q", cat.Version)
	}
	for _, r := range cat.Rules {
		if r.ID == "SDD-012" {
			if r.Profile != "minimal" || r.Baseline != "forbidden" || !r.Pinned || r.Check != "config_weakening_has_adr" {
				t.Errorf("SDD-012: %+v", r)
			}
			return
		}
	}
	t.Error("SDD-012 missing in the catalog")
}

// config-weakening AC-6: handbook, concept, README and CHANGELOG describe SDD-012 and pinned.
func TestSDD012Documented(t *testing.T) {
	for f, wants := range map[string][]string{
		"../../../handbuch/03-mechanismen.md": {"{#sdd-012}", "{#sdd-012-vertrag}", "pinned"},
		"../../../docs/sdd_konzept.md":        {"SDD-012", "pinned: true"},
		"../../../README.md":                  {"SDD-012"},
		"../../../CHANGELOG.md":               {"SDD-012", "pinned", "1.2.0"},
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

// config-weakening AC-3: an exclude that covers sdd.config.yaml itself cannot hide the change that adds it, and an ADR
// under an excluded path still counts (found before the v0.6.0 tag: exclude: ["**"] passed the gate).
func TestSDD012NotBlindedByExclude(t *testing.T) {
	for _, pat := range []string{"**", "sdd.config.yaml"} {
		weak := strings.Replace(baseConfig, "exclude: [vendor/**]", `exclude: [vendor/**, "`+pat+`"]`, 1)
		if msgs := runSDD012(t, "pre-merge", weak, true, cfgDiff, ""); len(msgs) != 1 || !strings.Contains(msgs[0], "exclude: +"+pat) {
			t.Errorf("exclude %s: %v", pat, msgs)
		}
		adr := "---\nstatus: accepted\nreferences: [sdd.config.yaml]\n---\n"
		if msgs := runSDD012(t, "pre-merge", weak, true, []string{"sdd.config.yaml", "docs/adr/0002-x.md"}, adr); len(msgs) != 0 {
			t.Errorf("exclude %s with ADR: %v", pat, msgs)
		}
	}
}
