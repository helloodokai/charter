package sources

import (
	"fmt"
	"os"

	"github.com/helloodokai/charter/internal/charter"
)

type FileSource struct{}

func NewFileSource() *FileSource {
	return &FileSource{}
}

func (s *FileSource) Fetch(path string) (charter.Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return charter.Source{}, fmt.Errorf("reading file %s: %w", path, err)
	}
	return charter.Source{
		Type: "file",
		URL:  path,
		Raw:  string(data),
	}, nil
}