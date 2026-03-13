package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eduardohitek/brag/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard to configure brag",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== brag setup wizard ===")
	fmt.Println()

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Storage GitHub token
	cfg.Storage.GithubToken = prompt(reader, "GitHub token for storage repo", cfg.Storage.GithubToken)
	cfg.Storage.Repo = prompt(reader, "Storage repo (owner/repo)", cfg.Storage.Repo)

	// GitHub sync
	fmt.Println()
	fmt.Println("--- GitHub sync (for PR/issue tracking) ---")
	cfg.GithubSync.Token = prompt(reader, "GitHub token for sync (can be same as storage)", cfg.GithubSync.Token)
	cfg.GithubSync.Username = prompt(reader, "GitHub username", cfg.GithubSync.Username)

	// Jira
	fmt.Println()
	fmt.Println("--- Jira integration (optional, press Enter to skip) ---")
	cfg.Jira.BaseURL = prompt(reader, "Jira base URL (e.g. https://company.atlassian.net)", cfg.Jira.BaseURL)
	if cfg.Jira.BaseURL != "" {
		cfg.Jira.Email = prompt(reader, "Jira email", cfg.Jira.Email)
		cfg.Jira.APIToken = prompt(reader, "Jira API token", cfg.Jira.APIToken)
	}

	// Anthropic
	fmt.Println()
	cfg.AnthropicAPIKey = prompt(reader, "Anthropic API key", cfg.AnthropicAPIKey)

	// Sync defaults
	fmt.Println()
	daysStr := prompt(reader, "Default sync window (days)", strconv.Itoa(cfg.Sync.DefaultDays))
	if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
		cfg.Sync.DefaultDays = days
	} else if cfg.Sync.DefaultDays == 0 {
		cfg.Sync.DefaultDays = 30
	}

	// OKRs
	fmt.Println()
	fmt.Println("--- OKRs (optional) ---")
	for {
		okrID := prompt(reader, "OKR ID (press Enter to finish adding OKRs)", "")
		if okrID == "" {
			break
		}
		okrTitle := prompt(reader, "OKR title", "")
		if okrTitle == "" {
			fmt.Println("OKR title is required. Skipping.")
			continue
		}
		// Check if already exists
		existing := cfg.FindOKR(okrID)
		if existing != nil {
			existing.Title = okrTitle
			existing.Active = true
		} else {
			cfg.OKRs = append(cfg.OKRs, config.OKR{
				ID:     okrID,
				Title:  okrTitle,
				Active: true,
			})
		}
		fmt.Printf("Added OKR: %s\n", okrID)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", config.ConfigPath())
	return nil
}

func prompt(reader *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, maskSecret(current))
	} else {
		fmt.Printf("%s: ", label)
	}

	line, _ := reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return current
	}
	return line
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
