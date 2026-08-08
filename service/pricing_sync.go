package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/model"
	"github.com/QuantumNous/ai-bridge/relay/pricing"
	"github.com/QuantumNous/ai-bridge/setting/ratio_setting"
)

var adapters []pricing.Adapter

// providerPriceCache maps model_name to the latest applied raw pricing
var (
	providerPriceCache     = make(map[string]model.ProviderPricing)
	providerPriceCacheLock sync.RWMutex
)

func init() {
	adapters = []pricing.Adapter{
		pricing.NewModelsDevAdapter(),
	}
}

// RegisterAdapter adds an adapter to the sync pool.
func RegisterAdapter(a pricing.Adapter) {
	adapters = append(adapters, a)
}

// SyncAllProviders fetches pricing from all registered adapters and stores pending diffs.
func SyncAllProviders(ctx context.Context) (int, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	var total int
	var firstErr error

	for _, a := range adapters {
		wg.Add(1)
		go func(adapter pricing.Adapter) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := adapter.Fetch(ctx)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", adapter.Name(), err)
				}
				mu.Unlock()
				common.SysLog(fmt.Sprintf("pricing_sync: %s fetch error: %v", adapter.Name(), err))
				return
			}

			stored, err := storeDiffs(adapter.Name(), result)
			if err != nil {
				common.SysLog(fmt.Sprintf("pricing_sync: %s store error: %v", adapter.Name(), err))
				return
			}
			mu.Lock()
			total += stored
			mu.Unlock()
			common.SysLog(fmt.Sprintf("pricing_sync: %s %d pending diffs", adapter.Name(), stored))
		}(a)
	}
	wg.Wait()
	return total, firstErr
}

func storeDiffs(provider string, result *pricing.PricingResult) (int, error) {
	stored := 0
	for modelName, mp := range result.Models {
		modelRatio, compRatio, cacheRatio := pricing.ConvertToRatio(mp)
		currentRatio, _, _ := ratio_setting.GetModelRatio(modelName)
		currentCompRatio := ratio_setting.GetCompletionRatio(modelName)

		if nearEq(modelRatio, currentRatio) && nearEq(compRatio, currentCompRatio) {
			continue
		}

		entry := model.ProviderPricing{
			Provider:             provider,
			ModelName:            modelName,
			RawInputPrice:        mp.InputPrice,
			RawOutputPrice:       mp.OutputPrice,
			RawCacheReadPrice:    mp.CacheReadPrice,
			Currency:             mp.Currency,
			UnitType:             mp.UnitType,
			CalculatedRatio:      modelRatio,
			CalculatedCompRatio:  compRatio,
			CalculatedCacheRatio: cacheRatio,
			FetchedAt:            result.FetchedAt,
			Status:               "pending",
		}

		var existing model.ProviderPricing
		err := model.DB.Where("provider = ? AND model_name = ? AND status = ?",
			provider, modelName, "pending").Order("fetched_at DESC").First(&existing).Error
		if err == nil {
			model.DB.Model(&existing).Updates(map[string]interface{}{
				"raw_input_price":        mp.InputPrice,
				"raw_output_price":       mp.OutputPrice,
				"raw_cache_read_price":   mp.CacheReadPrice,
				"calculated_ratio":       modelRatio,
				"calculated_comp_ratio":  compRatio,
				"calculated_cache_ratio": cacheRatio,
				"fetched_at":             result.FetchedAt,
			})
		} else {
			model.DB.Create(&entry)
		}
		stored++
	}
	return stored, nil
}

// GetPricingDiffs returns all pending pricing entries.
func GetPricingDiffs() ([]model.ProviderPricing, error) {
	var diffs []model.ProviderPricing
	err := model.DB.Where("status = ?", "pending").
		Order("model_name ASC, provider ASC").Find(&diffs).Error
	return diffs, err
}

// ApplyPricing applies selected pending entries: updates ratio maps, marks applied.
func ApplyPricing(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var entries []model.ProviderPricing
	if err := model.DB.Where("id IN ? AND status = ?", ids, "pending").Find(&entries).Error; err != nil {
		return fmt.Errorf("query pending entries: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	for _, e := range entries {
		rMap := ratio_setting.GetModelRatioCopy()
		rMap[e.ModelName] = e.CalculatedRatio
		rJSON, _ := common.Marshal(rMap)
		if err := ratio_setting.UpdateModelRatioByJSONString(string(rJSON)); err != nil {
			common.SysLog(fmt.Sprintf("pricing_sync: update model_ratio %s: %v", e.ModelName, err))
			continue
		}
		if e.CalculatedCompRatio != 0 {
			cMap := ratio_setting.GetCompletionRatioCopy()
			cMap[e.ModelName] = e.CalculatedCompRatio
			cJSON, _ := common.Marshal(cMap)
			_ = ratio_setting.UpdateCompletionRatioByJSONString(string(cJSON))
		}
		now := time.Now()
		model.DB.Model(&e).Updates(map[string]interface{}{
			"status": "applied", "applied_at": &now,
		})
	}
	return nil
}

// RejectPricing marks selected entries as rejected.
func RejectPricing(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return model.DB.Where("id IN ? AND status = ?", ids, "pending").
		Updates(map[string]interface{}{"status": "rejected"}).Error
}

// StartScheduler runs SyncAllProviders periodically.
func StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	time.Sleep(30 * time.Second)
	common.SysLog("pricing_sync: initial sync")
	n, err := SyncAllProviders(ctx)
	if err != nil {
		common.SysLog(fmt.Sprintf("pricing_sync: initial sync err: %v", err))
	} else {
		common.SysLog(fmt.Sprintf("pricing_sync: initial sync done (%d entries)", n))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			common.SysLog("pricing_sync: scheduled sync")
			n, err := SyncAllProviders(ctx)
			if err != nil {
				common.SysLog(fmt.Sprintf("pricing_sync: sync err: %v", err))
			} else {
				common.SysLog(fmt.Sprintf("pricing_sync: sync done (%d entries)", n))
			}
		}
	}
}

func nearEq(a, b float64) bool {
	const eps = 1e-6
	if a > b {
		return a-b < eps
	}
	return b-a < eps
}

// LookupProviderPrice returns the raw provider pricing for a model, if known.
func LookupProviderPrice(modelName string) (*model.ProviderPricing, bool) {
	providerPriceCacheLock.RLock()
	p, ok := providerPriceCache[modelName]
	providerPriceCacheLock.RUnlock()
	if ok {
		return &p, true
	}
	return nil, false
}

// LoadProviderPriceCache loads applied pricing entries into memory.
func LoadProviderPriceCache() error {
	var entries []model.ProviderPricing
	if err := model.DB.Where("status = ?", "applied").Find(&entries).Error; err != nil {
		return err
	}
	providerPriceCacheLock.Lock()
	for _, e := range entries {
		// Keep the latest entry per model
		if existing, ok := providerPriceCache[e.ModelName]; !ok || e.AppliedAt.After(*existing.AppliedAt) {
			providerPriceCache[e.ModelName] = e
		}
	}
	providerPriceCacheLock.Unlock()
	return nil
}
