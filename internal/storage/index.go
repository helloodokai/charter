package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/helloodokai/charter/internal/charter"
)

type IndexEntry struct {
	ID        string           `yaml:"id"`
	Goal      string           `yaml:"goal"`
	Status    charter.Status   `yaml:"status"`
	Risk      charter.Risk     `yaml:"risk"`
	CreatedAt time.Time        `yaml:"created_at"`
	UpdatedAt time.Time        `yaml:"updated_at"`
	Source    charter.Source   `yaml:"source"`
}

type Index struct {
	Charters []IndexEntry `yaml:"charters"`
}

func LoadIndex(dir string) (*Index, error) {
	path := filepath.Join(dir, "index.yaml")
	data, err := os.ReadFile(path)
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

func SaveIndex(dir string, idx *Index) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	path := filepath.Join(dir, "index.yaml")
	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshalling index: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

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

func ChartersDir(root string) string {
	return filepath.Join(root, ".charters")
}