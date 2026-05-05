package sources

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/helloodokai/charter/internal/charter"
)

// IssueSource fetches charter source material from a GitHub issue.
type IssueSource struct {
	client *github.Client
}

// NewIssueSource returns a new IssueSource using the given GitHub client.
func NewIssueSource(client *github.Client) *IssueSource {
	return &IssueSource{client: client}
}

// IssueInfo holds parsed data from a GitHub issue and its comments.
type IssueInfo struct {
	Owner    string
	Repo     string
	Number   int
	Title    string
	Body     string
	Comments []string
	URL      string
}

// ParseIssueURL extracts the owner, repo, and issue number from a GitHub issue URL.
func ParseIssueURL(url string) (owner, repo string, number int, err error) {
	parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
	if len(parts) < 4 {
		return "", "", 0, fmt.Errorf("invalid GitHub issue URL: %s", url)
	}
	owner = parts[0]
	repo = parts[1]
	if parts[2] != "issues" && parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("URL must be a GitHub issue: %s", url)
	}
	_, err = fmt.Sscanf(parts[3], "%d", &number)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue number in URL: %s", url)
	}
	return owner, repo, number, nil
}

// Fetch retrieves the GitHub issue and its comments, returning an IssueInfo.
func (s *IssueSource) Fetch(ctx context.Context, owner, repo string, number int) (*IssueInfo, error) {
	issue, _, err := s.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("fetching issue %s/%s#%d: %w", owner, repo, number, err)
	}

	comments := []string{}
	if s.client != nil {
		opts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}
		issueComments, _, err := s.client.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("fetching comments: %w", err)
		}
		for _, c := range issueComments {
			if c.Body != nil {
				comments = append(comments, *c.Body)
			}
		}
	}

	body := ""
	if issue.Body != nil {
		body = *issue.Body
	}
	title := ""
	if issue.Title != nil {
		title = *issue.Title
	}

	return &IssueInfo{
		Owner:    owner,
		Repo:     repo,
		Number:   number,
		Title:    title,
		Body:     body,
		Comments: comments,
		URL:      fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, number),
	}, nil
}

// ToSource converts the IssueInfo into a charter Source.
func (info *IssueInfo) ToSource() charter.Source {
	raw := info.Title + "\n\n" + info.Body
	if len(info.Comments) > 0 {
		raw += "\n\n--- Comments ---\n" + strings.Join(info.Comments, "\n---\n")
	}
	return charter.Source{
		Type: "github_issue",
		URL:  info.URL,
		Raw:  raw,
	}
}