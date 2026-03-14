package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/enricher"
	"github.com/eduardohitek/brag/internal/report"
	"github.com/eduardohitek/brag/internal/store"
)

var (
	reportFrom     string
	reportTo       string
	reportPeriod   string
	reportOKR      string
	reportProvider string
	reportModel    string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a performance review report",
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportFrom, "from", "", "Start date (YYYY-MM-DD)")
	reportCmd.Flags().StringVar(&reportTo, "to", "", "End date (YYYY-MM-DD)")
	reportCmd.Flags().StringVar(&reportPeriod, "period", "", "Period shorthand (e.g. Q1-2025)")
	reportCmd.Flags().StringVar(&reportOKR, "okr", "", "Filter by OKR ID")
	reportCmd.Flags().StringVar(&reportProvider, "provider", "", "AI provider (anthropic, openai)")
	reportCmd.Flags().StringVar(&reportModel, "model", "", "AI model override")
}

func runReport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Storage.GithubToken == "" || cfg.Storage.Repo == "" {
		return fmt.Errorf("storage not configured — run `brag init`")
	}

	s, err := store.New(cfg.Storage.GithubToken, cfg.Storage.Repo)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	opts := store.ListOptions{OKR: reportOKR}

	if reportPeriod != "" {
		from, to, err := parsePeriod(reportPeriod)
		if err != nil {
			return err
		}
		opts.From = &from
		opts.To = &to
	} else {
		if reportFrom != "" {
			t, err := time.Parse("2006-01-02", reportFrom)
			if err != nil {
				return fmt.Errorf("invalid --from date: %w", err)
			}
			opts.From = &t
		}
		if reportTo != "" {
			t, err := time.Parse("2006-01-02", reportTo)
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

	if len(entries) == 0 {
		fmt.Println("No entries found for the specified period.")
		return nil
	}

	fmt.Printf("Found %d entries. Generating narrative...\n", len(entries))

	provider, model := cfg.ResolveAIProvider(reportProvider, reportModel)
	enc, encErr := enricher.New(provider, model, cfg.AnthropicAPIKey, cfg.OpenAIAPIKey)
	if encErr != nil {
		fmt.Printf("Warning: AI not configured (%v). Generating without AI narrative.\n", encErr)
		enc = nil
	}

	var narrative string
	if enc != nil {
		// Enrich any entries that haven't been enriched yet
		for _, e := range entries {
			if e.Enriched != "" {
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
			_ = s.UpdateEntry(ctx, e)
			_ = saveToCacheLocal(e)
		}

		narrative, err = enc.SynthesizeReport(ctx, entries, &cfg.StarTrail)
		if err != nil {
			fmt.Printf("Warning: AI narrative generation failed: %v\nGenerating without AI narrative.\n", err)
		}
	}

	content := report.Generate(entries, narrative)

	// Save to GitHub repo
	filename := fmt.Sprintf("%s-report.md", time.Now().Format("2006-01-02"))
	if err := s.SaveReport(ctx, content, filename); err != nil {
		return fmt.Errorf("saving report to GitHub: %w", err)
	}

	fmt.Printf("\nReport saved to GitHub repo: reports/%s\n\n", filename)
	fmt.Println("--- Report Preview ---")
	// Print first 50 lines as preview
	lines := splitLines(content, 50)
	fmt.Println(lines)
	if len(content) > len(lines) {
		fmt.Println("... (truncated, see full report in GitHub repo)")
	}

	return nil
}

func splitLines(s string, maxLines int) string {
	count := 0
	result := ""
	for i, c := range s {
		if c == '\n' {
			count++
			if count >= maxLines {
				return s[:i]
			}
		}
		result = s[:i+1]
	}
	return result
}
