package rules

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
)

func TestDeclRegions(t *testing.T) {
	src := strings.Join([]string{
		"package x",                   // 0 out
		"// AC-1 freistehend",         // 1 out: blank line follows
		"",                            // 2 out
		"// AC-2 direkt darüber",      // 3 in (attached)
		"func TestA(t *testing.T) {",  // 4 in
		"\tt.Run(\"AC-3\", nil)",      // 5 in
		"}",                           // 6 in
		"",                            // 7 in (region runs to the next declaration)
		"func helper() {}",            // 8 in (still TestA's region by the contract)
		"#[test]",                     // 9 in (attribute of the next declaration)
		"func TestB(t *testing.T) {}", // 10 in
	}, "\n")
	got := declRegions(strings.Split(src, "\n"))
	want := []bool{false, false, false, true, true, true, true, true, true, true, true}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("regions\n got %v\nwant %v", got, want)
	}
}

// sdd-rules-v1-1 AC-8: an AC referenced in a test file must lie within a test declaration's region.
func TestSDD025(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/f.md", "---\nfeature: f\n---\n## Akzeptanzkriterien\n- AC-1 a\n- AC-2 b\n- AC-3 c\n- AC-4 d\n- AC-5 e\n")
	writeFile(t, root, "pkg/a_test.go", "package a\n\n// AC-1 freistehender Kommentar\n\n// AC-2 Doc-Kommentar\nfunc TestX(t *testing.T) {\n\tt.Run(\"AC-3\", nil)\n}\n")
	writeFile(t, root, "web/b.spec.ts", "// AC-4 Dateikopf\n\ndescribe('suite', () => {\n  it('does ac_5', () => {})\n})\n")
	cfg := featureConfig()
	cfg.Tests.Include = []string{"**/*_test.go", "**/*.spec.ts"}

	findings, err := checkSDD025(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-025"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range findings {
		got = append(got, strings.Fields(f.Message)[0])
	}
	if want := []string{"AC-1", "AC-4"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v (%+v)", got, want, findings)
	}

	// Unreferenced ACs are SDD-022's concern, not SDD-025's.
	writeFile(t, root, "docs/prd/f.md", "---\nfeature: f\n---\n## Akzeptanzkriterien\n- AC-9 nirgends\n")
	if findings, err = checkSDD025(&Context{RepoRoot: root, Config: cfg}, catalog.Rule{ID: "SDD-025"}); err != nil || len(findings) != 0 {
		t.Fatalf("unreferenced AC: %+v, %v", findings, err)
	}
}
