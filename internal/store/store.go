package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Store struct {
	client   *github.Client
	owner    string
	repo     string
	cacheDir string
}

func New(token, ownerRepo, cacheDir string) (*Store, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format %q, expected owner/repo", ownerRepo)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Store{
		client:   github.NewClient(tc),
		owner:    parts[0],
		repo:     parts[1],
		cacheDir: cacheDir,
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
	resp, _, err := s.client.Repositories.CreateFile(ctx, s.owner, s.repo, path, opts)
	if err != nil {
		return fmt.Errorf("creating file in repo: %w", err)
	}
	if s.cacheDir != "" && resp.Content != nil {
		_ = s.writeEntryToCache(e, path, resp.Content.GetSHA())
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
	resp, _, err := s.client.Repositories.UpdateFile(ctx, s.owner, s.repo, path, opts)
	if err != nil {
		return fmt.Errorf("updating file in repo: %w", err)
	}
	if s.cacheDir != "" && resp.Content != nil {
		_ = s.writeEntryToCache(e, path, resp.Content.GetSHA())
	}
	return nil
}

type ListOptions struct {
	Since        *time.Time
	From         *time.Time
	To           *time.Time
	Tag          string
	Source       string
	OKR          string
	Project      string
	NoOKR        bool
	ForceRefresh bool
}

func (s *Store) ListEntries(ctx context.Context, opts ListOptions) ([]*Entry, error) {
	meta := s.loadCacheMeta()

	_, dirContents, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, "entries", nil)
	if err != nil {
		if s.cacheDir != "" && len(meta.Entries) > 0 {
			fmt.Fprintln(os.Stderr, "[aviso: sem acesso ao GitHub — exibindo cache local]")
			return s.listEntriesFromCache(meta, opts)
		}
		return nil, fmt.Errorf("listing entries directory: %w", err)
	}

	var entries []*Entry
	for _, fc := range dirContents {
		if fc.GetType() != "file" || !strings.HasSuffix(fc.GetName(), ".json") {
			continue
		}

		var e *Entry
		cached, inCache := meta.Entries[fc.GetPath()]
		if !opts.ForceRefresh && inCache && cached.SHA == fc.GetSHA() && s.cacheDir != "" {
			e, err = s.readEntryFromCache(fc.GetName(), cached.SHA)
			if err != nil {
				e = nil // fallthrough to GitHub fetch on read error
			}
		}

		if e == nil {
			fileContent, _, _, err := s.client.Repositories.GetContents(ctx, s.owner, s.repo, fc.GetPath(), nil)
			if err != nil {
				continue
			}
			rawContent, _ := fileContent.GetContent()
			decoded, err := base64.StdEncoding.DecodeString(rawContent)
			if err != nil {
				cleaned := strings.ReplaceAll(rawContent, "\n", "")
				decoded, err = base64.StdEncoding.DecodeString(cleaned)
				if err != nil {
					continue
				}
			}
			var entry Entry
			if err := json.Unmarshal(decoded, &entry); err != nil {
				continue
			}
			entry.FileSHA = fileContent.GetSHA()
			e = &entry
			if s.cacheDir != "" {
				_ = s.writeEntryToCache(e, fc.GetPath(), fileContent.GetSHA())
			}
		}

		if !applyFilters(e, opts) {
			continue
		}
		entries = append(entries, e)
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
