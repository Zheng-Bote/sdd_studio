package rules

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

var (
	slugRegex          = regexp.MustCompile(`^[a-z0-9-]+$`)
	featureBranchRegex = regexp.MustCompile(`^(?:feature/([a-z0-9-]+)|feat/(?:[0-9]+-)?([a-z0-9-]+))$`)
)

// featureScope is the result of the feature resolution (concept section 5.1).
type featureScope struct {
	Slugs  []string
	Source string // "--feature", "Frontmatter im Diff", "Branch", or "" if unresolved
}

// gitSource returns the injected Git access or the CLI implementation for RepoRoot.
func (c *Context) gitSource() scan.Git {
	if c.Git == nil {
		c.Git = scan.CLIGit{Root: c.RepoRoot}
	}
	return c.Git
}

// changedFiles returns `git diff <base>...HEAD` without excluded paths; computed once per run.
func (c *Context) changedFiles() ([]string, error) {
	if c.diffDone {
		return c.diff, c.diffErr
	}
	c.diffDone = true
	g := c.gitSource()
	base, err := g.ResolveBaseRef(c.Config.BaseRef)
	if err != nil {
		c.diffErr = err
		return nil, err
	}
	files, err := g.ChangedFiles(base)
	if err != nil {
		c.diffErr = err
		return nil, err
	}
	for _, f := range files {
		if !scan.MatchAny(c.Config.Exclude, f) {
			c.diff = append(c.diff, f)
		}
	}
	return c.diff, nil
}

// featureArtifacts returns all Markdown artifacts below the PRD, design and plan directories; computed once per run.
func (c *Context) featureArtifacts() ([]scan.Artifact, error) {
	if c.artifactsDone {
		return c.artifacts, c.artifactsErr
	}
	c.artifactsDone = true
	seen := map[string]bool{}
	for _, dir := range []string{c.Config.Artifacts.PRD, c.Config.Artifacts.Design, c.Config.Artifacts.Plans} {
		if strings.TrimSpace(dir) == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		list, err := scan.ListArtifacts(c.RepoRoot, dir, c.Config.Exclude)
		if err != nil {
			c.artifactsErr = fmt.Errorf("Artefakte unter %s nicht lesbar: %w", dir, err)
			return nil, c.artifactsErr
		}
		c.artifacts = append(c.artifacts, list...)
	}
	return c.artifacts, nil
}

// resolveFeatures applies the resolution order --feature > frontmatter of artifacts in the diff > branch name.
func resolveFeatures(ctx *Context) (featureScope, error) {
	if strings.TrimSpace(ctx.Feature) != "" {
		var slugs []string
		for _, s := range strings.Split(ctx.Feature, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if !slugRegex.MatchString(s) {
				return featureScope{}, fmt.Errorf("ungültiger Feature-Slug %q (erlaubt: [a-z0-9-]+)", s)
			}
			slugs = append(slugs, s)
		}
		return featureScope{Slugs: uniqueSorted(slugs), Source: "--feature"}, nil
	}

	diff, err := ctx.changedFiles()
	if err != nil {
		return featureScope{}, err
	}
	artifacts, err := ctx.featureArtifacts()
	if err != nil {
		return featureScope{}, err
	}
	inDiff := map[string]bool{}
	for _, f := range diff {
		inDiff[f] = true
	}
	var slugs []string
	for _, a := range artifacts {
		if inDiff[a.Path] && slugRegex.MatchString(a.Frontmatter.Feature) {
			slugs = append(slugs, a.Frontmatter.Feature)
		}
	}
	if len(slugs) > 0 {
		return featureScope{Slugs: uniqueSorted(slugs), Source: "Frontmatter im Diff"}, nil
	}

	branch, err := ctx.gitSource().CurrentBranch()
	if err != nil {
		return featureScope{}, err
	}
	if m := featureBranchRegex.FindStringSubmatch(branch); m != nil {
		slug := m[1] + m[2]
		return featureScope{Slugs: []string{slug}, Source: "Branch " + branch}, nil
	}
	return featureScope{}, nil
}

// declaredIntent collects `touches` of all feature artifacts carrying the slug, plus --paths.
func declaredIntent(ctx *Context, slug string) ([]string, error) {
	artifacts, err := ctx.featureArtifacts()
	if err != nil {
		return nil, err
	}
	intent := append([]string(nil), ctx.Paths...)
	for _, a := range artifacts {
		if a.Frontmatter.Feature == slug {
			intent = append(intent, a.Frontmatter.Touches...)
		}
	}
	return uniqueSorted(intent), nil
}

// featureArtifactKind parameterizes the shared check behind SDD-020 (PRD) and SDD-030 (Tech Spec).
type featureArtifactKind struct {
	label     string // human-readable artifact name
	configKey string // key below artifacts in sdd.config.yaml
	dir       func(ctx *Context) string
}

var (
	prdKind    = featureArtifactKind{label: "PRD", configKey: "artifacts.prd", dir: func(c *Context) string { return c.Config.Artifacts.PRD }}
	designKind = featureArtifactKind{label: "Tech Spec", configKey: "artifacts.design", dir: func(c *Context) string { return c.Config.Artifacts.Design }}
)

func checkSDD020(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	return checkFeatureArtifact(ctx, rule, prdKind)
}

func checkSDD030(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	return checkFeatureArtifact(ctx, rule, designKind)
}

// checkFeatureArtifact (SDD-020/SDD-030): If the size threshold is exceeded, an artifact with `feature: <slug>` must exist
// in the configured directory. Only pre-implementation (declared intent) and pre-merge (diff) are evaluated.
func checkFeatureArtifact(ctx *Context, rule catalog.Rule, kind featureArtifactKind) ([]report.Finding, error) {
	if ctx.Gate != "pre-implementation" && ctx.Gate != "pre-merge" {
		return nil, nil
	}
	dir := strings.TrimSuffix(strings.TrimSpace(kind.dir(ctx)), "/")
	reportPath := dir
	if reportPath == "" {
		reportPath = "sdd.config.yaml"
	}

	scope, err := resolveFeatures(ctx)
	if err != nil {
		return nil, err
	}

	var diff []string
	if ctx.Gate == "pre-merge" {
		if diff, err = ctx.changedFiles(); err != nil {
			return nil, err
		}
	}

	if len(scope.Slugs) == 0 {
		paths := diff
		if ctx.Gate == "pre-implementation" {
			if len(ctx.Paths) == 0 {
				return []report.Finding{undeclaredIntent(rule, kind, reportPath, "", 0)}, nil
			}
			paths = ctx.Paths
		}
		th := scan.EvaluateThreshold(ctx.Config.Thresholds, paths)
		if !th.Exceeded {
			return nil, nil
		}
		return []report.Finding{{
			Path:        reportPath,
			Line:        1,
			Message:     fmt.Sprintf("Größenschwelle überschritten (%s), aber kein Feature auflösbar; %s fehlt — `sdd new --feature <slug>` ausführen", th.Reason, kind.label),
			ContentHash: HashContent(rule.ID + ":unresolved"),
		}}, nil
	}

	artifacts, err := ctx.featureArtifacts()
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for idx, slug := range scope.Slugs {
		paths := diff
		if ctx.Gate == "pre-implementation" {
			if paths, err = declaredIntent(ctx, slug); err != nil {
				return nil, err
			}
			if len(paths) == 0 {
				findings = append(findings, undeclaredIntent(rule, kind, reportPath, slug, idx))
				continue
			}
		}
		th := scan.EvaluateThreshold(ctx.Config.Thresholds, paths)
		if !th.Exceeded {
			continue
		}
		if dir == "" {
			findings = append(findings, report.Finding{
				Path:        reportPath,
				Line:        1,
				ByteOffset:  idx,
				Message:     fmt.Sprintf("Feature %s überschreitet die Größenschwelle (%s), aber %s ist nicht konfiguriert", slug, th.Reason, kind.configKey),
				ContentHash: HashContent(rule.ID + ":unconfigured:" + slug),
			})
			continue
		}
		if hasArtifactFor(artifacts, dir, slug) {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        reportPath,
			Line:        1,
			ByteOffset:  idx,
			Message:     fmt.Sprintf("Feature %s (aufgelöst über %s) überschreitet die Größenschwelle (%s), aber unter %s existiert kein %s mit `feature: %s` — `sdd new --feature %s` ausführen", slug, scope.Source, th.Reason, dir, kind.label, slug, slug),
			ContentHash: HashContent(rule.ID + ":missing:" + slug),
		})
	}
	return findings, nil
}

func undeclaredIntent(rule catalog.Rule, kind featureArtifactKind, path, slug string, idx int) report.Finding {
	subject := "kein Feature aufgelöst und kein --paths"
	if slug != "" {
		subject = fmt.Sprintf("Feature %s ohne `touches` und ohne --paths", slug)
	}
	return report.Finding{
		Severity:    "warn",
		Path:        path,
		Line:        1,
		ByteOffset:  idx,
		Message:     fmt.Sprintf("Änderungsabsicht nicht erklärt (%s); Schwelle nicht bewertbar, keine %s-Prüfung erfolgt", subject, kind.label),
		ContentHash: HashContent(rule.ID + ":undeclared:" + slug),
	}
}

func hasArtifactFor(artifacts []scan.Artifact, dir, slug string) bool {
	for _, a := range artifacts {
		if scan.UnderDir(a.Path, dir) && a.Frontmatter.Feature == slug {
			return true
		}
	}
	return false
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
