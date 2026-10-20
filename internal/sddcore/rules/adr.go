package rules

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/adrlock"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// adrLinks are the frontmatter fields through which an ADR names the files it concerns.
type adrLinks struct {
	References []string `yaml:"references"`
	ADR        any      `yaml:"adr"`
}

// names reports whether the ADR at adrPath names target in references or adr (repo-relative or relative to the ADR).
func (l adrLinks) names(adrPath, target string) bool {
	values := append([]string{}, l.References...)
	switch v := l.ADR.(type) {
	case string:
		values = append(values, v)
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				values = append(values, s)
			}
		}
	}
	for _, v := range values {
		if clean, err := scan.CleanRel(v); err == nil && clean == target {
			return true
		}
		if resolved, ok := resolveLink(adrPath, v); ok && resolved == target {
			return true
		}
	}
	return false
}

func readADRLinks(root, rel string) (adrLinks, bool) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return adrLinks{}, false
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return adrLinks{}, false
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return adrLinks{}, false
	}
	var l adrLinks
	if yaml.Unmarshal([]byte(text[4:4+end]), &l) != nil {
		return adrLinks{}, false
	}
	return l, true
}

// isADR reports whether rel is an ADR file (NNNN-*.md below artifacts.adr).
func isADR(ctx *Context, rel string) bool {
	dir := strings.TrimSpace(ctx.Config.Artifacts.ADR)
	return dir != "" && scan.UnderDir(rel, dir) && scan.IsADRFileName(path.Base(rel))
}

// checkSDD010 (SDD-010): A change of the constitution in the diff needs an ADR in the same diff that names it.
func checkSDD010(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	if ctx.Gate != "pre-implementation" && ctx.Gate != "pre-merge" {
		return nil, nil
	}
	constitution, ok := repoRelative(strings.TrimSpace(ctx.Config.Artifacts.Constitution))
	if !ok {
		return nil, nil // SDD-001 reports the configuration
	}
	diff, err := ctx.changedFiles()
	if err != nil {
		return nil, err
	}
	changed := false
	for _, f := range diff {
		if f == constitution {
			changed = true
		}
	}
	if !changed {
		return nil, nil
	}
	current, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(constitution)))
	if err != nil {
		return nil, nil // deleted: SDD-001
	}
	before, existed, err := ctx.gitSource().ShowFile(ctx.baseRef, constitution)
	if err != nil {
		return nil, err
	}
	if !existed || scan.NormalizeText(string(before)) == scan.NormalizeText(string(current)) {
		return nil, nil // first creation, or formatting only
	}
	for _, f := range diff {
		if !isADR(ctx, f) {
			continue
		}
		if links, ok := readADRLinks(ctx.RepoRoot, f); ok && links.names(f, constitution) {
			return nil, nil
		}
	}
	return []report.Finding{{
		Path:        constitution,
		Line:        1,
		Message:     fmt.Sprintf("Verfassung %s wurde gegenüber %s inhaltlich geändert, aber kein ADR im selben Diff nennt sie in references oder adr — `sdd new --type adr --title …` anlegen und references: [%s] setzen", constitution, ctx.baseRef, constitution),
		ContentHash: HashContent(rule.ID + ":" + constitution + ":" + scan.NormalizeText(string(current))),
	}}, nil
}

// checkSDD011 (SDD-011): Decided ADRs are immutable: their body must match the hash in .sdd/adr-lock.json.
func checkSDD011(ctx *Context, rule catalog.Rule) ([]report.Finding, error) {
	adrs, err := adrlock.Collect(ctx.RepoRoot, ctx.Config)
	if err != nil {
		return nil, err
	}
	lock, _, err := adrlock.Load(ctx.RepoRoot)
	if err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, a := range adrs {
		if !a.Decided {
			continue
		}
		locked, ok := lock.Entries[a.Path]
		switch {
		case !ok:
			findings = append(findings, report.Finding{
				Severity:    "warn",
				Path:        a.Path,
				Line:        1,
				Message:     fmt.Sprintf("Entschiedenes ADR %s ist noch nicht in %s gelockt — `sdd baseline` ausführen", a.Path, adrlock.RelPath),
				ContentHash: HashContent(rule.ID + ":unlocked:" + a.Path),
			})
		case locked != a.Hash:
			findings = append(findings, report.Finding{
				Path:        a.Path,
				Line:        1,
				Message:     fmt.Sprintf("ADR %s wurde nach der Entscheidung geändert; ADRs sind unveränderlich — Ablösung über ein neues ADR mit supersedes, bewusste Quittierung nur mit `sdd baseline --accept SDD-011`", a.Path),
				ContentHash: HashContent(rule.ID + ":changed:" + a.Path + ":" + a.Hash),
			})
		}
	}
	return findings, nil
}
