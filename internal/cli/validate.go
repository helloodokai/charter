package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
)

var validateCmd = &cobra.Command{
	Use:   "validate <charter.yaml>",
	Short: "Validate a charter for completeness and internal consistency",
	Long: `Validate an existing charter file. Checks for required fields,
internal consistency, and completeness. Reports field-level errors
with helpful messages.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	path := args[0]
	c, err := charter.Load(path)
	if err != nil {
		return fmt.Errorf("loading charter: %w", err)
	}

	errs := charter.Validate(c)
	if len(errs) == 0 {
		fmt.Fprintf(os.Stderr, "Charter %s is valid.\n", c.ID)
		fmt.Fprintf(os.Stderr, "  Goal: %s\n", c.Goal)
		fmt.Fprintf(os.Stderr, "  Status: %s\n", c.Status)
		fmt.Fprintf(os.Stderr, "  Risk: %s\n", c.Risk)
		fmt.Fprintf(os.Stderr, "  Acceptance criteria: %d\n", len(c.AcceptanceCriteria))
		fmt.Fprintf(os.Stderr, "  Non-goals: %d\n", len(c.NonGoals))
		return nil
	}

	fmt.Fprintf(os.Stderr, "Charter %s has %d validation error(s):\n\n", c.ID, len(errs))
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  %s\n", e.Error())
	}
	os.Exit(1)
	return nil
}