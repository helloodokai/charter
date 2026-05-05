package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/helloodokai/charter/internal/charter"
)

// IndexEntry represents a summary record of a charter in the index.
type IndexEntry struct {
	ID        string           `yaml:"id"`
	Goal      string           `yaml:"goal"`
	Status    charter.Status   `yaml:"status"`
	Risk      charter.Risk     `yaml:"risk"`
	CreatedAt time.Time        `yaml:"created_at"`
	UpdatedAt time.Time        `yaml:"updated_at"`
	Source    charter.Source   `yaml:"source"`
}

// Index is the collection of charter summary records stored on disk.
type Index struct {
	Charters []IndexEntry `yaml:"charters"`
}

// LoadIndex reads the charter index from the given directory.
func LoadIndex(dir string) (*Index, error) {
	path := filepath.Join(dir, "index.yaml")
	data, err := os.ReadFile(path) //nolint:gosec // expected: user-specified path
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, fmt.Errorf("reading index: %w", err)
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &idx, nil
}

// SaveIndex writes the charter index to the given directory.
func SaveIndex(dir string, idx *Index) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	path := filepath.Join(dir, "index.yaml")
	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshalling index: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// UpsertIndex adds or updates the charter's entry in the index.
func UpsertIndex(dir string, c *charter.Charter) error {
	idx, err := LoadIndex(dir)
	if err != nil {
		return err
	}
	entry := IndexEntry{
		ID:        c.ID,
		Goal:      c.Goal,
		Status:    c.Status,
		Risk:      c.Risk,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Source:    c.Source,
	}
	found := false
	for i, e := range idx.Charters {
		if e.ID == c.ID {
			idx.Charters[i] = entry
			found = true
			break
		}
	}
	if !found {
		idx.Charters = append(idx.Charters, entry)
	}
	return SaveIndex(dir, idx)
}

// ListByStatus returns index entries matching the given status, or all entries if status is empty.
func ListByStatus(dir string, status charter.Status) ([]IndexEntry, error) {
	idx, err := LoadIndex(dir)
	if err != nil {
		return nil, err
	}
	var result []IndexEntry
	for _, e := range idx.Charters {
		if status == "" || e.Status == status {
			result = append(result, e)
		}
	}
	return result, nil
}

// ChartersDir returns the path to the charters directory under the given root.
func ChartersDir(root string) string {
	return filepath.Join(root, ".charters")
}