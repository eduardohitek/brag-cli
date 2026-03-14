package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
)

type StarTrailConfig struct {
	FilePath    string `yaml:"file_path,omitempty"`
	CurrentRole string `yaml:"current_role,omitempty"`
	TargetRole  string `yaml:"target_role,omitempty"`
}

func (c *StarTrailConfig) IsConfigured() bool {
	return c != nil && c.FilePath != "" && c.CurrentRole != ""
}

func (c *StarTrailConfig) Content() (string, error) {
	if c == nil || c.FilePath == "" {
		return "", nil
	}
	data, err := os.ReadFile(c.FilePath)
	if err != nil {
		return "", fmt.Errorf("reading startrail file: %w", err)
	}
	return string(data), nil
}

type OKR struct {
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Active bool   `yaml:"active"`
}

type StorageConfig struct {
	GithubToken string `yaml:"github_token"`
	Repo        string `yaml:"repo"`
}

type GithubSyncConfig struct {
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
}

type JiraConfig struct {
	BaseURL  string `yaml:"base_url"`
	Email    string `yaml:"email"`
	APIToken string `yaml:"api_token"`
}

type SyncConfig struct {
	DefaultDays int `yaml:"default_days"`
}

type Config struct {
	AnthropicAPIKey string           `yaml:"anthropic_api_key"`
	OpenAIAPIKey    string           `yaml:"openai_api_key,omitempty"`
	AIProvider      string           `yaml:"ai_provider,omitempty"`
	AIModel         string           `yaml:"ai_model,omitempty"`
	Storage         StorageConfig    `yaml:"storage"`
	GithubSync      GithubSyncConfig `yaml:"github_sync"`
	Jira            JiraConfig       `yaml:"jira"`
	Sync            SyncConfig       `yaml:"sync"`
	OKRs            []OKR            `yaml:"okrs"`
	StarTrail       StarTrailConfig  `yaml:"star_trail,omitempty"`
}

func (c *Config) ResolveAIProvider(providerFlag, modelFlag string) (provider, model string) {
	provider = providerFlag
	if provider == "" {
		provider = c.AIProvider
	}
	if provider == "" {
		provider = ProviderAnthropic
	}
	model = modelFlag
	if model == "" {
		model = c.AIModel
	}
	return provider, model
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".brag")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func CacheDir() string {
	return filepath.Join(ConfigDir(), "cache")
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Sync: SyncConfig{DefaultDays: 30}}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	if err := os.MkdirAll(CacheDir(), 0700); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}

func (c *Config) ActiveOKRs() []OKR {
	var active []OKR
	for _, o := range c.OKRs {
		if o.Active {
			active = append(active, o)
		}
	}
	return active
}

func (c *Config) FindOKR(id string) *OKR {
	for i := range c.OKRs {
		if c.OKRs[i].ID == id {
			return &c.OKRs[i]
		}
	}
	return nil
}
