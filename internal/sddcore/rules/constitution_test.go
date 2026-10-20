package rules

import (
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
