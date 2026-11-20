package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// The three macro rules of the BMAD pipeline (ADR 0009). They run only with `pipeline: bmad`: without a declared macro
// level there is nothing to check, and every other project keeps its previous findings.

// bmadEnabled reports whether the macro rules apply.
func bmadEnabled(ctx *Context) bool { return ctx.Config.Pipeline == config.PipelineBMAD }

// initiativeFile is the initiative with its requirement IDs; nil if there is none.
type initiativeFile struct {
	rel  string
	data []byte
	ids  []string
}

// readInitiative reads the configured initiative. A missing file yields nil, not an error: SDD-080/081 then have
// nothing to measure against and stay silent.
func readInitiative(ctx *Context) (*initiativeFile, error) {
	rel, err := scan.CleanRel(ctx.Config.Artifacts.InitiativePath())
	if err != nil {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return nil, nil
	}
	return &initiativeFile{rel: rel, data: data, ids: scan.RequirementIDs(data)}, nil
}

// epicArtifacts returns the active artifacts below artifacts.prd that declare themselves epics.
func epicArtifacts(ctx *Context) ([]scan.Artifact, error) {
	spec := strings.TrimSpace(ctx.Config.Artifacts.PRD)
	if spec == "" {
		return nil, nil
	}
	list, err := scan.ListArtifacts(ctx.RepoRoot, spec, ctx.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", spec, err)
	}
	var out []scan.Artifact
	for _, a := range list {
		if a.Frontmatter.IsEpic() && !inactiveStatuses[a.Frontmatter.Status] {
			out = append(out, a)
		}
	}
	return out, nil
}

// keyLine returns the 1-based line of `key:` in the file's frontmatter, or 1.
func keyLine(root, rel, key string) int {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return 1
	}
	for i, line := range strings.Split(string(data), "\n") {
		if i > 0 && strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(line), key+":") {
			return i + 1
		}
	}
	return 1
}

// requirementLine returns the 1-based line of a requirement list item, or 1.
func requirementLine(data []byte, id string) int {
	for i, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "+") {
			continue
		}
		t = strings.TrimSpace(strings.TrimLeft(t, "-*+"))
		t = strings.TrimPrefix(t, "**")
		if strings.HasPrefix(t, id) && (len(t) == len(id) || isIDBoundary(t[len(id)])) {
			return i + 1
		}
	}
	return 1
}

func isIDBoundary(b byte) bool {
	return b == '*' || b == ' ' || b == '\t' || b == ':' || b == '(' || b == ','
}

// checkSDD080: every epic covers at least one requirement, and every requirement it names exists in the initiative.
func checkSDD080(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if !bmadEnabled(ctx) {
		return nil, nil
	}
	ini, err := readInitiative(ctx)
	if err != nil || ini == nil {
		return nil, err
	}
	epics, err := epicArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, id := range ini.ids {
		known[id] = true
	}
	var findings []report.Finding
	for _, e := range epics {
		var valid, unknown []string
		for _, c := range e.Frontmatter.Covers {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if known[c] {
				valid = append(valid, c)
				continue
			}
			unknown = append(unknown, c)
		}
		line := keyLine(ctx.RepoRoot, e.Path, "covers")
		var msg string
		switch {
		case len(valid) == 0 && len(unknown) == 0:
			msg = fmt.Sprintf("Epic %s nennt in covers keine Anforderung — mindestens eine Kennung aus %s eintragen", e.Path, ini.rel)
		case len(unknown) > 0:
			sort.Strings(unknown)
			msg = fmt.Sprintf("Epic %s nennt in covers Anforderungen, die %s nicht kennt: %s", e.Path, ini.rel, strings.Join(unknown, ", "))
		default:
			continue
		}
		findings = append(findings, report.Finding{
			Path:        e.Path,
			Line:        line,
			ByteOffset:  line,
			Message:     msg,
			ContentHash: HashContent(rule.ID + ":" + e.Path + ":" + msg),
		})
	}
	return findings, nil
}

// checkSDD081: every requirement of the initiative is covered by an epic or named in its `unplanned` list.
func checkSDD081(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if !bmadEnabled(ctx) {
		return nil, nil
	}
	ini, err := readInitiative(ctx)
	if err != nil || ini == nil || len(ini.ids) == 0 {
		return nil, err
	}
	epics, err := epicArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	if len(epics) == 0 {
		return nil, nil // nothing taken over yet: before the first epic every requirement would look uncovered
	}
	covered := map[string]bool{}
	for _, e := range epics {
		for _, c := range e.Frontmatter.Covers {
			covered[strings.TrimSpace(c)] = true
		}
	}
	unplanned := map[string]bool{}
	if fm, ok := scan.ParseFrontmatter(ini.data); ok {
		for _, u := range fm.Unplanned {
			unplanned[strings.TrimSpace(u)] = true
		}
	}
	var findings []report.Finding
	for _, id := range ini.ids {
		if covered[id] || unplanned[id] {
			continue
		}
		findings = append(findings, report.Finding{
			Path:        ini.rel,
			Line:        requirementLine(ini.data, id),
			Message:     fmt.Sprintf("Anforderung %s ist von keinem Epic übernommen — in covers eines Epics aufnehmen oder in unplanned der Initiative als nicht geplant nennen", id),
			ContentHash: HashContent(rule.ID + ":" + ini.rel + ":" + id),
		})
	}
	return findings, nil
}

var ticketSlugRegex = regexp.MustCompile(`(?m)^[ \t]*slug[ \t]*=[ \t]*"([^"]+)"`)

// checkSDD082: a `tickets.toml` index and the epic artifacts name the same slugs.
func checkSDD082(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if !bmadEnabled(ctx) {
		return nil, nil
	}
	indexes, err := scan.WalkFiles(ctx.RepoRoot, []string{"_bmad-output/**/tickets.toml"}, nil)
	if err != nil || len(indexes) == 0 {
		return nil, err
	}
	epics, err := epicArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	if len(epics) == 0 {
		return nil, nil // nothing taken over yet: the index is still BMAD's own list
	}
	inArtifacts := map[string]string{} // slug -> path
	for _, e := range epics {
		slug := e.Frontmatter.Feature
		if slug == "" {
			base := filepath.Base(e.Path)
			slug = strings.TrimSuffix(base, filepath.Ext(base))
		}
		inArtifacts[slug] = e.Path
	}
	var findings []report.Finding
	inIndex := map[string]bool{}
	for _, idx := range indexes {
		data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(idx)))
		if err != nil {
			return nil, fmt.Errorf("%s nicht lesbar: %w", idx, err)
		}
		for _, m := range ticketSlugRegex.FindAllSubmatchIndex(data, -1) {
			slug := string(data[m[2]:m[3]])
			inIndex[slug] = true
			if _, ok := inArtifacts[slug]; ok {
				continue
			}
			line := 1 + strings.Count(string(data[:m[0]]), "\n")
			msg := fmt.Sprintf("Index %s führt Epic %s, das keine Epic-Datei unter %s hat", idx, slug, ctx.Config.Artifacts.PRD)
			findings = append(findings, report.Finding{
				Path: idx, Line: line,
				Message:     msg,
				ContentHash: HashContent(rule.ID + ":" + idx + ":" + slug),
			})
		}
	}
	for slug, path := range inArtifacts {
		if inIndex[slug] {
			continue
		}
		msg := fmt.Sprintf("Epic %s fehlt im Index unter _bmad-output (Slug %s)", path, slug)
		findings = append(findings, report.Finding{
			Path: path, Line: 1,
			Message:     msg,
			ContentHash: HashContent(rule.ID + ":" + path + ":" + slug),
		})
	}
	return findings, nil
}
