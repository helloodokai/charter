package charter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	id := NewID()
	require.Contains(t, id, "ch-")
	require.Len(t, id, len("ch-2006-01-02-")+6)
}

func TestNew(t *testing.T) {
	src := Source{Type: "stdin", Raw: "test"}
	c := New("test goal", src, "tester")
	require.Equal(t, "1", c.SchemaVersion)
	require.Contains(t, c.ID, "ch-")
	require.Equal(t, "test goal", c.Goal)
	require.Equal(t, StatusDraft, c.Status)
	require.Equal(t, "tester", c.Authors[0])
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	src := Source{Type: "stdin", Raw: "test"}
	c := New("test goal", src, "tester")
	c.AcceptanceCriteria = []AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = RiskLow

	err := c.Save(tmp)
	require.NoError(t, err)

	loaded, err := Load(c.FilePath(tmp))
	require.NoError(t, err)
	require.Equal(t, c.ID, loaded.ID)
	require.Equal(t, "test goal", loaded.Goal)
	require.Equal(t, RiskLow, loaded.Risk)
	require.Len(t, loaded.AcceptanceCriteria, 1)
}

func TestValidateComplete(t *testing.T) {
	src := Source{Type: "stdin"}
	c := New("test goal", src, "tester")
	c.AcceptanceCriteria = []AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = RiskLow

	errs := Validate(c)
	require.Empty(t, errs)
}

func TestValidateIncomplete(t *testing.T) {
	c := &Charter{SchemaVersion: "1"}
	errs := Validate(c)
	require.NotEmpty(t, errs)

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}
	require.True(t, fields["goal"])
	require.True(t, fields["source.type"])
	require.True(t, fields["acceptance_criteria"])
}

func TestValidateHighRiskRequiresRollback(t *testing.T) {
	src := Source{Type: "stdin"}
	c := New("test goal", src, "tester")
	c.AcceptanceCriteria = []AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = RiskHigh

	errs := Validate(c)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Field == "rollback_plan" {
			found = true
		}
	}
	require.True(t, found)
}

func TestValidateBlockingUnknown(t *testing.T) {
	src := Source{Type: "stdin"}
	c := New("test goal", src, "tester")
	c.AcceptanceCriteria = []AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = RiskLow
	c.Status = StatusReady
	c.Unknowns = []Unknown{
		{ID: "unk-1", Question: "what about X?", Blocking: true},
	}

	errs := Validate(c)
	require.NotEmpty(t, errs)
}

func TestParseID(t *testing.T) {
	id, err := ParseID("ch-2026-05-04-abc123")
	require.NoError(t, err)
	require.Equal(t, "ch-2026-05-04-abc123", id)

	_, err = ParseID("invalid")
	require.Error(t, err)
}

func TestCharterFilePath(t *testing.T) {
	src := Source{Type: "stdin"}
	c := New("test goal", src, "tester")
	path := c.FilePath("/tmp/charters")
	require.Contains(t, path, "/tmp/charters/")
	require.Contains(t, path, c.ID)
}

func TestTimeFields(t *testing.T) {
	src := Source{Type: "stdin"}
	c := New("test goal", src, "tester")
	require.WithinDuration(t, time.Now(), c.CreatedAt, 2*time.Second)
	require.WithinDuration(t, time.Now(), c.UpdatedAt, 2*time.Second)
}