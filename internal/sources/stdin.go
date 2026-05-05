package sources

import (
	"io"
	"os"

	"github.com/helloodokai/charter/internal/charter"
)

type StdinSource struct{}

func NewStdinSource() *StdinSource {
	return &StdinSource{}
}

func (s *StdinSource) Fetch() (charter.Source, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return charter.Source{}, err
	}
	if stat.Size() == 0 {
		return charter.Source{
			Type: "stdin",
			Raw:  "",
		}, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return charter.Source{}, err
	}
	return charter.Source{
		Type: "stdin",
		Raw:  string(data),
	}, nil
}