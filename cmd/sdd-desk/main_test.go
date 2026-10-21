package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// sdd-desk-sidecar-name AC-1: --sdd, then the bundled sdd-studio-sdd next to sdd-desk, then sdd next to it, then PATH.
func TestFindSdd(t *testing.T) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	dir := t.TempDir()
	touch := func(name string) string {
		p := filepath.Join(dir, name+ext)
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	inPath := func(string) (string, error) { return "/opt/bin/sdd", nil }
	notInPath := func(string) (string, error) { return "", errors.New("not found") }

	if got := findSdd("", dir, inPath); got != "/opt/bin/sdd" {
		t.Errorf("nothing next to sdd-desk: %q", got)
	}
	if got := findSdd("", dir, notInPath); got != "sdd" {
		t.Errorf("nothing found: %q", got)
	}
	plain := touch("sdd")
	if got := findSdd("", dir, inPath); got != plain {
		t.Errorf("sdd next to sdd-desk must win over PATH: %q", got)
	}
	bundled := touch("sdd-studio-sdd")
	if got := findSdd("", dir, inPath); got != bundled {
		t.Errorf("bundled sidecar must win: %q", got)
	}
	if got := findSdd("/x/sdd", dir, inPath); got != "/x/sdd" {
		t.Errorf("--sdd must win: %q", got)
	}
	if err := os.Remove(bundled); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(bundled, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findSdd("", dir, inPath); got != plain {
		t.Errorf("a directory named sdd-studio-sdd is no executable: %q", got)
	}
}

// sdd-desk-turn-timeout AC-3: a single provider request has no shorter bound of its own than the turn.
func TestProviderClientHasNoOwnTimeout(t *testing.T) {
	if c := providerClient(); c.Timeout != 0 {
		t.Errorf("provider client timeout %v; the turn's context must bound requests", c.Timeout)
	}
}
