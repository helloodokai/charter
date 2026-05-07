package dialogue

import (
	"strings"
)

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