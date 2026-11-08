package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// checkSDD012 (SDD-012, ADR 0005): a weakening change of sdd.config.yaml against the merge base needs an ADR in the same
// diff that names the file, like SDD-010 for the constitution. Only fields with a clear direction count: a lower
// profile, a changed gate override, a new exclude pattern, loosened thresholds.
func checkSDD012(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if ctx.Gate != "pre-implementation" && ctx.Gate != "pre-merge" {
		return nil, nil
	}
	const cfgPath = config.DefaultConfigFileName
	diff, err := ctx.changedFilesUnfiltered() // exclude must not hide the change that adds it
	if err != nil {
		return nil, err
	}
	if !slices.Contains(diff, cfgPath) {
		return nil, nil
	}
	current, err := os.ReadFile(filepath.Join(ctx.RepoRoot, cfgPath))
	if err != nil {
		return nil, nil // without a configuration nothing runs at all
	}
	before, existed, err := ctx.gitSource().ShowFile(ctx.baseRef, cfgPath)
	if err != nil {
		return nil, err
	}
	if !existed {
		return nil, nil // first creation
	}
	oldCfg, err := config.Parse(before)
	if err != nil {
		return nil, nil // an unreadable base configuration cannot be compared
	}
	newCfg, err := config.Parse(current)
	if err != nil {
		return nil, nil // loading the configuration reports it
	}
	weak := configWeakenings(oldCfg, newCfg)
	if len(weak) == 0 {
		return nil, nil
	}
	for _, f := range diff {
		if !isADR(ctx, f) {
			continue
		}
		if links, ok := readADRLinks(ctx.RepoRoot, f); ok && links.names(f, cfgPath) {
			return nil, nil
		}
	}
	return []report.Finding{{
		Path: cfgPath,
		Line: 1,
		Message: fmt.Sprintf("%s schwächt die Prüfung gegenüber %s ab (%s), aber kein ADR im selben Diff nennt die Datei in references oder adr — ADR anlegen und references: [%s] setzen",
			cfgPath, ctx.baseRef, strings.Join(weak, "; "), cfgPath),
		ContentHash: HashContent(rule.ID + ":" + strings.Join(weak, "\n")),
	}}, nil
}

// configWeakenings lists the changes from old to new that weaken the checks, sorted.
func configWeakenings(old, new *config.Config) []string {
	var out []string
	if profileOrder(new.Profile) < profileOrder(old.Profile) {
		out = append(out, fmt.Sprintf("profile %s → %s", old.Profile, new.Profile))
	}
	for _, g := range []string{"pre-implementation", "pre-merge", "pre-release"} {
		if !sameSet(old.GateOverrideFor(g), new.GateOverrideFor(g)) {
			out = append(out, fmt.Sprintf("gates.%s.override geändert", strings.ReplaceAll(g, "-", "_")))
		}
	}
	for _, p := range added(old.Exclude, new.Exclude) {
		out = append(out, "exclude: +"+p)
	}
	for _, p := range added(new.Thresholds.Layers, old.Thresholds.Layers) {
		out = append(out, "thresholds.layers: −"+p)
	}
	oldT, newT := old.Thresholds.FeatureArtifactsRequiredWhen, new.Thresholds.FeatureArtifactsRequiredWhen
	for _, p := range added(newT.TouchedPaths, oldT.TouchedPaths) {
		out = append(out, "touched_paths: −"+p)
	}
	if newT.MinLayersTouched > oldT.MinLayersTouched {
		out = append(out, fmt.Sprintf("min_layers_touched %d → %d", oldT.MinLayersTouched, newT.MinLayersTouched))
	}
	sort.Strings(out)
	return out
}

// profileOrder ranks the profiles minimal < standard < regulated; an unknown profile ranks lowest.
func profileOrder(p string) int {
	return slices.Index([]string{"minimal", "standard", "regulated"}, p)
}

// added returns the entries of b that are not in a.
func added(a, b []string) []string {
	var out []string
	for _, x := range b {
		if !slices.Contains(a, x) {
			out = append(out, x)
		}
	}
	return out
}

func sameSet(a, b []string) bool {
	return len(added(a, b)) == 0 && len(added(b, a)) == 0
}
