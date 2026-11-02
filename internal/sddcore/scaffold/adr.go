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

// AcceptADRPlan decides a proposed ADR: status: proposed becomes status: accepted, nothing else changes. Deciding an
// ADR is a critical action (ADR 0003); callers must make sure a human confirmed exactly this call.
func AcceptADRPlan(root string, cfg *config.Config, rel string) (Plan, error) {
	clean, err := scan.CleanRel(rel)
	if err != nil {
		return Plan{}, err
	}
	dir := strings.TrimSpace(cfg.Artifacts.ADR)
	if dir == "" || !scan.InSpec(clean, dir) || !scan.IsADRFileName(path.Base(clean)) {
		return Plan{}, fmt.Errorf("%w: %s ist kein ADR (NNNN-*.md unter artifacts.adr)", ErrInvalidInput, clean)
	}
	target := filepath.Join(root, filepath.FromSlash(clean))
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() {
		return Plan{}, fmt.Errorf("%w: %s existiert nicht oder ist keine reguläre Datei", ErrInvalidInput, clean)
	}
	old, err := os.ReadFile(target)
	if err != nil {
		return Plan{}, err
	}
	if fm, ok := scan.ParseFrontmatter(old); !ok || fm.Status != "proposed" {
		return Plan{}, fmt.Errorf("%w: %s trägt status: %q; entscheiden lässt sich nur ein ADR mit status: proposed", ErrInvalidInput, clean, fm.Status)
	}
	content, err := setFrontmatter(string(old), "status", "accepted")
	if err != nil {
		return Plan{}, fmt.Errorf("%w: %s: %v", ErrInvalidInput, clean, err)
	}
	return Plan{Ops: []Op{{Path: clean, Content: []byte(content), Replace: true, Expect: old}}}, nil
}
