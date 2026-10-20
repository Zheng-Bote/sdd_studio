// Package scaffold is the only write path of sdd: it builds plans of file operations and applies them additively.
package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	sddstudio "github.com/Zheng-Bote/sdd_studio"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Op is a single planned file operation with a repository-relative, slash-separated path.
type Op struct {
	Path    string
	Content []byte
	// KeepDir marks a placeholder (.gitkeep) for a directory; it is only written if the directory does not exist yet.
	KeepDir bool
	// Skip, if set, records why the operation is planned but will not be executed (e.g. artifact exists).
	Skip string
	// Prepend inserts Content before the existing file content instead of creating a file. It only applies if the file
	// still equals Expect byte for byte; otherwise the operation is skipped (the file changed since it was planned).
	Prepend bool
	Expect  []byte
}

// Plan is an ordered list of operations.
type Plan struct {
	Ops []Op
}

// Status values reported by Apply.
const (
	StatusCreated   = "created"
	StatusPrepended = "prepended"
	StatusSkipped   = "skipped"
)

// Result is the outcome of one operation.
type Result struct {
	Path   string
	Status string
	Reason string
}

// ErrUnsafePath is returned when a planned path would leave the repository root.
var ErrUnsafePath = scan.ErrUnsafePath

// CleanRel normalizes a repository-relative path and rejects absolute paths and `..` escapes.
func CleanRel(p string) (string, error) {
	return scan.CleanRel(p)
}

// Validate checks every path of the plan before anything is written.
func (p Plan) Validate() error {
	for _, op := range p.Ops {
		if _, err := CleanRel(op.Path); err != nil {
			return err
		}
	}
	return nil
}

// Preview reports what Apply would do without touching the file system beyond reads.
func (p Plan) Preview(root string) ([]Result, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var out []Result
	for _, op := range p.Ops {
		out = append(out, previewOp(root, op))
	}
	return out, nil
}

func previewOp(root string, op Op) Result {
	if op.Skip != "" {
		return Result{Path: op.Path, Status: StatusSkipped, Reason: op.Skip}
	}
	target := filepath.Join(root, filepath.FromSlash(op.Path))
	if op.Prepend {
		current, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(current, op.Expect) {
			return Result{Path: op.Path, Status: StatusSkipped, Reason: "Datei seit der Planung verändert"}
		}
		return Result{Path: op.Path, Status: StatusPrepended}
	}
	if op.KeepDir {
		if _, err := os.Stat(filepath.Dir(target)); err == nil {
			return Result{Path: op.Path, Status: StatusSkipped, Reason: "Verzeichnis existiert"}
		}
	} else if _, err := os.Lstat(target); err == nil {
		return Result{Path: op.Path, Status: StatusSkipped, Reason: "Datei existiert"}
	}
	return Result{Path: op.Path, Status: StatusCreated}
}

// Apply executes the plan additively: existing files are never overwritten but reported as skipped.
// All paths are validated before the first write, and every parent directory must resolve inside root.
func (p Plan) Apply(root string) ([]Result, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("Repository-Wurzel %s nicht auflösbar: %w", root, err)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("Repository-Wurzel %s nicht auflösbar: %w", root, err)
	}
	var out []Result
	for _, op := range p.Ops {
		res := previewOp(root, op)
		if res.Status == StatusSkipped {
			out = append(out, res)
			continue
		}
		rel, _ := CleanRel(op.Path)
		target := filepath.Join(absRoot, filepath.FromSlash(rel))
		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return out, fmt.Errorf("Verzeichnis für %s nicht anlegbar: %w", rel, err)
		}
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return out, fmt.Errorf("Verzeichnis für %s nicht auflösbar: %w", rel, err)
		}
		if realDir != realRoot && !strings.HasPrefix(realDir, realRoot+string(filepath.Separator)) {
			return out, fmt.Errorf("%w: %s (über symbolischen Link)", ErrUnsafePath, rel)
		}
		if op.Prepend {
			if err := prependFile(target, op.Content, op.Expect); err != nil {
				return out, fmt.Errorf("%s nicht ergänzbar: %w", rel, err)
			}
			out = append(out, Result{Path: rel, Status: StatusPrepended})
			continue
		}
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			out = append(out, Result{Path: rel, Status: StatusSkipped, Reason: "Datei existiert"})
			continue
		}
		if err != nil {
			return out, fmt.Errorf("%s nicht schreibbar: %w", rel, err)
		}
		_, werr := f.Write(op.Content)
		cerr := f.Close()
		if werr != nil || cerr != nil {
			return out, fmt.Errorf("%s nicht schreibbar: %v", rel, errors.Join(werr, cerr))
		}
		out = append(out, Result{Path: rel, Status: StatusCreated})
	}
	return out, nil
}

// prependFile writes head+expect to target via a temporary file in the same directory and an atomic rename,
// keeping the original permissions. No existing content is removed.
func prependFile(target string, head, expect []byte) error {
	info, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("keine reguläre Datei")
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".sdd-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, werr := tmp.Write(append(append([]byte{}, head...), expect...))
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr, os.Chmod(tmp.Name(), info.Mode().Perm())); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// render executes an embedded template from templates/<name>.
func render(name string, data any) ([]byte, error) {
	src, err := fs.ReadFile(sddstudio.EmbeddedAssets, "templates/"+name)
	if err != nil {
		return nil, fmt.Errorf("Template %s nicht gefunden: %w", name, err)
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("Template %s ungültig: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("Template %s nicht ausführbar: %w", name, err)
	}
	return buf.Bytes(), nil
}
