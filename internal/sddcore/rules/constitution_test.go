package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

func runSDD001(t *testing.T, root, constitution string) []string {
	t.Helper()
	ctx := &Context{
		RepoRoot: root,
		Config:   &config.Config{Version: 1, Profile: "minimal", Artifacts: config.ArtifactsConfig{Constitution: constitution}},
	}
	findings, err := checkSDD001(ctx, catalog.Rule{ID: "SDD-001"})
	if err != nil {
		t.Fatalf("checkSDD001 returned error: %v", err)
	}
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.Message)
	}
	return msgs
}

// sdd-core-v1 AC-5: SDD-001 constitution present.
func TestSDD001(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("constitution.md", "# Verfassung\n")
	write("empty.md", "  \n\t\n")
	if err := os.Mkdir(filepath.Join(root, "dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name         string
		constitution string
		wantSubstr   string // "" means no finding expected
	}{
		{"present", "constitution.md", ""},
		{"present with dot prefix", "./constitution.md", ""},
		{"unconfigured", "", "nicht konfiguriert"},
		{"missing", "missing.md", "existiert nicht"},
		{"empty", "empty.md", "ist leer"},
		{"directory", "dir.md", "keine reguläre Datei"},
		{"parent escape", "../constitution.md", "außerhalb"},
		{"absolute", "/etc/passwd", "außerhalb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msgs := runSDD001(t, root, tc.constitution)
			if tc.wantSubstr == "" {
				if len(msgs) != 0 {
					t.Fatalf("expected no findings, got %v", msgs)
				}
				return
			}
			if len(msgs) != 1 || !strings.Contains(msgs[0], tc.wantSubstr) {
				t.Fatalf("expected one finding containing %q, got %v", tc.wantSubstr, msgs)
			}
		})
	}
}

// sdd-core-v1 AC-6: SDD-002a required constitution sections.
func TestSDD002a(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("full.md", "# V\n## Architektur/Layering\n### SECURITY\n#### strikte   regeln {#strikt}\n")
	write("spaced.md", "## Architektur / Layering\n## Security ##\n## Strikte Regeln\n")
	write("partial.md", "## Architektur/Layering\n```\n## Security\n```\n")
	write("grammar.sdd", "Spec: Demo\n")

	cases := []struct {
		name         string
		constitution string
		wantMissing  []string
	}{
		{"all sections", "full.md", nil},
		{"whitespace and closing hashes", "spaced.md", nil},
		{"heading in code fence does not count", "partial.md", []string{"Security", "Strikte Regeln"}},
		{".sdd is SDD-002b territory", "grammar.sdd", nil},
		{"missing file is SDD-001 territory", "missing.md", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &Context{
				RepoRoot: root,
				Config:   &config.Config{Version: 1, Profile: "minimal", Artifacts: config.ArtifactsConfig{Constitution: tc.constitution}},
			}
			findings, err := checkSDD002a(ctx, catalog.Rule{ID: "SDD-002a"})
			if err != nil {
				t.Fatalf("checkSDD002a returned error: %v", err)
			}
			if len(findings) != len(tc.wantMissing) {
				t.Fatalf("expected %d findings, got %+v", len(tc.wantMissing), findings)
			}
			for i, section := range tc.wantMissing {
				if !strings.Contains(findings[i].Message, `"`+section+`"`) {
					t.Errorf("finding %d: expected section %q, got %q", i, section, findings[i].Message)
				}
			}
		})
	}
}

// sdd-rules-v1-1 AC-9: a .sdd constitution follows the formal grammar; known labels are configurable.
func TestSDD002b(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok.sdd", "# Kommentar\n\nSpec: md-indexer\n\nArchitecture:\n  Layer A darf B nicht kennen.\n    Detail\nSecurity:\n  Nur lokal.\nCan modify:\n  - team-core\n")
	write("bad.sdd", "# SpecDD: md-indexer\n"+ // comment, skipped
		"## 1. System Intent\n"+ // 2: also a comment line
		"Architecture:\n"+ // 3: first content line is not Spec:
		"   drei Leerzeichen\n"+ // 4: odd indent
		"\tTab\n"+ // 5: tab
		"Roadmap:\n"+ // 6: unknown label
		"freier Text\n") // 7: no label
	write("custom.sdd", "Spec: x\nRoadmap:\n  ok\n")
	write("constitution.md", "# Markdown ist SDD-002a\n")

	run := func(file string, labels []string) []int {
		t.Helper()
		cfg := &config.Config{Version: 1, Profile: "minimal", Artifacts: config.ArtifactsConfig{Constitution: file}, ConstitutionSections: labels}
		findings, err := checkSDD002b(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-002b"})
		if err != nil {
			t.Fatal(err)
		}
		var lines []int
		for _, f := range findings {
			lines = append(lines, f.Line)
		}
		return lines
	}
	if got := run("ok.sdd", nil); len(got) != 0 {
		t.Errorf("valid grammar: findings at %v", got)
	}
	if got, want := run("bad.sdd", nil), []int{3, 4, 5, 6, 7}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("bad grammar: findings at %v, want %v", got, want)
	}
	if got := run("custom.sdd", nil); len(got) != 1 {
		t.Errorf("Roadmap is unknown by default, got %v", got)
	}
	if got := run("custom.sdd", []string{"roadmap"}); len(got) != 0 {
		t.Errorf("constitution_sections must override the labels (case-insensitive), got %v", got)
	}
	if got := run("constitution.md", nil); len(got) != 0 {
		t.Errorf("Markdown constitutions are SDD-002a territory, got %v", got)
	}
	if got := run("missing.sdd", nil); len(got) != 0 {
		t.Errorf("missing file is SDD-001 territory, got %v", got)
	}
}

// sdd-rules-003-040 AC-1: feature content in the constitution is reported line by line (heuristic, warn).
func TestSDD003(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("constitution.md", "---\nstatus: active\n---\n"+
		"# Verfassung\n"+ // 4
		"Feature-Inhalte (User Stories, Tasklisten) gehören in PRDs.\n"+ // 5: prose mention, no finding
		"## Security\n"+ // 6
		"1. Als Nutzer möchte ich mich anmelden.\n"+ // 7 user story
		"- [ ] Login bauen\n"+ // 8 task
		"- Erfüllt AC-3\n"+ // 9 AC
		"## Sprint 12\n"+ // 10 planning heading
		"## Release-Roadmap\n"+ // 11 planning heading
		"```\n- [ ] im Codeblock\n```\n"+
		"## Strikte Regeln\n"+
		"As an admin, I want audit logs.\n") // 16 user story (English)
	write("c.sdd", "Spec: x\n# Roadmap ist ein Kommentar\nArchitecture:\n  - [ ] Task im Label\nRoadmap:\n  später\n")

	run := func(file string) []int {
		t.Helper()
		cfg := &config.Config{Version: 1, Profile: "minimal", Artifacts: config.ArtifactsConfig{Constitution: file}}
		findings, err := checkSDD003(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-003"})
		if err != nil {
			t.Fatal(err)
		}
		var lines []int
		for _, f := range findings {
			lines = append(lines, f.Line)
		}
		return lines
	}
	if got, want := run("constitution.md"), []int{7, 8, 9, 10, 11, 16}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("markdown: findings at %v, want %v", got, want)
	}
	if got, want := run("c.sdd"), []int{4, 5}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf(".sdd: findings at %v, want %v", got, want)
	}
	if got := run("missing.md"); len(got) != 0 {
		t.Errorf("missing constitution is SDD-001 territory, got %v", got)
	}
}
