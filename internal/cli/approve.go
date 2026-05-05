package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/storage"
)

var approveCmd = &cobra.Command{
	Use:   "approve <charter.yaml>",
	Short: "Approve a charter, making it immutable",
	Long: `Approve a charter, transitioning it from ready to approved status.
Approved charters are immutable — they represent a contract that
should not be modified.`,
	Args: cobra.ExactArgs(1),
	RunE: runApprove,
}

func init() {
	rootCmd.AddCommand(approveCmd)
}

func runApprove(cmd *cobra.Command, args []string) error {
	path := args[0]
	c, err := charter.Load(path)
	if err != nil {
		return fmt.Errorf("loading charter: %w", err)
	}

	if c.Status == charter.StatusApproved {
		fmt.Fprintf(os.Stderr, "Charter %s is already approved.\n", c.ID)
		return nil
	}

	errs := charter.Validate(c)
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Cannot approve charter with validation errors:\n")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  %s\n", e.Error())
		}
		os.Exit(2)
		return nil
	}

	for _, u := range c.Unknowns {
		if u.Blocking {
			fmt.Fprintf(os.Stderr, "Cannot approve: blocking unknown remains: %s\n", u.Question)
			os.Exit(2)
			return nil
		}
	}

	c.Status = charter.StatusApproved
	c.UpdatedAt = time.Now().UTC()

	cfg := GetConfig()
	repoRoot, _ := cmd.Flags().GetString("repo-root")
	chartersDir := cfg.ChartersDir(repoRoot)

	if err := c.Save(chartersDir); err != nil {
		return fmt.Errorf("saving charter: %w", err)
	}
	if err := storage.UpsertIndex(chartersDir, c); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to update index:", err)
	}

	fmt.Fprintf(os.Stderr, "Charter %s approved.\n", c.ID)
	return nil
}