package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

const DefaultBaselineRelPath = ".sdd/baseline.json"

type File struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Gate        string `json:"gate"`
	RuleID      string `json:"rule_id"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	Ordinal     int    `json:"ordinal"`
	Line        int    `json:"line,omitempty"`
}

type identityKey struct {
	gate        string
	ruleID      string
	path        string
	contentHash string
	ordinal     int
}

// Load reads .sdd/baseline.json if present. Returns (nil, false, nil) if the file does not exist.
func Load(repoRoot string) (*File, bool, error) {
	p := filepath.Join(repoRoot, filepath.FromSlash(DefaultBaselineRelPath))
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Baseline %s nicht lesbar: %w", DefaultBaselineRelPath, err)
	}
	var bf File
	if err := json.Unmarshal(data, &bf); err != nil {
		return nil, false, fmt.Errorf("Baseline %s ungültig: %w", DefaultBaselineRelPath, err)
	}
	if bf.Version != 1 {
		return nil, false, fmt.Errorf("ungültige Baseline-Version %d in %s", bf.Version, DefaultBaselineRelPath)
	}
	return &bf, true, nil
}

// Save writes .sdd/baseline.json deterministically sorted.
func Save(repoRoot string, bf *File) error {
	bf.Version = 1
	sortEntries(bf.Entries)
	p := filepath.Join(repoRoot, filepath.FromSlash(DefaultBaselineRelPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("Verzeichnis .sdd konnte nicht angelegt werden: %w", err)
	}
	data, err := json.MarshalIndent(bf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p, data, 0o644)
}

// Apply marks matching error findings as "baselined" (unless the rule has baseline: forbidden).
// Returns the updated findings slice and the count of suppressed error findings.
func Apply(bf *File, cat *catalog.Catalog, gate string, findings []report.Finding) ([]report.Finding, int) {
	if bf == nil || len(bf.Entries) == 0 {
		return findings, 0
	}

	forbiddenRules := make(map[string]bool)
	for _, r := range cat.Rules {
		if r.Baseline == "forbidden" {
			forbiddenRules[r.ID] = true
		}
	}

	exactGateSet := make(map[identityKey]bool, len(bf.Entries))
	anyGateSet := make(map[identityKey]bool, len(bf.Entries))
	for _, e := range bf.Entries {
		exactGateSet[identityKey{
			gate:        e.Gate,
			ruleID:      e.RuleID,
			path:        e.Path,
			contentHash: e.ContentHash,
			ordinal:     e.Ordinal,
		}] = true
		anyGateSet[identityKey{
			gate:        "",
			ruleID:      e.RuleID,
			path:        e.Path,
			contentHash: e.ContentHash,
			ordinal:     e.Ordinal,
		}] = true
	}

	ordered := report.AssignOrdinalsAndSort(findings)
	suppressed := 0
	for i := range ordered {
		f := &ordered[i]
		if f.Severity != "error" || forbiddenRules[f.RuleID] {
			continue
		}
		var matched bool
		if gate != "" {
			matched = exactGateSet[identityKey{
				gate:        gate,
				ruleID:      f.RuleID,
				path:        f.Path,
				contentHash: f.ContentHash,
				ordinal:     f.Ordinal,
			}]
		} else {
			matched = anyGateSet[identityKey{
				gate:        "",
				ruleID:      f.RuleID,
				path:        f.Path,
				contentHash: f.ContentHash,
				ordinal:     f.Ordinal,
			}]
		}
		if matched {
			f.Status = "baselined"
			suppressed++
		}
	}
	return ordered, suppressed
}

// BuildEntriesFromFindings extracts baselineable error findings for a specific gate.
func BuildEntriesFromFindings(cat *catalog.Catalog, gate string, findings []report.Finding) []Entry {
	allowedRules := make(map[string]bool)
	for _, r := range cat.Rules {
		if r.Baseline == "allowed" {
			allowedRules[r.ID] = true
		}
	}

	ordered := report.AssignOrdinalsAndSort(findings)
	var entries []Entry
	for _, f := range ordered {
		if f.Severity != "error" || !allowedRules[f.RuleID] {
			continue
		}
		entries = append(entries, Entry{
			Gate:        gate,
			RuleID:      f.RuleID,
			Path:        f.Path,
			ContentHash: f.ContentHash,
			Ordinal:     f.Ordinal,
			Line:        f.Line,
		})
	}
	sortEntries(entries)
	return entries
}

// MergeEntries combines existing entries with new entries without duplicates.
func MergeEntries(existing, added []Entry) []Entry {
	seen := make(map[identityKey]Entry, len(existing)+len(added))
	for _, e := range existing {
		k := identityKey{gate: e.Gate, ruleID: e.RuleID, path: e.Path, contentHash: e.ContentHash, ordinal: e.Ordinal}
		seen[k] = e
	}
	for _, e := range added {
		k := identityKey{gate: e.Gate, ruleID: e.RuleID, path: e.Path, contentHash: e.ContentHash, ordinal: e.Ordinal}
		seen[k] = e
	}
	out := make([]Entry, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	sortEntries(out)
	return out
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Gate != entries[j].Gate {
			return entries[i].Gate < entries[j].Gate
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].RuleID != entries[j].RuleID {
			return entries[i].RuleID < entries[j].RuleID
		}
		if entries[i].Ordinal != entries[j].Ordinal {
			return entries[i].Ordinal < entries[j].Ordinal
		}
		return entries[i].ContentHash < entries[j].ContentHash
	})
}
