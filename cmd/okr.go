package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
)

var okrCmd = &cobra.Command{
	Use:   "okr",
	Short: "Manage OKRs",
}

var (
	okrAddID    string
	okrAddTitle string
)

var okrAddSubCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an OKR",
	RunE:  runOKRAdd,
}

var okrListSubCmd = &cobra.Command{
	Use:   "list",
	Short: "List all OKRs",
	RunE:  runOKRList,
}

var okrDeactivateSubCmd = &cobra.Command{
	Use:   "deactivate <id>",
	Short: "Mark an OKR as inactive",
	Args:  cobra.ExactArgs(1),
	RunE:  runOKRDeactivate,
}

func init() {
	okrAddSubCmd.Flags().StringVar(&okrAddID, "id", "", "OKR ID (required)")
	okrAddSubCmd.Flags().StringVar(&okrAddTitle, "title", "", "OKR title (required)")
	okrAddSubCmd.MarkFlagRequired("id")
	okrAddSubCmd.MarkFlagRequired("title")

	okrCmd.AddCommand(okrAddSubCmd, okrListSubCmd, okrDeactivateSubCmd)
}

func runOKRAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	existing := cfg.FindOKR(okrAddID)
	if existing != nil {
		existing.Title = okrAddTitle
		existing.Active = true
		fmt.Printf("Updated OKR: %s — %s\n", okrAddID, okrAddTitle)
	} else {
		cfg.OKRs = append(cfg.OKRs, config.OKR{
			ID:     okrAddID,
			Title:  okrAddTitle,
			Active: true,
		})
		fmt.Printf("Added OKR: %s — %s\n", okrAddID, okrAddTitle)
	}

	return config.Save(cfg)
}

func runOKRList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.OKRs) == 0 {
		fmt.Println("No OKRs configured.")
		return nil
	}

	fmt.Printf("%-20s %-8s %s\n", "ID", "Active", "Title")
	fmt.Println("------------------------------------------------------------")
	for _, o := range cfg.OKRs {
		active := "yes"
		if !o.Active {
			active = "no"
		}
		fmt.Printf("%-20s %-8s %s\n", o.ID, active, o.Title)
	}
	return nil
}

func runOKRDeactivate(cmd *cobra.Command, args []string) error {
	id := args[0]
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	okr := cfg.FindOKR(id)
	if okr == nil {
		return fmt.Errorf("OKR %q not found", id)
	}

	okr.Active = false
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("OKR %q marked as inactive.\n", id)
	return nil
}
