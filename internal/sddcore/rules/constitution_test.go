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
