package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/dialogue"
	"github.com/helloodokai/charter/internal/routing"
	"github.com/helloodokai/charter/internal/storage"
)

var specCmd = &cobra.Command{
	Use:   "spec [charter-id-or-path]",
	Short: "Generate a SPEC.md from a charter for an autonomous coding agent",
	Long: `Generate a complete, unambiguous software specification (SPEC.md) that
an autonomous coding agent can execute without additional human context.

If no charter ID or path is specified, the most recent charter is used.

The spec is saved alongside the charter as <id>.spec.md and referenced
from the charter YAML via the spec_file field.

Examples:
  charter spec
  charter spec ch-2026-05-04-abc123
  charter spec .charters/ch-2026-05-04-abc123.yaml
  charter spec --latest`,
	RunE: runSpec,
}

var specOut string
var specLatest bool

func init() {
	specCmd.Flags().StringVar(&specOut, "out", "", "write spec to a custom path instead of .charters/<id>.spec.md")
	specCmd.Flags().BoolVar(&specLatest, "latest", false, "use the most recent charter")
	rootCmd.AddCommand(specCmd)
}

func runSpec(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := GetConfig()
	profileName := GetProfileName(cmd)

	router, err := routing.NewRouter(cfg, profileName)
	if err != nil {
		return fmt.Errorf("initializing model router: %w", err)
	}

	repoRoot, _ := cmd.Flags().GetString("repo-root")
	chartersDir := cfg.ChartersDir(repoRoot)

	c, err := resolveCharter(args, chartersDir, specLatest)
	if err != nil {
		return err
	}

	var transcriptContent string
	if c.TranscriptFile != "" {
		data, readErr := os.ReadFile(charter.TranscriptFilePath(chartersDir, c.ID))
		if readErr != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not read transcript file:", readErr)
		} else {
			transcriptContent = string(data)
		}
	} else if len(c.Transcript) > 0 {
		transcriptContent = charter.FormatTranscript(c)
	}

	fmt.Fprintf(os.Stderr, "Generating SPEC.md ")

	spec, err := dialogue.GenerateSpec(ctx, c, transcriptContent, router, os.Stderr)
	if err != nil {
		return fmt.Errorf("generating spec: %w", err)
	}

	spec = strings.TrimSpace(spec)
	if spec == "" {
		return fmt.Errorf("spec generation returned empty content")
	}

	if specOut != "" {
		if err := os.WriteFile(specOut, []byte(spec), 0o644); err != nil { //nolint:gosec // user-specified output path
			return fmt.Errorf("writing spec: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nSpec written to: %s\n", specOut)
	} else {
		if err := charter.SaveSpec(chartersDir, c, spec); err != nil {
			return fmt.Errorf("saving spec: %w", err)
		}
		if saveErr := c.Save(chartersDir); saveErr != nil {
			return fmt.Errorf("saving charter: %w", saveErr)
		}
		fmt.Fprintf(os.Stderr, "\nSpec saved:  .charters/%s\n", c.SpecFile)
	}

	return nil
}

func resolveCharter(args []string, chartersDir string, latest bool) (*charter.Charter, error) {
	if len(args) > 0 {
		arg := args[0]

		if isFilePath(arg) {
			c, err := charter.Load(arg)
			if err != nil {
				return nil, fmt.Errorf("loading charter %s: %w", arg, err)
			}
			return c, nil
		}

		c, err := storage.LoadByID(chartersDir, arg)
		if err != nil {
			if strings.HasSuffix(arg, ".yaml") || strings.HasSuffix(arg, ".yml") {
				return nil, fmt.Errorf("loading charter: %w (did you mean to pass a file path? If so, use a path like .charters/%s)", err, arg)
			}
			return nil, fmt.Errorf("loading charter %s: %w", arg, err)
		}
		return c, nil
	}

	if !latest {
		entries, err := storage.ListByStatus(chartersDir, "")
		if err != nil {
			return nil, fmt.Errorf("listing charters: %w", err)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("no charters found in %s — run 'charter draft' first", chartersDir)
		}

		recent := entries[len(entries)-1]
		c, err := storage.LoadByID(chartersDir, recent.ID)
		if err != nil {
			return nil, fmt.Errorf("loading charter %s: %w", recent.ID, err)
		}
		fmt.Fprintf(os.Stderr, "Using charter: %s (%s)\n", recent.ID, recent.Goal)
		return c, nil
	}

	entries, err := storage.ListByStatus(chartersDir, "")
	if err != nil {
		return nil, fmt.Errorf("listing charters: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no charters found in %s — run 'charter draft' first", chartersDir)
	}

	recent := entries[len(entries)-1]
	c, err := storage.LoadByID(chartersDir, recent.ID)
	if err != nil {
		return nil, fmt.Errorf("loading charter %s: %w", recent.ID, err)
	}
	fmt.Fprintf(os.Stderr, "Using charter: %s (%s)\n", recent.ID, recent.Goal)
	return c, nil
}

func isFilePath(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
		return true
	}
	if strings.Contains(s, string(os.PathSeparator)) {
		return true
	}
	if strings.HasSuffix(strings.ToLower(s), ".yaml") || strings.HasSuffix(strings.ToLower(s), ".yml") {
		return true
	}
	return false
}