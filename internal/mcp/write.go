package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/guide"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/rules"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scaffold"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// maxWriteBytes bounds the content sdd_write accepts.
const maxWriteBytes = 1 << 20

// pathClass says which kind of file a repository path is for sdd_read and sdd_write.
type pathClass int

const (
	classNone pathClass = iota
	classArtifact
	classConfig
	classTest
)

// classify decides whether rel may be read or written by sdd_read/sdd_write: the constitution and Markdown artifacts,
// sdd.config.yaml, and test files (tests.include, empty = fallback list). .git, .sdd and exclude matches never are.
func classify(cfg *config.Config, rel string) (pathClass, error) {
	first, _, _ := strings.Cut(rel, "/")
	if first == ".git" || first == ".sdd" {
		return classNone, fmt.Errorf("nicht beschreibbar: %s liegt in %s/", rel, first)
	}
	if scan.MatchAny(cfg.Exclude, rel) {
		return classNone, fmt.Errorf("nicht beschreibbar: %s ist über exclude ausgenommen", rel)
	}
	if rel == config.DefaultConfigFileName {
		return classConfig, nil
	}
	a := cfg.Artifacts
	if c, err := scan.CleanRel(a.Constitution); err == nil && c == rel {
		return classArtifact, nil
	}
	if strings.EqualFold(path.Ext(rel), ".md") {
		for _, spec := range append([]string{a.ADR, a.PRD, a.Design, a.Plans, a.Archive}, a.Docs...) {
			if strings.TrimSpace(spec) != "" && scan.InSpec(rel, spec) {
				return classArtifact, nil
			}
		}
	}
	include := cfg.Tests.Include
	if len(include) == 0 {
		include = rules.FallbackTestGlobs
	}
	if scan.MatchAny(include, rel) {
		return classTest, nil
	}
	return classNone, fmt.Errorf("nicht beschreibbar: %s ist weder Artefakt, Konfiguration noch Testdatei (tests.include)", rel)
}

// readExisting returns the current content of rel; exists is false if there is no file. Anything but a regular file
// (directory, symbolic link, device) is an error.
func readExisting(root, rel string) (data []byte, exists bool, err error) {
	if err := insideRoot(root, rel); err != nil {
		return nil, false, err
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("nicht beschreibbar: %s ist keine reguläre Datei", rel)
	}
	data, err = os.ReadFile(target)
	return data, err == nil, err
}

// insideRoot rejects rel if its nearest existing parent directory resolves (through symbolic links) outside root.
func insideRoot(root, rel string) error {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel)))
	for {
		real, err := filepath.EvalSymlinks(dir)
		if err == nil {
			if real != realRoot && !strings.HasPrefix(real, realRoot+string(filepath.Separator)) {
				return fmt.Errorf("%w: %s (über symbolischen Link)", scan.ErrUnsafePath, rel)
			}
			return nil
		}
		parent := filepath.Dir(dir)
		if !errors.Is(err, fs.ErrNotExist) || dir == filepath.Clean(root) || parent == dir {
			return err
		}
		dir = parent
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func lineCount(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	return strings.Count(strings.TrimSuffix(string(data), "\n"), "\n") + 1
}

// checkArtifact applies what every artifact write must satisfy before anything is written: an ADR is never decided by
// a tool (ADR 0003, critical), a new archive-after lies not before today, and the rules report no error for the
// content at rel. It returns the remaining findings (warnings and infos).
func (s *Server) checkArtifact(cfg *config.Config, rel string, old []byte, exists bool, content string) ([]report.Finding, error) {
	newFM, _ := scan.ParseFrontmatter([]byte(content))
	var oldFM scan.Frontmatter
	if exists {
		oldFM, _ = scan.ParseFrontmatter(old)
	}
	if adr := strings.TrimSpace(cfg.Artifacts.ADR); adr != "" && scan.InSpec(rel, adr) &&
		newFM.Status == "accepted" && (!exists || oldFM.Status != "accepted") {
		return nil, errors.New("ein ADR entscheidet der Mensch: status: accepted setzt kein Werkzeug (ADR 0003, kritische Aktion)")
	}
	if after := strings.TrimSpace(newFM.ArchiveAfter); after != strings.TrimSpace(oldFM.ArchiveAfter) || !exists {
		if err := scaffold.CheckArchiveAfter(after, s.Now()); err != nil {
			return nil, err
		}
	}
	d, err := guide.ValidateDraft(s.Root, rel, content)
	if err != nil {
		return nil, err
	}
	if d.Errors > 0 {
		var msgs []string
		for _, f := range d.Findings {
			if f.Severity == "error" {
				msgs = append(msgs, fmt.Sprintf("%s (Zeile %d): %s", f.RuleID, f.Line, f.Message))
			}
		}
		return nil, fmt.Errorf("Entwurf für %s hat %d Fehler, nichts geschrieben: %s", rel, d.Errors, strings.Join(msgs, "; "))
	}
	return d.Findings, nil
}

type writeOp struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // create, update, unchanged, skip
	Reason  string `json:"reason,omitempty"`
	Diff    string `json:"diff,omitempty"`
	Content string `json:"content,omitempty"`
}

func (s *Server) toolWrite(raw json.RawMessage) (*toolResult, error) {
	a, err := args[struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		BaseSHA256 string `json:"base_sha256"`
		Apply      bool   `json:"apply"`
	}](raw)
	if err != nil {
		return nil, err
	}
	if a.Apply && !s.AllowWrite {
		return nil, errors.New("Schreiben ist nicht freigegeben: der Server läuft ohne --allow-write; ohne apply gibt es die Vorschau")
	}
	if len(a.Content) > maxWriteBytes {
		return nil, fmt.Errorf("Inhalt zu groß: %d Bytes (höchstens %d)", len(a.Content), maxWriteBytes)
	}
	rel, err := scan.CleanRel(a.Path)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(s.Root)
	if err != nil {
		return nil, err
	}
	class, err := classify(cfg, rel)
	if err != nil {
		return nil, err
	}
	old, exists, err := readExisting(s.Root, rel)
	if err != nil {
		return nil, err
	}
	if exists && strings.TrimSpace(a.BaseSHA256) != sha256Hex(old) {
		return nil, fmt.Errorf("base_sha256 fehlt oder ist veraltet; aktueller Stand von %s: %s — erst mit sdd_read lesen", rel, sha256Hex(old))
	}

	findings := []report.Finding{}
	switch class {
	case classArtifact:
		if findings, err = s.checkArtifact(cfg, rel, old, exists, a.Content); err != nil {
			return nil, err
		}
	case classConfig:
		if _, err := config.Parse([]byte(a.Content)); err != nil {
			return nil, fmt.Errorf("sdd.config.yaml nicht geschrieben: %w", err)
		}
	}

	op := writeOp{Path: rel, Action: "create", Content: a.Content}
	if exists {
		op = writeOp{Path: rel, Action: "update", Diff: scaffold.UnifiedDiff(rel, old, []byte(a.Content))}
		if op.Diff == "" {
			op.Action = "unchanged"
		}
	}
	if !a.Apply {
		return resultOf(map[string]any{"applied": false, "ops": []writeOp{op}, "findings": findings}), nil
	}

	status := "unchanged"
	if op.Action != "unchanged" {
		planned := scaffold.Op{Path: rel, Content: []byte(a.Content)}
		if exists {
			planned.Replace, planned.Expect = true, old
		}
		results, err := scaffold.Plan{Ops: []scaffold.Op{planned}}.Apply(s.Root)
		if err != nil {
			return nil, err
		}
		if results[0].Status == scaffold.StatusSkipped {
			return nil, fmt.Errorf("%s nicht geschrieben: %s", rel, results[0].Reason)
		}
		status = results[0].Status
	}
	written, _, err := readExisting(s.Root, rel)
	if err != nil {
		return nil, err
	}
	return resultOf(map[string]any{
		"applied": true, "path": rel, "status": status, "sha256": sha256Hex(written), "lines": lineCount(written),
		"diff": scaffold.UnifiedDiff(rel, old, written), "findings": findings,
	}), nil
}
