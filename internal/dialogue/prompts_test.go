package dialogue

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKickoffPromptIsShort(t *testing.T) {
	prompt := KickoffPrompt
	require.True(t, strings.Contains(prompt, "GOAL:"))
	require.True(t, strings.Contains(prompt, "CONTEXT:"))
	require.False(t, strings.Contains(prompt, "```"))
}

func TestNoPromptsContainCodeBlocks(t *testing.T) {
	prompts := map[string]string{
		"kickoff":       KickoffPrompt,
		"conversation":    ConversationPrompt,
		"non_goals":     AskNonGoalsPrompt,
		"edge_cases":    AskEdgeCasesPrompt,
		"blast_radius":  AskBlastRadiusPrompt,
		"constraints":   AskConstraintsPrompt,
		"extract":       ExtractPrompt,
		"synthesize":    SynthesizePrompt,
		"counterspec":   CounterSpecPrompt,
	}

	for name, prompt := range prompts {
		require.False(t, strings.Contains(prompt, "```"),
			"%s prompt must not contain code blocks", name)
	}
}
