package sources

import (
	"fmt"
	"os"

	"github.com/helloodokai/charter/internal/charter"
)

// FileSource reads charter source material from a local file.
type FileSource struct{}

// NewFileSource returns a new FileSource.
func NewFileSource() *FileSource {
	return &FileSource{}
}

// Fetch reads the file at the given path and returns its contents as a charter Source.
func (s *FileSource) Fetch(path string) (charter.Source, error) {
	data, err := os.ReadFile(path) //nolint:gosec // expected: user-specified file
	if err != nil {
		return charter.Source{}, fmt.Errorf("reading file %s: %w", path, err)
	}
	return charter.Source{
		Type: "file",
		URL:  path,
		Raw:  string(data),
	}, nil
}