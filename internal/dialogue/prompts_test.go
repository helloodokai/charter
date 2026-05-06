package dialogue

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKickoffPromptDoesNotSolveProblems(t *testing.T) {
	prompt := KickoffPrompt
	antiImplementationRules := []string{
		"NOT solve problems",
		"NOT an assistant",
		"do NOT solve",
	}
	for _, rule := range antiImplementationRules {
		require.True(t, strings.Contains(prompt, rule),
			"kickoff prompt must contain anti-implementation rule: %q", rule)
	}

	forbiddenContent := []string{
		"here's how",
		"step by step",
		"you can use",
		"example:",
		"```",
	}
	lower := strings.ToLower(prompt)
	for _, forbidden := range forbiddenContent {
		require.False(t, strings.Contains(lower, forbidden),
			"kickoff prompt must not contain implementation guidance: %q", forbidden)
	}
}

func TestConversationPromptDoesNotSolveProblems(t *testing.T) {
	prompt := ConversationPrompt
	antiRules := []string{
		"SPECIFICATION",
		"NEVER provide implementation",
		"NEVER write tutorials",
	}
	for _, rule := range antiRules {
		require.True(t, strings.Contains(prompt, rule),
			"conversation prompt must contain anti-implementation rule: %q", rule)
	}
}

func TestNonGoalsPromptAsksNotTells(t *testing.T) {
	prompt := AskNonGoalsPrompt
	require.True(t, strings.Contains(prompt, "SPECIFICATION"),
		"non-goals prompt must identify as specification engine")
	require.True(t, strings.Contains(prompt, "NEVER"),
		"non-goals prompt must contain NEVER rule")
	require.False(t, strings.Contains(strings.ToLower(prompt), "here's how"),
		"non-goals prompt must not contain how-to guidance")
}

func TestEdgeCasesPromptAsksNotSolves(t *testing.T) {
	prompt := AskEdgeCasesPrompt
	require.True(t, strings.Contains(prompt, "SPECIFICATION"),
		"edge cases prompt must identify as specification engine")
	require.True(t, strings.Contains(prompt, "NEVER"),
		"edge cases prompt must contain NEVER rule")
	require.False(t, strings.Contains(strings.ToLower(prompt), "here's how"),
		"edge cases prompt must not contain how-to guidance")
}

func TestBlastRadiusPromptAsksNotSolves(t *testing.T) {
	prompt := AskBlastRadiusPrompt
	require.True(t, strings.Contains(prompt, "SPECIFICATION"),
		"blast radius prompt must identify as specification engine")
	require.True(t, strings.Contains(prompt, "NEVER"),
		"blast radius prompt must contain NEVER rule")
}

func TestConstraintsPromptAsksNotSolves(t *testing.T) {
	prompt := AskConstraintsPrompt
	require.True(t, strings.Contains(prompt, "SPECIFICATION"),
		"constraints prompt must identify as specification engine")
	require.True(t, strings.Contains(prompt, "NEVER"),
		"constraints prompt must contain NEVER rule")
}

func TestExtractPromptDoesNotAddContent(t *testing.T) {
	prompt := ExtractPrompt
	require.True(t, strings.Contains(prompt, "NEVER"),
		"extract prompt must contain NEVER rule")
	require.True(t, strings.Contains(prompt, "ONLY what the user") || strings.Contains(prompt, "only what the user"),
		"extract prompt must specify user-only extraction")
}

func TestSynthesizePromptDoesNotAddImplementation(t *testing.T) {
	prompt := SynthesizePrompt
	require.True(t, strings.Contains(prompt, "NEVER add implementation"),
		"synthesize prompt must forbid adding implementation")
	require.True(t, strings.Contains(prompt, "No implementation details"),
		"synthesize prompt must say 'No implementation details'")
}

func TestCounterSpecPromptDoesNotSolve(t *testing.T) {
	prompt := CounterSpecPrompt
	require.True(t, strings.Contains(prompt, "NEVER suggest solutions"),
		"counterspec prompt must forbid suggesting solutions")
	require.True(t, strings.Contains(prompt, "SPECIFICATION AMBIGUITIES"),
		"counterspec prompt must focus on specification ambiguities")
}

func TestAllPromptsIdentifyAsCharter(t *testing.T) {
	prompts := map[string]string{
		"kickoff":       KickoffPrompt,
		"conversation":   ConversationPrompt,
		"non_goals":     AskNonGoalsPrompt,
		"edge_cases":    AskEdgeCasesPrompt,
		"blast_radius":  AskBlastRadiusPrompt,
		"constraints":   AskConstraintsPrompt,
		"extract":       ExtractPrompt,
		"synthesize":    SynthesizePrompt,
		"counterspec":   CounterSpecPrompt,
	}

	for name, prompt := range prompts {
		lower := strings.ToLower(prompt)
		require.True(t, strings.Contains(lower, "charter"),
			"%s prompt must identify as CHARTER", name)
	}
}

func TestNoPromptContainsCodeBlocks(t *testing.T) {
	prompts := map[string]string{
		"kickoff":       KickoffPrompt,
		"conversation":   ConversationPrompt,
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