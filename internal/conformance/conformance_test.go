package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/charter"
)

func TestBlastRadiusGrader(t *testing.T) {
	ch := &charter.Charter{
		BlastRadius: charter.BlastRadius{
			Files: []string{"src/auth/**"},
		},
	}

	diff := `+++ b/src/auth/login.go
--- a/src/auth/handler.go
+++ b/src/payments/billing.go
`
	findings := (&BlastRadiusGrader{}).Grade(ch, diff)
	require.NotEmpty(t, findings)

	found := false
	for _, f := range findings {
		if f.File == "src/payments/billing.go" {
			found = true
		}
	}
	require.True(t, found, "should flag file outside blast radius")
}

func TestBlastRadiusAllWithin(t *testing.T) {
	ch := &charter.Charter{
		BlastRadius: charter.BlastRadius{
			Files: []string{"src/auth/**", "src/payments/**"},
		},
	}

	diff := `+++ b/src/auth/login.go
+++ b/src/payments/billing.go
`
	findings := (&BlastRadiusGrader{}).Grade(ch, diff)
	require.Empty(t, findings, "all files within blast radius should pass")
}

func TestUnknownGatingGrader(t *testing.T) {
	ch := &charter.Charter{
		Unknowns: []charter.Unknown{
			{ID: "unk-1", Question: "what API version?", Blocking: true},
		},
	}

	findings := (&UnknownGatingGrader{}).Grade(ch, "")
	require.NotEmpty(t, findings)
	require.Equal(t, SeverityBlocking, findings[0].Severity)
}

func TestUnknownGatingNoBlocking(t *testing.T) {
	ch := &charter.Charter{
		Unknowns: []charter.Unknown{
			{ID: "unk-1", Question: "what API version?", Blocking: false},
		},
	}

	findings := (&UnknownGatingGrader{}).Grade(ch, "")
	require.Empty(t, findings)
}

func TestGradePassesClean(t *testing.T) {
	ch := &charter.Charter{
		ID:   "ch-2026-05-04-test",
		Goal: "test goal",
		BlastRadius: charter.BlastRadius{
			Files: []string{"src/**"},
		},
	}

	diff := "+++ b/src/feature.go\n"
	verdict := Grade(ch, diff)
	require.Equal(t, "pass", verdict.Status)
	require.Equal(t, 1.0, verdict.Score)
}

func TestGradeFailsBlocking(t *testing.T) {
	ch := &charter.Charter{
		ID:   "ch-2026-05-04-test",
		Goal: "test goal",
		Unknowns: []charter.Unknown{
			{ID: "unk-1", Question: "blocking unknown", Blocking: true},
		},
	}

	verdict := Grade(ch, "")
	require.Equal(t, "fail", verdict.Status)
	require.Equal(t, 0.0, verdict.Score)
}