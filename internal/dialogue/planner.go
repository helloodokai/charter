package dialogue

import (
	"context"
	"fmt"

	"github.com/helloodokai/charter/internal/models"
)

func (d *Dialogue) askAcceptanceCriteria(ctx context.Context, charterSummary string) (string, error) {
	prompt := `Given the charter summary below, propose 3-5 acceptance criteria. Each should be:
1. Testable — you can verify it passed or failed
2. Specific — no vague words like "works" or "is good"
3. Minimal — the smallest set that proves the goal is met

Format each as:
- <statement> (verification: test|manual|metric)

Charter:
` + charterSummary

	resp, err := d.routingStreamer.Complete(ctx, models.Mid, models.CompletionRequest{
		System: "You are a test engineer writing acceptance criteria for a coding agent.",
		Messages: []models.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("acceptance criteria LLM call: %w", err)
	}

	return resp.Content + "\n\nDo you want to add, modify, or remove any of these criteria?", nil
}

func (d *Dialogue) askRisk(ctx context.Context, charterSummary string) (string, error) {
	prompt := `Based on the charter summary, assess the risk level. Consider:
- How many files/services are touched?
- Are any critical paths affected?
- How reversible is the change?
- What's the blast radius?

Rate: low, medium, high, or critical.

Charter:
` + charterSummary

	resp, err := d.routingStreamer.Complete(ctx, models.Cheap, models.CompletionRequest{
		System: "You are a risk assessor for software changes. Be conservative.",
		Messages: []models.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("risk assessment LLM call: %w", err)
	}

	return resp.Content + "\n\nDo you agree with this risk assessment? If not, what should it be and why?", nil
}