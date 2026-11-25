package sddstudio_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeDryRun runs `make -n <target>` for one system and returns the commands it would run.
//
// It runs in a temporary copy of the Makefile with every file target created, so the dry run shows the install
// commands instead of a build. A fresh checkout (CI) has none of those files; relying on `--old-file` did not help
// there, because make still considers a missing file out of date.
func makeDryRun(t *testing.T, osName, target string) string {
	t.Helper()
	dir := t.TempDir()
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), makefile, 0o644); err != nil {
		t.Fatal(err)
	}
	// Everything `make` would otherwise build or fetch: the Go binaries, the app, the Tauri node_modules marker,
	// the embedded UI and the macOS bundle. Sources get an older timestamp than the targets, so the dry run shows
	// the install commands and not a build (since issue #35 the targets depend on their sources).
	for _, rel := range []string{
		"go.mod",
		"embed.go",
		"cmd/sdd/main.go",
		"cmd/sdd-desk/main.go",
		"internal/sddcore/x.go",
		"catalog/catalog.yaml",
		"handbuch/03-mechanismen.md",
		"templates/agents.md.tmpl",
		"apps/desk/package-lock.json",
		"apps/desk/src-tauri/src/lib.rs",
		"apps/desk/src-tauri/tauri.conf.json",
		"apps/desk/src-tauri/Cargo.toml",
	} {
		writeAt(t, filepath.Join(dir, filepath.FromSlash(rel)), 1_000)
	}
	for _, rel := range []string{
		"bin/sdd",
		"bin/sdd-desk",
		"apps/desk/src-tauri/target/release/sdd-desk-app",
		"apps/desk/node_modules/.package-lock.json",
		"internal/desk/webdist/index.html",
		"apps/desk/src-tauri/target/release/bundle/macos/SDD Studio Desktop.app/Contents/Info.plist",
	} {
		writeAt(t, filepath.Join(dir, filepath.FromSlash(rel)), 2_000)
	}

	bin := t.TempDir()
	args := []string{
		"-C", dir,
		"-n",
		"OS=" + osName,
		"BINDIR=" + filepath.Join(bin, "bin"),
		"DATADIR=" + filepath.Join(bin, "share"),
		"APPDIR=" + filepath.Join(bin, "Applications"),
		target,
	}
	cmd := exec.Command("make", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s OS=%s failed: %v\n%s", target, osName, err, out)
	}
	return string(out)
}

// build-linux-macos AC-2, AC-7: Linux installs the program, the desktop entry and the hicolor icon theme — and
// nothing that only macOS needs. Checked on a dry run, so no file is written here.
func TestMakeInstallLinux(t *testing.T) {
	linux := makeDryRun(t, "Linux", "install")
	for _, want := range []string{
		"/bin/sdd",
		"/bin/sdd-desk",
		"/bin/sdd-desk-app",
		"applications/io.github.zhengbote.sdddesk.desktop",
		"icons/hicolor",
	} {
		if !strings.Contains(linux, want) {
			t.Errorf("Linux install must mention %q:\n%s", want, linux)
		}
	}
	if strings.Contains(linux, "ditto") {
		t.Errorf("Linux install must not copy an .app bundle:\n%s", linux)
	}
}

// build-linux-macos AC-3, AC-7: macOS copies the .app bundle into APPDIR and writes neither a .desktop file nor a
// hicolor icon theme, which macOS does not read.
func TestMakeInstallMacOS(t *testing.T) {
	darwin := makeDryRun(t, "Darwin", "install")
	for _, want := range []string{
		"/bin/sdd",
		"/bin/sdd-desk",
		"/Applications/SDD Studio Desktop.app",
		// Ein früheres Install legte das Programm nach BINDIR; das wird beim Umstieg entfernt.
		"/bin/sdd-desk-app",
	} {
		if !strings.Contains(darwin, want) {
			t.Errorf("macOS install must mention %q:\n%s", want, darwin)
		}
	}
	for _, forbidden := range []string{".desktop", "hicolor"} {
		if strings.Contains(darwin, forbidden) {
			t.Errorf("macOS install must not write %q:\n%s", forbidden, darwin)
		}
	}
	// Regression: make trennt Ziele an Leerzeichen. "SDD Studio Desktop.app" unmaskiert wird zu drei Zielen mit
	// demselben Rezept — der Probelauf zeigte dann ein zusätzliches `make all` und führte ditto dreimal aus.
	if strings.Contains(darwin, "make all") {
		t.Errorf("the bundle target is word-split by make:\n%s", darwin)
	}
	if n := strings.Count(darwin, "ditto "); n != 1 {
		t.Errorf("the bundle must be copied once, got %d ditto calls:\n%s", n, darwin)
	}
}

// build-linux-macos AC-5: uninstall removes per system what install created; on macOS it also removes a sdd-desk-app
// binary that an earlier install left in BINDIR.
func TestMakeUninstallPerSystem(t *testing.T) {
	linux := makeDryRun(t, "Linux", "uninstall")
	for _, want := range []string{"bin/sdd", "bin/sdd-desk", "sdd-desk-app", ".desktop", "hicolor"} {
		if !strings.Contains(linux, want) {
			t.Errorf("Linux uninstall must remove %q:\n%s", want, linux)
		}
	}

	darwin := makeDryRun(t, "Darwin", "uninstall")
	for _, want := range []string{"bin/sdd", "bin/sdd-desk", "SDD Studio Desktop.app", "sdd-desk-app"} {
		if !strings.Contains(darwin, want) {
			t.Errorf("macOS uninstall must remove %q:\n%s", want, darwin)
		}
	}
	if strings.Contains(darwin, ".desktop") {
		t.Errorf("macOS uninstall must not mention a .desktop file:\n%s", darwin)
	}
}

// build-linux-macos AC-1: the app build asks Tauri for a bundle on macOS and for no bundle elsewhere, so Linux keeps
// its plain binary and macOS gets the .app the system starts.
func TestMakeAppBuildsBundleOnlyOnDarwin(t *testing.T) {
	darwin := makeDryRun(t, "Darwin", "desk-app")
	if !strings.Contains(darwin, "--bundles app") {
		t.Errorf("macOS app build must ask for the app bundle:\n%s", darwin)
	}
	linux := makeDryRun(t, "Linux", "desk-app")
	if !strings.Contains(linux, "--no-bundle") {
		t.Errorf("Linux app build must not bundle:\n%s", linux)
	}
}

// tauriConf is the part of apps/desk/src-tauri/tauri.conf.json this test relies on.
type tauriConf struct {
	ProductName    string `json:"productName"`
	MainBinaryName string `json:"mainBinaryName"`
	Version        string `json:"version"`
	Bundle         struct {
		ExternalBin []string `json:"externalBin"`
	} `json:"bundle"`
}

func loadTauriConf(t *testing.T) tauriConf {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("apps", "desk", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatal(err)
	}
	var conf tauriConf
	if err := json.Unmarshal(data, &conf); err != nil {
		t.Fatalf("tauri.conf.json unlesbar: %v", err)
	}
	return conf
}

// build-linux-macos AC-4: the bundle macOS installs is the one Tauri packs — named like productName, carrying the
// project version and the main binary plus both sidecars in Contents/MacOS.
func TestMacOSBundleMatchesTauriConfig(t *testing.T) {
	conf := loadTauriConf(t)
	if conf.ProductName == "" || conf.MainBinaryName == "" {
		t.Fatalf("tauri.conf.json without productName/mainBinaryName: %+v", conf)
	}

	pkg, err := os.ReadFile(filepath.Join("apps", "desk", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkgVersion struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(pkg, &pkgVersion); err != nil {
		t.Fatalf("package.json unlesbar: %v", err)
	}
	if conf.Version != pkgVersion.Version {
		t.Errorf("tauri.conf.json version %q != package.json version %q", conf.Version, pkgVersion.Version)
	}

	// Contents/MacOS holds the main binary and, next to it, the sidecars without their target-triple suffix.
	want := map[string]bool{conf.MainBinaryName: false}
	for _, bin := range conf.Bundle.ExternalBin {
		want[filepath.Base(bin)] = false
	}
	if len(want) != 3 {
		t.Fatalf("expected main binary and two sidecars, got %v", want)
	}

	bundle := filepath.Join("apps", "desk", "src-tauri", "target", "release", "bundle", "macos", conf.ProductName+".app")
	macosDir := filepath.Join(bundle, "Contents", "MacOS")
	entries, err := os.ReadDir(macosDir)
	if err != nil {
		// No bundle built here (CI, or a source checkout). The naming rule is still checked below.
		t.Logf("Bundle nicht gebaut, Inhalt ungeprüft: %v", err)
	} else {
		for _, e := range entries {
			if _, ok := want[e.Name()]; ok {
				want[e.Name()] = true
			}
		}
		for name, found := range want {
			if !found {
				t.Errorf("Bundle ohne %s in Contents/MacOS", name)
			}
		}
	}

	install := makeDryRun(t, "Darwin", "install")
	if !strings.Contains(install, conf.ProductName+".app") {
		t.Errorf("macOS install must copy %q, got:\n%s", conf.ProductName+".app", install)
	}
}

// build-linux-macos AC-6: the system is detected and can be overridden; the paths stay overridable.
func TestMakeSystemIsDetectedAndOverridable(t *testing.T) {
	// OS= wins over uname -s, which is what the other tests above rely on. print-os runs for real: under `make -n`
	// only the command line would be printed, not its output.
	cmd := exec.Command("make", "print-os", "OS=Darwin")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make print-os OS=Darwin failed: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "darwin" {
		t.Errorf("make print-os OS=Darwin = %q, want darwin", got)
	}

	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	// Collapse runs of blanks: the file aligns the assignments in columns, and the test cares about the names.
	mk := squashBlanks(string(data))
	for _, want := range []string{
		"OS ?= $(shell uname -s)",
		"PREFIX ?=",
		"BINDIR ?=",
		"DATADIR ?=",
		"APPDIR ?=",
	} {
		if !strings.Contains(mk, want) {
			t.Errorf("Makefile must keep %q", want)
		}
	}
}

// squashBlanks turns runs of spaces and tabs into one space, so a column-aligned assignment still matches.
func squashBlanks(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\t", " ")), " ")
}

// build-linux-macos AC-8: README and handbook say per system what install creates and how to remove it.
func TestInstallDocsCoverBothSystems(t *testing.T) {
	readDocs := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, path := range []string{"README.md", filepath.Join("handbuch", "04-agenten.md")} {
		text := readDocs(path)
		for _, want := range []string{"macOS", "Linux"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s must name %s", path, want)
			}
		}
	}
	readme := readDocs("README.md")
	for _, want := range []string{"Applications", "make uninstall"} {
		if !strings.Contains(readme, want) {
			t.Errorf("README must mention %q", want)
		}
	}
}

// --- Issue #35: the build targets must depend on their sources -------------------------------------------------

// makeSandbox copies the Makefile into a temporary tree with fake sources, so the build rules can be questioned
// without running a build.
func makeSandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), makefile, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"go.mod",
		"cmd/sdd/main.go",
		"cmd/sdd-desk/main.go",
		"internal/sddcore/x.go",
		"catalog/catalog.yaml",
		"templates/agents.md.tmpl",
	} {
		writeAt(t, filepath.Join(dir, filepath.FromSlash(rel)), 1_000)
	}
	return dir
}

// writeAt writes a small file and sets its modification time, so "newer than" is deterministic instead of racing
// the clock. An existing file keeps its content and only gets the new time.
func writeAt(t *testing.T, path string, unix int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, []byte("built\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ts := time.Unix(unix, 0)
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatal(err)
	}
}

// makeQuestion reports whether make considers the target up to date (0) or out of date (1). Recipes do not run.
// extra carries make variables, e.g. OS=Darwin for a rule that only exists on macOS.
func makeQuestion(t *testing.T, dir, target string, extra ...string) int {
	t.Helper()
	args := append([]string{"-C", dir, "-q"}, extra...)
	args = append(args, target)
	err := exec.Command("make", args...).Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return 1
	}
	t.Fatalf("make -q %s: %v", target, err)
	return -1
}

// Issue #35: bin/sdd must be rebuilt when a Go source is newer. Without a prerequisite, make calls an existing
// binary up to date no matter how old the sources are — and make install then copies it.
func TestBinaryDependsOnSources(t *testing.T) {
	dir := makeSandbox(t)
	binSdd := filepath.Join(dir, "bin", "sdd")

	// The binary is newer than every source: nothing to do.
	writeAt(t, binSdd, 2_000)
	if got := makeQuestion(t, dir, "bin/sdd"); got != 0 {
		t.Errorf("bin/sdd is up to date (exit %d), expected 0", got)
	}

	// A source changed afterwards: the binary is stale.
	writeAt(t, filepath.Join(dir, "cmd", "sdd", "main.go"), 3_000)
	if got := makeQuestion(t, dir, "bin/sdd"); got != 1 {
		t.Errorf("bin/sdd must be stale after a source change (exit %d), expected 1", got)
	}
}

// Issue #35: the catalog, the handbook and the templates are compiled into the binary (embed.go). A change there
// makes both Go programs stale — that is how a new AGENTS.md template reaches sdd init.
func TestBinaryDependsOnEmbeddedFiles(t *testing.T) {
	dir := makeSandbox(t)
	writeAt(t, filepath.Join(dir, "bin", "sdd"), 2_000)
	if got := makeQuestion(t, dir, "bin/sdd"); got != 0 {
		t.Fatalf("bin/sdd is up to date (exit %d), expected 0", got)
	}

	writeAt(t, filepath.Join(dir, "templates", "agents.md.tmpl"), 3_000)
	if got := makeQuestion(t, dir, "bin/sdd"); got != 1 {
		t.Errorf("bin/sdd must be stale after a template change (exit %d), expected 1", got)
	}
}

// Issue #35: make install builds before it copies. The dependency on `all` is what guarantees a fresh build —
// without it, an existing bin/sdd is copied unchanged no matter how old it is.
func TestMakeInstallDependsOnAll(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "install:") {
			continue
		}
		if !strings.Contains(line, "all") {
			t.Errorf("install must depend on all, got %q", line)
		}
	}
}

// --- Issue #39: make -q and make -n must not build ----------------------------------------------------------------

// Issue #39: no recipe may call the sub-make. GNU make runs a recipe line containing $(MAKE) even under -q and -n, so
// a question about a target started a full build — and make -q then reported exit 2 instead of 0 or 1.
func TestNoRecipeCallsSubMake(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.Contains(line, "$(MAKE)") {
			t.Errorf("Makefile:%d ruft $(MAKE) im Rezept; make -q und -n führen das aus: %q", i+1, strings.TrimSpace(line))
		}
	}
}

// bundleSandbox is a tree with every file the bundle target depends on. The bundle itself and the sources can be
// stamped separately, so "stale" is a fact of the test and not of the clock.
//
// It returns the tree and the bundle as a path relative to it: the Makefile names its targets relatively, and a
// question about an absolute path would find no rule and answer "nothing to be done" (exit 0) instead of "stale".
func bundleSandbox(t *testing.T) (dir, bundleRel string) {
	t.Helper()
	dir = t.TempDir()
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), makefile, 0o644); err != nil {
		t.Fatal(err)
	}
	sources := []string{
		"go.mod",
		"embed.go",
		"cmd/sdd/main.go",
		"cmd/sdd-desk/main.go",
		"internal/sddcore/x.go",
		"catalog/catalog.yaml",
		"handbuch/03-mechanismen.md",
		"templates/agents.md.tmpl",
		"apps/desk/src-tauri/src/lib.rs",
		"apps/desk/src-tauri/tauri.conf.json",
		"apps/desk/src-tauri/Cargo.toml",
		"internal/desk/webdist/index.html",
	}
	for _, rel := range sources {
		writeAt(t, filepath.Join(dir, filepath.FromSlash(rel)), 3_000)
	}
	bundleRel = filepath.Join("apps", "desk", "src-tauri", "target", "release", "bundle", "macos", "SDD Studio Desktop.app")
	if err := os.MkdirAll(filepath.Join(dir, bundleRel), 0o755); err != nil {
		t.Fatal(err)
	}
	// The Go programs and the app binary are file prerequisites of the bundle too.
	for _, rel := range []string{"bin/sdd", "bin/sdd-desk", "apps/desk/src-tauri/target/release/sdd-desk-app"} {
		writeAt(t, filepath.Join(dir, filepath.FromSlash(rel)), 3_000)
	}
	return dir, bundleRel
}

// Issue #39: asking whether the bundle is current must answer, not build. With a stale bundle make -q returns 1
// (out of date) — before the fix it returned 2, because the sub-make it started failed.
//
// OS=Darwin is passed because the bundle rule lives inside `ifeq ($(OS),Darwin)`: on a Linux host (CI) it would not
// exist at all, and make would answer "nothing to be done" (0) for an unknown target.
func TestMakeQuestionOnStaleBundleDoesNotBuild(t *testing.T) {
	dir, bundleRel := bundleSandbox(t)
	bundle := filepath.Join(dir, bundleRel)

	// Bundle newer than everything: current.
	writeAt(t, bundle, 4_000)
	if got := makeQuestion(t, dir, bundleRel, "OS=Darwin"); got != 0 {
		t.Fatalf("aktuelles Bundle: make -q = %d, erwartet 0", got)
	}

	// A source changed afterwards: stale, and make -q must say so without running anything.
	writeAt(t, filepath.Join(dir, "templates", "agents.md.tmpl"), 5_000)
	if got := makeQuestion(t, dir, bundleRel, "OS=Darwin"); got != 1 {
		t.Errorf("veraltetes Bundle: make -q = %d, erwartet 1 (2 = es wurde gebaut)", got)
	}
	// The bundle keeps its old timestamp — nothing rebuilt it.
	info, err := os.Stat(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().Unix() != 4_000 {
		t.Errorf("make -q hat das Bundle angefasst: mtime %d, erwartet 4000", info.ModTime().Unix())
	}
}

// Issue #39: the same for the Linux program. Its recipe used to be $(MAKE) all as well.
func TestMakeQuestionOnStaleAppBinaryDoesNotBuild(t *testing.T) {
	dir, _ := bundleSandbox(t)
	appRel := filepath.Join("apps", "desk", "src-tauri", "target", "release", "sdd-desk-app")
	appOut := filepath.Join(dir, appRel)

	writeAt(t, appOut, 4_000)
	if got := makeQuestion(t, dir, appRel); got != 0 {
		t.Fatalf("aktuelles Programm: make -q = %d, erwartet 0", got)
	}

	writeAt(t, filepath.Join(dir, "internal", "sddcore", "x.go"), 5_000)
	if got := makeQuestion(t, dir, appRel); got != 1 {
		t.Errorf("veraltetes Programm: make -q = %d, erwartet 1 (2 = es wurde gebaut)", got)
	}
}
