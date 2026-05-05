package dialogue

import (
	"context"
	"fmt"
	"strings"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/models"
)

// RunCounterSpec executes a counter-speculative analysis against the current charter using the frontier model.
func (d *Dialogue) RunCounterSpec(ctx context.Context) (*charter.CounterSpec, error) {
	summary := d.charterSummary()
	resp, err := d.routingStreamer.Complete(ctx, models.Frontier, models.CompletionRequest{
		System:   CounterSpecPrompt,
		Messages: []models.Message{{Role: "user", Content: summary}},
	})
	if err != nil {
		return nil, fmt.Errorf("counter-spec LLM call: %w", err)
	}
	cs := parseCounterSpec(resp.Content)
	return &cs, nil
}

// AskCounterSpecReview presents the counter-spec findings to the user for interactive review and filtering.
func (d *Dialogue) AskCounterSpecReview(ctx context.Context, cs *charter.CounterSpec) error {
	if d.nonInteractive {
		d.charter.CounterSpec = *cs
		return nil
	}

	fmt.Println("\n--- Counter-Spec Review ---")
	fmt.Println("The following misinterpretations were identified. Review each one:")

	var kept []string
	for _, m := range cs.Misinterpretations {
		fmt.Printf("\nMisinterpretation: %s\n", m)
		var confirm bool
		fmt.Print("Keep this in the charter? (y/n): ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		confirm = strings.ToLower(strings.TrimSpace(input)) == "y" || strings.ToLower(strings.TrimSpace(input)) == "yes"
		if confirm {
			kept = append(kept, m)
		}
	}

	d.charter.CounterSpec = charter.CounterSpec{
		Misinterpretations: kept,
		AmbiguitiesFlagged: cs.AmbiguitiesFlagged,
	}
	return nil
}