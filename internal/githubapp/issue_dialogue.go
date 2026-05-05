package githubapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/helloodokai/charter/internal/charter"
	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/dialogue"
	"github.com/helloodokai/charter/internal/routing"
	"github.com/helloodokai/charter/internal/sources"
	"github.com/helloodokai/charter/internal/storage"
)

// IssueDialogue orchestrates charter creation from a GitHub issue.
type IssueDialogue struct {
	client      *github.Client
	cfg         *config.Config
	router      *routing.Router
	repoRoot    string
	owner       string
	repo        string
	issueNumber int
	issueInfo   *sources.IssueInfo
}

// NewIssueDialogue creates an IssueDialogue for the given repository and issue.
func NewIssueDialogue(client *github.Client, cfg *config.Config, router *routing.Router, repoRoot, owner, repo string, issueNumber int) *IssueDialogue {
	return &IssueDialogue{
		client:      client,
		cfg:         cfg,
		router:      router,
		repoRoot:    repoRoot,
		owner:       owner,
		repo:        repo,
		issueNumber: issueNumber,
	}
}

// Run fetches the issue, runs the dialogue, and saves the resulting charter.
func (d *IssueDialogue) Run(ctx context.Context) (*charter.Charter, error) {
	src := sources.NewIssueSource(d.client)
	info, err := src.Fetch(ctx, d.owner, d.repo, d.issueNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching issue: %w", err)
	}
	d.issueInfo = info

	source := info.ToSource()
	c := charter.New(info.Title, source, "github-app")

	inputChan := make(chan string, 1)
	outputChan := make(chan string, 1)

	dlg := dialogue.New(c, d.router, d.cfg,
		dialogue.WithNonInteractive(false),
		dialogue.WithChannels(inputChan, outputChan),
	)

	go func() {
		for question := range outputChan {
			_, _, commentErr := d.client.Issues.CreateComment(ctx, d.owner, d.repo, d.issueNumber, &github.IssueComment{
				Body: github.String(question),
			})
			if commentErr != nil {
				slog.Error("posting question comment", "error", commentErr)
			}
		}
	}()

	result, err := dlg.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("running dialogue: %w", err)
	}

	chartersDir := storage.ChartersDir(d.repoRoot)
	if saveErr := result.Charter.Save(chartersDir); saveErr != nil {
		return nil, fmt.Errorf("saving charter: %w", saveErr)
	}
	if idxErr := storage.UpsertIndex(chartersDir, result.Charter); idxErr != nil {
		slog.Warn("updating index", "error", idxErr)
	}

	_, _, err = d.client.Issues.AddLabelsToIssue(ctx, d.owner, d.repo, d.issueNumber, []string{d.cfg.GitHub.HasLabel})
	if err != nil {
		slog.Warn("adding has-charter label", "error", err)
	}

	_, err = d.client.Issues.RemoveLabelForIssue(ctx, d.owner, d.repo, d.issueNumber, d.cfg.GitHub.NeedsLabel)
	if err != nil {
		slog.Warn("removing needs-charter label", "error", err)
	}

	slog.Info("charter created via GitHub App",
		"id", result.Charter.ID,
		"issue", d.issueNumber,
		"turns", result.TurnsUsed,
	)

	return result.Charter, nil
}

// PostDraftComment posts a summary comment of the drafted charter on the GitHub issue.
func PostDraftComment(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, c *charter.Charter) error {
	var sb strings.Builder
	sbPtr := &sb
	fmt.Fprintf(sbPtr, "## Charter: %s\n\n", c.ID)
	fmt.Fprintf(sbPtr, "**Goal:** %s\n\n", c.Goal)
	fmt.Fprintf(sbPtr, "**Status:** %s | **Risk:** %s\n\n", c.Status, c.Risk)

	if len(c.NonGoals) > 0 {
		sb.WriteString("**Non-goals:**\n")
		for _, ng := range c.NonGoals {
			fmt.Fprintf(sbPtr, "- %s\n", ng)
		}
		sb.WriteString("\n")
	}

	if len(c.AcceptanceCriteria) > 0 {
		sb.WriteString("**Acceptance Criteria:**\n")
		for _, ac := range c.AcceptanceCriteria {
			fmt.Fprintf(sbPtr, "- [ ] %s (%s)\n", ac.Statement, ac.Verification)
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(sbPtr, "Charter file: `.charters/%s.yaml`\n", c.ID)

	_, _, err := client.Issues.CreateComment(ctx, owner, repo, issueNumber, &github.IssueComment{
		Body: github.String(sb.String()),
	})
	return err
}