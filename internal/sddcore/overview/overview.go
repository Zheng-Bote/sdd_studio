// Package overview builds the project report of `sdd report`: maturity, findings, gates and baseline debt with trend.
package overview

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adopt"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// Git is the read-only access the report needs to find and read the comparison baseline.
type Git interface {
	ResolveBaseRef(configured string) (string, error)
	ShowFile(ref, path string) ([]byte, bool, error)
}

// Counts summarizes findings of one run.
type Counts struct {
	Error     int `json:"error"`
	Warn      int `json:"warn"`
	Info      int `json:"info"`
	Baselined int `json:"baselined"`
}

// GateStatus is the result of one gate, or the reason it could not run.
type GateStatus struct {
	Gate   string `json:"gate"`
	Counts Counts `json:"counts"`
	Error  string `json:"error,omitempty"`
}

// DebtRow counts baseline entries of one rule in one gate, now and at the comparison point.
type DebtRow struct {
	RuleID string `json:"rule_id"`
	Gate   string `json:"gate"`
	Now    int    `json:"now"`
	Before int    `json:"before"`
}

// Overview is the complete report.
type Overview struct {
	Profile    string           `json:"profile"`
	Maturity   adopt.Maturity   `json:"maturity"`
	Lint       Counts           `json:"lint"`
	Findings   []report.Finding `json:"findings"`
	Gates      []GateStatus     `json:"gates"`
	Since      string           `json:"since,omitempty"`
	SinceError string           `json:"since_error,omitempty"`
	Debt       []DebtRow        `json:"debt"`
	DebtNow    int              `json:"debt_now"`
	DebtBefore int              `json:"debt_before"`
	Added      []baseline.Entry `json:"added"`
}

// Grew reports whether the baseline gained entries since the comparison point (ratchet violation).
func (o *Overview) Grew() bool {
	return len(o.Added) > 0
}

func counts(r *report.Report) Counts {
	return Counts{Error: r.Summary.Error, Warn: r.Summary.Warn, Info: r.Summary.Info, Baselined: r.Summary.Suppressed}
}

// Build assembles the report. since is the comparison ref; empty means the merge base (base_ref).
// Only a missing or invalid configuration or an unreadable current baseline is an error.
func Build(root, since string, git Git) (*Overview, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	o := &Overview{Profile: cfg.Profile, Findings: []report.Finding{}, Debt: []DebtRow{}, Added: []baseline.Entry{}}

	lint, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root})
	if err != nil {
		return nil, err
	}
	o.Lint = counts(lint)
	for _, f := range lint.Findings {
		if f.Status != "baselined" && (f.Severity == "error" || f.Severity == "warn") {
			o.Findings = append(o.Findings, f)
		}
	}

	inv, err := adopt.Take(root, cfg.Profile)
	if err != nil {
		return nil, err
	}
	o.Maturity = adopt.Assess(root, inv, lint.Findings)

	for _, g := range catalog.AllGates {
		st := GateStatus{Gate: g}
		if rep, err := sddcore.RunCheck(sddcore.RunOptions{RepoRoot: root, Gate: g}); err != nil {
			st.Error = err.Error()
		} else {
			st.Counts = counts(rep)
		}
		o.Gates = append(o.Gates, st)
	}

	current, _, err := baseline.Load(root)
	if err != nil {
		return nil, err
	}
	var before *baseline.File
	if since == "" {
		if since, err = git.ResolveBaseRef(cfg.BaseRef); err != nil {
			o.SinceError = err.Error()
		}
	}
	if since != "" {
		o.Since = since
		data, ok, err := git.ShowFile(since, baseline.DefaultBaselineRelPath)
		switch {
		case err != nil:
			o.SinceError = err.Error()
		case ok:
			var bf baseline.File
			if err := json.Unmarshal(data, &bf); err != nil {
				o.SinceError = fmt.Sprintf("Baseline am Vergleichsstand %s ungültig: %v", since, err)
			} else {
				before = &bf
			}
		default:
			before = &baseline.File{Version: 1}
		}
	}
	o.debt(current, before)
	return o, nil
}

type entryKey struct {
	gate, rule, path, hash string
	ordinal                int
}

func keyOf(e baseline.Entry) entryKey {
	return entryKey{e.Gate, e.RuleID, e.Path, e.ContentHash, e.Ordinal}
}

// debt fills the per-rule table and the entries added since the comparison point (nil before: trend unknown).
func (o *Overview) debt(current, before *baseline.File) {
	type rowKey struct{ rule, gate string }
	rows := map[rowKey]*DebtRow{}
	row := func(e baseline.Entry) *DebtRow {
		k := rowKey{e.RuleID, e.Gate}
		if rows[k] == nil {
			rows[k] = &DebtRow{RuleID: e.RuleID, Gate: e.Gate}
		}
		return rows[k]
	}
	known := map[entryKey]bool{}
	if before != nil {
		for _, e := range before.Entries {
			row(e).Before++
			known[keyOf(e)] = true
			o.DebtBefore++
		}
	}
	if current != nil {
		for _, e := range current.Entries {
			row(e).Now++
			o.DebtNow++
			if before != nil && !known[keyOf(e)] {
				o.Added = append(o.Added, e)
			}
		}
	}
	for _, r := range rows {
		o.Debt = append(o.Debt, *r)
	}
	sort.Slice(o.Debt, func(i, j int) bool {
		if o.Debt[i].RuleID != o.Debt[j].RuleID {
			return o.Debt[i].RuleID < o.Debt[j].RuleID
		}
		return o.Debt[i].Gate < o.Debt[j].Gate
	})
}
