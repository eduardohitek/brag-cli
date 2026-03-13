package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	ghsync "github.com/eduardohitek/brag/internal/github"
	"github.com/eduardohitek/brag/internal/jira"
	"github.com/eduardohitek/brag/internal/store"
)

var syncDays int

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync GitHub PRs/issues and Jira tickets",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().IntVar(&syncDays, "days", 0, "Number of days to look back (default from config)")
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	days := cfg.Sync.DefaultDays
	if syncDays > 0 {
		days = syncDays
	}
	if days == 0 {
		days = 30
	}

	if cfg.Storage.GithubToken == "" || cfg.Storage.Repo == "" {
		return fmt.Errorf("storage GitHub token and repo must be configured (run `brag init`)")
	}

	s, err := store.New(cfg.Storage.GithubToken, cfg.Storage.Repo)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var githubCount, jiraCount int

	// Sync GitHub
	if cfg.GithubSync.Token != "" && cfg.GithubSync.Username != "" {
		fmt.Printf("Syncing GitHub (last %d days)...\n", days)
		ghClient := ghsync.New(cfg.GithubSync.Token, cfg.GithubSync.Username)

		prs, err := ghClient.FetchRecentPRs(ctx, days)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: GitHub PR sync failed: %v\n", err)
		} else {
			for _, e := range prs {
				n, err := processAndSaveEntry(ctx, e, s)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save PR entry: %v\n", err)
				}
				githubCount += n
			}
		}

		issues, err := ghClient.FetchRecentIssues(ctx, days)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: GitHub issue sync failed: %v\n", err)
		} else {
			for _, e := range issues {
				n, err := processAndSaveEntry(ctx, e, s)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save issue entry: %v\n", err)
				}
				githubCount += n
			}
		}
	} else {
		fmt.Println("GitHub sync skipped (not configured).")
	}

	// Sync Jira
	if cfg.Jira.BaseURL != "" && cfg.Jira.Email != "" && cfg.Jira.APIToken != "" {
		fmt.Printf("Syncing Jira (last %d days)...\n", days)
		jiraClient := jira.New(cfg.Jira.BaseURL, cfg.Jira.Email, cfg.Jira.APIToken)

		tickets, err := jiraClient.FetchRecentDoneTickets(ctx, days)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Jira sync failed: %v\n", err)
		} else {
			for _, e := range tickets {
				n, err := processAndSaveEntry(ctx, e, s)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save Jira entry: %v\n", err)
				}
				jiraCount += n
			}
		}
	} else {
		fmt.Println("Jira sync skipped (not configured).")
	}

	fmt.Printf("\nSync complete:\n")
	fmt.Printf("  GitHub: %d new entries\n", githubCount)
	fmt.Printf("  Jira:   %d new entries\n", jiraCount)

	return nil
}

// processAndSaveEntry checks for duplicates and saves a single entry.
// Returns 1 if a new entry was saved, 0 if skipped (duplicate).
func processAndSaveEntry(ctx context.Context, e *store.Entry, s *store.Store) (int, error) {
	// Check for duplicate
	if e.SourceURL != "" {
		exists, err := s.EntryExists(ctx, e.SourceURL)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, nil
		}
	}

	// Save to cache
	if err := saveToCacheLocal(e); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cache write failed: %v\n", err)
	}

	// Save to GitHub
	if err := s.SaveEntry(ctx, e); err != nil {
		return 0, err
	}

	fmt.Printf("  + [%s] %s\n", e.Source, e.Raw)
	return 1, nil
}
