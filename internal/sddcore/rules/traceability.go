package rules

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

var nonGoalsHeadingRegex = regexp.MustCompile(`^(?:nicht-?ziele|non-?goals|out of scope)$`)

// activeArtifacts returns the artifacts selected by spec that are neither archived nor superseded nor in the archive.
func activeArtifacts(ctx *Context, spec string) ([]scan.Artifact, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	list, err := scan.ListArtifacts(ctx.RepoRoot, spec, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", spec, err)
	}
	archive := strings.TrimSpace(ctx.Config.Artifacts.Archive)
	var out []scan.Artifact
	for _, a := range list {
		if inactiveStatuses[a.Frontmatter.Status] || (archive != "" && scan.InSpec(a.Path, archive)) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func readLines(ctx *Context, rel string) ([]markdownLine, error) {
	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("%s nicht lesbar: %w", rel, err)
	}
	return markdownLines(data), nil
}

// checkSDD023 (SDD-023): Every active PRD has a non-empty section "Nicht-Ziele" / "Non-Goals" / "Out of Scope".
func checkSDD023(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	prds, err := activeArtifacts(ctx, ctx.Config.Artifacts.PRD)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, prd := range prds {
		lines, err := readLines(ctx, prd.Path)
		if err != nil {
			return nil, err
		}
		found, filled, line := false, false, 1
		for _, l := range lines {
			if l.Heading > 0 {
				if found {
					break // the section ends at the next heading of any level
				}
				if nonGoalsHeadingRegex.MatchString(strings.ToLower(normalizedHeading(l.Text))) {
					found, line = true, l.Number
				}
				continue
			}
			if found && strings.TrimSpace(l.Text) != "" {
				filled = true
				break
			}
		}
		var msg string
		switch {
		case !found:
			msg = fmt.Sprintf("PRD %s hat keinen Abschnitt „Nicht-Ziele“ (Non-Goals, Out of Scope)", prd.Path)
		case !filled:
			msg = fmt.Sprintf("Abschnitt „Nicht-Ziele“ in %s ist leer", prd.Path)
		default:
			continue
		}
		findings = append(findings, report.Finding{Path: prd.Path, Line: line, Message: msg, ContentHash: HashContent(rule.ID + ":" + prd.Path + ":" + fmt.Sprint(found))})
	}
	return findings, nil
}

// normalizedHeading strips the ATX markers, an explicit {#anchor} and closing hashes from a heading line.
func normalizedHeading(line string) string {
	m := headingRegex.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	text := strings.TrimSpace(explicitAnchorRegex.ReplaceAllString(m[2], ""))
	return strings.TrimSpace(strings.TrimRight(text, "#"))
}

// checkSDD031 (SDD-031): Every AC ID of a feature's PRD occurs in at least one Tech Spec of the same feature.
// Features without a Tech Spec are left to SDD-030.
func checkSDD031(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	defs, err := definedACs(ctx)
	if err != nil || len(defs) == 0 {
		return nil, err
	}
	designs, err := activeArtifacts(ctx, ctx.Config.Artifacts.Design)
	if err != nil {
		return nil, err
	}
	specsOf := map[string][]string{}          // feature -> Tech Spec paths
	mentioned := map[string]map[string]bool{} // feature -> AC IDs mentioned in its Tech Specs
	for _, d := range designs {
		feature := d.Frontmatter.Feature
		if feature == "" {
			continue
		}
		lines, err := readLines(ctx, d.Path)
		if err != nil {
			return nil, err
		}
		specsOf[feature] = append(specsOf[feature], d.Path)
		if mentioned[feature] == nil {
			mentioned[feature] = map[string]bool{}
		}
		for _, l := range lines {
			for _, id := range acIDRegex.FindAllString(acTombstoneRegex.ReplaceAllString(l.Text, ""), -1) {
				mentioned[feature][id] = true
			}
		}
	}

	var findings []report.Finding
	for _, d := range defs {
		specs := specsOf[d.Feature]
		if len(specs) == 0 || mentioned[d.Feature][d.ID] {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        specs[0],
			Line:        1,
			ByteOffset:  d.ByteOffset,
			Message:     fmt.Sprintf("%s aus %s (Feature %s) wird in keiner Tech Spec des Features referenziert", d.ID, d.Path, d.Feature),
			ContentHash: HashContent(rule.ID + ":" + d.Feature + ":" + d.ID),
		})
	}
	return findings, nil
}

// indentOf counts leading whitespace; a tab counts as four spaces.
func indentOf(line string) int {
	n := 0
	for _, r := range line {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// checkSDD041 (SDD-041): Every task (checkbox line in an active plan) carries an AC ID in its line or continuation lines.
func checkSDD041(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	plans, err := activeArtifacts(ctx, ctx.Config.Artifacts.Plans)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, plan := range plans {
		lines, err := readLines(ctx, plan.Path)
		if err != nil {
			return nil, err
		}
		for i, l := range lines {
			if !checkboxRegex.MatchString(l.Text) {
				continue
			}
			indent := indentOf(l.Text)
			text := l.Text
			for _, c := range lines[i+1:] {
				if strings.TrimSpace(c.Text) == "" {
					continue
				}
				if c.Heading > 0 || indentOf(c.Text) <= indent || checkboxRegex.MatchString(c.Text) {
					break
				}
				text += "\n" + c.Text
			}
			if acIDRegex.MatchString(acTombstoneRegex.ReplaceAllString(text, "")) {
				continue
			}
			findings = append(findings, report.Finding{
				Path:        plan.Path,
				Line:        l.Number,
				ByteOffset:  l.ByteOffset,
				Message:     fmt.Sprintf("Task ohne AC-Kennung: %q", strings.TrimSpace(l.Text)),
				ContentHash: HashContent(rule.ID + ":" + plan.Path + ":" + l.Text),
			})
		}
	}
	return findings, nil
}

var (
	inlineLinkRegex = regexp.MustCompile(`!?\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	refLinkRegex    = regexp.MustCompile(`^\s{0,3}\[[^\]]+\]:\s*<?([^\s>]+)>?`)
	schemeRegex     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// LinksIn returns the repository-relative targets of the Markdown links in the file rel (external links and anchors
// excluded); an unreadable file has none.
func LinksIn(root, rel string) []string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		for _, t := range linkTargets(line) {
			if resolved, ok := resolveLink(rel, t); ok {
				out = append(out, resolved)
			}
		}
	}
	return out
}

// linkTargets extracts the targets of inline links and reference definitions in one line.
func linkTargets(line string) []string {
	var out []string
	for _, m := range inlineLinkRegex.FindAllStringSubmatch(line, -1) {
		out = append(out, m[1])
	}
	if m := refLinkRegex.FindStringSubmatch(line); m != nil {
		out = append(out, m[1])
	}
	return out
}

// resolveLink turns a link target in file from into a repository-relative path; ok is false for external links,
// pure anchors and targets outside the repository.
func resolveLink(from, target string) (string, bool) {
	if schemeRegex.MatchString(target) || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") {
		return "", false
	}
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		target = target[:i]
	}
	if unescaped, err := url.PathUnescape(target); err == nil {
		target = unescaped
	}
	if target == "" {
		return "", false
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimLeft(target, "/")
	} else {
		target = path.Join(path.Dir(from), target)
	}
	clean, err := scan.CleanRel(target)
	return clean, err == nil
}

// checkSDD051 (SDD-051): Active feature artifacts do not link to archived feature artifacts.
func checkSDD051(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	a := ctx.Config.Artifacts
	archive := strings.TrimSpace(a.Archive)
	var sources []scan.Artifact
	for _, spec := range []string{a.PRD, a.Design, a.Plans} {
		list, err := activeArtifacts(ctx, spec)
		if err != nil {
			return nil, err
		}
		sources = append(sources, list...)
	}

	archivedTarget := func(p string) bool {
		if archive != "" && scan.InSpec(p, archive) {
			return true
		}
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(p)))
		if err != nil {
			return false
		}
		fm, ok := scan.ParseFrontmatter(data)
		return ok && fm.Feature != "" && fm.Status == "archived"
	}

	var findings []report.Finding
	seen := map[string]bool{}
	for _, src := range sources {
		if seen[src.Path] {
			continue
		}
		seen[src.Path] = true
		lines, err := readLines(ctx, src.Path)
		if err != nil {
			return nil, err
		}
		for _, l := range lines {
			for _, target := range linkTargets(l.Text) {
				resolved, ok := resolveLink(src.Path, target)
				if !ok || !archivedTarget(resolved) {
					continue
				}
				findings = append(findings, report.Finding{
					Path:        src.Path,
					Line:        l.Number,
					ByteOffset:  l.ByteOffset,
					Message:     fmt.Sprintf("Aktives Feature-Artefakt verweist auf archiviertes Artefakt %s", resolved),
					ContentHash: HashContent(rule.ID + ":" + src.Path + ":" + resolved),
				})
			}
		}
	}
	return findings, nil
}

// featureOf returns the feature of an artifact: its feature: field, else the slug position of its spec.
func featureOf(a scan.Artifact, spec string) string {
	if a.Frontmatter.Feature != "" {
		return a.Frontmatter.Feature
	}
	return scan.SlugInSpec(a.Path, spec)
}

// checkSDD040 (SDD-040): A feature with tasks in an active plan has an active Tech Spec ("plan before the first task").
func checkSDD040(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	a := ctx.Config.Artifacts
	plans, err := activeArtifacts(ctx, a.Plans)
	if err != nil {
		return nil, err
	}
	designs, err := activeArtifacts(ctx, a.Design)
	if err != nil {
		return nil, err
	}
	hasDesign := map[string]bool{}
	for _, d := range designs {
		hasDesign[featureOf(d, a.Design)] = true
	}

	var findings []report.Finding
	for _, p := range plans {
		feature := featureOf(p, a.Plans)
		if hasDesign[feature] {
			continue
		}
		lines, err := readLines(ctx, p.Path)
		if err != nil {
			return nil, err
		}
		for _, l := range lines {
			if !checkboxRegex.MatchString(l.Text) {
				continue
			}
			findings = append(findings, report.Finding{
				Path:        p.Path,
				Line:        l.Number,
				ByteOffset:  l.ByteOffset,
				Message:     fmt.Sprintf("Feature %s hat Tasks, aber keine aktive Tech Spec unter %s — zuerst den technischen Plan anlegen (`sdd new --feature %s --type design`)", feature, a.Design, feature),
				ContentHash: HashContent(rule.ID + ":" + p.Path + ":" + feature),
			})
			break // one finding per plan, at its first task
		}
	}
	return findings, nil
}
