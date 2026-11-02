package scaffold

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// ArchivePlan moves every PRD, Tech Spec and Plan of feature outside the archive into artifacts.archive, with
// status: archived (and release, if given) in the frontmatter (handbook "Archivierung", SDD-050).
func ArchivePlan(root string, cfg *config.Config, feature, release string) (Plan, error) {
	if !scan.IsSlug(feature) {
		return Plan{}, fmt.Errorf("%w: Feature-Slug %q (erlaubt: [a-z0-9-]+)", ErrInvalidInput, feature)
	}
	archive, err := scan.CleanRel(strings.TrimSpace(cfg.Artifacts.Archive))
	if strings.TrimSpace(cfg.Artifacts.Archive) == "" || err != nil {
		return Plan{}, fmt.Errorf("%w: artifacts.archive ist nicht konfiguriert", ErrInvalidInput)
	}
	if scan.HasWildcard(archive) {
		return Plan{}, fmt.Errorf("%w: artifacts.archive %q ist ein Datei-Glob; archivieren braucht einen Ordner", ErrInvalidInput, archive)
	}
	release = strings.TrimSpace(release)
	if strings.ContainsAny(release, "\n\r") {
		return Plan{}, fmt.Errorf("%w: release %q", ErrInvalidInput, release)
	}

	var plan Plan
	a := cfg.Artifacts
	for _, spec := range []string{a.PRD, a.Design, a.Plans} {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		list, err := scan.ListArtifacts(root, spec, cfg.Exclude)
		if err != nil {
			return Plan{}, fmt.Errorf("Artefakte unter %s nicht lesbar: %w", spec, err)
		}
		base, err := specBase(spec)
		if err != nil {
			return Plan{}, err
		}
		for _, art := range list {
			if scan.InSpec(art.Path, archive) {
				continue
			}
			if f := art.Frontmatter.Feature; f != feature && (f != "" || scan.SlugInSpec(art.Path, spec) != feature) {
				continue
			}
			old, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(art.Path)))
			if err != nil {
				return Plan{}, err
			}
			content, err := setFrontmatter(string(old), "status", "archived")
			if err == nil && release != "" {
				content, err = setFrontmatter(content, "release", release)
			}
			if err != nil {
				return Plan{}, fmt.Errorf("%w: %s: %v", ErrInvalidInput, art.Path, err)
			}
			rel := strings.TrimPrefix(art.Path, path.Dir(base)+"/")
			if path.Dir(base) == "." {
				rel = art.Path
			}
			plan.Ops = append(plan.Ops, Op{Path: archive + "/" + rel, Content: []byte(content), MoveFrom: art.Path, Expect: old})
		}
	}
	if len(plan.Ops) == 0 {
		return Plan{}, fmt.Errorf("%w: keine Artefakte für Feature %s außerhalb von %s", ErrInvalidInput, feature, archive)
	}
	return plan, plan.Validate()
}

// specBase is the folder an artifact spec selects from: the folder itself, or the static prefix of a glob.
func specBase(spec string) (string, error) {
	clean, err := scan.CleanRel(spec)
	if err != nil {
		return "", err
	}
	if scan.HasWildcard(clean) {
		clean = strings.TrimSuffix(scan.StaticPrefix(clean), "/")
	}
	if clean == "" {
		return "", fmt.Errorf("%w: aus %q lässt sich kein Archivpfad ableiten", ErrInvalidInput, spec)
	}
	return clean, nil
}

// setFrontmatter sets `key: value` in the YAML frontmatter block of content: an existing top-level line `key:` is
// replaced, otherwise the line is added before the closing `---`. Everything else stays byte for byte.
func setFrontmatter(content, key, value string) (string, error) {
	nl := "\n"
	if strings.Contains(content, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return "", fmt.Errorf("kein Frontmatter-Block")
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		if line == "---" {
			lines = append(lines[:i], append([]string{key + ": " + value + nl}, lines[i:]...)...)
			return strings.Join(lines, ""), nil
		}
		if name, _, ok := strings.Cut(line, ":"); ok && name == key {
			lines[i] = key + ": " + value + nl
			return strings.Join(lines, ""), nil
		}
	}
	return "", fmt.Errorf("Frontmatter-Block nicht geschlossen")
}
