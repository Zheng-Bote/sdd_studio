package sddcore

import (
	"fmt"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/baseline"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
)

type RunOptions struct {
	RepoRoot string
	Gate     string
	Feature  string
	Paths    []string
}

// RunCheck loads config, catalog, and baseline, evaluates all active rules, and returns the report.
// Any configuration, catalog, or baseline load failure returns an error (CLI exits with code 2).
func RunCheck(opts RunOptions) (*report.Report, error) {
	cfg, err := config.Load(opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	return RunCheckWithConfig(opts, cfg)
}

// RunCheckWithConfig is RunCheck with an already loaded (or proposed) configuration.
func RunCheckWithConfig(opts RunOptions, cfg *config.Config) (*report.Report, error) {
	cat, findings, err := evaluate(opts, cfg)
	if err != nil {
		return nil, err
	}

	bf, hasBaseline, err := baseline.Load(opts.RepoRoot)
	if err != nil {
		return nil, err
	}

	suppressed := 0
	if hasBaseline {
		findings, suppressed = baseline.Apply(bf, cat, opts.Gate, findings)
	}

	rep := report.Build(cat.Version, cfg.Profile, opts.Gate, opts.Feature, hasBaseline, suppressed, findings)
	return &rep, nil
}

// BaselineEntries evaluates the given gates without applying an existing baseline and returns the baselineable entries.
func BaselineEntries(repoRoot string, cfg *config.Config, gates []string) ([]baseline.Entry, error) {
	var entries []baseline.Entry
	for _, g := range gates {
		cat, findings, err := evaluate(RunOptions{RepoRoot: repoRoot, Gate: g}, cfg)
		if err != nil {
			return nil, err
		}
		entries = append(entries, baseline.BuildEntriesFromFindings(cat, g, findings)...)
	}
	return baseline.MergeEntries(nil, entries), nil
}

func evaluate(opts RunOptions, cfg *config.Config) (*catalog.Catalog, []report.Finding, error) {
	cat, catPath, err := catalog.Load(opts.RepoRoot)
	if err != nil {
		return nil, nil, err
	}
	ctx := &rules.Context{
		RepoRoot:    opts.RepoRoot,
		Config:      cfg,
		Catalog:     cat,
		CatalogPath: catPath,
		Gate:        opts.Gate,
		Feature:     opts.Feature,
		Paths:       opts.Paths,
	}
	findings, err := rules.Evaluate(ctx, rules.NewRegistry())
	if err != nil {
		return nil, nil, fmt.Errorf("Fehler bei der Regelprüfung: %w", err)
	}
	return cat, findings, nil
}
