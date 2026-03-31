package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/eduardohitek/brag/internal/store"
)

const apiURL = "https://api.linear.app/graphql"

type Client struct {
	apiKey string
	teamID string
	http   *http.Client
}

func New(apiKey, teamID string) *Client {
	return &Client{
		apiKey: apiKey,
		teamID: teamID,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type linearIssue struct {
	ID          string    `json:"id"`
	Identifier  string    `json:"identifier"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	CompletedAt time.Time `json:"completedAt"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphqlResponse struct {
	Data struct {
		Issues struct {
			Nodes    []linearIssue `json:"nodes"`
			PageInfo pageInfo      `json:"pageInfo"`
		} `json:"issues"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

var queryWithoutTeam = `
query CompletedIssues($after: DateTime, $cursor: String) {
  issues(
    first: 100
    after: $cursor
    filter: {
      assignee: { isMe: { eq: true } }
      state: { type: { eq: "completed" } }
      completedAt: { gt: $after }
    }
  ) {
    nodes { id identifier title url completedAt }
    pageInfo { hasNextPage endCursor }
  }
}`

var queryWithTeam = `
query CompletedIssues($after: DateTime, $cursor: String, $teamId: ID!) {
  issues(
    first: 100
    after: $cursor
    filter: {
      assignee: { isMe: { eq: true } }
      state: { type: { eq: "completed" } }
      completedAt: { gt: $after }
      team: { id: { eq: $teamId } }
    }
  ) {
    nodes { id identifier title url completedAt }
    pageInfo { hasNextPage endCursor }
  }
}`

func (c *Client) FetchRecentCompletedIssues(ctx context.Context, days int) ([]*store.Entry, error) {
	after := time.Now().AddDate(0, 0, -days).UTC().Format(time.RFC3339)

	query := queryWithoutTeam
	if c.teamID != "" {
		query = queryWithTeam
	}

	var entries []*store.Entry
	var cursor *string

	for {
		vars := map[string]any{
			"after": after,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}
		if c.teamID != "" {
			vars["teamId"] = c.teamID
		}

		reqBody := graphqlRequest{
			Query:     query,
			Variables: vars,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling Linear request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("creating Linear request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling Linear API: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading Linear response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Linear API returned %d: %s", resp.StatusCode, string(respBody))
		}

		var gqlResp graphqlResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("parsing Linear response: %w", err)
		}

		if len(gqlResp.Errors) > 0 {
			return nil, fmt.Errorf("Linear API error: %s", gqlResp.Errors[0].Message)
		}

		for _, issue := range gqlResp.Data.Issues.Nodes {
			entries = append(entries, &store.Entry{
				ID:        issue.Identifier,
				Date:      issue.CompletedAt,
				Raw:       issue.Title,
				Source:    "linear",
				SourceURL: issue.URL,
			})
		}

		if !gqlResp.Data.Issues.PageInfo.HasNextPage {
			break
		}
		end := gqlResp.Data.Issues.PageInfo.EndCursor
		cursor = &end
	}

	return entries, nil
}
