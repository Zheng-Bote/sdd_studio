package guide_test

import (
	"reflect"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
)

// desk-guided-flow AC-1: languages by file count with markers, test patterns and comment; exclude, .git and .sdd are
// skipped; without source files the list is empty.
func TestStack(t *testing.T) {
	root := project(t)
	empty, err := guide.Stack(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Languages) != 0 || !empty.Configured {
		t.Fatalf("fresh project: %+v", empty)
	}
	for rel, content := range map[string]string{
		"go.mod": "module x\n", "main.go": "package main\n", "cmd/a/a.go": "package a\n", "cmd/a/a_test.go": "package a\n",
		"scripts/run.sh": "#!/bin/sh\n", "tools/CMakeLists.txt": "", "deep/x/y/CMakeLists.txt": "",
		"vendor/v/v.go": "package v\n", "vendor/v/w.go": "package v\n", "vendor/v/u.go": "package v\n", "vendor/v/t.go": "package v\n",
		".sdd/x.sh": "", ".git/hooks/pre-commit.sh": "",
	} {
		write(t, root, rel, content)
	}
	info, err := guide.Stack(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, l := range info.Languages {
		names = append(names, l.Name)
	}
	if !reflect.DeepEqual(names, []string{"Go", "Shell", "C/C++"}) {
		t.Fatalf("languages %v", names)
	}
	gol, sh, cpp := info.Languages[0], info.Languages[1], info.Languages[2]
	if gol.Files != 3 || !reflect.DeepEqual(gol.Markers, []string{"go.mod"}) || gol.Comment != "//" || gol.TestGlobs[0] != "**/*_test.go" || gol.Example != "// <slug> AC-1: <was der Test prüft>" {
		t.Errorf("Go: %+v", gol)
	}
	if sh.Files != 1 || sh.Comment != "#" {
		t.Errorf("Shell: %+v (.sdd and .git must not count)", sh)
	}
	if cpp.Files != 0 || !reflect.DeepEqual(cpp.Markers, []string{"tools/CMakeLists.txt"}) {
		t.Errorf("C/C++: %+v (markers only at the root or one level down)", cpp)
	}
}
