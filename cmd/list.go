package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/store"
)

var (
	listSince   string
	listFrom    string
	listTo      string
	listTag     string
	listSource  string
	listPeriod  string
	listOKR     string
	listProject string
	listNoOKR   bool
	listRefresh bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List accomplishment entries",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listSince, "since", "", "Show entries since this date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listFrom, "from", "", "Show entries from this date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listTo, "to", "", "Show entries to this date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().StringVar(&listSource, "source", "", "Filter by source (manual, github, jira)")
	listCmd.Flags().StringVar(&listPeriod, "period", "", "Period shorthand (e.g. Q1-2025, H1-2025)")
	listCmd.Flags().StringVar(&listOKR, "okr", "", "Filter by OKR ID")
	listCmd.Flags().StringVar(&listProject, "project", "", "Filter by GitHub project name")
	listCmd.Flags().BoolVar(&listNoOKR, "no-okr", false, "Show only entries without an OKR")
	listCmd.Flags().BoolVar(&listRefresh, "refresh", false, "Force refresh from GitHub, ignoring local cache")
}

func runList(cmd *cobra.Command, args []string) error {
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

	opts := store.ListOptions{
		Tag:          listTag,
		Source:       listSource,
		OKR:          listOKR,
		Project:      listProject,
		NoOKR:        listNoOKR,
		ForceRefresh: listRefresh,
	}

	if listPeriod != "" {
		from, to, err := parsePeriod(listPeriod)
		if err != nil {
			return err
		}
		opts.From = &from
		opts.To = &to
	} else {
		if listSince != "" {
			t, err := time.Parse("2006-01-02", listSince)
			if err != nil {
				return fmt.Errorf("invalid --since date: %w", err)
			}
			opts.Since = &t
		}
		if listFrom != "" {
			t, err := time.Parse("2006-01-02", listFrom)
			if err != nil {
				return fmt.Errorf("invalid --from date: %w", err)
			}
			opts.From = &t
		}
		if listTo != "" {
			t, err := time.Parse("2006-01-02", listTo)
			if err != nil {
				return fmt.Errorf("invalid --to date: %w", err)
			}
			opts.To = &t
		}
	}

	entries, err := s.ListEntries(ctx, opts)
	if err != nil {
		return fmt.Errorf("listing entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No entries found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tSOURCE\tPROJECT\tOKR\tTAGS\tSCORE\tSUMMARY")
	fmt.Fprintln(w, "----\t------\t-------\t---\t----\t-----\t-------")

	for _, e := range entries {
		okrStr := ""
		if e.OKR != nil {
			okrStr = e.OKR.ID
		}
		summary := e.Enriched
		if summary == "" {
			summary = e.Raw
		}
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			e.Date.Format("2006-01-02"),
			e.Source,
			e.GithubProject,
			okrStr,
			strings.Join(e.Tags, ","),
			e.ImpactScore,
			summary,
		)
	}
	w.Flush()
	fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d entries\n", len(entries))
	return nil
}

// parsePeriod parses shorthands like "Q1-2025" or "H1-2025" into date ranges.
func parsePeriod(period string) (from, to time.Time, err error) {
	upper := strings.ToUpper(period)
	parts := strings.SplitN(upper, "-", 2)
	if len(parts) != 2 {
		return from, to, fmt.Errorf("invalid period format %q, expected e.g. Q1-2025 or H1-2025", period)
	}
	quarter := parts[0]
	yearStr := parts[1]

	var year int
	if _, err = fmt.Sscanf(yearStr, "%d", &year); err != nil {
		return from, to, fmt.Errorf("invalid year in period: %w", err)
	}

	switch quarter {
	case "Q1":
		from = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 3, 31, 23, 59, 59, 0, time.UTC)
	case "Q2":
		from = time.Date(year, 4, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 6, 30, 23, 59, 59, 0, time.UTC)
	case "Q3":
		from = time.Date(year, 7, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 9, 30, 23, 59, 59, 0, time.UTC)
	case "Q4":
		from = time.Date(year, 10, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	case "H1":
		from = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 6, 30, 23, 59, 59, 0, time.UTC)
	case "H2":
		from = time.Date(year, 7, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	default:
		return from, to, fmt.Errorf("unknown period %q (expected Q1-Q4 or H1-H2)", quarter)
	}
	return from, to, nil
}
