package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	sddstudio "github.com/Zheng-Bote/sdd_studio"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adopt"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/overview"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	abs, _ := filepath.Abs(s.Root)
	writeJSON(w, http.StatusOK, map[string]any{
		"project": s.Project,
		"root":    abs,
		"editor":  strings.TrimSpace(os.Getenv("EDITOR")) != "",
		"gates":   catalog.AllGates,
	})
}

// handleOverview is `sdd report --format json`.
func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	o, err := overview.Build(s.Root, "", scan.CLIGit{Root: s.Root})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// handleFindings is `sdd lint --json` or `sdd gate <gate> --json`; never cached, so the view stays live.
func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	gate := r.URL.Query().Get("gate")
	if gate != "" && !validGate(gate) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("ungültiges Gate %q", gate))
		return
	}
	rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: s.Root, Gate: gate})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func validGate(g string) bool {
	for _, v := range catalog.AllGates {
		if v == g {
			return true
		}
	}
	return false
}

// artifactView is one row of the artifact tree.
type artifactView struct {
	Path         string     `json:"path"`
	Kind         adopt.Kind `json:"kind"`
	Archived     bool       `json:"archived"`
	Feature      string     `json:"feature,omitempty"`
	Status       string     `json:"status,omitempty"`
	ArchiveAfter string     `json:"archive_after,omitempty"`
	AgeDays      int        `json:"age_days"`
	Reason       string     `json:"reason"`
}

func (s *Server) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load(s.Root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	inv, err := adopt.Take(s.Root, cfg.Profile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]artifactView, 0, len(inv.Files))
	for _, f := range inv.Files {
		v := artifactView{Path: f.Path, Kind: f.Kind, Archived: f.Archived, Feature: f.Frontmatter.Feature,
			Status: f.Frontmatter.Status, ArchiveAfter: f.Frontmatter.ArchiveAfter, Reason: f.Reason}
		if info, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(f.Path))); err == nil {
			v.AgeDays = max(0, int(math.Floor(s.Now().Sub(info.ModTime()).Hours()/24))) // clock skew never yields a negative age
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

// opView shows one planned operation including the content to be written.
type opView struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // create, prepend, skip
	Reason  string `json:"reason,omitempty"`
	Content string `json:"content,omitempty"`
}

func planView(root string, plan scaffold.Plan) ([]opView, error) {
	preview, err := plan.Preview(root)
	if err != nil {
		return nil, err
	}
	out := make([]opView, 0, len(plan.Ops))
	for i, op := range plan.Ops {
		v := opView{Path: op.Path, Content: string(op.Content)}
		switch preview[i].Status {
		case scaffold.StatusSkipped:
			v.Action, v.Reason, v.Content = "skip", preview[i].Reason, ""
		case scaffold.StatusPrepended:
			v.Action = "prepend"
		default:
			v.Action = "create"
		}
		out = append(out, v)
	}
	return out, nil
}

type newRequest struct {
	Types        []string `json:"types"`
	Feature      string   `json:"feature"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Touches      []string `json:"touches"`
	ArchiveAfter string   `json:"archive_after"`
	Apply        bool     `json:"apply"`
}

// handleNew is `sdd new` (apply:false is the dry run).
func (s *Server) handleNew(w http.ResponseWriter, r *http.Request) {
	var req newRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cfg, err := config.Load(s.Root)
	if errors.Is(err, config.ErrConfigNotFound) {
		cfg, err = scaffold.DefaultConfig("standard"), nil
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	plan, err := scaffold.NewPlan(s.Root, cfg, scaffold.NewOptions{
		Types: req.Types, Feature: strings.TrimSpace(req.Feature), Title: req.Title, Slug: strings.TrimSpace(req.Slug),
		Touches: req.Touches, ArchiveAfter: strings.TrimSpace(req.ArchiveAfter), Today: s.Now(),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !req.Apply {
		ops, err := planView(s.Root, plan)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"applied": false, "ops": ops})
		return
	}
	results, err := plan.Apply(s.Root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": true, "results": results})
}

func (s *Server) proposal() (*adopt.Proposal, error) {
	profile := "standard"
	if cfg, err := config.Load(s.Root); err == nil {
		profile = cfg.Profile
	}
	return adopt.Propose(s.Root, adopt.Options{Profile: profile, Branch: "sdd/adopt", Project: s.Project, Today: s.Now()})
}

type stepView struct {
	Title  string   `json:"title"`
	Detail []string `json:"detail"`
	Ops    []opView `json:"ops"`
}

// handleAdoptPreview is `sdd adopt` (dry run) with the full content of every planned operation.
func (s *Server) handleAdoptPreview(w http.ResponseWriter, r *http.Request) {
	p, err := s.proposal()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	steps := []stepView{}
	for _, st := range p.Steps {
		ops, err := planView(s.Root, st.Plan)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		steps = append(steps, stepView{Title: st.Title, Detail: st.Detail, Ops: ops})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"report": adopt.RenderReport(p, nil), "maturity": p.Maturity, "steps": steps,
		"conflicts": p.Conflicts, "branch": p.Options.Branch, "baseline_planned": !p.Inventory.BaselineExists,
	})
}

type adoptApplyRequest struct {
	Branch string `json:"branch"`
}

// handleAdoptApply is `sdd adopt --apply --branch <branch>`.
func (s *Server) handleAdoptApply(w http.ResponseWriter, r *http.Request) {
	var req adoptApplyRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.proposal()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if b := strings.TrimSpace(req.Branch); b != "" {
		p.Options.Branch = b
	}
	res, err := adopt.Apply(s.Root, p, scan.CLIGit{Root: s.Root})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, adopt.ErrPrecondition) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"error": err.Error(), "result": res})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type openRequest struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// lineFirstEditors accept `+<line> <file>`.
var lineFirstEditors = map[string]bool{"vi": true, "vim": true, "nvim": true, "nano": true, "emacs": true, "emacsclient": true, "micro": true, "kak": true, "joe": true}

// handleOpen starts $EDITOR for a file inside the repository (no shell; arguments as a list).
func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	var req openRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rel, err := scan.CleanRel(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	abs := filepath.Join(s.Root, filepath.FromSlash(rel))
	if info, err := os.Stat(abs); err != nil || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, fmt.Errorf("%s ist keine Datei im Repository", rel))
		return
	}
	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("$EDITOR ist nicht gesetzt"))
		return
	}
	args := append([]string{}, editor[1:]...)
	if req.Line > 0 && lineFirstEditors[filepath.Base(editor[0])] {
		args = append(args, "+"+strconv.Itoa(req.Line))
	}
	args = append(args, abs)
	if err := s.Exec(editor[0], args...); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// handleHandbook renders the handbook (working tree first, embedded copy as fallback) as HTML.
func (s *Server) handleHandbook(w http.ResponseWriter, r *http.Request) {
	const rel = "handbuch/03-mechanismen.md"
	data, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(rel)))
	if err != nil {
		if data, err = fs.ReadFile(sddstudio.EmbeddedAssets, rel); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html lang=\"de\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>Handbuch — sdd</title><link rel=\"stylesheet\" href=\"/app.css\"></head><body class=\"handbook\"><main class=\"prose\">%s</main></body></html>", RenderMarkdown(data))
}
