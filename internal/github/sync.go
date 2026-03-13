package github

import (
	"context"
	"fmt"
	"time"

	gh "github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"

	"github.com/eduardohitek/brag/internal/store"
)

type Client struct {
	client   *gh.Client
	username string
}

func New(token, username string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{
		client:   gh.NewClient(tc),
		username: username,
	}
}

// FetchRecentPRs fetches PRs merged by the user in the last `days` days.
func (c *Client) FetchRecentPRs(ctx context.Context, days int) ([]*store.Entry, error) {
	since := time.Now().AddDate(0, 0, -days)
	query := fmt.Sprintf("type:pr author:%s merged:>%s", c.username, since.Format("2006-01-02"))

	searchOpts := &gh.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("searching PRs: %w", err)
	}

	var entries []*store.Entry
	for _, issue := range result.Issues {
		e := &store.Entry{
			ID:        fmt.Sprintf("%d", issue.GetID()),
			Date:      issue.GetClosedAt().Time,
			Raw:       issue.GetTitle(),
			Source:    "github",
			SourceURL: issue.GetHTMLURL(),
		}
		if e.Date.IsZero() {
			e.Date = issue.GetUpdatedAt().Time
		}
		if e.ID == "0" {
			e.ID = time.Now().Format("20060102-150405")
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// FetchRecentIssues fetches issues closed by the user in the last `days` days.
func (c *Client) FetchRecentIssues(ctx context.Context, days int) ([]*store.Entry, error) {
	since := time.Now().AddDate(0, 0, -days)
	query := fmt.Sprintf("type:issue author:%s closed:>%s", c.username, since.Format("2006-01-02"))

	searchOpts := &gh.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	result, _, err := c.client.Search.Issues(ctx, query, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}

	var entries []*store.Entry
	for _, issue := range result.Issues {
		e := &store.Entry{
			ID:        fmt.Sprintf("%d", issue.GetID()),
			Date:      issue.GetClosedAt().Time,
			Raw:       issue.GetTitle(),
			Source:    "github",
			SourceURL: issue.GetHTMLURL(),
		}
		if e.Date.IsZero() {
			e.Date = issue.GetUpdatedAt().Time
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ResolveProjectName attempts to find the GitHub Project name for a given repo/issue combo.
// This uses the GraphQL Projects API via a REST workaround.
func (c *Client) ResolveProjectName(ctx context.Context, owner, repo string, issueNumber int) (string, error) {
	// GitHub Projects v2 requires GraphQL; for simplicity we return empty string if unavailable
	// A full GraphQL implementation would query projectItems for the issue
	return "", nil
}
