package pricing

import (
	"context"
	"time"
)

// ModelPricing stores raw provider pricing for one model.
// All prices are in the provider's original currency (USD or RMB).
type ModelPricing struct {
	InputPrice     float64  // price per unit (typically per 1M tokens)
	OutputPrice    float64  // price per unit
	CacheReadPrice *float64 // optional: cache read price per unit
	Currency       string   // "USD" or "RMB"
	UnitType       string   // "per_1M_tokens", "per_1K_tokens", "per_image", "per_call"
}

// PricingResult holds the fetched pricing for all models from one provider.
type PricingResult struct {
	Provider  string
	Models    map[string]ModelPricing // model_name -> pricing
	FetchedAt time.Time
}

// Adapter fetches model pricing from a provider's API endpoint.
type Adapter interface {
	// Name returns the provider identifier (e.g. "openai", "claude", "openrouter").
	Name() string

	// Fetch retrieves current pricing from the provider.
	Fetch(ctx context.Context) (*PricingResult, error)
}

// ratio conversion constants, mirrored from setting/ratio_setting to avoid import cycle.
// 1 ratio unit = $0.002 / 1K tokens.
// $1 = 500 ratio units.  $X/1M tokens = X * 0.5 ratio.
const (
	USD2RMB = 7.3
	USD     = 500.0
	RMB     = USD / USD2RMB
)

// ConvertUSDPerMillionToRatio converts a USD-per-1M-tokens price to internal ratio.
func ConvertUSDPerMillionToRatio(price float64) float64 {
	return price * 0.5
}

// ConvertRMBPerMillionToRatio converts a RMB-per-1M-tokens price to internal ratio.
func ConvertRMBPerMillionToRatio(price float64) float64 {
	return price * 0.5 / USD2RMB
}

// ConvertRMBPerKToRatio converts a RMB-per-1K-tokens price to internal ratio.
func ConvertRMBPerKToRatio(price float64) float64 {
	return price * RMB
}

// ConvertToRatio converts a ModelPricing to internal ratio values.
// Returns modelRatio, completionRatio (output/input), and optional cacheRatio.
func ConvertToRatio(p ModelPricing) (modelRatio, completionRatio float64, cacheRatio *float64) {
	if p.UnitType == "" {
		p.UnitType = "per_1M_tokens"
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}

	if p.InputPrice <= 0 {
		return 0, 1, nil
	}

	switch p.UnitType {
	case "per_1M_tokens":
		switch p.Currency {
		case "RMB":
			modelRatio = ConvertRMBPerMillionToRatio(p.InputPrice)
		default:
			modelRatio = ConvertUSDPerMillionToRatio(p.InputPrice)
		}
	case "per_1K_tokens":
		switch p.Currency {
		case "RMB":
			modelRatio = ConvertRMBPerKToRatio(p.InputPrice)
		default:
			modelRatio = p.InputPrice * USD
		}
	default:
		modelRatio = ConvertUSDPerMillionToRatio(p.InputPrice)
	}

	modelRatio = roundRatio(modelRatio)

	if p.OutputPrice > 0 {
		completionRatio = roundRatio(p.OutputPrice / p.InputPrice)
	} else {
		completionRatio = 1.0
	}

	if p.CacheReadPrice != nil && *p.CacheReadPrice > 0 {
		cr := roundRatio(*p.CacheReadPrice / p.InputPrice)
		cacheRatio = &cr
	}

	return
}

func roundRatio(v float64) float64 {
	return float64(int(v*1e6+0.5)) / 1e6
}
