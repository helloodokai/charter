package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/routing"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check backend reachability and configuration",
	Long: `Check that your charter configuration is valid and all configured
backends are reachable. Verifies Ollama Cloud, Ollama Local, Anthropic,
and OpenAI API connectivity based on your profile settings.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	profileName := GetProfileName(cmd)

	fmt.Fprintf(os.Stderr, "charter doctor\n")
	fmt.Fprintf(os.Stderr, "================\n\n")
	fmt.Fprintf(os.Stderr, "Profile: %s\n", profileName)
	fmt.Fprintf(os.Stderr, "Charters dir: %s\n", cfg.Storage.ChartersDir)
	fmt.Fprintf(os.Stderr, "Turn budget: %d\n\n", cfg.Dialogue.TurnBudget)

	router, err := routing.NewRouter(cfg, profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing router: %v\n", err)
		os.Exit(1)
	}

	results := router.CheckReachable(cmd.Context())

	allOk := true
	for prov, err := range results {
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [FAIL] %s: %v\n", prov, err)
			allOk = false
		} else {
			fmt.Fprintf(os.Stderr, "  [ OK ] %s\n", prov)
		}
	}

	if !allOk {
		fmt.Fprintf(os.Stderr, "\nSome backends are unreachable. Check your API keys and network.\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\nAll backends reachable.\n")

	githubAppID := os.Getenv(cfg.GitHub.AppIDEnv)
	if githubAppID != "" {
		fmt.Fprintf(os.Stderr, "  GitHub App ID: %s (configured)\n", githubAppID)
	} else {
		fmt.Fprintf(os.Stderr, "  GitHub App ID: not set (optional, needed for App mode)\n")
	}

	return nil
}