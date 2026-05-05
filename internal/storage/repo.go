package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/helloodokai/charter/internal/charter"
)

// Save writes a charter to the given directory as a YAML file.
func Save(dir string, c *charter.Charter) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	path := filepath.Join(dir, c.ID+".yaml")
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling charter: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadByID reads a charter from the given directory by its ID.
func LoadByID(dir string, id string) (*charter.Charter, error) {
	path := filepath.Join(dir, id+".yaml")
	return charter.Load(path)
}