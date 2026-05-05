package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a repo for Charter",
	Long: `Create a .charter.toml config file with sensible defaults and a
.charters/ directory for storing charter files.

Run this once per repo to get started with Charter.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	repoRoot, _ := cmd.Flags().GetString("repo-root")
	if repoRoot == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		repoRoot = dir
	}

	cfgPath := filepath.Join(repoRoot, ".charter.toml")
	chartersDir := filepath.Join(repoRoot, ".charters")

	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Fprintf(os.Stderr, ".charter.toml already exists — edit it to customize.\n")
		return nil
	}

	cfg := config.Default()
	data, err := config.MarshalTOML(cfg)
	if err != nil {
		return fmt.Errorf("generating config: %w", err)
	}

	if writeErr := os.WriteFile(cfgPath, data, 0o644); writeErr != nil {
		return fmt.Errorf("writing .charter.toml: %w", writeErr)
	}
	fmt.Fprintf(os.Stderr, "  Created .charter.toml\n")

	if mkdirErr := os.MkdirAll(chartersDir, 0o750); mkdirErr != nil {
		return fmt.Errorf("creating .charters/: %w", mkdirErr)
	}
	fmt.Fprintf(os.Stderr, "  Created .charters/\n")

	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	f, openErr := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // expected perms for gitignore
	if openErr == nil {
		if _, err := f.WriteString("\n# Charter\n.charters/*.yaml\n!.charters/index.yaml\n"); err != nil {
			_ = f.Close()
			return fmt.Errorf("writing .gitignore: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing .gitignore: %w", err)
		}
		fmt.Fprintf(os.Stderr, "  Updated .gitignore\n")
	}

	fmt.Fprintf(os.Stderr, "\nCharter initialized! Next steps:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  1. Set your API key:\n")
	fmt.Fprintf(os.Stderr, "       export OLLAMA_API_KEY=your-key-here\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  2. Draft your first charter:\n")
	fmt.Fprintf(os.Stderr, "       charter draft 'Add rate limiting to the public API'\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  3. Edit .charter.toml to customize models and paths.\n")

	return nil
}