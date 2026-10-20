// Package adrlock maintains .sdd/adr-lock.json, the content hashes of decided ADRs (SDD-011).
// Only `sdd baseline` and `sdd adopt` write it; lint and gates read it.
package adrlock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// RelPath is the lockfile location.
const RelPath = ".sdd/adr-lock.json"

// File maps ADR paths to the hash of their normalized body. Entries of deleted ADRs stay (history).
type File struct {
	Version int               `json:"version"`
	Entries map[string]string `json:"entries"`
}

// ADR is one ADR file with its current hash.
type ADR struct {
	Path    string
	Status  string
	Hash    string
	Decided bool // status is not "proposed"
}

// Hash is the SHA-256 over the normalized text without frontmatter, so status changes stay possible.
func Hash(data []byte) string {
	sum := sha256.Sum256([]byte(scan.NormalizeText(scan.Body(data))))
	return hex.EncodeToString(sum[:])
}

// Collect lists the ADRs (NNNN-*.md below artifacts.adr) with their hashes.
func Collect(root string, cfg *config.Config) ([]ADR, error) {
	if cfg.Artifacts.ADR == "" {
		return nil, nil
	}
	list, err := scan.ListArtifacts(root, cfg.Artifacts.ADR, cfg.Exclude)
	if err != nil {
		return nil, fmt.Errorf("ADRs unter %s nicht lesbar: %w", cfg.Artifacts.ADR, err)
	}
	var out []ADR
	for _, a := range list {
		if !scan.IsADRFileName(path.Base(a.Path)) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(a.Path)))
		if err != nil {
			return nil, fmt.Errorf("%s nicht lesbar: %w", a.Path, err)
		}
		out = append(out, ADR{Path: a.Path, Status: a.Frontmatter.Status, Hash: Hash(data), Decided: a.Frontmatter.Status != "proposed"})
	}
	return out, nil
}

// Load reads the lockfile; ok is false if it does not exist.
func Load(root string) (*File, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RelPath)))
	if errors.Is(err, os.ErrNotExist) {
		return &File{Version: 1, Entries: map[string]string{}}, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s nicht lesbar: %w", RelPath, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil || f.Version != 1 {
		return nil, false, fmt.Errorf("%s ungültig (erwartet version 1)", RelPath)
	}
	if f.Entries == nil {
		f.Entries = map[string]string{}
	}
	return &f, true, nil
}

// Marshal renders the lockfile deterministically (map keys are sorted by encoding/json).
func (f *File) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Update adds decided ADRs that are not locked yet. With accept, changed hashes of locked ADRs are replaced too —
// the only way to acknowledge a changed ADR (`sdd baseline --accept SDD-011`).
func Update(f *File, adrs []ADR, accept bool) (added, updated int) {
	for _, a := range adrs {
		if !a.Decided {
			continue
		}
		old, locked := f.Entries[a.Path]
		switch {
		case !locked:
			f.Entries[a.Path] = a.Hash
			added++
		case old != a.Hash && accept:
			f.Entries[a.Path] = a.Hash
			updated++
		}
	}
	return added, updated
}
