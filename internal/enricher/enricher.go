package enricher

import (
	"fmt"

	"github.com/eduardohitek/brag/internal/config"
)

// EnrichedResult is the structured output returned by all providers.
type EnrichedResult struct {
	Enriched    string   `json:"enriched"`
	Tags        []string `json:"tags"`
	ImpactScore int      `json:"impact_score"`
	OKRID       *string  `json:"okr_id"`
}

// Enricher is a backward-compatible alias for AnthropicProvider.
type Enricher = AnthropicProvider

// New constructs the appropriate Provider for the given provider name.
// provider and model may be empty strings; defaults are applied automatically.
func New(provider, model, anthropicKey, openAIKey string) (Provider, error) {
	switch provider {
	case config.ProviderAnthropic, "":
		if anthropicKey == "" {
			return nil, fmt.Errorf("anthropic_api_key not configured — run `brag init`")
		}
		return newAnthropicProvider(anthropicKey, model), nil
	case config.ProviderOpenAI:
		if openAIKey == "" {
			return nil, fmt.Errorf("openai_api_key not configured — run `brag init`")
		}
		return newOpenAIProvider(openAIKey, model), nil
	default:
		return nil, fmt.Errorf("unknown ai_provider %q — supported: anthropic, openai", provider)
	}
}
