package mcp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

const artifactScheme = "sdd://artifact/"

type resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType"`
}

// artifactPaths lists the readable artifacts: constitution, ADRs and all feature artifacts incl. the archive.
func (s *Server) artifactPaths() ([]string, *config.Config, error) {
	cfg, err := config.Load(s.Root)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var out []string
	if c, err := scan.CleanRel(cfg.Artifacts.Constitution); err == nil {
		if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(c))); err == nil {
			seen[c] = true
			out = append(out, c)
		}
	}
	a := cfg.Artifacts
	for _, spec := range []string{a.ADR, a.PRD, a.Design, a.Plans, a.Archive} {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		list, err := scan.ListArtifacts(s.Root, spec, cfg.Exclude)
		if err != nil {
			return nil, nil, err
		}
		for _, x := range list {
			if !seen[x.Path] {
				seen[x.Path] = true
				out = append(out, x.Path)
			}
		}
	}
	sort.Strings(out)
	return out, cfg, nil
}

func (s *Server) listResources() (any, *rpcError) {
	res := []resource{
		{URI: "sdd://constitution", Name: "constitution", Title: "Verfassung", Description: "Die Verfassung des Projekts (ranghöchstes Artefakt).", MimeType: "text/markdown"},
		{URI: "sdd://catalog", Name: "catalog", Title: "Regelkatalog", Description: "Alle Regeln mit Schweregrad, Gates und Prüfvertrag.", MimeType: "application/yaml"},
		{URI: "sdd://handbook", Name: "handbook", Title: "Handbuch", Description: "Mechanismen, Gates und Prüfverträge aller Regeln.", MimeType: "text/markdown"},
	}
	paths, _, err := s.artifactPaths()
	if err != nil {
		return nil, &rpcError{codeInternalError, err.Error()}
	}
	for _, p := range paths {
		res = append(res, resource{URI: artifactScheme + p, Name: p, MimeType: "text/markdown"})
	}
	return map[string]any{"resources": res}, nil
}

func (s *Server) readResource(uri string) (any, *rpcError) {
	var text, mime string
	switch {
	case uri == "sdd://constitution":
		cfg, err := config.Load(s.Root)
		if err != nil {
			return nil, &rpcError{codeInternalError, err.Error()}
		}
		data, err := readRel(s.Root, cfg.Artifacts.Constitution)
		if err != nil {
			return nil, &rpcError{codeInvalidParams, err.Error()}
		}
		text, mime = data, "text/markdown"
	case uri == "sdd://catalog":
		data, err := readRel(s.Root, catalog.DefaultCatalogRelPath)
		if err != nil {
			cat, _, cerr := catalog.Load(s.Root) // embedded catalog
			if cerr != nil {
				return nil, &rpcError{codeInternalError, cerr.Error()}
			}
			data = fmt.Sprintf("# eingebetteter Katalog, Version %s mit %d Regeln\n", cat.Version, len(cat.Rules))
		}
		text, mime = data, "application/yaml"
	case uri == "sdd://handbook":
		data, err := readRel(s.Root, "handbuch/03-mechanismen.md")
		if err != nil {
			return nil, &rpcError{codeInvalidParams, err.Error()}
		}
		text, mime = data, "text/markdown"
	case strings.HasPrefix(uri, artifactScheme):
		rel, err := scan.CleanRel(strings.TrimPrefix(uri, artifactScheme))
		if err != nil {
			return nil, &rpcError{codeInvalidParams, err.Error()}
		}
		paths, _, perr := s.artifactPaths()
		if perr != nil {
			return nil, &rpcError{codeInternalError, perr.Error()}
		}
		allowed := false
		for _, p := range paths {
			allowed = allowed || p == rel
		}
		if !allowed || !(strings.EqualFold(path.Ext(rel), ".md") || strings.EqualFold(path.Ext(rel), ".sdd")) {
			return nil, &rpcError{codeInvalidParams, "kein Artefakt: " + rel}
		}
		data, err := readRel(s.Root, rel)
		if err != nil {
			return nil, &rpcError{codeInvalidParams, err.Error()}
		}
		text, mime = data, "text/markdown"
	default:
		return nil, &rpcError{codeInvalidParams, "unbekannte Ressource " + uri}
	}
	return map[string]any{"contents": []map[string]any{{"uri": uri, "mimeType": mime, "text": text}}}, nil
}

func readRel(root, rel string) (string, error) {
	clean, err := scan.CleanRel(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return "", fmt.Errorf("%s nicht lesbar: %w", clean, err)
	}
	return string(data), nil
}
