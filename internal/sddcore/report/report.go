package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

const (
	SchemaVersion = 1
	ToolVersion   = "0.1.0"
)

type BaselineSummary struct {
	Applied    bool `json:"applied"`
	Suppressed int  `json:"suppressed"`
}

type Summary struct {
	Error      int `json:"error"`
	Warn       int `json:"warn"`
	Info       int `json:"info"`
	Suppressed int `json:"suppressed"`
}

type Finding struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Ordinal     int    `json:"ordinal"`
	Message     string `json:"message"`
	HandbookURL string `json:"handbook_url"`

	// Internal fields for deterministic ordering and baseline identity (not serialized in JSON report)
	ByteOffset  int    `json:"-"`
	ContentHash string `json:"-"`
}

type Report struct {
	SchemaVersion  int             `json:"schema_version"`
	ToolVersion    string          `json:"tool_version"`
	CatalogVersion string          `json:"catalog_version"`
	Profile        string          `json:"profile"`
	Gate           string          `json:"gate,omitempty"`
	Feature        string          `json:"feature,omitempty"`
	Baseline       BaselineSummary `json:"baseline"`
	Findings       []Finding       `json:"findings"`
	Summary        Summary         `json:"summary"`
}

// AssignOrdinalsAndSort sorts findings deterministically by (Path, ByteOffset, RuleID)
// and assigns the 0-based Ordinal per (Path, RuleID) sorted by ascending ByteOffset (section 5.2).
func AssignOrdinalsAndSort(findings []Finding) []Finding {
	if len(findings) == 0 {
		return []Finding{}
	}
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)

	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		if sorted[i].ByteOffset != sorted[j].ByteOffset {
			return sorted[i].ByteOffset < sorted[j].ByteOffset
		}
		if sorted[i].Line != sorted[j].Line {
			return sorted[i].Line < sorted[j].Line
		}
		return sorted[i].RuleID < sorted[j].RuleID
	})

	type fileRuleKey struct {
		path   string
		ruleID string
	}
	counters := make(map[fileRuleKey]int)
	for i := range sorted {
		k := fileRuleKey{path: sorted[i].Path, ruleID: sorted[i].RuleID}
		sorted[i].Ordinal = counters[k]
		counters[k]++
		if sorted[i].Status == "" {
			sorted[i].Status = "new"
		}
	}
	return sorted
}

// Build creates a Report with computed summary counts.
func Build(catalogVersion, profile, gate, feature string, baselineApplied bool, suppressed int, findings []Finding) Report {
	ordered := AssignOrdinalsAndSort(findings)
	var sum Summary
	sum.Suppressed = suppressed
	for _, f := range ordered {
		if f.Status == "baselined" {
			continue
		}
		switch f.Severity {
		case "error":
			sum.Error++
		case "warn":
			sum.Warn++
		case "info":
			sum.Info++
		}
	}
	return Report{
		SchemaVersion:  SchemaVersion,
		ToolVersion:    ToolVersion,
		CatalogVersion: catalogVersion,
		Profile:        profile,
		Gate:           gate,
		Feature:        feature,
		Baseline: BaselineSummary{
			Applied:    baselineApplied,
			Suppressed: suppressed,
		},
		Findings: ordered,
		Summary:  sum,
	}
}

// ExitCode returns 1 if there is at least one new error finding, otherwise 0.
func (r Report) ExitCode() int {
	if r.Summary.Error > 0 {
		return 1
	}
	return 0
}

func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func (r Report) WriteText(w io.Writer) error {
	gateLabel := r.Gate
	if gateLabel == "" {
		gateLabel = "default (lint)"
	}
	if _, err := fmt.Fprintf(w, "SDD Prüflauf — Profil: %s | Gate: %s | Katalog: v%s\n", r.Profile, gateLabel, r.CatalogVersion); err != nil {
		return err
	}
	for _, f := range r.Findings {
		if f.Status == "baselined" {
			continue
		}
		if _, err := fmt.Fprintf(w, "[%s] %s (%s:%d #%d): %s\n", f.Severity, f.RuleID, f.Path, f.Line, f.Ordinal, f.Message); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "Ergebnis: %d error, %d warn, %d info (%d baselined)\n",
		r.Summary.Error, r.Summary.Warn, r.Summary.Info, r.Summary.Suppressed)
	return err
}
