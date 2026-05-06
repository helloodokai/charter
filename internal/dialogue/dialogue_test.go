package dialogue

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
)

func TestParseGoalContext(t *testing.T) {
	goal, ctx, err := parseGoalContext("GOAL: Add a login page\nCONTEXT: The app currently has no authentication.")
	require.NoError(t, err)
	require.Equal(t, "Add a login page", goal)
	require.Equal(t, "The app currently has no authentication.", ctx)
}

func TestParseGoalContextFallback(t *testing.T) {
	goal, ctx, _ := parseGoalContext("Add a login page\nThis is context.")
	require.Equal(t, "Add a login page", goal)
	require.Equal(t, "This is context.", ctx)
}

func TestSplitLines(t *testing.T) {
	items := splitLines("- First item with enough length\n- Second item with some length\n- Third item with adequate length")
	require.Len(t, items, 3)
	require.Contains(t, items[0], "First item")
}

func TestParseBlastRadius(t *testing.T) {
	br := parseBlastRadius("FILES:\n- src/auth/**\n- src/api/**\nSERVICES:\n- auth-service\nDATA:\n- users_table")
	require.Equal(t, []string{"src/auth/**", "src/api/**"}, br.Files)
	require.Equal(t, []string{"auth-service"}, br.Services)
	require.Equal(t, []string{"users_table"}, br.Data)
}

func TestParseConstraints(t *testing.T) {
	c := parseConstraints("PERFORMANCE: p99 < 100ms\nSECURITY: must use OAuth2\nCOMPATIBILITY: keep v1 API\nSTYLE: follow existing patterns\nDEPENDENCIES: Go 1.22+")
	require.Equal(t, []string{"p99 < 100ms"}, c.Performance)
	require.Equal(t, []string{"must use OAuth2"}, c.Security)
	require.Equal(t, []string{"keep v1 API"}, c.Compatibility)
	require.Equal(t, []string{"follow existing patterns"}, c.Style)
	require.Equal(t, []string{"Go 1.22+"}, c.Dependencies)
}

func TestParseCounterSpec(t *testing.T) {
	content := `MISINTERPRETATION 1: I would add a logout button instead of login
WHAT I'D BUILD: logout handler
WHY IT'S WRONG: user asked for login

MISINTERPRETATION 2: I would skip error handling
WHAT I'D BUILD: happy path only
WHY IT'S WRONG: production needs error handling

AMBIGUITIES FLAGGED:
- What does "login" mean exactly?
- Which auth provider?`
	cs := parseCounterSpec(content)
	require.Len(t, cs.Misinterpretations, 2)
	require.Contains(t, cs.Misinterpretations[0], "logout")
	require.Len(t, cs.AmbiguitiesFlagged, 2)
}

// --- Field tracking tests ---

func TestIsFieldFilledEmptyCharter(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := &Dialogue{charter: c, cfg: defaultTestConfig()}

	require.False(t, d.isFieldFilled(fieldGoal))
	require.False(t, d.isFieldFilled(fieldContext))
	require.False(t, d.isFieldFilled(fieldNonGoals))
	require.False(t, d.isFieldFilled(fieldAcceptance))
	require.False(t, d.isFieldFilled(fieldEdgeCases))
	require.False(t, d.isFieldFilled(fieldBlastRadius))
	require.False(t, d.isFieldFilled(fieldConstraints))
	require.False(t, d.isFieldFilled(fieldUnknowns))
	require.False(t, d.isFieldFilled(fieldRisk))
}

func TestIsFieldFilledPopulatedCharter(t *testing.T) {
	c := charter.New("Add login", charter.Source{Type: "stdin"}, "tester")
	c.Context = "Some context"
	c.NonGoals = []string{"no admin panel"}
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "works", Verification: "test"},
	}
	c.EdgeCases = []string{"timeout"}
	c.BlastRadius = charter.BlastRadius{
		Files:    []string{"src/auth/**"},
		Services: []string{"auth-service"},
	}
	c.Constraints = charter.Constraints{
		Performance:  []string{"p99 < 100ms"},
		Security:     []string{"OAuth2"},
		Compatibility: []string{"v1 API"},
		Style:        []string{"existing patterns"},
		Dependencies: []string{"Go 1.22"},
	}
	c.Unknowns = []charter.Unknown{
		{ID: "unk-1", Question: "what about X?", Blocking: true},
	}
	c.Risk = charter.RiskLow

	d := &Dialogue{charter: c, cfg: defaultTestConfig()}

	require.True(t, d.isFieldFilled(fieldGoal))
	require.True(t, d.isFieldFilled(fieldContext))
	require.True(t, d.isFieldFilled(fieldNonGoals))
	require.True(t, d.isFieldFilled(fieldAcceptance))
	require.True(t, d.isFieldFilled(fieldEdgeCases))
	require.True(t, d.isFieldFilled(fieldBlastRadius))
	require.True(t, d.isFieldFilled(fieldConstraints))
	require.True(t, d.isFieldFilled(fieldUnknowns))
	require.True(t, d.isFieldFilled(fieldRisk))
}

func TestMissingFields(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := &Dialogue{charter: c, cfg: defaultTestConfig(), output: io.Discard}

	missing := d.missingFields()
	require.Contains(t, missing, "Goal")
	require.Contains(t, missing, "Context")
	require.Contains(t, missing, "Non-Goals")
	require.Contains(t, missing, "Acceptance Criteria")
	require.Contains(t, missing, "Edge Cases")
	require.Contains(t, missing, "Blast Radius")
	require.Contains(t, missing, "Constraints")
	require.Contains(t, missing, "Unknowns")
	require.Contains(t, missing, "Risk Assessment")
}

func TestMissingFieldsWithPopulatedCharter(t *testing.T) {
	c := charter.New("Add login", charter.Source{Type: "stdin"}, "tester")
	c.Context = "Some context"
	c.NonGoals = []string{"no admin panel"}
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "works", Verification: "test"},
	}
	c.EdgeCases = []string{"timeout"}
	c.BlastRadius = charter.BlastRadius{
		Files:    []string{"src/auth/**"},
		Services: []string{"auth-service"},
	}
	c.Constraints = charter.Constraints{
		Performance:  []string{"p99 < 100ms"},
		Security:     []string{"OAuth2"},
		Compatibility: []string{"v1 API"},
		Style:        []string{"existing patterns"},
		Dependencies: []string{"Go 1.22"},
	}
	c.Unknowns = []charter.Unknown{
		{ID: "unk-1", Question: "what about X?", Blocking: true},
	}
	c.Risk = charter.RiskLow
	c.RollbackPlan = "revert the deployment"

	d := &Dialogue{charter: c, cfg: defaultTestConfig(), output: io.Discard}
	missing := d.missingFields()
	require.Empty(t, missing)
}

func TestFilledFields(t *testing.T) {
	c := charter.New("Add login", charter.Source{Type: "stdin"}, "tester")
	d := &Dialogue{charter: c, cfg: defaultTestConfig(), output: io.Discard}

	filled := d.filledFields()
	require.Contains(t, filled, "Goal")
	require.NotContains(t, filled, "Context")
}

func TestNextFieldToDiscuss(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := &Dialogue{charter: c, cfg: defaultTestConfig()}

	field := d.nextFieldToDiscuss()
	require.Equal(t, fieldGoal, field)

	c.Goal = "Add login"
	field = d.nextFieldToDiscuss()
	require.Equal(t, fieldContext, field)
}

func TestNextFieldToDiscussAllFilled(t *testing.T) {
	c := charter.New("Add login", charter.Source{Type: "stdin"}, "tester")
	c.Context = "Some context"
	c.NonGoals = []string{"no admin"}
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "works", Verification: "test"},
	}
	c.EdgeCases = []string{"timeout"}
	c.BlastRadius = charter.BlastRadius{Files: []string{"src/**"}}
	c.Constraints = charter.Constraints{Performance: []string{"p99 < 100ms"}}
	c.Unknowns = []charter.Unknown{{ID: "unk-1", Question: "what?", Blocking: true}}
	c.Risk = charter.RiskLow
	c.RollbackPlan = "just revert"

	d := &Dialogue{charter: c, cfg: defaultTestConfig()}
	field := d.nextFieldToDiscuss()
	require.Equal(t, fieldName(""), field, "all fields filled, should return empty string")
}

// --- Option tests ---

func TestWithResumeOption(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := New(c, nil, defaultTestConfig(), WithResume(true))
	require.True(t, d.resumeMode)
}

func TestWithResumeFalseOption(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := New(c, nil, defaultTestConfig(), WithResume(false))
	require.False(t, d.resumeMode)
}

func TestWithChartersDirOption(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := New(c, nil, defaultTestConfig(), WithChartersDir("/tmp/charters-test"))
	require.Equal(t, "/tmp/charters-test", d.chartersDir)
}

func TestWithOutputOption(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	buf := &bytes.Buffer{}
	d := New(c, nil, defaultTestConfig(), WithOutput(buf))
	require.Equal(t, buf, d.output)
}

// --- Transcript persistence tests ---

func TestPersistTranscriptNoOpWhenDirEmpty(t *testing.T) {
	c := charter.New("test goal", charter.Source{Type: "stdin"}, "tester")
	c.Transcript = []charter.TranscriptTurn{
		{Role: "human", Content: "hello"},
	}

	d := &Dialogue{
		charter:     c,
		transcript:  c.Transcript,
		cfg:         defaultTestConfig(),
		chartersDir: "",
		output:      io.Discard,
	}

	d.persistTranscript()
}

func TestPersistTranscriptSavesToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	c := charter.New("test goal", charter.Source{Type: "stdin"}, "tester")
	c.Transcript = []charter.TranscriptTurn{
		{Role: "human", Content: "hello"},
		{Role: "tool", Content: "response"},
	}

	d := &Dialogue{
		charter:     c,
		transcript:  c.Transcript,
		cfg:         defaultTestConfig(),
		chartersDir: tmpDir,
		output:      io.Discard,
	}

	d.persistTranscript()

	loaded, err := charter.Load(c.FilePath(tmpDir))
	require.NoError(t, err)
	require.Equal(t, "test goal", loaded.Goal)
	require.Len(t, loaded.Transcript, 2)
	require.Equal(t, "human", loaded.Transcript[0].Role)
	require.Equal(t, "hello", loaded.Transcript[0].Content)
	require.Equal(t, "tool", loaded.Transcript[1].Role)
	require.Equal(t, "response", loaded.Transcript[1].Content)
}

func TestPersistTranscriptUpdatesOnDiskEachCall(t *testing.T) {
	tmpDir := t.TempDir()
	c := charter.New("test goal", charter.Source{Type: "stdin"}, "tester")

	d := &Dialogue{
		charter:     c,
		transcript:  nil,
		cfg:         defaultTestConfig(),
		chartersDir: tmpDir,
		output:      io.Discard,
	}

	d.transcript = append(d.transcript, charter.TranscriptTurn{Role: "human", Content: "first"})
	d.persistTranscript()

	loaded, err := charter.Load(c.FilePath(tmpDir))
	require.NoError(t, err)
	require.Len(t, loaded.Transcript, 1)

	d.transcript = append(d.transcript, charter.TranscriptTurn{Role: "tool", Content: "second"})
	d.persistTranscript()

	loaded, err = charter.Load(c.FilePath(tmpDir))
	require.NoError(t, err)
	require.Len(t, loaded.Transcript, 2)
}

// --- streamComplete tests ---

type mockRouterClient struct {
	resp *models.CompletionResponse
	err  error
}

func (m *mockRouterClient) Stream(ctx context.Context, tier models.Tier, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil && m.resp.Content != "" {
		_, _ = w.Write([]byte(m.resp.Content))
	}
	return m.resp, nil
}

func (m *mockRouterClient) Complete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (*models.CompletionResponse, error) {
	return m.resp, m.err
}

type mockStreamingRouter struct {
	client *mockRouterClient
}

func (m *mockStreamingRouter) Stream(ctx context.Context, tier models.Tier, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
	return m.client.Stream(ctx, tier, req, w)
}

func (m *mockStreamingRouter) Complete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (*models.CompletionResponse, error) {
	return m.client.Complete(ctx, tier, req)
}

var _ routerStreamer = (*mockStreamingRouter)(nil)

func TestStreamCompleteCallsRouterAndReturnsContent(t *testing.T) {
	mockClient := &mockRouterClient{
		resp: &models.CompletionResponse{
			Content: "Hello from stream",
			Model:   "test-model",
			Usage:   models.Usage{InputTokens: 10, OutputTokens: 5},
		},
	}

	router := &mockStreamingRouter{client: mockClient}
	buf := &bytes.Buffer{}

	d := &Dialogue{
		charter: charter.New("test", charter.Source{Type: "stdin"}, "tester"),
		cfg:     defaultTestConfig(),
		output:  buf,
		routingStreamer: router,
	}

	content, err := d.streamComplete(context.Background(), models.Mid, models.CompletionRequest{
		Messages: []models.Message{{Role: "user", Content: "hello"}},
	})

	require.NoError(t, err)
	require.Equal(t, "Hello from stream", content)
}

func TestStreamCompleteReturnsErrorOnRouterFailure(t *testing.T) {
	mockClient := &mockRouterClient{
		err: context.DeadlineExceeded,
	}

	router := &mockStreamingRouter{client: mockClient}
	buf := &bytes.Buffer{}

	d := &Dialogue{
		charter: charter.New("test", charter.Source{Type: "stdin"}, "tester"),
		cfg:     defaultTestConfig(),
		output:  buf,
		routingStreamer: router,
	}

	content, err := d.streamComplete(context.Background(), models.Mid, models.CompletionRequest{
		Messages: []models.Message{{Role: "user", Content: "hello"}},
	})

	require.Error(t, err)
	require.Equal(t, "", content)
}

func defaultTestConfig() *config.Config {
	return config.Default()
}