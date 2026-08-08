package pricing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/ai-bridge/common"
)

// OpenRouterAdapter fetches pricing from OpenRouter's /v1/models endpoint.
type OpenRouterAdapter struct {
	APIKey string
	client *http.Client
}

func NewOpenRouterAdapter(apiKey string) *OpenRouterAdapter {
	return &OpenRouterAdapter{
		APIKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *OpenRouterAdapter) Name() string {
	return "openrouter"
}

func (a *OpenRouterAdapter) Fetch(ctx context.Context) (*PricingResult, error) {
	url := "https://openrouter.ai/api/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("openrouter: build request: %w", err)
	}
	if a.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("openrouter: read body: %w", err)
	}

	return a.parse(bodyBytes)
}

// OpenRouter returns pricing in USD per token — convert to per 1M tokens for ModelPricing.
type orResp struct {
	Data []struct {
		ID      string `json:"id"`
		Pricing struct {
			Prompt         string `json:"prompt"`
			Completion     string `json:"completion"`
			InputCacheRead string `json:"input_cache_read"`
		} `json:"pricing"`
	} `json:"data"`
}

func (a *OpenRouterAdapter) parse(body []byte) (*PricingResult, error) {
	var or orResp
	if err := common.Unmarshal(body, &or); err != nil {
		return nil, fmt.Errorf("openrouter: decode: %w", err)
	}

	result := &PricingResult{
		Provider:  a.Name(),
		Models:    make(map[string]ModelPricing),
		FetchedAt: time.Now(),
	}

	for _, m := range or.Data {
		promptPrice, pErr := parseFloat(m.Pricing.Prompt)
		completionPrice, cErr := parseFloat(m.Pricing.Completion)

		if pErr != nil && cErr != nil {
			continue
		}
		if promptPrice < 0 || completionPrice < 0 {
			continue
		}
		if promptPrice == 0 && completionPrice == 0 {
			result.Models[m.ID] = ModelPricing{Currency: "USD", UnitType: "per_1M_tokens"}
			continue
		}
		if promptPrice <= 0 {
			continue
		}

		mp := ModelPricing{
			InputPrice:  promptPrice * 1_000_000, // per-token -> per-1M
			OutputPrice: completionPrice * 1_000_000,
			Currency:    "USD",
			UnitType:    "per_1M_tokens",
		}

		if cp, err := parseFloat(m.Pricing.InputCacheRead); err == nil && cp >= 0 {
			v := cp * 1_000_000
			mp.CacheReadPrice = &v
		}

		result.Models[m.ID] = mp
	}

	return result, nil
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
