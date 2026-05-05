package storage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/charter"
)

func TestLoadIndexEmpty(t *testing.T) {
	tmp := t.TempDir()
	idx, err := LoadIndex(tmp)
	require.NoError(t, err)
	require.Empty(t, idx.Charters)
}

func TestSaveAndLoadIndex(t *testing.T) {
	tmp := t.TempDir()
	idx := &Index{
		Charters: []IndexEntry{
			{
				ID:     "ch-2026-05-04-abc123",
				Goal:   "test goal",
				Status: charter.StatusDraft,
				Risk:   charter.RiskLow,
			},
		},
	}

	err := SaveIndex(tmp, idx)
	require.NoError(t, err)

	loaded, err := LoadIndex(tmp)
	require.NoError(t, err)
	require.Len(t, loaded.Charters, 1)
	require.Equal(t, "ch-2026-05-04-abc123", loaded.Charters[0].ID)
}

func TestUpsertIndex(t *testing.T) {
	tmp := t.TempDir()
	src := charter.Source{Type: "stdin", Raw: "test"}
	c := charter.New("test goal", src, "tester")
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = charter.RiskLow

	err := UpsertIndex(tmp, c)
	require.NoError(t, err)

	entries, err := ListByStatus(tmp, charter.StatusDraft)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, c.ID, entries[0].ID)
}

func TestSaveAndLoadByID(t *testing.T) {
	tmp := t.TempDir()
	src := charter.Source{Type: "stdin", Raw: "test"}
	c := charter.New("test goal", src, "tester")
	c.AcceptanceCriteria = []charter.AcceptanceCriterion{
		{ID: "ac-1", Statement: "it works", Verification: "test"},
	}
	c.Risk = charter.RiskLow

	err := Save(tmp, c)
	require.NoError(t, err)

	loaded, err := LoadByID(tmp, c.ID)
	require.NoError(t, err)
	require.Equal(t, c.ID, loaded.ID)
	require.Equal(t, "test goal", loaded.Goal)
}