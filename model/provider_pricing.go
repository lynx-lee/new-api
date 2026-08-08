package model

import "time"

// ProviderPricing stores synced pricing data from provider APIs.
// Status values: "pending" (unreviewed), "applied" (live), "rejected" (dismissed).
type ProviderPricing struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	Provider             string     `gorm:"index;size:64" json:"provider"`
	ModelName            string     `gorm:"index;size:256" json:"model_name"`
	RawInputPrice        float64    `json:"raw_input_price"`
	RawOutputPrice       float64    `json:"raw_output_price"`
	RawCacheReadPrice    *float64   `json:"raw_cache_read_price,omitempty"`
	Currency             string     `gorm:"size:8;default:USD" json:"currency"`
	UnitType             string     `gorm:"size:32;default:per_1M_tokens" json:"unit_type"`
	CalculatedRatio      float64    `json:"calculated_ratio"`
	CalculatedCompRatio  float64    `json:"calculated_comp_ratio"`
	CalculatedCacheRatio *float64   `json:"calculated_cache_ratio,omitempty"`
	FetchedAt            time.Time  `gorm:"index" json:"fetched_at"`
	AppliedAt            *time.Time `json:"applied_at,omitempty"`
	Status               string     `gorm:"index;size:16;default:pending" json:"status"`
	Version              string     `gorm:"size:64" json:"version,omitempty"`
}

func (ProviderPricing) TableName() string {
	return "provider_pricings"
}
