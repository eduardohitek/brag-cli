package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type fileMeta struct {
	SHA     string `json:"sha"`
	EntryID string `json:"entry_id"`
}

type cacheMeta struct {
	Entries map[string]fileMeta `json:"entries"`
}

func (s *Store) metaPath() string {
	return filepath.Join(s.cacheDir, ".meta.json")
}

func (s *Store) loadCacheMeta() *cacheMeta {
	m := &cacheMeta{Entries: make(map[string]fileMeta)}
	if s.cacheDir == "" {
		return m
	}
	data, err := os.ReadFile(s.metaPath())
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, m)
	if m.Entries == nil {
		m.Entries = make(map[string]fileMeta)
	}
	return m
}

func (s *Store) saveCacheMeta(m *cacheMeta) error {
	if err := os.MkdirAll(s.cacheDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(), data, 0600)
}

func (s *Store) writeEntryToCache(e *Entry, githubPath, sha string) error {
	if err := os.MkdirAll(s.cacheDir, 0700); err != nil {
		return err
	}
	filename := filepath.Base(githubPath)
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.cacheDir, filename), data, 0600); err != nil {
		return err
	}
	m := s.loadCacheMeta()
	m.Entries[githubPath] = fileMeta{SHA: sha, EntryID: e.ID}
	return s.saveCacheMeta(m)
}

func (s *Store) readEntryFromCache(filename, sha string) (*Entry, error) {
	data, err := os.ReadFile(filepath.Join(s.cacheDir, filename))
	if err != nil {
		return nil, err
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	e.FileSHA = sha
	return &e, nil
}

func (s *Store) listEntriesFromCache(m *cacheMeta, opts ListOptions) ([]*Entry, error) {
	var entries []*Entry
	for githubPath, fm := range m.Entries {
		filename := filepath.Base(githubPath)
		if !strings.HasSuffix(filename, ".json") {
			continue
		}
		e, err := s.readEntryFromCache(filename, fm.SHA)
		if err != nil {
			continue
		}
		if !applyFilters(e, opts) {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func applyFilters(e *Entry, opts ListOptions) bool {
	if opts.Since != nil && e.Date.Before(*opts.Since) {
		return false
	}
	if opts.From != nil && e.Date.Before(*opts.From) {
		return false
	}
	if opts.To != nil && e.Date.After(*opts.To) {
		return false
	}
	if opts.Source != "" && e.Source != opts.Source {
		return false
	}
	if opts.OKR != "" && (e.OKR == nil || e.OKR.ID != opts.OKR) {
		return false
	}
	if opts.NoOKR && e.OKR != nil {
		return false
	}
	if opts.Project != "" && e.GithubProject != opts.Project {
		return false
	}
	if opts.Tag != "" && !containsTag(e.Tags, opts.Tag) {
		return false
	}
	return true
}

