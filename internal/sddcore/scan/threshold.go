package scan

import (
	"fmt"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
)

// ThresholdResult describes whether the feature size threshold (SDD-020/SDD-030) is exceeded and why.
type ThresholdResult struct {
	Exceeded      bool
	TouchedPaths  []string // touched_paths globs that were hit
	LayersTouched []string // layer globs that were hit
	Reason        string
}

// EvaluateThreshold applies the disjunctive threshold from sdd.config.yaml to a set of paths or path globs:
// exceeded if any touched_paths glob overlaps, OR at least min_layers_touched distinct layers overlap.
// min_layers_touched <= 0 disables the layer condition.
func EvaluateThreshold(t config.ThresholdsConfig, paths []string) ThresholdResult {
	var res ThresholdResult
	trig := t.FeatureArtifactsRequiredWhen
	for _, tp := range trig.TouchedPaths {
		if overlapsAny(tp, paths) {
			res.TouchedPaths = append(res.TouchedPaths, tp)
		}
	}
	for _, layer := range t.Layers {
		if overlapsAny(layer, paths) {
			res.LayersTouched = append(res.LayersTouched, layer)
		}
	}

	switch {
	case len(res.TouchedPaths) > 0:
		res.Exceeded = true
		res.Reason = fmt.Sprintf("touched_paths getroffen: %v", res.TouchedPaths)
	case trig.MinLayersTouched > 0 && len(res.LayersTouched) >= trig.MinLayersTouched:
		res.Exceeded = true
		res.Reason = fmt.Sprintf("%d Layer berührt (Schwelle %d): %v", len(res.LayersTouched), trig.MinLayersTouched, res.LayersTouched)
	}
	return res
}

func overlapsAny(pattern string, paths []string) bool {
	for _, p := range paths {
		if Overlaps(pattern, p) {
			return true
		}
	}
	return false
}
