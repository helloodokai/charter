package dialogue

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

var renderer *glamour.TermRenderer

func init() {
	var err error
	renderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		renderer, _ = glamour.NewTermRenderer()
	}
}

func renderMarkdown(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	rendered, err := renderer.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimSpace(rendered)
}

func compactMarkdown(text string) string {
	var b strings.Builder
	lines := strings.Split(text, "\n")
	prevEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !prevEmpty {
				b.WriteByte('\n')
			}
			prevEmpty = true
			continue
		}
		prevEmpty = false
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}