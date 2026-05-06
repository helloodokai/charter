package dialogue

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
	"github.com/helloodokai/charter/internal/routing"
)

// KickoffPrompt is the embedded system prompt for the initial charter kickoff turn.
//go:embed prompts/kickoff.md
var KickoffPrompt string

// AcknowledgePrompt is the embedded system prompt for the field acknowledgment step.
//go:embed prompts/acknowledge.md
var AcknowledgePrompt string

// AskNonGoalsPrompt is the embedded system prompt for eliciting non-goals from the user.
//go:embed prompts/ask_non_goals.md
var AskNonGoalsPrompt string

// AskEdgeCasesPrompt is the embedded system prompt for identifying edge cases.
//go:embed prompts/ask_edge_cases.md
var AskEdgeCasesPrompt string

// AskBlastRadiusPrompt is the embedded system prompt for analyzing blast radius.
//go:embed prompts/ask_blast_radius.md
var AskBlastRadiusPrompt string

// AskConstraintsPrompt is the embedded system prompt for inferring project constraints.
//go:embed prompts/ask_constraints.md
var AskConstraintsPrompt string

// SynthesizePrompt is the embedded system prompt for the synthesis pass.
//go:embed prompts/synthesize.md
var SynthesizePrompt string

// CounterSpecPrompt is the embedded system prompt for the counter-speculative analysis.
//go:embed prompts/counterspec.md
var CounterSpecPrompt string

// ConversationPrompt is the embedded system prompt for the conversation-driven dialogue mode.
//go:embed prompts/conversation.md
var ConversationPrompt string

// ExtractPrompt is the embedded system prompt for extracting structured data from user responses.
//go:embed prompts/extract.md
var ExtractPrompt string

type routerStreamer interface {
	Stream(ctx context.Context, tier models.Tier, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error)
	Complete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (*models.CompletionResponse, error)
}

type fieldName string

const (
	fieldGoal        fieldName = "goal"
	fieldContext     fieldName = "context"
	fieldNonGoals    fieldName = "non_goals"
	fieldAcceptance  fieldName = "acceptance_criteria"
	fieldEdgeCases   fieldName = "edge_cases"
	fieldBlastRadius fieldName = "blast_radius"
	fieldConstraints fieldName = "constraints"
	fieldUnknowns    fieldName = "unknowns"
	fieldRisk        fieldName = "risk"
	fieldRollback    fieldName = "rollback"
)

var fieldOrder = []fieldName{
	fieldGoal,
	fieldContext,
	fieldNonGoals,
	fieldAcceptance,
	fieldEdgeCases,
	fieldBlastRadius,
	fieldConstraints,
	fieldUnknowns,
	fieldRisk,
	fieldRollback,
}

var fieldLabels = map[fieldName]string{
	fieldGoal:        "Goal",
	fieldContext:     "Context",
	fieldNonGoals:    "Non-Goals",
	fieldAcceptance:  "Acceptance Criteria",
	fieldEdgeCases:   "Edge Cases",
	fieldBlastRadius: "Blast Radius",
	fieldConstraints: "Constraints",
	fieldUnknowns:    "Unknowns",
	fieldRisk:        "Risk Assessment",
	fieldRollback:    "Rollback Plan",
}

var fieldDescriptions = map[fieldName]string{
	fieldGoal:        "the one-sentence objective of this charter",
	fieldContext:     "background information an outsider would need",
	fieldNonGoals:    "what this charter explicitly does NOT cover",
	fieldAcceptance:  "testable criteria that prove the goal is met",
	fieldEdgeCases:   "boundary conditions and failure scenarios",
	fieldBlastRadius: "files, services, and data stores this change touches",
	fieldConstraints:  "performance, security, compatibility, and style requirements",
	fieldUnknowns:    "open questions that could block progress",
	fieldRisk:        "how risky this change is (low/medium/high/critical) and why",
	fieldRollback:    "how to safely revert if something goes wrong",
}

// Dialogue drives an interactive conversation that fills out a Charter through guided turns.
type Dialogue struct {
	charter         *charter.Charter
	transcript      []charter.TranscriptTurn
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
	version         string
	conversation    []chatTurn
	skippedFields   map[fieldName]bool
}

type chatTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Result holds the outcome of a completed dialogue session.
type Result struct {
	Charter   *charter.Charter
	TurnsUsed int
}

var (
	styleHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleThink   = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("8"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleAccent  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleWarn    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleField   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	styleDone    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// New creates a Dialogue ready to fill the given charter.
func New(c *charter.Charter, router *routing.Router, cfg *config.Config, opts ...Option) *Dialogue {
	d := &Dialogue{
		charter:         c,
		transcript:      c.Transcript,
		budget:           cfg.Dialogue.TurnBudget,
		router:           router,
		routingStreamer:  router,
		cfg:              cfg,
		output:           os.Stderr,
		skippedFields:    make(map[fieldName]bool),
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

// WithNonInteractive disables interactive prompts.
func WithNonInteractive(v bool) Option {
	return func(d *Dialogue) { d.nonInteractive = v }
}

// WithChannels sets input/output channels for programmatic dialogue.
func WithChannels(input chan string, output chan string) Option {
	return func(d *Dialogue) {
		d.inputChan = input
		d.outputChan = output
	}
}

// WithOutput sets the writer for dialogue output.
func WithOutput(w io.Writer) Option {
	return func(d *Dialogue) { d.output = w }
}

// WithChartersDir sets the directory for persisting charters between turns.
func WithChartersDir(dir string) Option {
	return func(d *Dialogue) { d.chartersDir = dir }
}

// WithResume enables resume mode, skipping fields already filled.
func WithResume(v bool) Option {
	return func(d *Dialogue) { d.resumeMode = v }
}

// WithVersion sets the charter version string for display in the header.
func WithVersion(v string) Option {
	return func(d *Dialogue) { d.version = v }
}

// Run executes the dialogue session as a conversation driven by the LLM.
func (d *Dialogue) Run(ctx context.Context) (*Result, error) {
	d.transcript = d.charter.Transcript
	for _, t := range d.transcript {
		d.conversation = append(d.conversation, chatTurn{Role: t.Role, Content: t.Content})
	}

	missing := d.missingFields()
	filled := d.filledFields()

	header := " Charter Dialogue "
	if d.version != "" {
		header = fmt.Sprintf(" Charter v%s ", d.version)
	}
	fmt.Fprintf(d.output, "\n%s\n", styleHeader.Render(header))
	if len(filled) > 0 {
		fmt.Fprintf(d.output, "%s\n", styleDone.Render(fmt.Sprintf("  Already set: %s", strings.Join(filled, ", "))))
	}
	if len(missing) > 0 {
		fmt.Fprintf(d.output, "%s\n", styleDim.Render(fmt.Sprintf("  Need: %s", strings.Join(missing, ", "))))
	}
	fmt.Fprintf(d.output, "%s\n\n", styleDim.Render(fmt.Sprintf("  Turns: 0/%d", d.budget)))

	if d.nonInteractive {
		return d.runNonInteractive(ctx)
	}

	return d.runConversation(ctx)
}

	func (d *Dialogue) runConversation(ctx context.Context) (*Result, error) {
	for d.turn < d.budget {
		field := d.nextFieldToDiscuss()
		if field == "" {
			break
		}

		fmt.Fprintf(d.output, "\n%s\n", styleField.Render(fmt.Sprintf("▸ %s", fieldLabels[field])))
		fmt.Fprintf(d.output, "%s\n", styleDim.Render(fmt.Sprintf("  Discussing %s — %s", fieldLabels[field], fieldDescriptions[field])))

		question, err := d.generateFieldQuestion(ctx, field)
		if err != nil {
			return nil, fmt.Errorf("turn %d: generating question for %s: %w", d.turn, field, err)
		}

		answer, err := d.askUser(ctx, question)
		if err != nil {
			return nil, fmt.Errorf("turn %d: asking user: %w", d.turn, err)
		}

		if strings.TrimSpace(answer) == "" {
			d.skippedFields[field] = true
			fmt.Fprintf(d.output, "%s\n", styleDim.Render(fmt.Sprintf("  Skipped %s", fieldLabels[field])))
			d.turn++
			continue
		}

		if strings.TrimSpace(strings.ToLower(answer)) == "done" {
			break
		}

		if err := d.extractField(field, answer); err != nil {
			slog.Warn("extraction had issues, keeping raw content", "field", field, "error", err)
		}

		if err := d.acknowledge(ctx, field, answer); err != nil {
			slog.Warn("acknowledge step failed", "field", field, "error", err)
		}

		d.turn++
		d.persistTranscript()

		filled := d.filledFields()
		missing := d.missingFields()
		fmt.Fprintf(d.output, "%s\n", styleDivider.Render("──────────────────────────────────────"))
		if len(filled) > 0 {
			fmt.Fprintf(d.output, "%s  ", styleDone.Render("✓"))
		}
		fmt.Fprintf(d.output, "%s", strings.Join(filled, ", "))
		if len(missing) > 0 {
			fmt.Fprintf(d.output, "  %s%s", styleDim.Render("⋯ "), strings.Join(missing, ", "))
		}
		fmt.Fprintf(d.output, "\n%s\n", styleDim.Render(fmt.Sprintf("  Turn %d/%d", d.turn, d.budget)))
	}

	return d.finalize(ctx)
}

func (d *Dialogue) runNonInteractive(ctx context.Context) (*Result, error) {
	for _, field := range fieldOrder {
		if d.isFieldFilled(field) {
			continue
		}
		question, err := d.generateFieldQuestion(ctx, field)
		if err != nil {
			return nil, fmt.Errorf("generating question for %s: %w", field, err)
		}
		d.transcript = append(d.transcript, charter.TranscriptTurn{
			Role: "tool", At: time.Now().UTC(), Content: question,
		})
		d.turn++
	}
	return d.finalize(ctx)
}

func (d *Dialogue) finalize(ctx context.Context) (*Result, error) {
	fmt.Fprintf(d.output, "\n%s ", styleThink.Render("Synthesizing charter"))
	if err := d.synthesize(ctx); err != nil {
		fmt.Fprintf(d.output, "\n%s\n", styleWarn.Render("⚠ Synthesis had issues, charter may be incomplete"))
		slog.Warn("synthesis failed", "error", err)
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

	fmt.Fprintf(d.output, "\n%s\n", styleHeader.Render(" ✓ Charter Complete "))
	fmt.Fprintf(d.output, "  Goal: %s\n", d.charter.Goal)
	fmt.Fprintf(d.output, "  Status: %s\n", d.charter.Status)
	fmt.Fprintf(d.output, "  Risk: %s\n", d.charter.Risk)
	fmt.Fprintf(d.output, "  Turns used: %d\n\n", d.turn)

	return &Result{
		Charter:   d.charter,
		TurnsUsed: d.turn,
	}, nil
}

func (d *Dialogue) nextFieldToDiscuss() fieldName {
	for _, field := range fieldOrder {
		if d.skippedFields[field] {
			continue
		}
		if !d.isFieldFilled(field) {
			return field
		}
	}
	return ""
}

func (d *Dialogue) isFieldFilled(field fieldName) bool {
	switch field {
	case fieldGoal:
		return d.charter.Goal != ""
	case fieldContext:
		return d.charter.Context != ""
	case fieldNonGoals:
		return len(d.charter.NonGoals) > 0
	case fieldAcceptance:
		return len(d.charter.AcceptanceCriteria) > 0
	case fieldEdgeCases:
		return len(d.charter.EdgeCases) > 0
	case fieldBlastRadius:
		return len(d.charter.BlastRadius.Files) > 0 || len(d.charter.BlastRadius.Services) > 0 || len(d.charter.BlastRadius.Data) > 0
	case fieldConstraints:
		return len(d.charter.Constraints.Performance) > 0 || len(d.charter.Constraints.Security) > 0 || len(d.charter.Constraints.Compatibility) > 0 || len(d.charter.Constraints.Style) > 0 || len(d.charter.Constraints.Dependencies) > 0
	case fieldUnknowns:
		return len(d.charter.Unknowns) > 0
	case fieldRisk:
		return d.charter.Risk != ""
	case fieldRollback:
		shouldAskRollback := d.charter.Risk == charter.RiskHigh || d.charter.Risk == charter.RiskCritical ||
			d.cfg.Dialogue.AskForRollback == "high" || d.cfg.Dialogue.AskForRollback == "medium" ||
			d.cfg.Dialogue.AskForRollback == "all"
		if !shouldAskRollback {
			return true
		}
		return d.charter.RollbackPlan != ""
	default:
		return true
	}
}

func (d *Dialogue) missingFields() []string {
	var missing []string
	for _, field := range fieldOrder {
		if !d.isFieldFilled(field) {
			missing = append(missing, fieldLabels[field])
		}
	}
	return missing
}

func (d *Dialogue) filledFields() []string {
	var filled []string
	for _, field := range fieldOrder {
		if d.isFieldFilled(field) {
			filled = append(filled, fieldLabels[field])
		}
	}
	return filled
}

func (d *Dialogue) generateFieldQuestion(ctx context.Context, field fieldName) (string, error) {
	switch field {
	case fieldGoal:
		if d.charter.Goal != "" {
			return fmt.Sprintf("**Goal:** %s\n\nIs this correct, or would you like to refine it?", d.charter.Goal), nil
		}
		return "**Goal:** What specific outcome are you trying to achieve? Describe it in one sentence starting with a verb.", nil

	case fieldContext:
		if d.charter.Context != "" {
			return fmt.Sprintf("**Context:** %s\n\nAnything to add or change?", d.charter.Context), nil
		}
		return "**Context:** What background would a new team member need to understand why this work matters?", nil

	case fieldNonGoals:
		return "**Non-Goals:** What is this work explicitly NOT going to do? What should an agent avoid building?", nil

	case fieldAcceptance:
		return "**Acceptance Criteria:** How will you know this is done? List specific, testable conditions.", nil

	case fieldEdgeCases:
		return "**Edge Cases:** What could break this? List boundary conditions, failures, and unusual scenarios.", nil

	case fieldBlastRadius:
		return "**Blast Radius:** What files, services, or data stores will this change touch?", nil

	case fieldConstraints:
		return "**Constraints:** Are there performance, security, style, or compatibility limits an agent must respect?", nil

	case fieldUnknowns:
		return "**Unknowns:** Any open questions that could block this work? What are you not sure about?", nil

	case fieldRisk:
		return "**Risk:** How risky is this change? (low / medium / high / critical)", nil

	case fieldRollback:
		return "**Rollback Plan:** If something goes wrong, how do you safely revert?", nil
	}

	return "", fmt.Errorf("unknown field: %s", field)
}

func (d *Dialogue) askUser(ctx context.Context, question string) (string, error) {
	d.transcript = append(d.transcript, charter.TranscriptTurn{
		Role: "tool", At: time.Now().UTC(), Content: question,
	})
	d.conversation = append(d.conversation, chatTurn{Role: "assistant", Content: question})

	if d.inputChan != nil && d.outputChan != nil {
		d.outputChan <- question
		select {
		case answer := <-d.inputChan:
			d.transcript = append(d.transcript, charter.TranscriptTurn{
				Role: "human", At: time.Now().UTC(), Content: answer,
			})
			d.conversation = append(d.conversation, chatTurn{Role: "user", Content: answer})
			return answer, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	fmt.Fprintf(d.output, "\n%s\n", styleDim.Render("  Press Enter to skip, type \"done\" to finish"))
	fmt.Fprintf(d.output, "%s ", styleAccent.Render("▸"))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading user input: %w", err)
	}
	answer := strings.TrimSpace(line)

	d.transcript = append(d.transcript, charter.TranscriptTurn{
		Role: "human", At: time.Now().UTC(), Content: answer,
	})
	d.conversation = append(d.conversation, chatTurn{Role: "user", Content: answer})

	return answer, nil
}

func (d *Dialogue) extractField(field fieldName, answer string) error {
	if answer == "" {
		return nil
	}

	switch field {
	case fieldGoal, fieldContext:
		return d.extractGoalAndContext(answer)
	case fieldNonGoals:
		return d.extractNonGoals(answer)
	case fieldAcceptance:
		return d.extractAcceptanceCriteria(answer)
	case fieldEdgeCases:
		return d.extractEdgeCases(answer)
	case fieldBlastRadius:
		return d.extractBlastRadius(answer)
	case fieldConstraints:
		return d.extractConstraints(answer)
	case fieldUnknowns:
		return d.extractUnknowns(answer)
	case fieldRisk:
		return d.extractRisk(answer)
	case fieldRollback:
		d.charter.RollbackPlan = answer
		return nil
	default:
		return nil
	}
}

func (d *Dialogue) acknowledge(ctx context.Context, field fieldName, answer string) error {
	summary := d.charterSummary()
	userMsg := fmt.Sprintf("Field: %s\nUser's answer: %s\n\nCurrent charter state:\n%s", fieldLabels[field], answer, summary)

	fmt.Fprintf(d.output, "%s ", styleThink.Render("  Reviewing"))

	resp, err := d.routingStreamer.Complete(ctx, models.Cheap, models.CompletionRequest{
		System:   AcknowledgePrompt,
		Messages: []models.Message{{Role: "user", Content: userMsg}},
	})
	if err != nil {
		return fmt.Errorf("acknowledge LLM call: %w", err)
	}

	content := strings.TrimSpace(resp.Content)

	if strings.Contains(content, "Got it, moving on") {
		fmt.Fprintf(d.output, "%s\n", styleDone.Render("✓"))
		return nil
	}

	fmt.Fprintf(d.output, "\n%s\n", styleAccent.Render(content))
	return nil
}

func (d *Dialogue) streamComplete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (string, error) {
	resp, err := d.routingStreamer.Stream(ctx, tier, req, io.Discard)
	if err != nil {
		return "", err
	}

	// Print a brief loading indicator
	fmt.Fprintf(d.output, "  ")

	cleaned := d.guardResponse(resp.Content)
	fmt.Fprintf(d.output, "%s", cleaned)
	fmt.Fprintf(d.output, "\n\n")

	// Show truncation warning if content was stripped
	if len(cleaned) < len(resp.Content)-10 {
		fmt.Fprintf(d.output, "%s\n", styleWarn.Render("  ⚠ Response contained implementation content and was cleaned"))
	}

	return cleaned, nil
}

func (d *Dialogue) guardResponse(content string) string {
	lower := strings.ToLower(content)
	forbiddenPatterns := []string{
		"```",                    // code blocks
		"here's how",          // tutorials
		"for example",         // examples
		"step 1",              // step-by-step
		"# step",              // steps
		"## ",                 // markdown headings (tutorials use these)
	}
	for _, p := range forbiddenPatterns {
		if strings.Contains(lower, p) {
			slog.Warn("LLM generated implementation content", "pattern", p, "truncating", true)
			// Truncate at the start of the implementation content
			idx := strings.Index(lower, p)
			if idx > 0 {
				return strings.TrimSpace(content[:idx])
			}
		}
	}

	// If response is very long, it's likely a tutorial
	if len(content) > 1200 {
		// Try to find the last question mark before the cutoff
		lastQ := strings.LastIndex(content[:1200], "?")
		if lastQ > 0 {
			return strings.TrimSpace(content[:lastQ+1])
		}
		return content[:1200] + "\n[response truncated — LLM generated too much content]"
	}

	return content
}

func (d *Dialogue) sourceSummary() string { //nolint:unused
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
		fmt.Fprintf(d.output, "\n%s\n", styleWarn.Render(fmt.Sprintf("Misinterpretation: %s", m)))
		fmt.Fprintf(d.output, "%s ", styleAccent.Render("Keep this in the charter? (y/n):"))
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "y" || line == "yes" {
			kept = append(kept, m)
		}
	}

	d.charter.CounterSpec.Misinterpretations = kept
	d.charter.CounterSpec.AmbiguitiesFlagged = cs.AmbiguitiesFlagged
	return nil
}

func (d *Dialogue) enhanceFromSynthesis(synthesis string) {
}

func (d *Dialogue) askAcceptanceCriteria(ctx context.Context, charterSummary string) (string, error) { //nolint:unused
	prompt := `Given the charter summary below, propose 3-5 acceptance criteria. Each should be:
1. Testable — you can verify it passed or failed
2. Specific — no vague words like "works" or "is good"
3. Minimal — the smallest set that proves the goal is met

Format each as:
- <statement> (verification: test|manual|metric)

Charter:
` + charterSummary

	resp, err := d.routingStreamer.Stream(ctx, models.Mid, models.CompletionRequest{
		System:   "You are a test engineer writing acceptance criteria for a coding agent.",
		Messages: []models.Message{{Role: "user", Content: prompt}},
	}, d.output)
	if err != nil {
		return "", fmt.Errorf("acceptance criteria LLM call: %w", err)
	}

	fmt.Fprintf(d.output, "\n")
	return resp.Content + "\n\nDo you want to add, modify, or remove any of these criteria?", nil
}

func (d *Dialogue) askRisk(ctx context.Context, charterSummary string) (string, error) { //nolint:unused
	prompt := `Based on the charter summary, assess the risk level. Consider:
- How many files/services are touched?
- Are any critical paths affected?
- How reversible is the change?
- What's the blast radius?

Rate: low, medium, high, or critical.

Charter:
` + charterSummary

	resp, err := d.routingStreamer.Stream(ctx, models.Cheap, models.CompletionRequest{
		System:   "You are a risk assessor for software changes. Be conservative.",
		Messages: []models.Message{{Role: "user", Content: prompt}},
	}, d.output)
	if err != nil {
		return "", fmt.Errorf("risk assessment LLM call: %w", err)
	}

	fmt.Fprintf(d.output, "\n")
	return resp.Content + "\n\nDo you agree with this risk assessment? If not, what should it be and why?", nil
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