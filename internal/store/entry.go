package store

import "time"

type OKRRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Entry struct {
	ID            string    `json:"id"`
	Date          time.Time `json:"date"`
	Raw           string    `json:"raw"`
	Enriched      string    `json:"enriched"`
	Tags          []string  `json:"tags"`
	Source        string    `json:"source"` // manual | github | jira
	SourceURL     string    `json:"source_url,omitempty"`
	GithubProject string    `json:"github_project,omitempty"`
	OKR           *OKRRef   `json:"okr,omitempty"`
	ImpactScore   int       `json:"impact_score"`
	FileSHA       string    `json:"-"`
}
