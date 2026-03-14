package enricher

import (
	"context"

	"github.com/eduardohitek/brag/internal/config"
)

// Provider is the interface satisfied by all AI backend implementations.
type Provider interface {
	Enrich(ctx context.Context, raw string, okrs []config.OKR, startrail *config.StarTrailConfig) (*EnrichedResult, error)
	SynthesizeReport(ctx context.Context, entries interface{}, startrail *config.StarTrailConfig) (string, error)
}
