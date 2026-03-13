package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
)

var (
	startrailFile        string
	startrailCurrentRole string
	startrailTargetRole  string
)

var startrailCmd = &cobra.Command{
	Use:   "startrail",
	Short: "Manage career ladder (StarTrail) configuration",
}

var startrailSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Configure the StarTrail document and roles",
	RunE:  runStarTrailSet,
}

var startrailShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current StarTrail configuration",
	RunE:  runStarTrailShow,
}

var startrailClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove StarTrail configuration",
	RunE:  runStarTrailClear,
}

func init() {
	startrailSetCmd.Flags().StringVar(&startrailFile, "file", "", "Path to the StarTrail/career ladder document (required)")
	startrailSetCmd.Flags().StringVar(&startrailCurrentRole, "current-role", "", "Your current role (required)")
	startrailSetCmd.Flags().StringVar(&startrailTargetRole, "target-role", "", "Your target role (optional)")
	_ = startrailSetCmd.MarkFlagRequired("file")
	_ = startrailSetCmd.MarkFlagRequired("current-role")

	startrailCmd.AddCommand(startrailSetCmd)
	startrailCmd.AddCommand(startrailShowCmd)
	startrailCmd.AddCommand(startrailClearCmd)
}

func runStarTrailSet(cmd *cobra.Command, args []string) error {
	// Validate file exists and is readable
	data, err := os.ReadFile(startrailFile)
	if err != nil {
		return fmt.Errorf("cannot read file %q: %w", startrailFile, err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.StarTrail = config.StarTrailConfig{
		FilePath:    startrailFile,
		CurrentRole: startrailCurrentRole,
		TargetRole:  startrailTargetRole,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("StarTrail configured successfully.")
	fmt.Printf("  File:         %s\n", startrailFile)
	fmt.Printf("  Current role: %s\n", startrailCurrentRole)
	if startrailTargetRole != "" {
		fmt.Printf("  Target role:  %s\n", startrailTargetRole)
	}

	// Show first 5 lines of the document as confirmation
	fmt.Println("\nDocument preview (first 5 lines):")
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for i := 0; i < 5 && scanner.Scan(); i++ {
		fmt.Printf("  %s\n", scanner.Text())
	}

	return nil
}

func runStarTrailShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	st := &cfg.StarTrail
	if !st.IsConfigured() {
		fmt.Println("StarTrail is not configured. Use `brag startrail set` to configure it.")
		return nil
	}

	fmt.Println("StarTrail configuration:")
	fmt.Printf("  File:         %s\n", st.FilePath)
	fmt.Printf("  Current role: %s\n", st.CurrentRole)
	if st.TargetRole != "" {
		fmt.Printf("  Target role:  %s\n", st.TargetRole)
	}

	content, err := st.Content()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read StarTrail file: %v\n", err)
		return nil
	}

	fmt.Println("\nDocument preview (first 10 lines):")
	scanner := bufio.NewScanner(strings.NewReader(content))
	for i := 0; i < 10 && scanner.Scan(); i++ {
		fmt.Printf("  %s\n", scanner.Text())
	}

	return nil
}

func runStarTrailClear(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.StarTrail = config.StarTrailConfig{}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("StarTrail configuration cleared.")
	return nil
}
