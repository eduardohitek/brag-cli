package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/eduardohitek/brag/internal/store"
)

type Client struct {
	baseURL  string
	email    string
	apiToken string
	http     *http.Client
}

func New(baseURL, email, apiToken string) *Client {
	return &Client{
		baseURL:  baseURL,
		email:    email,
		apiToken: apiToken,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string     `json:"key"`
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Summary     string    `json:"summary"`
	Updated     time.Time `json:"updated"`
	ResolutionDate *time.Time `json:"resolutiondate"`
	Self        string    `json:"self"`
}

func (c *Client) FetchRecentDoneTickets(ctx context.Context, days int) ([]*store.Entry, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	jql := url.QueryEscape(fmt.Sprintf(
		`assignee = currentUser() AND status = Done AND updated >= "%s" ORDER BY updated DESC`,
		since,
	))

	apiURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=100&fields=summary,updated,resolutiondate", c.baseURL, jql)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Jira request: %w", err)
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Jira API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Jira response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body))
	}

	var searchResp jiraSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing Jira response: %w", err)
	}

	var entries []*store.Entry
	for _, issue := range searchResp.Issues {
		date := issue.Fields.Updated
		if issue.Fields.ResolutionDate != nil {
			date = *issue.Fields.ResolutionDate
		}
		e := &store.Entry{
			ID:        issue.Key,
			Date:      date,
			Raw:       issue.Fields.Summary,
			Source:    "jira",
			SourceURL: fmt.Sprintf("%s/browse/%s", c.baseURL, issue.Key),
		}
		entries = append(entries, e)
	}
	return entries, nil
}
