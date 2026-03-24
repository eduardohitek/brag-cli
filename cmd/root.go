package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "brag",
	Version: Version,
	Short:   "Track your engineering accomplishments for performance reviews",
	Long: `brag helps software engineers capture accomplishments, sync GitHub PRs/issues
and Jira tickets, enrich entries with AI, and generate performance review reports.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(okrCmd)
	rootCmd.AddCommand(startrailCmd)
	rootCmd.AddCommand(enrichCmd)
	rootCmd.AddCommand(syncCacheCmd)
}
