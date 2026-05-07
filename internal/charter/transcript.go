package charter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var roleLabels = map[string]string{
	"human":  "You",
	"tool":   "Charter",
	"system": "System",
}

// FormatTranscript renders the charter's transcript as a human-readable Markdown document.
func FormatTranscript(c *Charter) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Charter Transcript: %s\n\n", c.Goal)
	fmt.Fprintf(&b, "- **ID**: %s\n", c.ID)
	fmt.Fprintf(&b, "- **Created**: %s\n", c.CreatedAt.Format(time.RFC1123))
	fmt.Fprintf(&b, "- **Author**: %s\n", strings.Join(c.Authors, ", "))
	if c.Source.Type != "" {
		fmt.Fprintf(&b, "- **Source**: %s\n", c.Source.Type)
	}
	fmt.Fprintf(&b, "\n---\n\n")

	for i, turn := range c.Transcript {
		label := turn.Role
		if l, ok := roleLabels[turn.Role]; ok {
			label = l
		}
		ts := turn.At.Format("15:04:05")
		fmt.Fprintf(&b, "## [%s] %s (%s)\n\n", fmt.Sprintf("%d", i+1), label, ts)
		fmt.Fprintf(&b, "%s\n\n", turn.Content)
	}

	return b.String()
}

// TranscriptFilePath returns the path to the transcript markdown file for the given charter ID.
func TranscriptFilePath(dir string, id string) string {
	return filepath.Join(dir, id+".transcript.md")
}

// SaveTranscript writes the charter transcript as a Markdown file and sets TranscriptFile on the charter.
func SaveTranscript(dir string, c *Charter) error {
	if len(c.Transcript) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	content := FormatTranscript(c)
	path := TranscriptFilePath(dir, c.ID)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing transcript %s: %w", path, err)
	}
	c.TranscriptFile = filepath.Base(path)
	return nil
}

// SaveSpec writes a SPEC.md file for the charter and sets SpecFile on the charter.
func SaveSpec(dir string, c *Charter, content string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating charters dir: %w", err)
	}
	path := SpecFilePath(dir, c.ID)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing spec %s: %w", path, err)
	}
	c.SpecFile = filepath.Base(path)
	return nil
}

// SpecFilePath returns the path to the spec markdown file for the given charter ID.
func SpecFilePath(dir string, id string) string {
	return filepath.Join(dir, id+".spec.md")
}