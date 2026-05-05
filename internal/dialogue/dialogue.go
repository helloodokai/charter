package dialogue

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
	"github.com/helloodokai/charter/internal/routing"
)

//go:embed prompts/kickoff.md
var KickoffPrompt string

//go:embed prompts/ask_non_goals.md
var AskNonGoalsPrompt string

//go:embed prompts/ask_edge_cases.md
var AskEdgeCasesPrompt string

//go:embed prompts/ask_blast_radius.md
var AskBlastRadiusPrompt string

//go:embed prompts/ask_constraints.md
var AskConstraintsPrompt string

//go:embed prompts/synthesize.md
var SynthesizePrompt string

//go:embed prompts/counterspec.md
var CounterSpecPrompt string

type GapType string

const (
	GapGoal          GapType = "goal"
	GapContext       GapType = "context"
	GapNonGoals      GapType = "non_goals"
	GapAcceptance    GapType = "acceptance_criteria"
	GapEdgeCases     GapType = "edge_cases"
	GapBlastRadius   GapType = "blast_radius"
	GapConstraints   GapType = "constraints"
	GapUnknowns      GapType = "unknowns"
	GapRisk          GapType = "risk"
	GapRollback      GapType = "rollback"
	GapSynthesize    GapType = "synthesize"
	GapCounterSpec   GapType = "counterspec"
	GapDone          GapType = "done"
)

type Gap struct {
	Type     GapType
	Priority int
	Prompt   string
	Field    string
}

type Dialogue struct {
	charter     *charter.Charter
	transcript  []charter.TranscriptTurn
	gaps        []Gap
	turn        int
	budget      int
	router      *routing.Router
	cfg         *config.Config
	nonInteractive bool
	inputChan   chan string
	outputChan  chan string
}

type DialogueResult struct {
	Charter   *charter.Charter
	GapsLeft  []Gap
	TurnsUsed int
}

func New(c *charter.Charter, router *routing.Router, cfg *config.Config, opts ...Option) *Dialogue {
	d := &Dialogue{
		charter:    c,
		transcript: c.Transcript,
		budget:      cfg.Dialogue.TurnBudget,
		router:      router,
		cfg:         cfg,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

type Option func(*Dialogue)

func WithBudget(n int) Option {
	return func(d *Dialogue) { d.budget = n }
}

func WithNonInteractive(v bool) Option {
	return func(d *Dialogue) { d.nonInteractive = v }
}

func WithChannels(input chan string, output chan string) Option {
	return func(d *Dialogue) {
		d.inputChan = input
		d.outputChan = output
	}
}

func (d *Dialogue) Run(ctx context.Context) (*DialogueResult, error) {
	d.gaps = d.planGaps()

	for d.turn < d.budget && len(d.gaps) > 0 {
		gap := d.gaps[0]
		if gap.Type == GapDone {
			break
		}

		question, err := d.generateQuestion(ctx, gap)
		if err != nil {
			return nil, fmt.Errorf("turn %d: generating question for %s: %w", d.turn, gap.Type, err)
		}

		answer, err := d.askUser(ctx, question)
		if err != nil {
			return nil, fmt.Errorf("turn %d: asking user: %w", d.turn, err)
		}

		if strings.TrimSpace(strings.ToLower(answer)) == "done" {
			break
		}

		if err := d.extract(ctx, gap, answer); err != nil {
			return nil, fmt.Errorf("turn %d: extracting %s: %w", d.turn, gap.Type, err)
		}

		d.gaps = d.gaps[1:]
		d.turn++
	}

	if err := d.synthesize(ctx); err != nil {
		slog.Warn("synthesis failed, charter may be incomplete", "error", err)
	}

	if d.cfg.Dialogue.RequireCounterSpec {
		if err := d.counterspec(ctx); err != nil {
			slog.Warn("counter-spec pass failed", "error", err)
		}
	}

	d.charter.Transcript = d.transcript
	d.charter.UpdatedAt = time.Now().UTC()

	if d.charter.Status == charter.StatusDraft && len(d.charter.AcceptanceCriteria) > 0 && d.charter.Goal != "" {
		d.charter.Status = charter.StatusReady
	}

	return &DialogueResult{
		Charter:   d.charter,
		GapsLeft:  d.gaps,
		TurnsUsed: d.turn,
	}, nil
}

func (d *Dialogue) planGaps() []Gap {
	var gaps []Gap

	if d.charter.Goal == "" {
		gaps = append(gaps, Gap{Type: GapGoal, Priority: 0})
	}
	if d.charter.Context == "" {
		gaps = append(gaps, Gap{Type: GapContext, Priority: 1})
	}
	gaps = append(gaps, Gap{Type: GapNonGoals, Priority: 2})
	gaps = append(gaps, Gap{Type: GapAcceptance, Priority: 3})
	gaps = append(gaps, Gap{Type: GapEdgeCases, Priority: 4})
	if len(d.charter.BlastRadius.Files) == 0 && len(d.charter.BlastRadius.Services) == 0 {
		gaps = append(gaps, Gap{Type: GapBlastRadius, Priority: 5})
	}
	gaps = append(gaps, Gap{Type: GapConstraints, Priority: 6})
	if len(d.charter.Unknowns) == 0 {
		gaps = append(gaps, Gap{Type: GapUnknowns, Priority: 7})
	}
	gaps = append(gaps, Gap{Type: GapRisk, Priority: 8})

	shouldAskRollback := d.charter.Risk == charter.RiskHigh || d.charter.Risk == charter.RiskCritical ||
		d.cfg.Dialogue.AskForRollback == "high" || d.cfg.Dialogue.AskForRollback == "medium" ||
		d.cfg.Dialogue.AskForRollback == "all"
	if shouldAskRollback && d.charter.RollbackPlan == "" {
		gaps = append(gaps, Gap{Type: GapRollback, Priority: 9})
	}

	gaps = append(gaps, Gap{Type: GapSynthesize, Priority: 10})
	gaps = append(gaps, Gap{Type: GapCounterSpec, Priority: 11})

	return gaps
}

func (d *Dialogue) generateQuestion(ctx context.Context, gap Gap) (string, error) {
	charterSummary := d.charterSummary()

	switch gap.Type {
	case GapGoal, GapContext:
		resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
			System:   KickoffPrompt,
			Messages: []models.Message{{Role: "user", Content: d.sourceSummary()}},
		})
		if err != nil {
			return "", fmt.Errorf("kickoff LLM call: %w", err)
		}
		return resp.Content, nil

	case GapNonGoals:
		resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
			System:   AskNonGoalsPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("non-goals LLM call: %w", err)
		}
		return resp.Content, nil

	case GapAcceptance:
		return d.askAcceptanceCriteria(ctx, charterSummary)

	case GapEdgeCases:
		resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
			System:   AskEdgeCasesPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("edge cases LLM call: %w", err)
		}
		return resp.Content, nil

	case GapBlastRadius:
		resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
			System:   AskBlastRadiusPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("blast radius LLM call: %w", err)
		}
		return resp.Content, nil

	case GapConstraints:
		resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
			System:   AskConstraintsPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("constraints LLM call: %w", err)
		}
		return resp.Content, nil

	case GapUnknowns:
		return "Are there any open questions or unknowns that could block this work? List anything you're not sure about, and whether it's blocking or can be resolved later.", nil

	case GapRisk:
		return d.askRisk(ctx, charterSummary)

	case GapRollback:
		return "This charter targets high-or-above risk. What's the rollback plan if something goes wrong? Describe how to safely revert.", nil

	case GapSynthesize, GapCounterSpec:
		return "", nil

	default:
		return "", fmt.Errorf("unknown gap type: %s", gap.Type)
	}
}

func (d *Dialogue) askUser(ctx context.Context, question string) (string, error) {
	d.transcript = append(d.transcript, charter.TranscriptTurn{
		Role:    "tool",
		At:      time.Now().UTC(),
		Content: question,
	})

	if d.nonInteractive {
		return "", nil
	}

	if d.inputChan != nil && d.outputChan != nil {
		d.outputChan <- question
		select {
		case answer := <-d.inputChan:
			d.transcript = append(d.transcript, charter.TranscriptTurn{
				Role:    "human",
				At:      time.Now().UTC(),
				Content: answer,
			})
			return answer, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	var answer string
	field := huh.NewText().
		Title(question).
		Value(&answer).
		CharLimit(4000)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("reading user input: %w", err)
	}

	d.transcript = append(d.transcript, charter.TranscriptTurn{
		Role:    "human",
		At:      time.Now().UTC(),
		Content: answer,
	})

	return answer, nil
}

func (d *Dialogue) extract(ctx context.Context, gap Gap, answer string) error {
	if answer == "" {
		return nil
	}

	switch gap.Type {
	case GapGoal, GapContext:
		return d.extractGoalAndContext(ctx, answer)
	case GapNonGoals:
		return d.extractNonGoals(ctx, answer)
	case GapAcceptance:
		return d.extractAcceptanceCriteria(ctx, answer)
	case GapEdgeCases:
		return d.extractEdgeCases(ctx, answer)
	case GapBlastRadius:
		return d.extractBlastRadius(ctx, answer)
	case GapConstraints:
		return d.extractConstraints(ctx, answer)
	case GapUnknowns:
		return d.extractUnknowns(answer)
	case GapRisk:
		return d.extractRisk(ctx, answer)
	case GapRollback:
		d.charter.RollbackPlan = answer
		return nil
	default:
		return nil
	}
}

func (d *Dialogue) sourceSummary() string {
	s := d.charter.Source
	if s.Raw != "" {
		return s.Raw
	}
	if s.URL != "" {
		return fmt.Sprintf("Source: %s (%s)", s.Type, s.URL)
	}
	return "No source material provided."
}

func (d *Dialogue) charterSummary() string {
	var b strings.Builder
	if d.charter.Goal != "" {
		b.WriteString(fmt.Sprintf("Goal: %s\n", d.charter.Goal))
	}
	if d.charter.Context != "" {
		b.WriteString(fmt.Sprintf("Context: %s\n", d.charter.Context))
	}
	if len(d.charter.NonGoals) > 0 {
		b.WriteString("Non-goals:\n")
		for _, ng := range d.charter.NonGoals {
			b.WriteString(fmt.Sprintf("- %s\n", ng))
		}
	}
	if len(d.charter.AcceptanceCriteria) > 0 {
		b.WriteString("Acceptance criteria:\n")
		for _, ac := range d.charter.AcceptanceCriteria {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", ac.Statement, ac.Verification))
		}
	}
	if len(d.charter.EdgeCases) > 0 {
		b.WriteString("Edge cases:\n")
		for _, ec := range d.charter.EdgeCases {
			b.WriteString(fmt.Sprintf("- %s\n", ec))
		}
	}
	if b.Len() == 0 {
		return "No charter content yet."
	}
	return b.String()
}

func (d *Dialogue) synthesize(ctx context.Context) error {
	summary := d.charterSummary()
	resp, err := d.router.Complete(ctx, models.Mid, models.CompletionRequest{
		System:   SynthesizePrompt,
		Messages: []models.Message{{Role: "user", Content: summary}},
	})
	if err != nil {
		return fmt.Errorf("synthesis: %w", err)
	}

	d.enhanceFromSynthesis(resp.Content)
	return nil
}

func (d *Dialogue) counterspec(ctx context.Context) error {
	summary := d.charterSummary()
	resp, err := d.router.Complete(ctx, models.Frontier, models.CompletionRequest{
		System:   CounterSpecPrompt,
		Messages: []models.Message{{Role: "user", Content: summary}},
	})
	if err != nil {
		return fmt.Errorf("counter-spec: %w", err)
	}

	content := resp.Content

	if d.nonInteractive {
		d.charter.CounterSpec = parseCounterSpec(content)
		return nil
	}

	cs := parseCounterSpec(content)
	var kept []string
	for _, m := range cs.Misinterpretations {
		keep := false
		confirm := huh.NewConfirm().
			Title(fmt.Sprintf("Misinterpretation: %s\nKeep this in the charter?", m)).
			Value(&keep)
		if err := huh.NewForm(huh.NewGroup(confirm)).Run(); err != nil {
			slog.Warn("counter-spec confirm failed", "error", err)
			continue
		}
		if keep {
			kept = append(kept, m)
		}
	}

	d.charter.CounterSpec.Misinterpretations = kept
	d.charter.CounterSpec.AmbiguitiesFlagged = cs.AmbiguitiesFlagged
	return nil
}

func (d *Dialogue) enhanceFromSynthesis(synthesis string) {
}

func parseCounterSpec(content string) charter.CounterSpec {
	var cs charter.CounterSpec
	lines := strings.Split(content, "\n")
	var inAmbiguity bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MISINTERPRETATION") {
			inAmbiguity = false
			after := strings.SplitN(line, ":", 2)
			if len(after) > 1 {
				cs.Misinterpretations = append(cs.Misinterpretations, strings.TrimSpace(after[1]))
			}
		} else if strings.HasPrefix(line, "AMBIGUITIES") {
			inAmbiguity = true
		} else if inAmbiguity && strings.HasPrefix(line, "-") {
			cs.AmbiguitiesFlagged = append(cs.AmbiguitiesFlagged, strings.TrimPrefix(line, "- "))
		}
	}
	return cs
}

