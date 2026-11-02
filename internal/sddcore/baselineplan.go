package sddcore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adrlock"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

// ADRLockRule is quitted through the ADR lockfile instead of baseline entries (`sdd baseline --accept SDD-011`).
const ADRLockRule = "SDD-011"

// BaselineOptions are the options of `sdd baseline`.
type BaselineOptions struct {
	Gate   string // only this gate; "" = all
	Prune  bool   // drop entries that no longer occur
	Accept string // add only new findings of this rule; SDD-011 re-accepts changed ADRs
}

// BaselinePlan is what `sdd baseline` would write: the new baseline and lockfile, and what changes.
type BaselinePlan struct {
	File       *baseline.File
	Added      []baseline.Entry
	Removed    []baseline.Entry
	Lock       *adrlock.File
	WriteLock  bool
	Locked     []string // ADRs locked for the first time
	Reaccepted []string // locked ADRs whose changed hash is acknowledged
}

// PlanBaseline computes `sdd baseline` without writing anything. Rules with `baseline: forbidden` cannot be accepted
// (SDD-011 is acknowledged through the lockfile instead).
func PlanBaseline(root string, opts BaselineOptions) (*BaselinePlan, error) {
	if opts.Gate != "" && !validGate(opts.Gate) {
		return nil, catalog.InvalidGateError(opts.Gate)
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	cat, _, err := catalog.Load(root)
	if err != nil {
		return nil, err
	}
	if opts.Accept != "" {
		var found *catalog.Rule
		for i := range cat.Rules {
			if cat.Rules[i].ID == opts.Accept {
				found = &cat.Rules[i]
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("unbekannte Regel %q", opts.Accept)
		}
		if found.Baseline == "forbidden" && found.ID != ADRLockRule {
			return nil, fmt.Errorf("Regel %s trägt baseline: forbidden und darf nicht unterdrückt werden", opts.Accept)
		}
	}

	gates := catalog.AllGates
	if opts.Gate != "" {
		gates = []string{opts.Gate}
	}
	current, err := BaselineEntries(root, cfg, gates)
	if err != nil {
		return nil, err
	}
	if opts.Accept != "" {
		var filtered []baseline.Entry
		for _, e := range current {
			if e.RuleID == opts.Accept {
				filtered = append(filtered, e)
			}
		}
		current = filtered
	}
	existing, hasExisting, err := baseline.Load(root)
	if err != nil {
		return nil, err
	}

	var result []baseline.Entry
	switch {
	case opts.Prune:
		if !hasExisting {
			result = []baseline.Entry{}
		} else {
			active := map[string]bool{}
			for _, e := range current {
				active[entryKey(e)] = true
			}
			for _, e := range existing.Entries {
				if (opts.Gate != "" && e.Gate != opts.Gate) || active[entryKey(e)] {
					result = append(result, e)
				}
			}
		}
	case opts.Accept != "" && hasExisting:
		result = baseline.MergeEntries(existing.Entries, current)
	case opts.Gate != "" && hasExisting:
		var kept []baseline.Entry
		for _, e := range existing.Entries {
			if e.Gate != opts.Gate {
				kept = append(kept, e)
			}
		}
		result = baseline.MergeEntries(kept, current)
	default:
		result = baseline.MergeEntries(nil, current)
	}

	plan := &BaselinePlan{File: &baseline.File{Version: 1, Entries: result}}
	var before []baseline.Entry
	if hasExisting {
		before = existing.Entries
	}
	plan.Added, plan.Removed = entryDiff(before, result)

	lock, hasLock, err := adrlock.Load(root)
	if err != nil {
		return nil, err
	}
	adrs, err := adrlock.Collect(root, cfg)
	if err != nil {
		return nil, err
	}
	old := map[string]string{}
	for k, v := range lock.Entries {
		old[k] = v
	}
	added, updated := adrlock.Update(lock, adrs, opts.Accept == ADRLockRule)
	for path, hash := range lock.Entries {
		prev, was := old[path]
		switch {
		case !was:
			plan.Locked = append(plan.Locked, path)
		case prev != hash:
			plan.Reaccepted = append(plan.Reaccepted, path)
		}
	}
	sort.Strings(plan.Locked)
	sort.Strings(plan.Reaccepted)
	plan.Lock, plan.WriteLock = lock, !hasLock || added > 0 || updated > 0
	return plan, nil
}

// Apply writes the baseline and, if it changed or did not exist, the lockfile.
func (p *BaselinePlan) Apply(root string) error {
	if err := baseline.Save(root, p.File); err != nil {
		return err
	}
	if !p.WriteLock {
		return nil
	}
	data, err := p.Lock.Marshal()
	if err == nil {
		err = os.WriteFile(filepath.Join(root, filepath.FromSlash(adrlock.RelPath)), data, 0o644)
	}
	if err != nil {
		return fmt.Errorf("%s nicht schreibbar: %w", adrlock.RelPath, err)
	}
	return nil
}

// Summary names the effect in one line, e.g. for the button of a confirmation.
func (p *BaselinePlan) Summary() string {
	count := func(entries []baseline.Entry) string {
		by := map[string]int{}
		var ids []string
		for _, e := range entries {
			if by[e.RuleID] == 0 {
				ids = append(ids, e.RuleID)
			}
			by[e.RuleID]++
		}
		sort.Strings(ids)
		var parts []string
		for _, id := range ids {
			parts = append(parts, fmt.Sprintf("%s ×%d", id, by[id]))
		}
		if len(parts) == 0 {
			return ""
		}
		return " (" + strings.Join(parts, ", ") + ")"
	}
	s := fmt.Sprintf("Baseline: %d Einträge aufnehmen%s, %d entfernen%s", len(p.Added), count(p.Added), len(p.Removed), count(p.Removed))
	if len(p.Locked) > 0 || len(p.Reaccepted) > 0 {
		s += fmt.Sprintf("; ADR-Lockfile: %d neu locken, %d Änderungen quittieren", len(p.Locked), len(p.Reaccepted))
	}
	return s
}

func entryKey(e baseline.Entry) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", e.Gate, e.RuleID, e.Path, e.ContentHash, e.Ordinal)
}

// entryDiff returns the entries only in after (added) and only in before (removed).
func entryDiff(before, after []baseline.Entry) (added, removed []baseline.Entry) {
	b, a := map[string]bool{}, map[string]bool{}
	for _, e := range before {
		b[entryKey(e)] = true
	}
	for _, e := range after {
		a[entryKey(e)] = true
		if !b[entryKey(e)] {
			added = append(added, e)
		}
	}
	for _, e := range before {
		if !a[entryKey(e)] {
			removed = append(removed, e)
		}
	}
	return added, removed
}

func validGate(g string) bool {
	for _, v := range catalog.AllGates {
		if v == g {
			return true
		}
	}
	return false
}
