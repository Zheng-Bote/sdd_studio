package guide

import (
	"fmt"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
)

// Archive plans `sdd archive`: every acceptance criterion of the feature must be referenced by a test (as SDD-022
// counts it) — a feature without tests is not done and is not archived. Every artifact must be movable; a plan that
// would move only some of them is refused.
func Archive(root, feature, release string) (scaffold.Plan, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return scaffold.Plan{}, err
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
