package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/storage"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List charters",
	Long:  `List charters in the repo, optionally filtered by status.`,
	RunE:  runLs,
}

var lsStatus string

func init() {
	lsCmd.Flags().StringVar(&lsStatus, "status", "", "filter by status: draft, ready, approved, archived")
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	repoRoot, _ := cmd.Flags().GetString("repo-root")
	chartersDir := cfg.ChartersDir(repoRoot)

	var status charter.Status
	switch lsStatus {
	case "draft":
		status = charter.StatusDraft
	case "ready":
		status = charter.StatusReady
	case "approved":
		status = charter.StatusApproved
	case "archived":
		status = charter.StatusArchived
	default:
		status = ""
	}

	entries, err := storage.ListByStatus(chartersDir, status)
	if err != nil {
		return fmt.Errorf("listing charters: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No charters found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tGOAL\tSTATUS\tRISK\tCREATED")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.ID,
			truncateGoal(e.Goal),
			e.Status,
			e.Risk,
			e.CreatedAt.Format("2006-01-02"),
		)
	}
	w.Flush()
	return nil
}

func truncateGoal(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}