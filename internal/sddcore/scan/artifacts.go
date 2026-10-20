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

// ListArtifacts returns all *.md files (sorted by path, without exclude matches) selected by an artifact spec:
// a directory (walked recursively) or a file glob such as `specs/features/*/spec.md`.
// A missing directory yields no artifacts and no error.
func ListArtifacts(repoRoot, spec string, exclude []string) ([]Artifact, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	spec, err := CleanRel(spec)
	if err != nil {
		return nil, err
	}
	dir, glob := spec, ""
	if HasWildcard(spec) {
		dir, glob = strings.TrimSuffix(StaticPrefix(spec), "/"), spec
		if dir == "" {
			dir = "."
		}
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
			if rel != "." && (d.Name() == ".git" || MatchAny(exclude, rel)) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.EqualFold(filepath.Ext(rel), ".md") || MatchAny(exclude, rel) {
			return nil
		}
		if glob != "" && !Match(glob, rel) {
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

// InSpec reports whether path is selected by an artifact spec (directory or file glob).
func InSpec(path, spec string) bool {
	if HasWildcard(spec) {
		return Match(spec, path)
	}
	return UnderDir(path, spec)
}

// TargetPath derives the file for slug from an artifact spec: `<dir>/<slug>.md` for a directory, or the glob with its
// single whole-segment `*` replaced by slug (`specs/features/*/spec.md` -> `specs/features/<slug>/spec.md`).
func TargetPath(spec, slug string) (string, error) {
	clean, err := CleanRel(spec)
	if err != nil {
		return "", err
	}
	if !HasWildcard(clean) {
		return clean + "/" + slug + ".md", nil
	}
	segments := strings.Split(clean, "/")
	star := -1
	for i, seg := range segments {
		switch {
		case seg == "*" && star < 0:
			star = i
		case HasWildcard(seg):
			return "", fmt.Errorf("aus %q lässt sich kein Zielpfad ableiten (genau ein `*` als ganzes Pfadsegment nötig)", spec)
		}
	}
	if star < 0 {
		return "", fmt.Errorf("aus %q lässt sich kein Zielpfad ableiten (genau ein `*` als ganzes Pfadsegment nötig)", spec)
	}
	segments[star] = slug
	return strings.Join(segments, "/"), nil
}

// SlugInSpec returns the path segment matched by the single `*` of a glob spec, or the file name without extension
// for a directory spec.
func SlugInSpec(path, spec string) string {
	clean, err := CleanRel(spec)
	if err == nil && HasWildcard(clean) {
		segs, psegs := strings.Split(clean, "/"), strings.Split(path, "/")
		if len(segs) == len(psegs) {
			for i, seg := range segs {
				if seg == "*" {
					return psegs[i]
				}
			}
		}
	}
	base := path[strings.LastIndex(path, "/")+1:]
	return strings.TrimSuffix(base, filepath.Ext(base))
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
