package adopt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adrlock"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// ErrPrecondition marks a refused apply (not a Git repository, dirty worktree, detached HEAD, existing branch).
var ErrPrecondition = errors.New("Voraussetzung für sdd adopt --apply nicht erfüllt")

// Commit records one applied step.
type Commit struct {
	Step  string
	Hash  string
	Paths []string
}

// Result is the outcome of an applied take-over.
type Result struct {
	FromBranch      string
	Branch          string
	Commits         []Commit
	BaselineEntries int
	Skipped         []scaffold.Result
}

// Apply executes the proposal on a new branch with one commit per step (ADR 0001). On error the branch keeps the
// commits completed so far; Result lists them.
func Apply(root string, p *Proposal, git scan.GitWriter) (*Result, error) {
	branch := p.Options.Branch
	if !git.IsRepo() {
		return nil, fmt.Errorf("%w: kein Git-Repository", ErrPrecondition)
	}
	dirty, err := git.DirtyPaths()
	if err != nil {
		return nil, err
	}
	var relevant []string
	for _, d := range dirty {
		if !scan.MatchAny(p.Inventory.Config.Exclude, d) {
			relevant = append(relevant, d)
		}
	}
	if len(relevant) > 0 {
		return nil, fmt.Errorf("%w: Arbeitsbaum nicht sauber (%s)", ErrPrecondition, strings.Join(relevant, ", "))
	}
	from, err := git.CurrentBranch()
	if err != nil {
		return nil, err
	}
	if from == "" {
		return nil, fmt.Errorf("%w: HEAD ist losgelöst; zuerst einen Branch auschecken", ErrPrecondition)
	}
	exists, err := git.BranchExists(branch)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: Branch %s existiert bereits", ErrPrecondition, branch)
	}
	if err := git.CreateBranch(branch); err != nil {
		return nil, err
	}

	res := &Result{FromBranch: from, Branch: branch}
	commitStep := func(title string, plan scaffold.Plan) error {
		results, err := plan.Apply(root)
		if err != nil {
			return fmt.Errorf("Schritt %s: %w", title, err)
		}
		var paths []string
		for _, r := range results {
			if r.Status == scaffold.StatusSkipped {
				res.Skipped = append(res.Skipped, r)
				continue
			}
			paths = append(paths, r.Path)
		}
		if len(paths) == 0 {
			return nil
		}
		hash, err := git.Commit("sdd adopt: "+title, paths)
		if err != nil {
			return fmt.Errorf("Schritt %s: %w", title, err)
		}
		res.Commits = append(res.Commits, Commit{Step: title, Hash: hash, Paths: paths})
		return nil
	}

	for _, s := range p.Steps {
		if err := commitStep(s.Title, s.Plan); err != nil {
			return res, err
		}
	}

	// The baseline is recorded after configuration and frontmatter, before any gate runs (concept section 6).
	if !p.Inventory.BaselineExists {
		entries, err := sddcore.BaselineEntries(root, p.Inventory.Config, catalog.AllGates)
		if err != nil {
			return res, fmt.Errorf("Schritt Baseline: %w", err)
		}
		if entries == nil {
			entries = []baseline.Entry{}
		}
		data, err := json.MarshalIndent(baseline.File{Version: 1, Entries: entries}, "", "  ")
		if err != nil {
			return res, err
		}
		res.BaselineEntries = len(entries)
		ops := []scaffold.Op{{Path: baseline.DefaultBaselineRelPath, Content: append(data, '\n')}}
		// The ADR lockfile (SDD-011) belongs to the same recorded state; an existing one is never touched.
		lock, hasLock, err := adrlock.Load(root)
		if err != nil {
			return res, fmt.Errorf("Schritt Baseline: %w", err)
		}
		if !hasLock {
			adrs, err := adrlock.Collect(root, p.Inventory.Config)
			if err != nil {
				return res, fmt.Errorf("Schritt Baseline: %w", err)
			}
			adrlock.Update(lock, adrs, false)
			lockData, err := lock.Marshal()
			if err != nil {
				return res, err
			}
			ops = append(ops, scaffold.Op{Path: adrlock.RelPath, Content: lockData})
		}
		if err := commitStep("Baseline", scaffold.Plan{Ops: ops}); err != nil {
			return res, err
		}
	}

	content := RenderReport(p, res)
	if err := commitStep("Bericht", scaffold.Plan{Ops: []scaffold.Op{{Path: ReportPath, Content: []byte(content)}}}); err != nil {
		return res, err
	}
	return res, nil
}
