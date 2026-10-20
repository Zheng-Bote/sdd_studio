package ui

import (
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

const (
	testAddr  = "127.0.0.1:47999"
	testToken = "test-token"
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

// project creates an initialized sdd project in a Git repository with one commit.
func project(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	root := t.TempDir()
	plan, err := scaffold.InitPlan(scaffold.InitOptions{Profile: "standard", Project: "demo", CI: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(root); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "init")
	return root
}

func server(root string) *Server {
	s := New(root, testAddr, testToken, "demo")
	s.Now = func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }
	return s
}

type reqOpt func(*http.Request)

func withHost(h string) reqOpt      { return func(r *http.Request) { r.Host = h } }
func withHeader(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }
func withoutHeader(k string) reqOpt { return func(r *http.Request) { r.Header.Del(k) } }
func writeHeaders() []reqOpt {
	return []reqOpt{withHeader("X-SDD-Token", testToken), withHeader("Origin", "http://"+testAddr), withHeader("Content-Type", "application/json")}
}

func do(t *testing.T, s *Server, method, path, body string, opts ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://"+testAddr+path, strings.NewReader(body))
	req.Host = testAddr
	for _, o := range opts {
		o(req)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func post(t *testing.T, s *Server, path, body string, extra ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, s, http.MethodPost, path, body, append(writeHeaders(), extra...)...)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("invalid JSON (%d): %v\n%s", rec.Code, err, rec.Body.String())
	}
}

// sdd-ui AC-1: the listener is bound to 127.0.0.1 and every start creates a fresh random token.
func TestListenBindsLoopbackWithToken(t *testing.T) {
	ln, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	if host != "127.0.0.1" || port == "0" {
		t.Fatalf("bound to %s", ln.Addr())
	}
	a, _ := NewToken()
	b, _ := NewToken()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a) || a == b {
		t.Fatalf("tokens %q %q", a, b)
	}
}

// sdd-ui AC-2: any other Host is rejected, including localhost (DNS rebinding).
func TestRejectsForeignHost(t *testing.T) {
	s := server(project(t))
	for _, host := range []string{"evil.example:47999", "localhost:47999", "127.0.0.1", "127.0.0.1:1"} {
		if rec := do(t, s, http.MethodGet, "/api/meta", "", withHost(host)); rec.Code != http.StatusForbidden {
			t.Errorf("Host %q: got %d", host, rec.Code)
		}
	}
	if rec := do(t, s, http.MethodGet, "/api/meta", ""); rec.Code != http.StatusOK {
		t.Errorf("own host: got %d", rec.Code)
	}
}

// sdd-ui AC-2: security headers, no CORS, no external resources, escaped handbook.
func TestSecurityHeadersAndNoExternalResources(t *testing.T) {
	s := server(project(t))
	for _, path := range []string{"/", "/app.js", "/api/meta", "/handbook"} {
		rec := do(t, s, http.MethodGet, path, "", withHeader("Origin", "http://evil.example"))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d", path, rec.Code)
		}
		hdr := rec.Header()
		if !strings.Contains(hdr.Get("Content-Security-Policy"), "default-src 'self'") || hdr.Get("X-Content-Type-Options") != "nosniff" || hdr.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: missing security headers %v", path, hdr)
		}
		if hdr.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("%s: CORS header present", path)
		}
	}
	external := regexp.MustCompile(`(?i)(?:https?:)?//[a-z0-9.-]+\.[a-z]{2,}`)
	err := fs.WalkDir(webFS, "web", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := fs.ReadFile(webFS, p)
		if m := external.FindString(string(data)); m != "" {
			t.Errorf("%s references an external resource: %s", p, m)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	hb := do(t, s, http.MethodGet, "/handbook", "").Body.String()
	if !strings.Contains(hb, `id="sdd-022"`) || !strings.Contains(hb, `id="sdd-022-vertrag"`) {
		t.Errorf("handbook anchors missing")
	}
}

func TestRenderMarkdownEscapes(t *testing.T) {
	out := RenderMarkdown([]byte("# Titel <x> {#t}\n\nText <script>alert(1)</script> [js](javascript:alert(1)) [ok](#t) `<b>`\n"))
	for _, bad := range []string{"<script>", "javascript:", "<x>", "<b>"} {
		if strings.Contains(out, bad) {
			t.Errorf("unescaped %q in %s", bad, out)
		}
	}
	if !strings.Contains(out, `<h1 id="t">`) || !strings.Contains(out, `<a href="#t">ok</a>`) {
		t.Errorf("unexpected rendering %s", out)
	}
}

// sdd-ui AC-3: writes need token, own Origin and JSON; reads need no token.
func TestWritesNeedTokenAndOrigin(t *testing.T) {
	root := project(t)
	s := server(root)
	body := `{"feature":"x","apply":true}`
	cases := map[string][]reqOpt{
		"no token":       {withoutHeader("X-SDD-Token")},
		"wrong token":    {withHeader("X-SDD-Token", "nope")},
		"no origin":      {withoutHeader("Origin")},
		"foreign origin": {withHeader("Origin", "http://evil.example")},
	}
	for name, opts := range cases {
		if rec := post(t, s, "/api/new", body, opts...); rec.Code != http.StatusForbidden {
			t.Errorf("%s: got %d", name, rec.Code)
		}
	}
	if rec := post(t, s, "/api/new", body, withHeader("Content-Type", "text/plain")); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("form-like post: got %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/x.md")); err == nil {
		t.Fatalf("a rejected request wrote a file")
	}
	if rec := do(t, s, http.MethodGet, "/api/findings", ""); rec.Code != http.StatusOK {
		t.Errorf("read without token: %d", rec.Code)
	}
}

// sdd-ui AC-4: overview, artifacts with status/age/deadline, findings with gate filter.
func TestReadEndpoints(t *testing.T) {
	root := project(t)
	write(t, root, "docs/prd/billing.md", "---\nfeature: billing\nstatus: active\narchive-after: 2026-01-31\n---\n## Akzeptanzkriterien\n- ohne Kennung\n")
	s := server(root)

	var o map[string]any
	decodeBody(t, do(t, s, http.MethodGet, "/api/overview", ""), &o)
	if _, ok := o["maturity"]; !ok || len(o["gates"].([]any)) != 3 {
		t.Errorf("overview %v", o)
	}

	var arts []artifactView
	decodeBody(t, do(t, s, http.MethodGet, "/api/artifacts", ""), &arts)
	found := false
	for _, a := range arts {
		if a.Path == "docs/prd/billing.md" {
			found = a.Kind == "PRD" && a.Status == "active" && a.ArchiveAfter == "2026-01-31" && a.AgeDays >= 0
		}
	}
	if !found {
		t.Errorf("artifact row missing or incomplete: %+v", arts)
	}

	var rep struct {
		Gate     string `json:"gate"`
		Findings []struct {
			RuleID      string `json:"rule_id"`
			Status      string `json:"status"`
			HandbookURL string `json:"handbook_url"`
		} `json:"findings"`
	}
	decodeBody(t, do(t, s, http.MethodGet, "/api/findings?gate=pre-merge", ""), &rep)
	if rep.Gate != "pre-merge" || len(rep.Findings) == 0 || rep.Findings[0].HandbookURL == "" || rep.Findings[0].Status == "" {
		t.Errorf("findings %+v", rep)
	}
	if rec := do(t, s, http.MethodGet, "/api/findings?gate=bogus", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid gate: %d", rec.Code)
	}
}

// sdd-ui AC-5: "new" previews files with content and writes only on apply, never overwriting.
func TestNewPreviewThenApply(t *testing.T) {
	root := project(t)
	s := server(root)
	var preview struct {
		Applied bool     `json:"applied"`
		Ops     []opView `json:"ops"`
	}
	rec := post(t, s, "/api/new", `{"feature":"billing","touches":["cmd/**"],"apply":false}`)
	decodeBody(t, rec, &preview)
	if rec.Code != http.StatusOK || preview.Applied || len(preview.Ops) != 2 || !strings.Contains(preview.Ops[0].Content, "feature: billing") {
		t.Fatalf("preview %d %+v", rec.Code, preview)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err == nil {
		t.Fatalf("preview wrote a file")
	}
	if rec := post(t, s, "/api/new", `{"feature":"billing","apply":true}`); rec.Code != http.StatusOK {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "docs/prd/billing.md")); err != nil {
		t.Fatalf("apply did not write: %v", err)
	}
	decodeBody(t, post(t, s, "/api/new", `{"feature":"billing","apply":false}`), &preview)
	for _, op := range preview.Ops {
		if op.Action != "skip" {
			t.Errorf("existing artifacts must be skipped: %+v", op)
		}
	}
	if rec := post(t, s, "/api/new", `{"feature":"Bad Slug","apply":true}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid input: %d", rec.Code)
	}
}

// sdd-ui AC-5: the migration assistant previews every step and applies on a new branch only.
func TestAdoptPreviewThenApply(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	write(t, root, "specs/features/001-billing/spec.md", "# Billing\n")
	write(t, root, "specs/features/001-billing/plan.md", "# Plan\n")
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "init")
	s := server(root)

	var a struct {
		Steps []stepView `json:"steps"`
	}
	decodeBody(t, do(t, s, http.MethodGet, "/api/adopt", ""), &a)
	titles := []string{}
	for _, st := range a.Steps {
		titles = append(titles, st.Title)
	}
	if strings.Join(titles, ",") != "Konfiguration,Verfassung,Frontmatter" || a.Steps[2].Ops[0].Action != "prepend" || a.Steps[2].Ops[0].Content == "" {
		t.Fatalf("preview steps %+v", a.Steps)
	}
	if git(t, root, "status", "--porcelain") != "" {
		t.Fatalf("preview changed the worktree")
	}

	rec := post(t, s, "/api/adopt/apply", `{"branch":"sdd/adopt-ui"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	if got := git(t, root, "rev-parse", "--abbrev-ref", "HEAD"); got != "sdd/adopt-ui" {
		t.Errorf("branch %q", got)
	}
	git(t, root, "switch", "-q", "main")
	if rec := post(t, s, "/api/adopt/apply", `{"branch":"sdd/adopt-ui"}`); rec.Code != http.StatusConflict {
		t.Errorf("existing branch: %d", rec.Code)
	}
}

// sdd-ui AC-5: there is no endpoint that edits the baseline or the ADR lockfile.
func TestNoBaselineEndpoint(t *testing.T) {
	s := server(project(t))
	for _, path := range []string{"/api/baseline", "/api/baseline/accept", "/api/adr-lock"} {
		if rec := post(t, s, path, `{}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
	src, _ := os.ReadFile("server.go")
	if strings.Contains(strings.ToLower(string(src)), "baseline") {
		t.Errorf("server routes mention the baseline")
	}
}

// sdd-ui AC-6: findings are computed per request from disk, so polling sees changes without reload.
func TestFindingsReflectDiskChanges(t *testing.T) {
	root := project(t)
	s := server(root)
	count := func() int {
		var rep struct {
			Findings []any `json:"findings"`
		}
		decodeBody(t, do(t, s, http.MethodGet, "/api/findings", ""), &rep)
		return len(rep.Findings)
	}
	before := count()
	write(t, root, "docs/prd/x.md", "---\nfeature: x\n---\n## Akzeptanzkriterien\n- ohne Kennung\n")
	if after := count(); after <= before {
		t.Fatalf("findings did not change: %d -> %d", before, after)
	}
	js, _ := fs.ReadFile(webFS, "web/app.js")
	if !strings.Contains(string(js), "setInterval(") || !strings.Contains(string(js), `cache: "no-store"`) {
		t.Errorf("app.js does not poll uncached")
	}
}

// sdd-ui AC-7: the command palette lists every view; $EDITOR opens only files inside the repository.
func TestPaletteReachesAllViews(t *testing.T) {
	js, _ := fs.ReadFile(webFS, "web/app.js")
	html, _ := fs.ReadFile(webFS, "web/index.html")
	for _, v := range []string{"overview", "artifacts", "findings", "new", "adopt"} {
		if !regexp.MustCompile(`\b`+v+`: \{ title:`).Match(js) || !strings.Contains(string(html), `data-view="`+v+`"`) {
			t.Errorf("view %s missing in palette or navigation", v)
		}
	}
	if !strings.Contains(string(js), "Object.entries(views).map") || !strings.Contains(string(js), `e.key.toLowerCase() === "k"`) {
		t.Errorf("palette not built from all views or Ctrl+K missing")
	}
}

// sdd-ui AC-7: opening a finding in $EDITOR is token-protected and limited to files in the repository.
func TestOpenStaysInRepository(t *testing.T) {
	root := project(t)
	s := server(root)
	var called []string
	s.Exec = func(name string, args ...string) error { called = append([]string{name}, args...); return nil }
	t.Setenv("EDITOR", "nvim")

	for _, body := range []string{`{"path":"../etc/passwd","line":1}`, `{"path":"/etc/passwd","line":1}`, `{"path":"missing.md","line":1}`, `{"path":"docs","line":1}`} {
		if rec := post(t, s, "/api/open", body); rec.Code == http.StatusNoContent {
			t.Errorf("%s: accepted", body)
		}
	}
	if rec := post(t, s, "/api/open", `{"path":"constitution.md","line":3}`, withoutHeader("X-SDD-Token")); rec.Code != http.StatusForbidden {
		t.Errorf("without token: %d", rec.Code)
	}
	if called != nil {
		t.Fatalf("editor started for a rejected request: %v", called)
	}
	if rec := post(t, s, "/api/open", `{"path":"constitution.md","line":3}`); rec.Code != http.StatusNoContent {
		t.Fatalf("open: %d %s", rec.Code, rec.Body.String())
	}
	if len(called) != 3 || called[0] != "nvim" || called[1] != "+3" || called[2] != filepath.Join(root, "constitution.md") {
		t.Errorf("editor call %v", called)
	}
	t.Setenv("EDITOR", "")
	if rec := post(t, s, "/api/open", `{"path":"constitution.md","line":3}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty $EDITOR: %d", rec.Code)
	}
}
