package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/store"
)

var (
	addProject string
	addOKR     string
)

var addCmd = &cobra.Command{
	Use:   "add <note>",
	Short: "Add a manual accomplishment entry",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addProject, "project", "", "GitHub project name")
	addCmd.Flags().StringVar(&addOKR, "okr", "", "OKR ID to associate with this entry")
}

func runAdd(cmd *cobra.Command, args []string) error {
	raw := args[0]
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	now := time.Now().UTC()
	e := &store.Entry{
		ID:            now.Format("20060102-150405"),
		Date:          now,
		Raw:           raw,
		Source:        "manual",
		GithubProject: addProject,
	}

	// Handle OKR assignment
	if addOKR != "" {
		okr := cfg.FindOKR(addOKR)
		if okr == nil {
			return fmt.Errorf("OKR %q not found in config", addOKR)
		}
		e.OKR = &store.OKRRef{ID: okr.ID, Title: okr.Title}
	}

	// Save to local cache first
	if err := saveToCacheLocal(e); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save to local cache: %v\n", err)
	}

	// Push to GitHub repo
	if cfg.Storage.GithubToken != "" && cfg.Storage.Repo != "" {
		s, err := store.New(cfg.Storage.GithubToken, cfg.Storage.Repo)
		if err != nil {
			return fmt.Errorf("creating store: %w", err)
		}
		if err := s.SaveEntry(ctx, e); err != nil {
			return fmt.Errorf("saving entry to GitHub: %w", err)
		}
		fmt.Println("Entry saved to GitHub repo.")
	} else {
		fmt.Println("Entry saved to local cache only (GitHub storage not configured).")
	}
	fmt.Println("Run `brag enrich` to enrich with AI.")

	// Print result
	fmt.Println()
	fmt.Printf("ID:           %s\n", e.ID)
	fmt.Printf("Date:         %s\n", e.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("Raw:          %s\n", e.Raw)
	if e.Enriched != "" {
		fmt.Printf("Enriched:     %s\n", e.Enriched)
	}
	if len(e.Tags) > 0 {
		fmt.Printf("Tags:         %v\n", e.Tags)
	}
	if e.ImpactScore > 0 {
		fmt.Printf("Impact Score: %d/5\n", e.ImpactScore)
	}
	if e.OKR != nil {
		fmt.Printf("OKR:          %s — %s\n", e.OKR.ID, e.OKR.Title)
	}
	if e.GithubProject != "" {
		fmt.Printf("Project:      %s\n", e.GithubProject)
	}

	return nil
}

func saveToCacheLocal(e *store.Entry) error {
	cacheDir := config.CacheDir()
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}
	filename := fmt.Sprintf("%s-%s.json", e.Date.UTC().Format("2006-01-02-150405"), e.Source)
	path := filepath.Join(cacheDir, filename)
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
