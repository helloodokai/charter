package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/google/go-github/v66/github"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/dialogue"
	"github.com/helloodokai/charter/internal/routing"
	"github.com/helloodokai/charter/internal/sources"
	"github.com/helloodokai/charter/internal/storage"
)

var draftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Draft a charter via interactive Socratic dialogue",
	Long: `Draft a charter from a GitHub issue, file, or stdin. CHARTER runs an
interactive Socratic dialogue to harden your intent into a machine-readable spec.

Examples:
  charter draft --issue https://github.com/org/repo/issues/42
  charter draft --from requirements.txt
  echo "Add login page" | charter draft --stdin
  charter draft --resume ch-2026-05-04-abc123`,
	RunE: runDraft,
}

var draftIssue string
var draftFile string
var draftStdin bool
var draftOut string
var draftNonInteractive bool
var draftTurnBudget int
var draftResume string
var draftNoTranscript bool

func init() {
	draftCmd.Flags().StringVar(&draftIssue, "issue", "", "GitHub issue URL")
	draftCmd.Flags().StringVar(&draftFile, "from", "", "read source from a file")
	draftCmd.Flags().BoolVar(&draftStdin, "stdin", false, "read source from stdin")
	draftCmd.Flags().StringVar(&draftOut, "out", "", "output path (defaults to .charters/<id>.yaml)")
	draftCmd.Flags().BoolVar(&draftNonInteractive, "non-interactive", false, "run without user interaction (for CI)")
	draftCmd.Flags().IntVar(&draftTurnBudget, "turn-budget", 0, "override default turn budget")
	draftCmd.Flags().StringVar(&draftResume, "resume", "", "resume an existing charter by ID")
	draftCmd.Flags().BoolVar(&draftNoTranscript, "no-transcript", false, "omit transcript from output")

	rootCmd.AddCommand(draftCmd)
}

func runDraft(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := GetConfig()
	profileName := GetProfileName(cmd)

	router, err := routing.NewRouter(cfg, profileName)
	if err != nil {
		return fmt.Errorf("initializing model router: %w", err)
	}

	var c *charter.Charter
	var source charter.Source

	if draftResume != "" {
		repoRoot, _ := cmd.Flags().GetString("repo-root")
		chartersDir := cfg.ChartersDir(repoRoot)
		var loadErr error
		c, loadErr = storage.LoadByID(chartersDir, draftResume)
		if loadErr != nil {
			return fmt.Errorf("loading charter %s: %w", draftResume, loadErr)
		}
		source = c.Source
	} else {
		source, err = resolveSource(cmd)
		if err != nil {
			return err
		}

		goal := "Draft charter from source"
		if source.Raw != "" {
			goal = extractGoal(source.Raw)
		}
		c = charter.New(goal, source, currentUser())
	}

	opts := []dialogue.Option{
		dialogue.WithNonInteractive(draftNonInteractive),
	}
	if draftTurnBudget > 0 {
		opts = append(opts, dialogue.WithBudget(draftTurnBudget))
	}

	dlg := dialogue.New(c, router, cfg, opts...)
	result, err := dlg.Run(ctx)
	if err != nil {
		return fmt.Errorf("dialogue failed: %w", err)
	}

	if draftNoTranscript {
		result.Charter.Transcript = nil
	}

	repoRoot, _ := cmd.Flags().GetString("repo-root")
	chartersDir := cfg.ChartersDir(repoRoot)

	if draftOut != "" {
		if saveErr := result.Charter.Save(filepath.Dir(draftOut)); saveErr != nil {
			return fmt.Errorf("saving charter: %w", saveErr)
		}
		fmt.Fprintln(os.Stderr, "Charter saved to:", draftOut)
	} else {
		if saveErr := result.Charter.Save(chartersDir); saveErr != nil {
			return fmt.Errorf("saving charter: %w", saveErr)
		}
		if idxErr := storage.UpsertIndex(chartersDir, result.Charter); idxErr != nil {
			fmt.Fprintln(os.Stderr, "Warning: failed to update index:", idxErr)
		}
		fmt.Fprintf(os.Stderr, "Charter saved: .charters/%s.yaml\n", result.Charter.ID)
	}

	fmt.Fprintf(os.Stderr, "  Goal: %s\n", result.Charter.Goal)
	fmt.Fprintf(os.Stderr, "  Status: %s\n", result.Charter.Status)
	fmt.Fprintf(os.Stderr, "  Risk: %s\n", result.Charter.Risk)
	fmt.Fprintf(os.Stderr, "  Turns used: %d\n", result.TurnsUsed)

	return nil
}

func resolveSource(cmd *cobra.Command) (charter.Source, error) {
	count := 0
	if draftIssue != "" {
		count++
	}
	if draftFile != "" {
		count++
	}
	if draftStdin {
		count++
	}
	if count > 1 {
		return charter.Source{}, fmt.Errorf("specify only one source: --issue, --from, or --stdin")
	}

	switch {
	case draftIssue != "":
		owner, repo, number, err := sources.ParseIssueURL(draftIssue)
		if err != nil {
			return charter.Source{}, err
		}
		client := newGitHubClient()
		src := sources.NewIssueSource(client)
		info, err := src.Fetch(context.Background(), owner, repo, number)
		if err != nil {
			return charter.Source{}, fmt.Errorf("fetching issue: %w", err)
		}
		return info.ToSource(), nil

	case draftFile != "":
		src := sources.NewFileSource()
		return src.Fetch(draftFile)

	case draftStdin:
		src := sources.NewStdinSource()
		return src.Fetch()

	default:
		if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
			src := sources.NewStdinSource()
			return src.Fetch()
		}
		return charter.Source{}, fmt.Errorf("specify a source: --issue URL, --from FILE, or --stdin")
	}
}

func extractGoal(raw string) string {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return truncate(line, 120)
		}
	}
	return "Draft charter"
}

func currentUser() string {
	u := os.Getenv("USER")
	if u == "" {
		u = os.Getenv("USERNAME")
	}
	if u == "" {
		u = "unknown"
	}
	return u
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func newGitHubClient() *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_ACCESS_TOKEN")
	}
	if token != "" {
		return github.NewTokenClient(context.Background(), token)
	}
	return github.NewClient(nil)
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}