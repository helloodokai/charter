package dialogue

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
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

func TestParseNonStructuredList(t *testing.T) {
	items := parseNonStructuredList("- First item with enough length\n- Second item with some length\n- Third item with adequate length")
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

func TestPlanGaps(t *testing.T) {
	c := charter.New("", charter.Source{Type: "stdin"}, "tester")
	d := &Dialogue{charter: c, cfg: defaultTestConfig()}
	gaps := d.planGaps()
	require.NotEmpty(t, gaps)
	require.Equal(t, GapGoal, gaps[0].Type)
}

func TestPlanGapsWithGoal(t *testing.T) {
	c := charter.New("Add login page", charter.Source{Type: "stdin"}, "tester")
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "works", Verification: "test"},
	}
	d := &Dialogue{charter: c, cfg: defaultTestConfig()}
	gaps := d.planGaps()
	require.Equal(t, GapContext, gaps[0].Type)
}

func defaultTestConfig() *config.Config {
	return config.Default()
}