package enricher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eduardohitek/brag/internal/config"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"
const model = "claude-sonnet-4-20250514"

type EnrichedResult struct {
	Enriched    string `json:"enriched"`
	Tags        []string `json:"tags"`
	ImpactScore int    `json:"impact_score"`
	OKRID       *string `json:"okr_id"`
}

type Enricher struct {
	apiKey string
}

func New(apiKey string) *Enricher {
	return &Enricher{apiKey: apiKey}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e *Enricher) Enrich(ctx context.Context, raw string, okrs []config.OKR, startrail *config.StarTrailConfig) (*EnrichedResult, error) {
	systemPrompt := `Você é um coach de carreira técnica. Dado um texto bruto de um engenheiro de software e uma lista de OKRs ativos, retorne um objeto JSON com:
- "enriched": declaração de conquista no formato STAR (1-2 frases)
- "tags": array com valores de: confiabilidade, velocidade, liderança, mentoria, entrega, qualidade, impacto
- "impact_score": inteiro de 1 a 5
- "okr_id": string ou null — preencha apenas se a nota claramente e com alta confiança corresponder a um dos OKRs fornecidos, caso contrário null

Retorne apenas JSON válido, sem markdown.`

	if startrail.IsConfigured() {
		content, err := startrail.Content()
		if err == nil && content != "" {
			startrailSection := fmt.Sprintf(`

Considere também as competências da trilha de carreira abaixo ao avaliar a atividade:
- Cargo atual: %s`, startrail.CurrentRole)
			if startrail.TargetRole != "" {
				startrailSection += fmt.Sprintf("\n- Cargo alvo: %s", startrail.TargetRole)
			}
			startrailSection += fmt.Sprintf(`

Trilha de carreira:
---
%s
---
Ao gerar "enriched" e o "impact_score", leve em conta como essa atividade demonstra ou avança as competências esperadas para o cargo atual (e para o cargo alvo, se configurado).`, content)
			systemPrompt += startrailSection
		}
	}

	okrJSON, _ := json.Marshal(okrs)
	userMsg := fmt.Sprintf("Nota bruta: %s\n\nOKRs ativos: %s", raw, string(okrJSON))

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 512,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userMsg},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("Anthropic API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Anthropic API")
	}

	text := strings.TrimSpace(apiResp.Content[0].Text)
	// Strip markdown code fences if present
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result EnrichedResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing enriched result JSON: %w\nRaw: %s", err, text)
	}

	return &result, nil
}

func (e *Enricher) SynthesizeReport(ctx context.Context, entries interface{}, startrail *config.StarTrailConfig) (string, error) {
	systemPrompt := `Você é um coach de carreira técnica ajudando um engenheiro de software a escrever uma autoavaliação de desempenho.
Com base em uma lista de registros de trabalho enriquecidos, gere um relatório narrativo coeso adequado para uma avaliação de desempenho.
Estruture o relatório com:
1. Seções agrupadas por OKR primeiro (uma seção por OKR com suas entradas)
2. Uma seção "Conquistas Gerais" para entradas sem OKR

Use formatação markdown. Seja específico, impactante e profissional. Foque em resultados e impacto no negócio.`

	if startrail.IsConfigured() {
		roleNote := fmt.Sprintf("\n\nAo escrever a narrativa, mencione o progresso em competências relevantes para o cargo atual (%s)", startrail.CurrentRole)
		if startrail.TargetRole != "" {
			roleNote += fmt.Sprintf(" e para o cargo alvo (%s)", startrail.TargetRole)
		}
		roleNote += "."
		systemPrompt += roleNote
	}

	entriesJSON, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling entries: %w", err)
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: fmt.Sprintf("Entradas:\n%s", string(entriesJSON))},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("Anthropic API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic API")
	}

	return apiResp.Content[0].Text, nil
}
