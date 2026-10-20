package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Artifact is a Markdown file below a configured artifact directory together with its parsed frontmatter.
type Artifact struct {
	Path           string // slash-separated, relative to the repository root
	Frontmatter    Frontmatter
	HasFrontmatter bool
}

// ListArtifacts walks dir (relative to repoRoot) and returns all *.md files not matched by exclude, sorted by path.
// A missing directory yields no artifacts and no error.
func ListArtifacts(repoRoot, dir string, exclude []string) ([]Artifact, error) {
	dir = strings.TrimSuffix(normalizePath(dir), "/")
	if dir == "" {
		return nil, nil
	}
	base := filepath.Join(repoRoot, filepath.FromSlash(dir))
	var out []Artifact
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == base {
				return fs.SkipAll
			}
			return err
		}
		rel, relErr := filepath.Rel(repoRoot, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if MatchAny(exclude, rel) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.EqualFold(filepath.Ext(rel), ".md") || MatchAny(exclude, rel) {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		fm, ok := ParseFrontmatter(data)
		out = append(out, Artifact{Path: rel, Frontmatter: fm, HasFrontmatter: ok})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// UnderDir reports whether the slash-separated path lies below dir.
func UnderDir(path, dir string) bool {
	dir = strings.TrimSuffix(normalizePath(dir), "/")
	return dir != "" && strings.HasPrefix(normalizePath(path), dir+"/")
}
