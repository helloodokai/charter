package dialogue

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
	"github.com/helloodokai/charter/internal/routing"
)

//go:embed prompts/kickoff.md
// KickoffPrompt is the embedded system prompt for the initial charter kickoff turn.
var KickoffPrompt string

//go:embed prompts/ask_non_goals.md
// AskNonGoalsPrompt is the embedded system prompt for eliciting non-goals from the user.
var AskNonGoalsPrompt string

//go:embed prompts/ask_edge_cases.md
// AskEdgeCasesPrompt is the embedded system prompt for identifying edge cases.
var AskEdgeCasesPrompt string

//go:embed prompts/ask_blast_radius.md
// AskBlastRadiusPrompt is the embedded system prompt for analyzing blast radius.
var AskBlastRadiusPrompt string

//go:embed prompts/ask_constraints.md
// AskConstraintsPrompt is the embedded system prompt for inferring project constraints.
var AskConstraintsPrompt string

//go:embed prompts/synthesize.md
// SynthesizePrompt is the embedded system prompt for the synthesis pass.
var SynthesizePrompt string

//go:embed prompts/counterspec.md
// CounterSpecPrompt is the embedded system prompt for the counter-speculative analysis.
var CounterSpecPrompt string

// GapType enumerates the categories of information the dialogue must fill.
type GapType string

// Gap types the dialogue tracks.
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

// Gap represents a missing piece of information that the dialogue should fill.
type Gap struct {
	Type     GapType
	Priority int
	Prompt   string
	Field    string
}

type routerStreamer interface {
	Stream(ctx context.Context, tier models.Tier, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error)
	Complete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (*models.CompletionResponse, error)
}

// Dialogue drives an interactive session that fills out a Charter through guided prompts.
type Dialogue struct {
	charter         *charter.Charter
	transcript      []charter.TranscriptTurn
	gaps            []Gap
	turn            int
	budget          int
	router          *routing.Router
	routingStreamer routerStreamer
	cfg             *config.Config
	nonInteractive  bool
	inputChan       chan string
	outputChan      chan string
	output          io.Writer
	chartersDir     string
	resumeMode      bool
}

// Result holds the outcome of a completed dialogue session.
type Result struct {
	Charter   *charter.Charter
	GapsLeft  []Gap
	TurnsUsed int
}

var (
	styleGap     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleThink   = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("8"))
	styleSection = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleAccent  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleWarn    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
)

func gapLabel(g GapType) string {
	switch g {
	case GapGoal:
		return "Goal"
	case GapContext:
		return "Context"
	case GapNonGoals:
		return "Non-Goals"
	case GapAcceptance:
		return "Acceptance Criteria"
	case GapEdgeCases:
		return "Edge Cases"
	case GapBlastRadius:
		return "Blast Radius"
	case GapConstraints:
		return "Constraints"
	case GapUnknowns:
		return "Unknowns"
	case GapRisk:
		return "Risk Assessment"
	case GapRollback:
		return "Rollback Plan"
	case GapSynthesize:
		return "Synthesis"
	case GapCounterSpec:
		return "Counter-Spec Review"
	default:
		return string(g)
	}
}

// New creates a Dialogue ready to fill the given charter.
func New(c *charter.Charter, router *routing.Router, cfg *config.Config, opts ...Option) *Dialogue {
	d := &Dialogue{
		charter:         c,
		transcript:      c.Transcript,
		budget:          cfg.Dialogue.TurnBudget,
		router:          router,
		routingStreamer: router,
		cfg:             cfg,
		output:          os.Stderr,
	}
	for _, opt := range opts {
		opt(d)
	}
	if d.routingStreamer == nil {
		d.routingStreamer = router
	}
	return d
}

// Option configures a Dialogue.
type Option func(*Dialogue)

// WithBudget sets the maximum number of dialogue turns.
func WithBudget(n int) Option {
	return func(d *Dialogue) { d.budget = n }
}

// WithNonInteractive disables interactive prompts when true.
func WithNonInteractive(v bool) Option {
	return func(d *Dialogue) { d.nonInteractive = v }
}

// WithChannels sets input/output channels for programmatic dialogue interaction.
func WithChannels(input chan string, output chan string) Option {
	return func(d *Dialogue) {
		d.inputChan = input
		d.outputChan = output
	}
}

// WithOutput sets the writer used for dialogue output.
func WithOutput(w io.Writer) Option {
	return func(d *Dialogue) { d.output = w }
}

// WithChartersDir sets the directory where charters are persisted between turns.
func WithChartersDir(dir string) Option {
	return func(d *Dialogue) { d.chartersDir = dir }
}

// WithResume enables resume mode, skipping gaps already filled in the charter.
func WithResume(v bool) Option {
	return func(d *Dialogue) { d.resumeMode = v }
}

// Run executes the dialogue session, iterating through gaps until the budget is exhausted or all gaps are filled.
func (d *Dialogue) Run(ctx context.Context) (*Result, error) {
	d.gaps = d.planGaps()

	fmt.Fprintf(d.output, "\n%s\n", styleSection.Render(" Charter Dialogue "))
	fmt.Fprintf(d.output, "%s\n\n", styleDim.Render(fmt.Sprintf("Turns: 0/%d | Gaps: %d", d.budget, len(d.gaps))))

	for d.turn < d.budget && len(d.gaps) > 0 {
		gap := d.gaps[0]
		if gap.Type == GapDone {
			break
		}

		fmt.Fprintf(d.output, "\n%s\n", styleGap.Render(fmt.Sprintf("▸ Step %d: %s", d.turn+1, gapLabel(gap.Type))))

		question, err := d.generateQuestion(ctx, gap)
		if err != nil {
			return nil, fmt.Errorf("turn %d: generating question for %s: %w", d.turn, gap.Type, err)
		}

		answer, err := d.askUser(ctx, gap, question)
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

		d.persistTranscript()

		fmt.Fprintf(d.output, "%s\n", styleDim.Render(fmt.Sprintf("  Turns: %d/%d | Remaining: %d", d.turn, d.budget, len(d.gaps))))
	}

	fmt.Fprintf(d.output, "\n%s ", styleThink.Render("Synthesizing charter"))
	if err := d.synthesize(ctx); err != nil {
		fmt.Fprintf(d.output, "\n%s\n", styleWarn.Render("⚠ Synthesis had issues, charter may be incomplete"))
		slog.Warn("synthesis failed, charter may be incomplete", "error", err)
	} else {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("done."))
	}

	if d.cfg.Dialogue.RequireCounterSpec {
		fmt.Fprintf(d.output, "\n%s ", styleThink.Render("Running counter-spec review"))
		if err := d.counterspec(ctx); err != nil {
			fmt.Fprintf(d.output, "\n%s\n", styleWarn.Render("⚠ Counter-spec pass failed"))
			slog.Warn("counter-spec pass failed", "error", err)
		} else {
			fmt.Fprintf(d.output, "%s\n", styleDim.Render("done."))
		}
	}

	d.charter.Transcript = d.transcript
	d.charter.UpdatedAt = time.Now().UTC()

	if d.charter.Status == charter.StatusDraft && len(d.charter.AcceptanceCriteria) > 0 && d.charter.Goal != "" {
		d.charter.Status = charter.StatusReady
	}

	fmt.Fprintf(d.output, "\n%s\n", styleSection.Render(" ✓ Charter Complete "))
	fmt.Fprintf(d.output, "  Goal: %s\n", d.charter.Goal)
	fmt.Fprintf(d.output, "  Status: %s\n", d.charter.Status)
	fmt.Fprintf(d.output, "  Risk: %s\n", d.charter.Risk)
	fmt.Fprintf(d.output, "  Turns used: %d\n\n", d.turn)

	return &Result{
		Charter:   d.charter,
		GapsLeft:  d.gaps,
		TurnsUsed: d.turn,
	}, nil
}

func (d *Dialogue) planGaps() []Gap {
	var gaps []Gap

	if d.charter.Goal == "" {
		gaps = append(gaps, Gap{Type: GapGoal, Priority: 0})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Goal already set — skipping"))
	}
	if d.charter.Context == "" {
		gaps = append(gaps, Gap{Type: GapContext, Priority: 1})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Context already set — skipping"))
	}
	if len(d.charter.NonGoals) == 0 || !d.resumeMode {
		gaps = append(gaps, Gap{Type: GapNonGoals, Priority: 2})
	} else {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Non-goals already set — skipping"))
	}
	if len(d.charter.AcceptanceCriteria) == 0 || !d.resumeMode {
		gaps = append(gaps, Gap{Type: GapAcceptance, Priority: 3})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Acceptance criteria already set — skipping"))
	}
	if len(d.charter.EdgeCases) == 0 || !d.resumeMode {
		gaps = append(gaps, Gap{Type: GapEdgeCases, Priority: 4})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Edge cases already set — skipping"))
	}
	if len(d.charter.BlastRadius.Files) == 0 && len(d.charter.BlastRadius.Services) == 0 {
		gaps = append(gaps, Gap{Type: GapBlastRadius, Priority: 5})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Blast radius already set — skipping"))
	}
	constraintsEmpty := len(d.charter.Constraints.Performance) == 0 && len(d.charter.Constraints.Security) == 0 && len(d.charter.Constraints.Compatibility) == 0 && len(d.charter.Constraints.Style) == 0 && len(d.charter.Constraints.Dependencies) == 0
	if constraintsEmpty || !d.resumeMode {
		gaps = append(gaps, Gap{Type: GapConstraints, Priority: 6})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Constraints already set — skipping"))
	}
	if len(d.charter.Unknowns) == 0 {
		gaps = append(gaps, Gap{Type: GapUnknowns, Priority: 7})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Unknowns already set — skipping"))
	}
	if d.charter.Risk == "" || !d.resumeMode {
		gaps = append(gaps, Gap{Type: GapRisk, Priority: 8})
	} else if d.resumeMode {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Risk already assessed — skipping"))
	}

	shouldAskRollback := d.charter.Risk == charter.RiskHigh || d.charter.Risk == charter.RiskCritical ||
		d.cfg.Dialogue.AskForRollback == "high" || d.cfg.Dialogue.AskForRollback == "medium" ||
		d.cfg.Dialogue.AskForRollback == "all"
	if shouldAskRollback && d.charter.RollbackPlan == "" {
		gaps = append(gaps, Gap{Type: GapRollback, Priority: 9})
	} else if d.resumeMode && d.charter.RollbackPlan != "" {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render("  ✓ Rollback plan already set — skipping"))
	}

	gaps = append(gaps, Gap{Type: GapSynthesize, Priority: 10})
	gaps = append(gaps, Gap{Type: GapCounterSpec, Priority: 11})

	return gaps
}

func (d *Dialogue) generateQuestion(ctx context.Context, gap Gap) (string, error) {
	charterSummary := d.charterSummary()

	switch gap.Type {
	case GapGoal, GapContext:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Thinking..."))
		resp, err := d.streamComplete(ctx, models.Mid, models.CompletionRequest{
			System:   KickoffPrompt,
			Messages: []models.Message{{Role: "user", Content: d.sourceSummary()}},
		})
		if err != nil {
			return "", fmt.Errorf("kickoff LLM call: %w", err)
		}
		return resp, nil

	case GapNonGoals:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Generating non-goals..."))
		resp, err := d.streamComplete(ctx, models.Mid, models.CompletionRequest{
			System:   AskNonGoalsPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("non-goals LLM call: %w", err)
		}
		return resp, nil

	case GapAcceptance:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Creating acceptance criteria..."))
		return d.askAcceptanceCriteria(ctx, charterSummary)

	case GapEdgeCases:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Identifying edge cases..."))
		resp, err := d.streamComplete(ctx, models.Mid, models.CompletionRequest{
			System:   AskEdgeCasesPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("edge cases LLM call: %w", err)
		}
		return resp, nil

	case GapBlastRadius:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Analyzing blast radius..."))
		resp, err := d.streamComplete(ctx, models.Mid, models.CompletionRequest{
			System:   AskBlastRadiusPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("blast radius LLM call: %w", err)
		}
		return resp, nil

	case GapConstraints:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Inferring constraints..."))
		resp, err := d.streamComplete(ctx, models.Mid, models.CompletionRequest{
			System:   AskConstraintsPrompt,
			Messages: []models.Message{{Role: "user", Content: charterSummary}},
		})
		if err != nil {
			return "", fmt.Errorf("constraints LLM call: %w", err)
		}
		return resp, nil

	case GapUnknowns:
		return "Are there any open questions or unknowns that could block this work? List anything you're not sure about, and whether it's blocking or can be resolved later.", nil

	case GapRisk:
		fmt.Fprintf(d.output, "%s\n", styleThink.Render("  Assessing risk level..."))
		return d.askRisk(ctx, charterSummary)

	case GapRollback:
		return "This charter targets high-or-above risk. What's the rollback plan if something goes wrong? Describe how to safely revert.", nil

	case GapSynthesize, GapCounterSpec:
		return "", nil

	default:
		return "", fmt.Errorf("unknown gap type: %s", gap.Type)
	}
}

func (d *Dialogue) streamComplete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (string, error) {
	fmt.Fprintf(d.output, "\n")
	resp, err := d.routingStreamer.Stream(ctx, tier, req, d.output)
	fmt.Fprintf(d.output, "\n\n")
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (d *Dialogue) askUser(ctx context.Context, gap Gap, question string) (string, error) {
	if question != "" {
		d.transcript = append(d.transcript, charter.TranscriptTurn{
			Role:    "tool",
			At:      time.Now().UTC(),
			Content: question,
		})
	}

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

	fmt.Fprintf(d.output, "%s\n\n", styleAccent.Render("Your response:"))

	var answer string
	field := huh.NewText().
		Title("Your response:").
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
	bPtr := &b
	if d.charter.Goal != "" {
		fmt.Fprintf(bPtr, "Goal: %s\n", d.charter.Goal)
	}
	if d.charter.Context != "" {
		fmt.Fprintf(bPtr, "Context: %s\n", d.charter.Context)
	}
	if len(d.charter.NonGoals) > 0 {
		b.WriteString("Non-goals:\n")
		for _, ng := range d.charter.NonGoals {
			fmt.Fprintf(bPtr, "- %s\n", ng)
		}
	}
	if len(d.charter.AcceptanceCriteria) > 0 {
		b.WriteString("Acceptance criteria:\n")
		for _, ac := range d.charter.AcceptanceCriteria {
			fmt.Fprintf(bPtr, "- %s (%s)\n", ac.Statement, ac.Verification)
		}
	}
	if len(d.charter.EdgeCases) > 0 {
		b.WriteString("Edge cases:\n")
		for _, ec := range d.charter.EdgeCases {
			fmt.Fprintf(bPtr, "- %s\n", ec)
		}
	}
	if b.Len() == 0 {
		return "No charter content yet."
	}
	return b.String()
}

func (d *Dialogue) synthesize(ctx context.Context) error {
	summary := d.charterSummary()
	fmt.Fprintf(d.output, "\n")
	resp, err := d.routingStreamer.Stream(ctx, models.Mid, models.CompletionRequest{
		System:   SynthesizePrompt,
		Messages: []models.Message{{Role: "user", Content: summary}},
	}, d.output)
	fmt.Fprintf(d.output, "\n")
	if err != nil {
		return fmt.Errorf("synthesis: %w", err)
	}

	d.enhanceFromSynthesis(resp.Content)
	return nil
}

func (d *Dialogue) counterspec(ctx context.Context) error {
	summary := d.charterSummary()
	fmt.Fprintf(d.output, "\n")
	resp, err := d.routingStreamer.Stream(ctx, models.Frontier, models.CompletionRequest{
		System:   CounterSpecPrompt,
		Messages: []models.Message{{Role: "user", Content: summary}},
	}, d.output)
	fmt.Fprintf(d.output, "\n\n")
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
		fmt.Fprintf(d.output, "\n%s\n", styleWarn.Render(fmt.Sprintf("Misinterpretation: %s", m)))
		confirm := huh.NewConfirm().
			Title("Keep this in the charter?").
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

func (d *Dialogue) persistTranscript() {
	if d.chartersDir == "" {
		return
	}
	d.charter.Transcript = d.transcript
	d.charter.UpdatedAt = time.Now().UTC()
	if err := d.charter.Save(d.chartersDir); err != nil {
		slog.Warn("failed to persist transcript", "error", err)
	}
}