package dialogue

import (
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

// StreamingMarkdownWriter accumulates streamed LLM tokens and renders
// each completed paragraph as styled terminal markdown before flushing.
type StreamingMarkdownWriter struct {
	mu      sync.Mutex
	buf     strings.Builder
	output  io.Writer
	flushed int
	pR      *glamour.TermRenderer
}

// NewStreamingMarkdownWriter creates a writer that renders markdown
// paragraphs to output as they complete during streaming.
func NewStreamingMarkdownWriter(output io.Writer) *StreamingMarkdownWriter {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	return &StreamingMarkdownWriter{output: output, pR: r}
}

func (s *StreamingMarkdownWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err = s.buf.Write(p)
	if err != nil {
		return n, err
	}
	s.tryFlush()
	return n, nil
}

func (s *StreamingMarkdownWriter) tryFlush() {
	content := s.buf.String()
	for {
		idx := findParagraphBreak(content, s.flushed)
		if idx == -1 {
			return
		}
		paragraph := content[s.flushed:idx]
		s.flushed = idx
		rendered := s.renderParagraph(paragraph)
		if rendered != "" {
			s.output.Write([]byte(rendered))
			s.output.Write([]byte("\n"))
		}
	}
}

// Finish flushes any remaining buffered content and returns the full raw text.
func (s *StreamingMarkdownWriter) Finish() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := strings.TrimSpace(s.buf.String()[s.flushed:])
	if remaining != "" {
		rendered := s.renderParagraph(remaining)
		s.output.Write([]byte(rendered))
		s.output.Write([]byte("\n"))
	}
	return s.buf.String()
}

func (s *StreamingMarkdownWriter) renderParagraph(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if s.pR != nil {
		rendered, err := s.pR.Render(text)
		if err == nil {
			return strings.TrimSpace(rendered)
		}
	}
	return text
}

func findParagraphBreak(content string, start int) int {
	for i := start; i < len(content)-1; i++ {
		if content[i] == '\n' && content[i+1] == '\n' {
			return i + 2
		}
	}
	if len(content)-start > 300 {
		end := min(len(content), start+300)
		lastDot := strings.LastIndex(content[start:end], ". ")
		if lastDot > 50 {
			return start + lastDot + 2
		}
		lastNewline := strings.LastIndex(content[start:end], "\n")
		if lastNewline > 20 {
			return start + lastNewline + 1
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}