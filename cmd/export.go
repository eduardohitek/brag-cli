package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
	"github.com/eduardohitek/brag/internal/exporter"
	"github.com/eduardohitek/brag/internal/store"
)

var (
	exportFormat string
	exportReport string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a report as PDF or Markdown",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "pdf", "Output format: pdf or md")
	exportCmd.Flags().StringVar(&exportReport, "report", "", "Report filename (default: latest)")
}

func runExport(cmd *cobra.Command, args []string) error {
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

	// Determine which report to export
	reportFile := exportReport
	if reportFile == "" {
		reports, err := s.ListReports(ctx)
		if err != nil {
			return fmt.Errorf("listing reports: %w", err)
		}
		if len(reports) == 0 {
			return fmt.Errorf("no reports found — run `brag report` first")
		}
		sort.Strings(reports)
		reportFile = reports[len(reports)-1] // latest
		fmt.Printf("Using latest report: %s\n", reportFile)
	}

	// Fetch report content
	content, err := s.GetReport(ctx, reportFile)
	if err != nil {
		return fmt.Errorf("fetching report: %w", err)
	}

	// Determine output filename
	base := strings.TrimSuffix(reportFile, ".md")
	var outputPath string

	switch strings.ToLower(exportFormat) {
	case "md", "markdown":
		outputPath = filepath.Join(".", base+".md")
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing markdown file: %w", err)
		}
	case "pdf":
		outputPath = filepath.Join(".", base+".pdf")
		fmt.Println("Generating PDF (this may take a moment)...")
		timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		if err := exporter.ExportPDF(timeoutCtx, content, outputPath); err != nil {
			return fmt.Errorf("generating PDF: %w\n\nTip: PDF export requires Google Chrome or Chromium to be installed.", err)
		}
	default:
		return fmt.Errorf("unknown format %q, expected pdf or md", exportFormat)
	}

	absPath, _ := filepath.Abs(outputPath)
	fmt.Printf("Exported: %s\n", absPath)
	return nil
}
