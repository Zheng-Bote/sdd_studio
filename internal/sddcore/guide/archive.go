package guide

import (
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Archive plans `sdd archive`: every acceptance criterion of the feature must be referenced by a test (as SDD-022
// counts it) — a feature without tests is not done and is not archived. Every artifact must be movable; a plan that
// would move only some of them is refused. A feature whose PRD is still a draft is not done either
// (archive-not-draft): archiving it would hide an unfinished feature from sdd next and the checks.
func Archive(root, feature, release string) (scaffold.Plan, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return scaffold.Plan{}, err
	}
	drafts, err := draftPRDs(root, cfg, feature)
	if err != nil {
		return scaffold.Plan{}, err
	}
	if len(drafts) > 0 {
		return scaffold.Plan{}, fmt.Errorf("%w: Feature %s nicht archivierbar: %s hat status: draft — archiviert wird erst nach Umsetzung und Merge (status: implemented), vor dem Release",
			scaffold.ErrInvalidInput, feature, strings.Join(drafts, ", "))
	}
	trace, err := rules.Trace(&rules.Context{RepoRoot: root, Config: cfg}, feature)
	if err != nil {
		return scaffold.Plan{}, err
	}
	var untested []string
	for _, ac := range trace {
		if len(ac.Tests) == 0 {
			untested = append(untested, ac.ID)
		}
	}
	if len(untested) > 0 {
		return scaffold.Plan{}, fmt.Errorf("%w: Feature %s nicht archivierbar: %s ohne Testverweis (SDD-022)", scaffold.ErrInvalidInput, feature, strings.Join(untested, ", "))
	}
	plan, err := scaffold.ArchivePlan(root, cfg, feature, release)
	if err != nil {
		return scaffold.Plan{}, err
	}
	preview, err := plan.Preview(root)
	if err != nil {
		return scaffold.Plan{}, err
	}
	for _, r := range preview {
		if r.Status == scaffold.StatusSkipped {
			return scaffold.Plan{}, fmt.Errorf("%w: %s: %s — nichts verschoben", scaffold.ErrInvalidInput, r.Path, r.Reason)
		}
	}
	return plan, nil
}

// draftPRDs are the PRDs of feature outside the archive with status: draft; a PRD belongs to the feature by feature:
// or, without the field, by its file name, as in scaffold.ArchivePlan.
func draftPRDs(root string, cfg *config.Config, feature string) ([]string, error) {
	list, err := scan.ListArtifacts(root, cfg.Artifacts.PRD, cfg.Exclude)
	if err != nil {
		return nil, err
	}
	archive := strings.TrimSpace(cfg.Artifacts.Archive)
	var drafts []string
	for _, a := range list {
		if archive != "" && scan.InSpec(a.Path, archive) {
			continue
		}
		if f := a.Frontmatter.Feature; f != feature && (f != "" || scan.SlugInSpec(a.Path, cfg.Artifacts.PRD) != feature) {
			continue
		}
		if a.Frontmatter.Status == "draft" {
			drafts = append(drafts, a.Path)
		}
	}
	return drafts, nil
}
