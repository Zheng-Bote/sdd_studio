package adopt_test

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adopt"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

var today = time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)

func options() adopt.Options {
	return adopt.Options{Profile: "standard", Branch: "sdd/adopt", Project: "pilot", Today: today}
}

// isolateGit makes git independent of the developer's configuration (hooks, signing, identity).
func isolateGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": empty, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.com", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.com",
		"GITHUB_BASE_REF": "", "CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "", "GITHUB_HEAD_REF": "", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME": "", "CI_COMMIT_BRANCH": "",
	} {
		t.Setenv(k, v)
	}
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// specKitRepo reproduces the pilot finding from the concept: Spec Kit features without frontmatter or AC IDs,
// some without plan/tasks, no archive, no constitution, no sdd configuration.
func specKitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "README.md", "# Pilot\n")
	write(t, root, "docs/notes/recherche.md", "# Recherche\n")
	write(t, root, "docs/architecture/adr/0001-storage.md", "---\nstatus: accepted\n---\n# ADR\n")
	write(t, root, "specs/features/001-billing-export/spec.md", "# Billing\n\n## Akzeptanzkriterien\n\n- Export als CSV\n")
	write(t, root, "specs/features/001-billing-export/plan.md", "# Plan\n")
	write(t, root, "specs/features/001-billing-export/tasks.md", "# Tasks\n- [ ] CSV\n")
	write(t, root, "specs/features/002-search/spec.md", "# Search\n")
	write(t, root, "specs/features/003-done/spec.md", "---\nfeature: done\nstatus: implemented\n---\n# Done\n")
	write(t, root, "specs/features/004-odd/spec.md", "---\ntitle: ohne feature\n---\n# Odd\n")
	write(t, root, "main.go", "package main\n")
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "initial")
	return root
}

// bmadRepo reproduces a BMAD v6 tree (ADR 0009): the tool tree `_bmad/`, the ledger `_bmad-output/` with an
// initiative of numbered requirements, two epics that cover them, a ticket index, a forge report and memlogs. No
// feature artifacts, no constitution, no sdd configuration — the state a project is in after the BMAD planning.
func bmadRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "README.md", "# Gallery\n")
	write(t, root, "_bmad/config.toml", "[core]\nproject_name = \"Gallery\"\n")
	write(t, root, "_bmad/bmm/v6-shims/README.md", "# shim\n")
	write(t, root, "_bmad-output/initiative-gallery/.memlog.md", "# memlog\n")
	write(t, root, "_bmad-output/initiative-gallery/forge-gallery/forge-gallery.md", "# Geschärfte Idee\n")
	write(t, root, "_bmad-output/initiative-gallery/tickets/tickets.toml",
		"[[epic]]\nid = 1\nslug = \"epic-plattform\"\n\n[[epic]]\nid = 2\nslug = \"epic-import\"\n")
	write(t, root, "_bmad-output/initiative-gallery/initiative-gallery.md",
		"---\ntype: initiative\ntitle: Gallery\n---\n\n# Gallery\n\n## Requirements\n\n"+
			"- R1 (forge §Scope): Die Gruppe meldet sich an.\n- R2 (forge §Scope): Bilder kommen ins Archiv.\n"+
			"- R3 (forge §Scope): Nur veröffentlichte Alben sind öffentlich.\n\n## Boundaries\n\n- Nicht in V1: Slideshow.\n")
	write(t, root, "_bmad-output/initiative-gallery/tickets/epic-plattform/epic-plattform.md",
		"---\ntype: epic\ntitle: Plattform\ncovers: [\"R1\"]\nrefined: false\n---\n\n# Plattform\n\n## Done when\n\n- Der Admin legt ein Konto an.\n")
	write(t, root, "_bmad-output/initiative-gallery/tickets/epic-import/epic-import.md",
		"---\ntype: epic\ntitle: Import\ncovers: [\"R2\", \"R9\"]\nrefined: false\n---\n\n# Import\n")
	write(t, root, "main.go", "package main\n")
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "initial")
	return root
}

// snapshot hashes every file outside .git.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// sdd-adopt AC-2: every Markdown file gets exactly one type with a reason; no pattern means Doku/Exploration.
func TestClassify(t *testing.T) {
	isolateGit(t)
	root := specKitRepo(t)
	inv, err := adopt.Take(root, "standard")
	if err != nil {
		t.Fatal(err)
	}
	if inv.ConfigExists || !strings.Contains(inv.Layout, "Spec Kit") || inv.Config.Artifacts.ADR != "docs/architecture/adr" {
		t.Fatalf("unexpected proposal: layout %q, artifacts %+v", inv.Layout, inv.Config.Artifacts)
	}
	got := map[string]adopt.Kind{}
	for _, f := range inv.Files {
		if f.Reason == "" {
			t.Errorf("%s: no reason", f.Path)
		}
		got[f.Path] = f.Kind
	}
	want := map[string]adopt.Kind{
		"README.md":                                  adopt.KindDoc,
		"docs/notes/recherche.md":                    adopt.KindDoc,
		"docs/architecture/adr/0001-storage.md":      adopt.KindADR,
		"specs/features/001-billing-export/spec.md":  adopt.KindPRD,
		"specs/features/001-billing-export/plan.md":  adopt.KindDesign,
		"specs/features/001-billing-export/tasks.md": adopt.KindPlan,
		"specs/features/002-search/spec.md":          adopt.KindPRD,
		"specs/features/003-done/spec.md":            adopt.KindPRD,
		"specs/features/004-odd/spec.md":             adopt.KindPRD,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("classification\n got %v\nwant %v", got, want)
	}
}

// sdd-adopt AC-3: L0-L4 with one justification per criterion and exactly one next step.
func TestMaturity(t *testing.T) {
	isolateGit(t)
	root := specKitRepo(t)
	p, err := adopt.Propose(root, options())
	if err != nil {
		t.Fatal(err)
	}
	if p.Maturity.Level != 0 || p.Maturity.Next == nil || p.Maturity.Next.Name != "Verfassung vorhanden" {
		t.Fatalf("pilot: want L0 with next step constitution, got L%d %+v", p.Maturity.Level, p.Maturity.Next)
	}
	for _, c := range p.Maturity.Criteria {
		if c.Detail == "" && c.Name != "Archiv konfiguriert" {
			t.Errorf("criterion %q has no justification", c.Name)
		}
	}

	// A project initialized by sdd with a PRD, a Tech Spec and AC IDs reaches L2; next is CI.
	root2 := t.TempDir()
	write(t, root2, "sdd.config.yaml", "version: 1\nprofile: standard\nartifacts:\n  constitution: constitution.md\n  prd: docs/prd\n  design: docs/design\n  archive: docs/archive\n")
	write(t, root2, "constitution.md", "# V\n## Architektur/Layering\n## Security\n## Strikte Regeln\n")
	write(t, root2, "docs/prd/a.md", "---\nfeature: a\nstatus: active\n---\n## Akzeptanzkriterien\n- AC-1 x\n")
	write(t, root2, "docs/design/a.md", "---\nfeature: a\nstatus: active\n---\n")
	p, err = adopt.Propose(root2, options())
	if err != nil {
		t.Fatal(err)
	}
	if p.Maturity.Level != 2 || p.Maturity.Next == nil || p.Maturity.Next.Name != "CI ruft sdd gate auf" {
		t.Fatalf("initialized project: want L2 with next step CI, got L%d %+v", p.Maturity.Level, p.Maturity.Next)
	}

	// sdd-adopt AC-3: archiving after a release keeps L2 — archived PRDs and Tech Specs still show the separation.
	for _, rel := range []string{"docs/prd/a.md", "docs/design/a.md"} {
		data, _ := os.ReadFile(filepath.Join(root2, rel))
		write(t, root2, strings.Replace(rel, "docs/", "docs/archive/", 1), strings.Replace(string(data), "status: active", "status: archived", 1))
		if err := os.Remove(filepath.Join(root2, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if p, err = adopt.Propose(root2, options()); err != nil || p.Maturity.Level != 2 {
		t.Fatalf("after archiving: want L2, got %+v, %v", p.Maturity, err)
	}
}

// sdd-adopt AC-4, AC-7: the dry run writes nothing and shows every planned change.
func TestDryRunWritesNothing(t *testing.T) {
	isolateGit(t)
	root := specKitRepo(t)
	before := snapshot(t, root)
	p, err := adopt.Propose(root, options())
	if err != nil {
		t.Fatal(err)
	}
	report := adopt.RenderReport(p, nil)
	if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("dry run changed files")
	}
	for _, want := range []string{
		"**Probelauf**",
		"## Reifegrad: L0",
		"sdd.config.yaml anlegen (Spec Kit",
		"constitution.md aus dem Template anlegen",
		"specs/features/001-billing-export/spec.md ← `feature: 001-billing-export`, `status: active` (PRD)",
		"specs/features/002-search/spec.md ← `feature: 002-search`, `status: active` (PRD)",
		"specs/features/004-odd/spec.md hat Frontmatter, aber kein `feature:`",
		"- `specs/features/003-done/spec.md`",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("dry-run report misses %q\n%s", want, report)
		}
	}
}

// sdd-adopt AC-5, AC-6, AC-7: one commit per step on a new branch, main untouched, nothing deleted,
// baseline before the first gate with only baselineable errors, archive step reported not executed.
func TestApplyCommitsEachStep(t *testing.T) {
	isolateGit(t)
	root := specKitRepo(t)
	mainHead := git(t, root, "rev-parse", "main")
	p, err := adopt.Propose(root, options())
	if err != nil {
		t.Fatal(err)
	}
	res, err := adopt.Apply(root, p, scan.CLIGit{Root: root})
	if err != nil {
		t.Fatal(err)
	}

	if got := git(t, root, "rev-parse", "--abbrev-ref", "HEAD"); got != "sdd/adopt" {
		t.Fatalf("current branch %q", got)
	}
	if got := git(t, root, "rev-parse", "main"); got != mainHead {
		t.Fatalf("main moved")
	}
	subjects := strings.Split(git(t, root, "log", "--reverse", "--format=%s", "main..sdd/adopt"), "\n")
	// agent-files AC-4: the agent files are a step of their own, right after the constitution.
	wantSubjects := []string{"sdd adopt: Konfiguration", "sdd adopt: Verfassung", "sdd adopt: Agenten-Dateien", "sdd adopt: Frontmatter", "sdd adopt: Baseline", "sdd adopt: Bericht"}
	if !reflect.DeepEqual(subjects, wantSubjects) || len(res.Commits) != len(wantSubjects) {
		t.Fatalf("commits %v, want %v", subjects, wantSubjects)
	}
	if got := git(t, root, "show", "--name-only", "--format=", "HEAD~2"); got != "specs/features/001-billing-export/plan.md\nspecs/features/001-billing-export/spec.md\nspecs/features/001-billing-export/tasks.md\nspecs/features/002-search/spec.md" {
		t.Errorf("frontmatter commit touches %q", got)
	}
	for _, line := range strings.Split(git(t, root, "diff", "--name-status", "main...sdd/adopt"), "\n") {
		if !strings.HasPrefix(line, "A\t") && !strings.HasPrefix(line, "M\t") {
			t.Errorf("only additions and modifications allowed, got %q", line)
		}
	}
	spec, _ := os.ReadFile(filepath.Join(root, "specs/features/002-search/spec.md"))
	if string(spec) != "---\nfeature: 002-search\nstatus: active\n---\n\n# Search\n" {
		t.Errorf("unexpected spec content %q", spec)
	}
	odd, _ := os.ReadFile(filepath.Join(root, "specs/features/004-odd/spec.md"))
	if string(odd) != "---\ntitle: ohne feature\n---\n# Odd\n" {
		t.Errorf("conflicting file was changed: %q", odd)
	}

	var bf baseline.File
	data, err := os.ReadFile(filepath.Join(root, ".sdd/baseline.json"))
	if err != nil || json.Unmarshal(data, &bf) != nil || len(bf.Entries) == 0 || res.BaselineEntries != len(bf.Entries) {
		t.Fatalf("baseline: %v %+v", err, bf)
	}
	for _, e := range bf.Entries {
		if e.RuleID == "SDD-050" || strings.HasPrefix(e.RuleID, "SDD-07") {
			t.Errorf("forbidden rule in baseline: %+v", e)
		}
	}
	// sdd-rules-v1-1 AC-6: the baseline step also locks the decided ADRs, in the same commit.
	if got := git(t, root, "show", "--name-only", "--format=", "HEAD~1"); got != ".sdd/adr-lock.json\n.sdd/baseline.json" {
		t.Errorf("baseline commit touches %q", got)
	}
	if lock, err := os.ReadFile(filepath.Join(root, ".sdd/adr-lock.json")); err != nil || !strings.Contains(string(lock), "docs/architecture/adr/0001-storage.md") {
		t.Errorf("ADR lockfile: %v %s", err, lock)
	}
	// The baseline makes the adopted repository pass; only the non-baselineable archive warning remains.
	rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Error != 0 {
		t.Errorf("errors after adopt: %+v", rep.Findings)
	}
	done, _ := os.ReadFile(filepath.Join(root, "specs/features/003-done/spec.md"))
	if !strings.Contains(string(done), "status: implemented") {
		t.Errorf("archive step must not be executed")
	}

	report, err := os.ReadFile(filepath.Join(root, adopt.ReportPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Angewendet am 2026-10-20 auf Branch `sdd/adopt` (ausgehend von `main`)", "Commit `" + res.Commits[0].Hash + "`", "Baseline: ", "- `specs/features/003-done/spec.md`"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("report misses %q", want)
		}
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("worktree not clean after apply: %q", status)
	}
}

// sdd-adopt AC-5: refuse without writing on a dirty worktree, an existing branch, or outside Git.
func TestApplyRefusesDirtyTreeAndExistingBranch(t *testing.T) {
	isolateGit(t)
	cases := map[string]func(t *testing.T, root string){
		"dirty worktree":  func(t *testing.T, root string) { write(t, root, "README.md", "# geändert\n") },
		"untracked file":  func(t *testing.T, root string) { write(t, root, "neu.txt", "x") },
		"existing branch": func(t *testing.T, root string) { git(t, root, "branch", "sdd/adopt") },
		"detached HEAD":   func(t *testing.T, root string) { git(t, root, "checkout", "-q", "--detach") },
		"no repository":   func(t *testing.T, root string) { _ = os.RemoveAll(filepath.Join(root, ".git")) },
	}
	for name, prepare := range cases {
		t.Run(name, func(t *testing.T) {
			root := specKitRepo(t)
			prepare(t, root)
			before := snapshot(t, root)
			p, err := adopt.Propose(root, options())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := adopt.Apply(root, p, scan.CLIGit{Root: root}); !errors.Is(err, adopt.ErrPrecondition) {
				t.Fatalf("expected ErrPrecondition, got %v", err)
			}
			if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
				t.Errorf("refused apply changed files")
			}
		})
	}
}

// bmad-import AC-3: a BMAD v6 tree is recognised — one initiative, every epic, pipeline bmad in the proposed
// configuration — and the tool tree stays out of the inventory.
func TestBMADInventory(t *testing.T) {
	isolateGit(t)
	root := bmadRepo(t)
	inv, err := adopt.Take(root, "standard")
	if err != nil {
		t.Fatal(err)
	}
	if inv.BMAD == nil {
		t.Fatal("BMAD tree not detected")
	}
	if got := inv.Config.Pipeline; got != config.PipelineBMAD {
		t.Errorf("pipeline = %q, want %q", got, config.PipelineBMAD)
	}
	if got, want := inv.Config.Artifacts.InitiativePath(), "docs/initiative.md"; got != want {
		t.Errorf("artifacts.initiative = %q, want %q — the ledger is the source, not the artifact", got, want)
	}
	if !strings.Contains(inv.Layout, "BMAD v6") || !strings.Contains(inv.Layout, "2 Epics") || !strings.Contains(inv.Layout, "initiative-gallery.md") {
		t.Errorf("layout = %q", inv.Layout)
	}

	kinds := map[string]adopt.Kind{}
	for _, f := range inv.Files {
		kinds[f.Path] = f.Kind
	}
	want := map[string]adopt.Kind{
		"_bmad-output/initiative-gallery/initiative-gallery.md":                    adopt.KindInitiative,
		"_bmad-output/initiative-gallery/tickets/epic-plattform/epic-plattform.md": adopt.KindEpic,
		"_bmad-output/initiative-gallery/tickets/epic-import/epic-import.md":       adopt.KindEpic,
		"_bmad-output/initiative-gallery/forge-gallery/forge-gallery.md":           adopt.KindDoc,
	}
	for p, k := range want {
		if kinds[p] != k {
			t.Errorf("%s classified %q, want %q", p, kinds[p], k)
		}
	}
	for _, p := range []string{"_bmad/config.toml", "_bmad/bmm/v6-shims/README.md", "_bmad-output/initiative-gallery/.memlog.md"} {
		if _, ok := kinds[p]; ok {
			t.Errorf("%s belongs to the BMAD tool tree and must stay out of the inventory", p)
		}
	}
	if !slices.Contains(inv.Config.Exclude, "_bmad/**") || !slices.Contains(inv.Config.Exclude, "**/.memlog.md") {
		t.Errorf("proposed exclude misses the BMAD tool tree: %v", inv.Config.Exclude)
	}
}

// bmad-import AC-3: a repository without _bmad/ or _bmad-output/ keeps the previous behaviour.
func TestBMADAbsentUnchanged(t *testing.T) {
	isolateGit(t)
	root := specKitRepo(t)
	inv, err := adopt.Take(root, "standard")
	if err != nil {
		t.Fatal(err)
	}
	if inv.BMAD != nil || inv.Config.Pipeline != "" {
		t.Fatalf("BMAD detected in a Spec Kit repository: %+v", inv.BMAD)
	}
	for _, f := range inv.Files {
		if f.Kind == adopt.KindInitiative || f.Kind == adopt.KindEpic {
			t.Errorf("%s got a macro kind", f.Path)
		}
	}
}

// bmad-import AC-4: the take-over adds no feature frontmatter to macro artifacts and reports no conflict for them.
func TestBMADFrontmatterUntouched(t *testing.T) {
	isolateGit(t)
	root := bmadRepo(t)
	before := snapshot(t, root)
	p, err := adopt.Propose(root, options())
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range p.Steps {
		for _, op := range step.Plan.Ops {
			if strings.HasPrefix(op.Path, "_bmad-output/") {
				t.Errorf("step %q touches the BMAD ledger: %s", step.Title, op.Path)
			}
		}
	}
	for _, c := range p.Conflicts {
		if strings.Contains(c, "_bmad-output/") {
			t.Errorf("conflict on a macro artifact: %s", c)
		}
	}
	if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Error("Propose changed files")
	}
}
