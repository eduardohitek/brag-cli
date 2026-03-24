package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/store"
)

var syncCacheCmd = &cobra.Command{
	Use:   "sync-cache",
	Short: "Push local cache entries to GitHub storage",
	Long:  `Reads all entries from the local cache (~/.brag/cache/) and uploads any that are not already present in the configured GitHub storage repository.`,
	RunE:  runSyncCache,
}

func runSyncCache(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Storage.GithubToken == "" || cfg.Storage.Repo == "" {
		return fmt.Errorf("GitHub storage not configured — run `brag init`")
	}

	s, err := store.New(cfg.Storage.GithubToken, cfg.Storage.Repo)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	// Fetch all existing GitHub entries once and build lookup maps
	fmt.Println("Fetching existing entries from GitHub...")
	existing, err := s.ListEntries(ctx, store.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing GitHub entries: %w", err)
	}
	uploadedIDs := make(map[string]bool, len(existing))
	uploadedURLs := make(map[string]bool)
	for _, e := range existing {
		uploadedIDs[e.ID] = true
		if e.SourceURL != "" {
			uploadedURLs[e.SourceURL] = true
		}
	}

	// Read cache files
	cacheDir := config.CacheDir()
	dirEntries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Local cache is empty — nothing to upload.")
			return nil
		}
		return fmt.Errorf("reading cache dir: %w", err)
	}

	var uploaded, skipped, failed int
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}

		path := filepath.Join(cacheDir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: reading %s: %v\n", de.Name(), err)
			failed++
			continue
		}

		var e store.Entry
		if err := json.Unmarshal(data, &e); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: parsing %s: %v\n", de.Name(), err)
			failed++
			continue
		}

		// Deduplication: by SourceURL for synced entries, by ID for manual entries
		if e.SourceURL != "" {
			if uploadedURLs[e.SourceURL] {
				skipped++
				continue
			}
		} else {
			if uploadedIDs[e.ID] {
				skipped++
				continue
			}
		}

		if err := s.SaveEntry(ctx, &e); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: uploading %s: %v\n", e.ID, err)
			failed++
			continue
		}

		uploadedIDs[e.ID] = true
		if e.SourceURL != "" {
			uploadedURLs[e.SourceURL] = true
		}
		fmt.Printf("  + [%s] %s\n", e.Source, e.Raw)
		uploaded++
	}

	fmt.Printf("\nSync-cache complete:\n")
	fmt.Printf("  Uploaded: %d\n", uploaded)
	fmt.Printf("  Skipped (already in GitHub): %d\n", skipped)
	if failed > 0 {
		fmt.Printf("  Failed: %d\n", failed)
	}
	return nil
}
