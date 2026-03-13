package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eduardohitek/brag/internal/store"
)

// Generate creates a structured Markdown report from the given entries.
// Groups by OKR first, then by unassigned entries grouped by impact category.
func Generate(entries []*store.Entry, narrative string) string {
	var sb strings.Builder

	now := time.Now()
	sb.WriteString(fmt.Sprintf("# Avaliação de Desempenho — %s\n\n", now.Format("January 2006")))
	sb.WriteString(fmt.Sprintf("_Generated: %s_\n\n", now.Format("2006-01-02 15:04:05")))
	sb.WriteString("---\n\n")

	if narrative != "" {
		sb.WriteString(narrative)
		sb.WriteString("\n\n---\n\n")
	}

	// Group by OKR
	okrMap := make(map[string][]*store.Entry)
	var unassigned []*store.Entry
	var okrOrder []string
	seenOKR := make(map[string]bool)

	for _, e := range entries {
		if e.OKR != nil {
			if !seenOKR[e.OKR.ID] {
				seenOKR[e.OKR.ID] = true
				okrOrder = append(okrOrder, e.OKR.ID)
			}
			okrMap[e.OKR.ID] = append(okrMap[e.OKR.ID], e)
		} else {
			unassigned = append(unassigned, e)
		}
	}

	// Sort entries within each OKR by date desc
	for _, id := range okrOrder {
		ents := okrMap[id]
		sort.Slice(ents, func(i, j int) bool { return ents[i].Date.After(ents[j].Date) })
		title := id
		if len(ents) > 0 && ents[0].OKR != nil {
			title = fmt.Sprintf("%s — %s", id, ents[0].OKR.Title)
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", title))
		for _, e := range ents {
			renderEntry(&sb, e)
		}
	}

	// Unassigned entries, grouped by impact category
	if len(unassigned) > 0 {
		sb.WriteString("## Conquistas Gerais\n\n")
		sort.Slice(unassigned, func(i, j int) bool {
			return unassigned[i].ImpactScore > unassigned[j].ImpactScore
		})
		for _, e := range unassigned {
			renderEntry(&sb, e)
		}
	}

	return sb.String()
}

func renderEntry(sb *strings.Builder, e *store.Entry) {
	sb.WriteString(fmt.Sprintf("### %s\n", e.Date.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Fonte:** %s", e.Source))
	if e.SourceURL != "" {
		sb.WriteString(fmt.Sprintf(" | [Link](%s)", e.SourceURL))
	}
	if e.GithubProject != "" {
		sb.WriteString(fmt.Sprintf(" | **Projeto:** %s", e.GithubProject))
	}
	sb.WriteString(fmt.Sprintf(" | **Impacto:** %d/5\n\n", e.ImpactScore))
	if e.Enriched != "" {
		sb.WriteString(e.Enriched + "\n\n")
	} else {
		sb.WriteString(e.Raw + "\n\n")
	}
	if len(e.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("_Tags: %s_\n\n", strings.Join(e.Tags, ", ")))
	}
}
