package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/conformance"
)

var conformanceCmd = &cobra.Command{
	Use:   "conformance <charter.yaml>",
	Short: "Grade a diff against a charter",
	Long: `Grade a git diff against a charter to check conformance.
Produces structured findings that acig can consume.

Examples:
  charter conformance .charters/ch-2026-05-04-abc123.yaml --diff HEAD..feature
  charter conformance .charters/ch-2026-05-04-abc123.yaml --diff main..HEAD
  git diff main..HEAD | charter conformance .charters/ch-2026-05-04-abc123.yaml --diff -`,
	Args: cobra.ExactArgs(1),
	RunE: runConformance,
}

var confDiff string
var confFormat string
var confOut string

func init() {
	conformanceCmd.Flags().StringVar(&confDiff, "diff", "", "git ref range (e.g. HEAD..feature) or '-' for stdin")
	conformanceCmd.Flags().StringVar(&confFormat, "format", "md", "output format: json, md, both")
	conformanceCmd.Flags().StringVar(&confOut, "out", "", "output path (defaults to stdout)")
	rootCmd.AddCommand(conformanceCmd)
}

func runConformance(cmd *cobra.Command, args []string) error {
	path := args[0]
	c, err := charter.Load(path)
	if err != nil {
		return fmt.Errorf("loading charter: %w", err)
	}

	var diffContent string

	switch {
	case confDiff == "-":
		stdinData, stdinErr := io.ReadAll(os.Stdin)
		if stdinErr != nil {
			return fmt.Errorf("reading diff from stdin: %w", stdinErr)
		}
		diffContent = string(stdinData)

	case confDiff != "":
		diffContent, err = getDiff(confDiff)
		if err != nil {
			return fmt.Errorf("getting diff: %w", err)
		}

	default:
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			pipedData, pipedErr := io.ReadAll(os.Stdin)
			if pipedErr != nil {
				return fmt.Errorf("reading diff from stdin: %w", pipedErr)
			}
			diffContent = string(pipedData)
		} else {
			return fmt.Errorf("--diff is required (e.g. --diff HEAD..feature, or pipe diff via stdin)")
		}
	}

	verdict := conformance.Grade(c, diffContent)

	switch confFormat {
	case "json":
		return outputJSON(verdict, confOut)
	case "md", "markdown":
		return outputMarkdown(verdict, confOut)
	case "both":
		_ = outputJSON(verdict, confOut+".json")
		return outputMarkdown(verdict, confOut+".md")
	default:
		return outputMarkdown(verdict, confOut)
	}
}

func getDiff(refRange string) (string, error) {
	cmd := exec.Command("git", "diff", refRange)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running git diff %s: %w", refRange, err)
	}
	return string(out), nil
}

func outputJSON(v *conformance.Verdict, path string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if path != "" {
		return os.WriteFile(path, data, 0o644)
	}
	fmt.Println(string(data))
	return nil
}

func outputMarkdown(v *conformance.Verdict, path string) error {
	var output string
	output += fmt.Sprintf("# Conformance Report: %s\n\n", v.CharterID)
	output += fmt.Sprintf("**Goal:** %s\n\n", v.Goal)
	output += fmt.Sprintf("**Status:** %s | **Score:** %.1f/1.0\n\n", v.Status, v.Score)

	if len(v.Findings) == 0 {
		output += "No conformance issues found.\n"
	} else {
		output += "## Findings\n\n"
		for _, f := range v.Findings {
			output += fmt.Sprintf("- **[%s]** %s\n", f.Severity, f.Message)
			if f.Detail != "" {
				output += fmt.Sprintf("  - %s\n", f.Detail)
			}
		}
	}

	if path != "" {
		return os.WriteFile(path, []byte(output), 0o644)
	}
	fmt.Println(output)
	return nil
}