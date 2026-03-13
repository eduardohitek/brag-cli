package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Store struct {
	client *github.Client
	owner  string
	repo   string
}

func New(token, ownerRepo string) (*Store, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/repo", ownerRepo)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Store{
		client: github.NewClient(tc),
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

func entryPath(e *Entry) string {
	ts := e.Date.UTC().Format("2006-01-02-150405")
	return fmt.Sprintf("entries/%s-%s.json", ts, e.Source)
}

func (s *Store) SaveEntry(ctx context.Context, e *Entry) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}
	path := entryPath(e)
	encoded := base64.StdEncoding.EncodeToString(data)
	msg := fmt.Sprintf("add entry %s", e.ID)
	opts := &github.RepositoryContentFileOptions{
		Message: &msg,
		Content: []byte(encoded),
	}
	_, _, err = s.client.Repositories.CreateFile(ctx, s.owner, s.repo, path, opts)
	if err != nil {
		return fmt.Errorf("creating file in repo: %w", err)
	}
	return nil
}

func (s *Store) UpdateEntry(ctx context.Context, e *Entry) error {
	if e.FileSHA == "" {
		return fmt.Errorf("entry %s has no FileSHA; load via ListEntries before updating", e.ID)
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}
	path := entryPath(e)
	encoded := base64.StdEncoding.EncodeToString(data)
	msg := fmt.Sprintf("enrich entry %s", e.ID)
	opts := &github.RepositoryContentFileOptions{
		Message: &msg,
		Content: []byte(encoded),
		SHA:     &e.FileSHA,
	}
	_, _, err = s.client.Repositories.UpdateFile(ctx, s.owner, s.repo, path, opts)
	if err != nil {
		return fmt.Errorf("updating file in repo: %w", err)
	}
	return nil
}

type ListOptions struct {
	Since   *time.Time
	From    *time.Time
	To      *time.Time
	Tag     string
	Source  string
	OKR     string
	Project string
	NoOKR   bool
}

func (s *Store) ListEntries(ctx context.Context, opts ListOptions) ([]*Entry, error) {
	_, dirContents, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, "entries", nil)
	if err != nil {
		return nil, fmt.Errorf("listing entries directory: %w", err)
	}

	var entries []*Entry
	for _, fc := range dirContents {
		if fc.GetType() != "file" || !strings.HasSuffix(fc.GetName(), ".json") {
			continue
		}
		fileContent, _, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, fc.GetPath(), nil)
		if err != nil {
			continue
		}
		rawContent, _ := fileContent.GetContent()
		decoded, err := base64.StdEncoding.DecodeString(rawContent)
		if err != nil {
			// GitHub sometimes returns content with newlines; strip them
			cleaned := strings.ReplaceAll(rawContent, "\n", "")
			decoded, err = base64.StdEncoding.DecodeString(cleaned)
			if err != nil {
				continue
			}
		}
		var e Entry
		if err := json.Unmarshal(decoded, &e); err != nil {
			continue
		}
		e.FileSHA = fileContent.GetSHA()

		// Apply filters
		if opts.Since != nil && e.Date.Before(*opts.Since) {
			continue
		}
		if opts.From != nil && e.Date.Before(*opts.From) {
			continue
		}
		if opts.To != nil && e.Date.After(*opts.To) {
			continue
		}
		if opts.Source != "" && e.Source != opts.Source {
			continue
		}
		if opts.OKR != "" && (e.OKR == nil || e.OKR.ID != opts.OKR) {
			continue
		}
		if opts.NoOKR && e.OKR != nil {
			continue
		}
		if opts.Project != "" && e.GithubProject != opts.Project {
			continue
		}
		if opts.Tag != "" && !containsTag(e.Tags, opts.Tag) {
			continue
		}

		entries = append(entries, &e)
	}
	return entries, nil
}

func (s *Store) SaveReport(ctx context.Context, content, filename string) error {
	path := "reports/" + filename
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	msg := fmt.Sprintf("add report %s", filename)
	opts := &github.RepositoryContentFileOptions{
		Message: &msg,
		Content: []byte(encoded),
	}
	_, _, err := s.client.Repositories.CreateFile(ctx, s.owner, s.repo, path, opts)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	return nil
}

func (s *Store) ListReports(ctx context.Context) ([]string, error) {
	_, dirContents, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, "reports", nil)
	if err != nil {
		return nil, fmt.Errorf("listing reports directory: %w", err)
	}
	var names []string
	for _, fc := range dirContents {
		if fc.GetType() == "file" {
			names = append(names, fc.GetName())
		}
	}
	return names, nil
}

func (s *Store) GetReport(ctx context.Context, filename string) (string, error) {
	path := "reports/" + filename
	fileContent, _, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("getting report: %w", err)
	}
	rawContent, _ := fileContent.GetContent()
	cleaned := strings.ReplaceAll(rawContent, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("decoding report: %w", err)
	}
	return string(decoded), nil
}

func (s *Store) EntryExists(ctx context.Context, sourceURL string) (bool, error) {
	entries, err := s.ListEntries(ctx, ListOptions{})
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.SourceURL == sourceURL {
			return true, nil
		}
	}
	return false, nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
