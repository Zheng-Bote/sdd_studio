package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
)

// checkSDD001 (SDD-001): The file configured in artifacts.constitution exists, is a regular file inside the repository and is not empty.
func checkSDD001(ctx *Context, selfRule catalog.Rule) ([]report.Finding, error) {
	rel := strings.TrimSpace(ctx.Config.Artifacts.Constitution)
	if rel == "" {
		return []report.Finding{{
			Path:        config.DefaultConfigFileName,
			Line:        1,
			Message:     "artifacts.constitution ist nicht konfiguriert",
			ContentHash: HashContent("constitution:unconfigured"),
		}}, nil
	}

	cleanRel, ok := repoRelative(rel)
	if !ok {
		return []report.Finding{{
			Path:        config.DefaultConfigFileName,
			Line:        1,
			Message:     fmt.Sprintf("artifacts.constitution %q liegt außerhalb des Repository-Wurzelverzeichnisses", rel),
			ContentHash: HashContent("constitution:outside:" + rel),
		}}, nil
	}

	info, err := os.Stat(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []report.Finding{{
				Path:        cleanRel,
				Line:        1,
				Message:     fmt.Sprintf("Verfassung %s existiert nicht", cleanRel),
				ContentHash: HashContent("constitution:missing:" + cleanRel),
			}}, nil
		}
		return nil, fmt.Errorf("Verfassung %s nicht lesbar: %w", cleanRel, err)
	}
	if !info.Mode().IsRegular() {
		return []report.Finding{{
			Path:        cleanRel,
			Line:        1,
			Message:     fmt.Sprintf("Verfassung %s ist keine reguläre Datei", cleanRel),
			ContentHash: HashContent("constitution:notregular:" + cleanRel),
		}}, nil
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepoRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		return nil, fmt.Errorf("Verfassung %s nicht lesbar: %w", cleanRel, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return []report.Finding{{
			Path:        cleanRel,
			Line:        1,
			Message:     fmt.Sprintf("Verfassung %s ist leer", cleanRel),
			ContentHash: HashContent("constitution:empty:" + cleanRel),
		}}, nil
	}
	return nil, nil
}

// repoRelative normalizes a configured path and reports whether it stays inside the repository root.
func repoRelative(p string) (string, bool) {
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}
