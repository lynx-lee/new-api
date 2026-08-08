package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/model"
	"github.com/QuantumNous/ai-bridge/relay/pricing"
	"github.com/QuantumNous/ai-bridge/setting/ratio_setting"
)

var (
	adapters   []pricing.Adapter
	adaptersMu sync.RWMutex

	providerPriceCache     = make(map[string]model.ProviderPricing)
	providerPriceCacheLock sync.RWMutex
)

func init() {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()
	adapters = []pricing.Adapter{
		pricing.NewModelsDevAdapter(),
		pricing.NewClaudeAdapter(),
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		adapters = append(adapters, pricing.NewOpenRouterAdapter(key))
	}
}

// RegisterAdapter adds an adapter to the sync pool. Thread-safe.
func RegisterAdapter(a pricing.Adapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()
	adapters = append(adapters, a)
}

func getAdapters() []pricing.Adapter {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()
	return adapters
}

// SyncAllProviders fetches pricing from all registered adapters and stores pending diffs.
func SyncAllProviders(ctx context.Context) (int, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	var total int
	var errs []error

	for _, a := range getAdapters() {
		wg.Add(1)
		go func(adapter pricing.Adapter) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("%s: panic: %v", adapter.Name(), r))
					mu.Unlock()
					common.SysLog(fmt.Sprintf("pricing_sync: %s panic recovered: %v", adapter.Name(), r))
				}
			}()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := adapter.Fetch(ctx)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", adapter.Name(), err))
				mu.Unlock()
				common.SysLog(fmt.Sprintf("pricing_sync: %s fetch error: %v", adapter.Name(), err))
				return
			}

			stored, err := storeDiffs(adapter.Name(), result)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: store: %w", adapter.Name(), err))
				mu.Unlock()
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
	return total, errors.Join(errs...)
}

func storeDiffs(provider string, result *pricing.PricingResult) (int, error) {
	stored := 0
	var errs []error
	for modelName, mp := range result.Models {
		modelRatio, compRatio, cacheRatio := pricing.ConvertToRatio(mp)
		currentRatio, _, _ := ratio_setting.GetModelRatio(modelName)
		currentCompRatio := ratio_setting.GetCompletionRatio(modelName)
		currentCacheRatio, _ := ratio_setting.GetCacheRatio(modelName)

		// Only consider completion ratio changed if provider actually supplied an output price
		compChanged := mp.OutputPrice > 0 && !nearEq(compRatio, currentCompRatio)
		cacheChanged := cacheRatio != nil && !nearEq(*cacheRatio, currentCacheRatio)

		if nearEq(modelRatio, currentRatio) && !compChanged && !cacheChanged {
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
			updateErr := model.DB.Model(&existing).Updates(map[string]interface{}{
				"raw_input_price":        mp.InputPrice,
				"raw_output_price":       mp.OutputPrice,
				"raw_cache_read_price":   mp.CacheReadPrice,
				"calculated_ratio":       modelRatio,
				"calculated_comp_ratio":  compRatio,
				"calculated_cache_ratio": cacheRatio,
				"fetched_at":             result.FetchedAt,
			}).Error
			if updateErr != nil {
				errs = append(errs, fmt.Errorf("update %s/%s: %w", provider, modelName, updateErr))
				continue
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			createErr := model.DB.Create(&entry).Error
			if createErr != nil {
				errs = append(errs, fmt.Errorf("create %s/%s: %w", provider, modelName, createErr))
				continue
			}
		} else {
			errs = append(errs, fmt.Errorf("query %s/%s: %w", provider, modelName, err))
			continue
		}
		stored++
	}
	return stored, errors.Join(errs...)
}

// GetPricingDiffs returns all pending pricing entries.
func GetPricingDiffs() ([]model.ProviderPricing, error) {
	var diffs []model.ProviderPricing
	err := model.DB.Where("status = ?", "pending").
		Order("model_name ASC, provider ASC").Find(&diffs).Error
	return diffs, err
}

// ApplyPricing applies selected pending entries: updates ratio maps, marks applied.
// Updates are batched: one map mutation per ratio type to avoid N-round-trip races.
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

	// Build batched map updates — one copy per ratio type, apply once
	modelRatioMap := ratio_setting.GetModelRatioCopy()
	compRatioMap := ratio_setting.GetCompletionRatioCopy()
	cacheRatioMap := ratio_setting.GetCacheRatioCopy()

	modelUpdated, compUpdated, cacheUpdated := false, false, false

	for _, e := range entries {
		// Normalize model name before inserting into ratio maps
		normalizedName := ratio_setting.FormatMatchingModelName(e.ModelName)

		modelRatioMap[normalizedName] = e.CalculatedRatio
		modelUpdated = true

		// Only update completion ratio if provider actually supplied an output price
		if e.RawOutputPrice > 0 && e.CalculatedCompRatio != 0 {
			compRatioMap[normalizedName] = e.CalculatedCompRatio
			compUpdated = true
		}

		// Update cache ratio if provided
		if e.CalculatedCacheRatio != nil && *e.CalculatedCacheRatio > 0 {
			cacheRatioMap[normalizedName] = *e.CalculatedCacheRatio
			cacheUpdated = true
		}
	}

	// Apply batched updates
	if modelUpdated {
		rJSON, err := common.Marshal(modelRatioMap)
		if err != nil {
			return fmt.Errorf("marshal model ratio map: %w", err)
		}
		if err := ratio_setting.UpdateModelRatioByJSONString(string(rJSON)); err != nil {
			return fmt.Errorf("update model ratio: %w", err)
		}
	}
	if compUpdated {
		cJSON, err := common.Marshal(compRatioMap)
		if err != nil {
			return fmt.Errorf("marshal completion ratio map: %w", err)
		}
		if err := ratio_setting.UpdateCompletionRatioByJSONString(string(cJSON)); err != nil {
			return fmt.Errorf("update completion ratio: %w", err)
		}
	}
	if cacheUpdated {
		crJSON, err := common.Marshal(cacheRatioMap)
		if err != nil {
			return fmt.Errorf("marshal cache ratio map: %w", err)
		}
		if err := ratio_setting.UpdateCacheRatioByJSONString(string(crJSON)); err != nil {
			return fmt.Errorf("update cache ratio: %w", err)
		}
	}

	// Mark entries as applied
	now := time.Now()
	if err := model.DB.Model(&model.ProviderPricing{}).
		Where("id IN ? AND status = ?", ids, "pending").
		Updates(map[string]interface{}{"status": "applied", "applied_at": &now}).Error; err != nil {
		return fmt.Errorf("mark applied: %w", err)
	}

	// Refresh provider price cache so cost display uses current pricing
	_ = LoadProviderPriceCache()

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

// StartScheduler runs SyncAllProviders periodically. Respects context cancellation.
func StartScheduler(ctx context.Context, interval time.Duration) {
	// Respect context cancellation during initial delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	common.SysLog("pricing_sync: initial sync")
	n, err := SyncAllProviders(ctx)
	if err != nil {
		common.SysLog(fmt.Sprintf("pricing_sync: initial sync err: %v", err))
	} else {
		common.SysLog(fmt.Sprintf("pricing_sync: initial sync done (%d entries)", n))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

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
		if existing, ok := providerPriceCache[e.ModelName]; !ok ||
			existing.AppliedAt == nil ||
			(e.AppliedAt != nil && e.AppliedAt.After(*existing.AppliedAt)) {
			providerPriceCache[e.ModelName] = e
		}
	}
	providerPriceCacheLock.Unlock()
	return nil
}
