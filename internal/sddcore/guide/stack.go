package guide

import (
	"errors"
	"path"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// language describes how one language is recognized and how its tests carry acceptance criteria.
type language struct {
	Name      string
	Exts      []string
	Markers   []string // exact file names; ".csproj" style entries match by extension
	TestGlobs []string
	Comment   string
}

// languages is fixed on purpose: detection is deterministic and offline, by file names only.
var languages = []language{
	{"Go", []string{".go"}, []string{"go.mod"}, []string{"**/*_test.go"}, "//"},
	{"Rust", []string{".rs"}, []string{"Cargo.toml"}, []string{"tests/**/*.rs", "src/**/*.rs"}, "//"},
	{"C/C++", []string{".c", ".cc", ".cpp", ".cxx", ".h", ".hpp"}, []string{"CMakeLists.txt", "meson.build"}, []string{"tests/**/*.cpp", "test/**/*.cpp", "**/*_test.cpp"}, "//"},
	{"Python", []string{".py"}, []string{"pyproject.toml", "setup.py", "requirements.txt"}, []string{"**/test_*.py", "**/*_test.py"}, "#"},
	{"TypeScript/JavaScript", []string{".ts", ".tsx", ".js", ".jsx", ".mjs"}, []string{"package.json", "tsconfig.json"}, []string{"**/*.spec.ts", "**/*.test.ts", "**/*.test.js"}, "//"},
	{"Java/Kotlin", []string{".java", ".kt"}, []string{"pom.xml", "build.gradle", "build.gradle.kts"}, []string{"**/src/test/**/*.java", "**/src/test/**/*.kt"}, "//"},
	{"C#", []string{".cs"}, []string{".csproj"}, []string{"**/*Tests.cs"}, "//"},
	{"Shell", []string{".sh", ".bash"}, nil, []string{"tests/**/*.bats", "tests/**/*.sh"}, "#"},
}

// Language is one detected language of the repository.
type Language struct {
	Name      string   `json:"name"`
	Files     int      `json:"files"`
	Markers   []string `json:"markers"`
	TestGlobs []string `json:"test_globs"`
	Comment   string   `json:"comment"`
	Example   string   `json:"example"`
}

// StackInfo is what sdd_stack reports.
type StackInfo struct {
	Files        int        `json:"files"` // all counted files, without .git, .sdd and exclude matches
	Languages    []Language `json:"languages"`
	TestsInclude []string   `json:"tests_include"`
	Configured   bool       `json:"configured"`
}

// Stack detects the languages of the repository by file extensions and marker files (no file contents), skipping
// .git, .sdd and exclude matches, sorted by number of source files.
func Stack(root string) (*StackInfo, error) {
	info := &StackInfo{Languages: []Language{}, TestsInclude: []string{}}
	cfg, err := config.Load(root)
	switch {
	case errors.Is(err, config.ErrConfigNotFound):
		cfg = scaffold.DefaultConfig("standard")
	case err != nil:
		return nil, err
	default:
		info.Configured = true
		if cfg.Tests.Include != nil {
			info.TestsInclude = cfg.Tests.Include
		}
	}
	files, err := scan.WalkFiles(root, []string{"**"}, append(append([]string{}, cfg.Exclude...), ".sdd/**"))
	if err != nil {
		return nil, err
	}
	info.Files = len(files)
	for _, l := range languages {
		found := Language{Name: l.Name, Markers: []string{}, TestGlobs: l.TestGlobs, Comment: l.Comment,
			Example: l.Comment + " <slug> AC-1: <was der Test prüft>"}
		markers := map[string]bool{}
		for _, f := range files {
			base, depth := path.Base(f), strings.Count(f, "/")
			for _, e := range l.Exts {
				if strings.EqualFold(path.Ext(f), e) {
					found.Files++
					break
				}
			}
			if depth > 1 {
				continue
			}
			for _, m := range l.Markers {
				if base == m || (strings.HasPrefix(m, ".") && strings.HasSuffix(base, m)) {
					markers[f] = true
				}
			}
		}
		for m := range markers {
			found.Markers = append(found.Markers, m)
		}
		sort.Strings(found.Markers)
		if found.Files > 0 || len(found.Markers) > 0 {
			info.Languages = append(info.Languages, found)
		}
	}
	sort.Slice(info.Languages, func(i, j int) bool {
		a, b := info.Languages[i], info.Languages[j]
		if a.Files != b.Files {
			return a.Files > b.Files
		}
		return a.Name < b.Name
	})
	return info, nil
}
