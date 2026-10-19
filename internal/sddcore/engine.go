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

	cat, catPath, err := catalog.Load(opts.RepoRoot)
	if err != nil {
		return nil, err
	}

	bf, hasBaseline, err := baseline.Load(opts.RepoRoot)
	if err != nil {
		return nil, err
	}

	reg := rules.NewRegistry()
	ctx := &rules.Context{
		RepoRoot:    opts.RepoRoot,
		Config:      cfg,
		Catalog:     cat,
		CatalogPath: catPath,
		Gate:        opts.Gate,
		Feature:     opts.Feature,
		Paths:       opts.Paths,
	}

	findings, err := rules.Evaluate(ctx, reg)
	if err != nil {
		return nil, fmt.Errorf("Fehler bei der Regelprüfung: %w", err)
	}

	suppressed := 0
	if hasBaseline {
		findings, suppressed = baseline.Apply(bf, cat, opts.Gate, findings)
	}

	rep := report.Build(cat.Version, cfg.Profile, opts.Gate, opts.Feature, hasBaseline, suppressed, findings)
	return &rep, nil
}
