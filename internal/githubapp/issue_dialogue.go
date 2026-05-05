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
			_, _, err := d.client.Issues.CreateComment(ctx, d.owner, d.repo, d.issueNumber, &github.IssueComment{
				Body: github.String(question),
			})
			if err != nil {
				slog.Error("posting question comment", "error", err)
			}
		}
	}()

	result, err := dlg.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("running dialogue: %w", err)
	}

	chartersDir := storage.ChartersDir(d.repoRoot)
	if err := result.Charter.Save(chartersDir); err != nil {
		return nil, fmt.Errorf("saving charter: %w", err)
	}
	if err := storage.UpsertIndex(chartersDir, result.Charter); err != nil {
		slog.Warn("updating index", "error", err)
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

func PostDraftComment(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, c *charter.Charter) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Charter: %s\n\n", c.ID))
	sb.WriteString(fmt.Sprintf("**Goal:** %s\n\n", c.Goal))
	sb.WriteString(fmt.Sprintf("**Status:** %s | **Risk:** %s\n\n", c.Status, c.Risk))

	if len(c.NonGoals) > 0 {
		sb.WriteString("**Non-goals:**\n")
		for _, ng := range c.NonGoals {
			sb.WriteString(fmt.Sprintf("- %s\n", ng))
		}
		sb.WriteString("\n")
	}

	if len(c.AcceptanceCriteria) > 0 {
		sb.WriteString("**Acceptance Criteria:**\n")
		for _, ac := range c.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s (%s)\n", ac.Statement, ac.Verification))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Charter file: `.charters/%s.yaml`\n", c.ID))

	_, _, err := client.Issues.CreateComment(ctx, owner, repo, issueNumber, &github.IssueComment{
		Body: github.String(sb.String()),
	})
	return err
}