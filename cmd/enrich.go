package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/enricher"
	"github.com/eduardohitek/brag/internal/store"
)

var (
	enrichFrom     string
	enrichTo       string
	enrichPeriod   string
	enrichAll      bool
	enrichProvider string
	enrichModel    string
)

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich entries with AI (STAR format, tags, impact score)",
	RunE:  runEnrich,
}

func init() {
	enrichCmd.Flags().StringVar(&enrichFrom, "from", "", "Start date (YYYY-MM-DD)")
	enrichCmd.Flags().StringVar(&enrichTo, "to", "", "End date (YYYY-MM-DD)")
	enrichCmd.Flags().StringVar(&enrichPeriod, "period", "", "Period shorthand (e.g. Q1-2025)")
	enrichCmd.Flags().BoolVar(&enrichAll, "all", false, "Re-enrich already enriched entries")
	enrichCmd.Flags().StringVar(&enrichProvider, "provider", "", "AI provider (anthropic, openai)")
	enrichCmd.Flags().StringVar(&enrichModel, "model", "", "AI model override")
}

func runEnrich(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Storage.GithubToken == "" || cfg.Storage.Repo == "" {
		return fmt.Errorf("storage not configured — run `brag init`")
	}

	s, err := store.New(cfg.Storage.GithubToken, cfg.Storage.Repo, config.CacheDir())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	opts := store.ListOptions{}

	if enrichPeriod != "" {
		from, to, err := parsePeriod(enrichPeriod)
		if err != nil {
			return err
		}
		opts.From = &from
		opts.To = &to
	} else {
		if enrichFrom != "" {
			t, err := time.Parse("2006-01-02", enrichFrom)
			if err != nil {
				return fmt.Errorf("invalid --from date: %w", err)
			}
			opts.From = &t
		}
		if enrichTo != "" {
			t, err := time.Parse("2006-01-02", enrichTo)
			if err != nil {
				return fmt.Errorf("invalid --to date: %w", err)
			}
			opts.To = &t
		}
	}

	fmt.Println("Fetching entries...")
	entries, err := s.ListEntries(ctx, opts)
	if err != nil {
		return fmt.Errorf("listing entries: %w", err)
	}

	provider, model := cfg.ResolveAIProvider(enrichProvider, enrichModel)
	enc, err := enricher.New(provider, model, cfg.AnthropicAPIKey, cfg.OpenAIAPIKey)
	if err != nil {
		return err
	}
	var count int

	for _, e := range entries {
		if !enrichAll && e.Enriched != "" {
			continue
		}

		result, err := enc.Enrich(ctx, e.Raw, cfg.ActiveOKRs(), &cfg.StarTrail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: enrichment failed for %q: %v\n", e.Raw, err)
			continue
		}

		e.Enriched = result.Enriched
		e.Tags = result.Tags
		e.ImpactScore = result.ImpactScore
		if result.OKRID != nil && *result.OKRID != "" && e.OKR == nil {
			okr := cfg.FindOKR(*result.OKRID)
			if okr != nil {
				e.OKR = &store.OKRRef{ID: okr.ID, Title: okr.Title}
			}
		}

		if err := s.UpdateEntry(ctx, e); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update entry in GitHub: %v\n", err)
			continue
		}

		if err := saveToCacheLocal(e); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cache write failed: %v\n", err)
		}

		fmt.Printf("  ✓ [%s] %s\n", e.Source, e.Raw)
		count++
	}

	fmt.Printf("\n%d entries enriched.\n", count)
	return nil
}
