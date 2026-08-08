package pricing

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/QuantumNous/ai-bridge/common"
)

// ModelsDevAdapter fetches pricing from models.dev /api.json.
type ModelsDevAdapter struct {
	client *http.Client
}

func NewModelsDevAdapter() *ModelsDevAdapter {
	return &ModelsDevAdapter{client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *ModelsDevAdapter) Name() string { return "models.dev" }

func (a *ModelsDevAdapter) Fetch(ctx context.Context) (*PricingResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://models.dev/api.json", nil)
	if err != nil {
		return nil, fmt.Errorf("models.dev: build request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("models.dev: http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("models.dev: read body: %w", err)
	}
	return a.parse(body)
}

type mdProvider struct {
	Models map[string]struct {
		Cost struct {
			Input     *float64 `json:"input"`
			Output    *float64 `json:"output"`
			CacheRead *float64 `json:"cache_read"`
		} `json:"cost"`
	} `json:"models"`
}

type mdPick struct {
	Provider  string
	Input     float64
	Output    *float64
	CacheRead *float64
}

func (a *ModelsDevAdapter) parse(body []byte) (*PricingResult, error) {
	var upstream map[string]mdProvider
	if err := common.Unmarshal(body, &upstream); err != nil {
		return nil, fmt.Errorf("models.dev: decode: %w", err)
	}
	providers := make([]string, 0, len(upstream))
	for p := range upstream {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	selected := make(map[string]mdPick)
	for _, prov := range providers {
		pd := upstream[prov]
		names := make([]string, 0, len(pd.Models))
		for n := range pd.Models {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, name := range names {
			cost := pd.Models[name].Cost
			if cost.Input == nil || math.IsNaN(*cost.Input) || math.IsInf(*cost.Input, 0) || *cost.Input < 0 {
				continue
			}
			pick := mdPick{Provider: prov, Input: *cost.Input}
			if cost.Output != nil && validFloat(*cost.Output) {
				v := *cost.Output
				pick.Output = &v
			}
			if cost.CacheRead != nil && validFloat(*cost.CacheRead) {
				v := *cost.CacheRead
				pick.CacheRead = &v
			}
			cur, ok := selected[name]
			if !ok || betterPick(cur, pick) {
				selected[name] = pick
			}
		}
	}

	result := &PricingResult{
		Provider:  a.Name(),
		Models:    make(map[string]ModelPricing),
		FetchedAt: time.Now(),
	}
	for name, pick := range selected {
		mp := ModelPricing{
			InputPrice:  pick.Input,
			Currency:    "USD",
			UnitType:    "per_1M_tokens",
		}
		if pick.Output != nil {
			mp.OutputPrice = *pick.Output
		}
		if pick.CacheRead != nil {
			v := *pick.CacheRead
			mp.CacheReadPrice = &v
		}
		result.Models[name] = mp
	}
	return result, nil
}

func validFloat(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

func betterPick(cur, next mdPick) bool {
	curNZ := cur.Input > 0
	nextNZ := next.Input > 0
	if curNZ != nextNZ {
		return nextNZ
	}
	if nextNZ && !nearlyEq(next.Input, cur.Input) {
		return next.Input < cur.Input
	}
	return next.Provider < cur.Provider
}

func nearlyEq(a, b float64) bool {
	const eps = 1e-9
	if a > b {
		return a-b < eps
	}
	return b-a < eps
}
