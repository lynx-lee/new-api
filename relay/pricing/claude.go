package pricing

import (
	"context"
	"time"
)

// ClaudeAdapter returns pricing from a hardcoded map. Anthropic has no public pricing API.
type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter { return &ClaudeAdapter{} }
func (a *ClaudeAdapter) Name() string  { return "claude" }

func (a *ClaudeAdapter) Fetch(_ context.Context) (*PricingResult, error) {
	result := &PricingResult{
		Provider:  a.Name(),
		Models:    make(map[string]ModelPricing),
		FetchedAt: time.Now(),
	}
	for model, p := range claudePrices {
		mp := ModelPricing{
			InputPrice:  p.input,
			OutputPrice: p.output,
			Currency:    "USD",
			UnitType:    "per_1M_tokens",
		}
		if p.cacheRead > 0 {
			v := p.cacheRead
			mp.CacheReadPrice = &v
		}
		result.Models[model] = mp
	}
	return result, nil
}

type claudeP struct{ input, output, cacheRead float64 }

// Last updated: 2026-08. Source: https://www.anthropic.com/pricing
var claudePrices = map[string]claudeP{
	"claude-opus-4-7":            {15, 75, 0},
	"claude-opus-4-6":            {15, 75, 0},
	"claude-opus-4-5-20251101":   {15, 75, 0},
	"claude-opus-4-20250514":     {15, 75, 0},
	"claude-opus-4-1-20250805":   {15, 75, 0},
	"claude-sonnet-4-5-20250929": {3, 15, 0},
	"claude-sonnet-4-20250514":   {3, 15, 0},
	"claude-haiku-4-5-20251001":  {1, 5, 0},
	"claude-3-7-sonnet-20250219": {3, 15, 0},
	"claude-3-5-sonnet-20241022": {3, 15, 0},
	"claude-3-5-sonnet-20240620": {3, 15, 0},
	"claude-3-5-haiku-20241022":  {1, 5, 0},
	"claude-3-opus-20240229":     {15, 75, 0},
	"claude-3-sonnet-20240229":   {3, 15, 0},
	"claude-3-haiku-20240307":    {0.25, 1.25, 0},
}
