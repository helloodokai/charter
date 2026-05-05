package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/config"
)

var cfg *config.Config

var Version = "dev"
var Commit = "none"
var Date = "unknown"

var rootCmd = &cobra.Command{
	Use:   "charter",
	Short: "Turn fuzzy intent into hardened, machine-readable specifications",
	Long: `CHARTER turns fuzzy human intent — a half-written GitHub issue, a Slack
thread, a "hey can you…" message — into a hardened, versioned, machine-readable
specification that downstream coding agents consume as a contract.`,
	Version: Version,
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
	rootCmd.PersistentFlags().String("repo-root", "", "repository root directory (defaults to cwd)")
	rootCmd.PersistentFlags().String("profile", "", "model profile: cloud, local (defaults to config)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func GetConfig() *config.Config {
	return cfg
}

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