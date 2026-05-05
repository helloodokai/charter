package charter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateValidCharter(t *testing.T) {
	c, err := Load("../../testdata/fixtures/valid_charter.yaml")
	require.NoError(t, err)

	errs := Validate(c)
	require.Empty(t, errs, "valid charter should have no validation errors: %v", errs)
}

func TestValidateIncompleteCharter(t *testing.T) {
	c, err := Load("../../testdata/fixtures/incomplete_charter.yaml")
	require.NoError(t, err)

	errs := Validate(c)
	require.NotEmpty(t, errs)

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}
	require.True(t, fields["goal"])
	require.True(t, fields["acceptance_criteria"])
}

func TestValidateAmbiguousCharter(t *testing.T) {
	c, err := Load("../../testdata/fixtures/ambiguous_charter.yaml")
	require.NoError(t, err)

	require.Equal(t, "Improve performance", c.Goal)
	require.Len(t, c.NonGoals, 0)
	require.Len(t, c.CounterSpec.Misinterpretations, 0)
	require.Empty(t, c.BlastRadius.Files)
}