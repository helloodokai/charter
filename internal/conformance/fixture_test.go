package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/charter"
)

func TestGradeAgainstValidCharter(t *testing.T) {
	c, err := charter.Load("../../testdata/fixtures/valid_charter.yaml")
	require.NoError(t, err)

	diff := `+++ b/src/auth/login.go
--- a/src/auth/handler.go
`
	verdict := Grade(c, diff)
	require.Equal(t, "pass", verdict.Status)
}

func TestGradeViolationOfNonGoals(t *testing.T) {
	c, err := charter.Load("../../testdata/fixtures/valid_charter.yaml")
	require.NoError(t, err)

	diff := `+++ b/src/auth/password_reset.go
+++ b/src/api/payments/billing.go
+++ b/src/auth/auth_provider.go
`
	verdict := Grade(c, diff)
	require.Equal(t, "fail", verdict.Status)
}

func TestCounterSpecOnAmbiguousCharter(t *testing.T) {
	c, err := charter.Load("../../testdata/fixtures/ambiguous_charter.yaml")
	require.NoError(t, err)

	require.Empty(t, c.CounterSpec.Misinterpretations)
	require.Equal(t, "Improve performance", c.Goal)
}