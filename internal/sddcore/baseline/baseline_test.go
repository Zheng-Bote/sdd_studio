package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An empty baseline is written as "entries": [] — sdd baseline --prune removing the last entry wrote null.
func TestSaveWritesEmptyListNotNull(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, &File{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DefaultBaselineRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"entries": []`) || strings.Contains(string(data), "null") {
		t.Errorf("baseline: %s", data)
	}
}
