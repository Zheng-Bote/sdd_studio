package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	if strings.TrimSpace(dir) == "" {
		return nil, nil
	}
	dir, err := CleanRel(dir)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(repoRoot, filepath.FromSlash(dir))
	var out []Artifact
	err = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
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

// WalkFiles returns all regular files below repoRoot (slash-separated, sorted) that match include and not exclude.
// The .git directory is always skipped.
func WalkFiles(repoRoot string, include, exclude []string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(repoRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(repoRoot, p)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if d.Name() == ".git" || MatchAny(exclude, rel) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && MatchAny(include, rel) && !MatchAny(exclude, rel) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

// IsSlug reports whether s is a valid feature or artifact slug: lowercase ASCII, digits and hyphens.
func IsSlug(s string) bool {
	return slugRegex.MatchString(s)
}

// ErrUnsafePath is returned for configured or planned paths that would leave the repository root.
var ErrUnsafePath = errors.New("Pfad liegt außerhalb des Repository-Wurzelverzeichnisses")

// CleanRel normalizes a repository-relative path and rejects empty and absolute paths as well as `..` escapes.
func CleanRel(p string) (string, error) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if p == "" || strings.HasPrefix(p, "/") || filepath.IsAbs(p) || filepath.VolumeName(p) != "" {
		return "", fmt.Errorf("%w: %q", ErrUnsafePath, p)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: %q", ErrUnsafePath, p)
	}
	return clean, nil
}
