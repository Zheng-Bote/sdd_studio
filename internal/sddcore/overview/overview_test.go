package overview_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/overview"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

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
		"GITHUB_BASE_REF": "", "CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "",
	} {
		t.Setenv(k, v)
	}
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
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

// project initializes an sdd project whose PRD has acceptance criteria without IDs (SDD-021 debt).
func project(t *testing.T, criteria ...string) string {
	t.Helper()
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo", CI: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	writePRD(t, root, criteria...)
	return root
}

func writePRD(t *testing.T, root string, criteria ...string) {
	t.Helper()
	body := "---\nfeature: billing\nstatus: active\n---\n## Akzeptanzkriterien\n"
	for _, c := range criteria {
		body += "- " + c + "\n"
	}
	write(t, root, "docs/prd/billing.md", body)
	write(t, root, "docs/design/billing.md", "---\nfeature: billing\nstatus: active\n---\n")
}

func rebaseline(t *testing.T, root string) {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := sddcore.BaselineEntries(root, cfg, catalog.AllGates)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root, &baseline.File{Version: 1, Entries: entries}); err != nil {
		t.Fatal(err)
	}
}

// sdd-report AC-1: maturity with next step, lint findings by severity, and all three gates.
func TestBuildShowsMaturityLintAndGates(t *testing.T) {
	isolateGit(t)
	root := project(t, "AC-1 mit Kennung", "ohne Kennung")
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "init")

	o, err := overview.Build(root, "", scan.CLIGit{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if o.Maturity.Level != 1 || o.Maturity.Next == nil || o.Maturity.Next.Name != "Akzeptanzkriterien mit Kennung" {
		t.Errorf("maturity: L%d, next %+v", o.Maturity.Level, o.Maturity.Next)
	}
	if o.Lint.Error == 0 || len(o.Findings) == 0 {
		t.Errorf("expected open lint errors, got %+v", o.Lint)
	}
	if len(o.Gates) != 3 {
		t.Fatalf("expected three gates, got %+v", o.Gates)
	}
	for i, g := range []string{"pre-implementation", "pre-merge", "pre-release"} {
		if o.Gates[i].Gate != g || o.Gates[i].Error != "" {
			t.Errorf("gate %d: %+v", i, o.Gates[i])
		}
	}
	if o.Gates[1].Counts.Error == 0 {
		t.Errorf("pre-merge should report the SDD-021 error, got %+v", o.Gates[1])
	}
	var md bytes.Buffer
	if err := overview.RenderMarkdown(&md, o); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Reifegrad **L1**", "**Nächster Schritt (L2):**", "| `pre-merge` |", "- **error** SDD-021 `docs/prd/billing.md:"} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("markdown misses %q\n%s", want, md.String())
		}
	}
}

// sdd-report AC-2, AC-3: debt per rule and gate against --since, and a growing baseline is highlighted.
func TestDebtTrendAgainstSince(t *testing.T) {
	isolateGit(t)
	root := project(t, "eins ohne Kennung")
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "init")
	rebaseline(t, root)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "baseline")

	// Unchanged: no growth.
	o, err := overview.Build(root, "main", scan.CLIGit{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if o.SinceError != "" || o.DebtNow != 1 || o.DebtBefore != 1 || o.Grew() {
		t.Fatalf("unchanged: %+v", o)
	}

	// A new criterion without ID is baselined on a branch: the ratchet is violated.
	git(t, root, "switch", "-q", "-c", "feature/x")
	writePRD(t, root, "eins ohne Kennung", "zwei ohne Kennung")
	rebaseline(t, root)
	o, err = overview.Build(root, "main", scan.CLIGit{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if o.DebtNow != 2 || o.DebtBefore != 1 || !o.Grew() || len(o.Added) != 1 || o.Added[0].RuleID != "SDD-021" {
		t.Fatalf("grown: now %d before %d added %+v", o.DebtNow, o.DebtBefore, o.Added)
	}
	if len(o.Debt) != 1 || o.Debt[0] != (overview.DebtRow{RuleID: "SDD-021", Gate: "pre-merge", Now: 2, Before: 1}) {
		t.Errorf("debt table %+v", o.Debt)
	}
	var md bytes.Buffer
	if err := overview.RenderMarkdown(&md, o); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2 Baseline-Einträge, am Vergleichsstand `main`: 1 (+1, gewachsen)", "**Verstoß gegen die Ratchet-Regel:**", "> - SDD-021 `docs/prd/billing.md` (Gate `pre-merge`"} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("markdown misses %q\n%s", want, md.String())
		}
	}

	// Fixing the criteria shrinks the debt: no violation.
	writePRD(t, root, "AC-1 eins", "AC-2 zwei")
	rebaseline(t, root)
	if o, err = overview.Build(root, "main", scan.CLIGit{Root: root}); err != nil || o.DebtNow != 0 || o.Grew() {
		t.Fatalf("shrunk: %+v, %v", o, err)
	}
}

// sdd-report AC-4: JSON output; without Git the gates that need it and the trend degrade with a reason.
func TestReportDegradesWithoutGit(t *testing.T) {
	isolateGit(t)
	root := project(t, "ohne Kennung") // not a Git repository
	o, err := overview.Build(root, "", scan.CLIGit{Root: root})
	if err != nil {
		t.Fatalf("report must not fail without Git: %v", err)
	}
	if o.SinceError == "" || o.Gates[1].Error == "" {
		t.Errorf("expected reasons for trend and pre-merge, got since=%q gates=%+v", o.SinceError, o.Gates)
	}
	if o.Gates[2].Error != "" {
		t.Errorf("pre-release needs no Git: %+v", o.Gates[2])
	}

	var buf bytes.Buffer
	if err := overview.RenderJSON(&buf, o); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"profile", "maturity", "lint", "findings", "gates", "since_error", "debt", "debt_now", "added"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("JSON misses %q", key)
		}
	}
	var md bytes.Buffer
	if err := overview.RenderMarkdown(&md, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md.String(), "Trend nicht ermittelbar:") || !strings.Contains(md.String(), "nicht ausführbar:") {
		t.Errorf("markdown misses degradation notes\n%s", md.String())
	}
}
