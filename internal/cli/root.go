package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/config"
)

var cfg *config.Config

// Version is set at build time via ldflags.
var Version = "dev"

// Commit is set at build time via ldflags.
var Commit = "none"

// Date is set at build time via ldflags.
var Date = "unknown"

var rootCmd = &cobra.Command{
	Use:   "charter",
	Short: "Turn fuzzy intent into hardened, machine-readable specifications",
	Long: `CHARTER turns fuzzy human intent — a half-written GitHub issue, a Slack
thread, a "hey can you…" message — into a hardened, versioned, machine-readable
specification that downstream coding agents consume as a contract.`,
	Version:             Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := cmd.Flags().GetString("repo-root")
		if repoRoot == "" {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot = dir
		}
		var err error
		cfg, err = config.FindAndLoad(repoRoot)
		return err
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("charter v{{.Version}} (commit: %s, built: %s)\n", Commit, Date))
	rootCmd.PersistentFlags().String("repo-root", "", "repository root directory (defaults to cwd)")
	rootCmd.PersistentFlags().String("profile", "", "model profile: cloud, local (defaults to config)")
}

// Execute runs the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetConfig returns the loaded configuration.
func GetConfig() *config.Config {
	return cfg
}

// GetProfileName returns the model profile name from the CLI flag or the config default.
func GetProfileName(cmd *cobra.Command) string {
	profile, _ := cmd.Flags().GetString("profile")
	if profile != "" {
		return profile
	}
	if cfg != nil {
		return cfg.Models.DefaultProfile
	}
	return "cloud"
}