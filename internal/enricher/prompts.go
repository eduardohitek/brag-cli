package enricher

import (
	"fmt"

	"github.com/eduardohitek/brag/internal/config"
)

func buildEnrichSystemPrompt(startrail *config.StarTrailConfig) string {
	prompt := `Você é um coach de carreira técnica. Dado um texto bruto de um engenheiro de software e uma lista de OKRs ativos, retorne um objeto JSON com:
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
			prompt += startrailSection
		}
	}

	return prompt
}

func buildReportSystemPrompt(startrail *config.StarTrailConfig) string {
	prompt := `Você é um coach de carreira técnica ajudando um engenheiro de software a escrever uma autoavaliação de desempenho.
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
		prompt += roleNote
	}

	return prompt
}
