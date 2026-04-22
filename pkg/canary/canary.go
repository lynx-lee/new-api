// Package canary provides canary/gray-release routing for AI model channels.
// It supports weight-based traffic splitting and tag-based user targeting
// to safely roll out new channels or models.
package canary

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/model"
)

const (
	// CanaryTagKey is the context key for storing matched canary group.
	CanaryTagKey = "canary_tag"
)

// Strategy defines how canary routing distributes traffic across channel groups.
type Strategy int

const (
	StrategyDefault  Strategy = iota // No canary, use existing selection logic
	StrategyWeighted                 // Weight-based percentage split
	StrategyTagged                   // Tag-based user/group routing
)

func (s Strategy) String() string {
	switch s {
	case StrategyWeighted:
		return "weighted"
	case StrategyTagged:
		return "tagged"
	default:
		return "default"
	}
}

// CanaryGroup represents a set of channels that receive canary traffic.
type CanaryGroup struct {
	Name        string   `json:"name"`         // e.g. "canary-v2", "stable"
	Weight      int      `json:"weight"`       // percentage 0-100 (for weighted strategy)
	Tags        []string `json:"tags"`         // user tags that match this group (for tagged strategy)
	ChannelIDs  []int    `json:"channel_ids"`   // specific channel IDs in this group
	ModelFilter string   `json:"model_filter"` // optional: only apply to models matching prefix
	Enabled     bool     `json:"enabled"`
}

// CanaryConfig holds canary settings for a specific model or group.
type CanaryConfig struct {
	ModelName    string        `json:"model_name"`    // target model (or "*" for all)
	UserGroup    string        `json:"user_group"`   // which user group this applies to ("*" for all)
	Strategy    Strategy      `json:"strategy"`
	Groups      []CanaryGroup `json:"groups"`
	Enabled     bool          `json:"enabled"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Manager manages canary configurations and routing decisions.
type Manager struct {
	configs sync.Map // map[string]*CanaryConfig, key = configKey(modelName:userGroup)
	mu      sync.RWMutex
	randSrc *rand.Rand
}

var globalManager *Manager
var managerOnce sync.Once

// InitManager initializes the global canary manager.
func InitManager() {
	managerOnce.Do(func() {
		globalManager = &Manager{
			randSrc: rand.New(rand.NewSource(time.Now().UnixNano())),
		}
		common.SysLog("canary release manager initialized")
	})
}

// GetManager returns the global canary manager instance.
func GetManager() *Manager {
	return globalManager
}

// SetConfig registers a new canary configuration.
func (m *Manager) SetConfig(config CanaryConfig) {
	key := m.configKey(config.ModelName, config.UserGroup)
	m.configs.Store(key, &config)
}

// RemoveConfig removes a canary configuration by model+group key.
func (m *Manager) RemoveConfig(modelName, userGroup string) {
	key := m.configKey(modelName, userGroup)
	m.configs.Delete(key)
}

// GetConfig retrieves the canary config for a given model and user group.
func (m *Manager) GetConfig(modelName, userGroup string) *CanaryConfig {
	if !common.CanaryEnabled {
		return nil
	}

	// Exact match first
	if v, ok := m.configs.Load(m.configKey(modelName, userGroup)); ok {
		c := v.(*CanaryConfig)
		if c.Enabled {
			return c
		}
	}
	// Wildcard group match
	if v, ok := m.configs.Load(m.configKey(modelName, "*")); ok {
		c := v.(*CanaryConfig)
		if c.Enabled {
			return c
		}
	}
	// Wildcard model match
	if v, ok := m.configs.Load(m.configKey("*", userGroup)); ok {
		c := v.(*CanaryConfig)
		if c.Enabled {
			return c
		}
	}
	return nil
}

// SelectChannel applies canary logic to select from candidate channels.
// Returns (selected channel IDs, canary tag name, or nil if no canary rule matches).
//
// Integration point: call this in middleware/distributor.go after getting candidate
// channels but before final random selection. If non-nil result is returned,
// use those channel IDs instead of the full candidate list.
func (m *Manager) SelectChannel(modelName, userGroup string, userTags []string, allCandidates []int) ([]int, string) {
	if !common.CanaryEnabled || len(allCandidates) == 0 {
		return nil, ""
	}

	config := m.GetConfig(modelName, userGroup)
	if config == nil {
		return nil, ""
	}

	switch config.Strategy {
	case StrategyWeighted:
		return m.selectByWeight(config, allCandidates)
	case StrategyTagged:
		return m.selectByTag(config, userTags, allCandidates)
	default:
		return nil, ""
	}
}

// selectByWeight routes traffic based on configured weights per group.
func (m *Manager) selectByWeight(config *CanaryConfig, candidates []int) ([]int, string) {
	var totalWeight int
	for _, g := range config.Groups {
		if !g.Enabled {
			continue
		}
		totalWeight += g.Weight
	}
	if totalWeight == 0 {
		return nil, ""
	}

	roll := m.randSrc.Intn(totalWeight)
	var cumulative int
	for _, group := range config.Groups {
		if !group.Enabled {
			continue
		}
		cumulative += group.Weight
		if roll < cumulative {
			channels := m.resolveChannels(group, candidates)
			if len(channels) > 0 {
				return channels, group.Name
			}
			break
		}
	}
	return nil, ""
}

// selectByTag routes traffic based on user tag matching.
func (m *Manager) selectByTag(config *CanaryConfig, userTags []string, candidates []int) ([]int, string) {
	tagSet := make(map[string]bool)
	for _, t := range userTags {
		tagSet[t] = true
	}

	// Sort groups by priority: more specific tags first
	groups := make([]CanaryGroup, len(config.Groups))
	copy(groups, config.Groups)
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i].Tags) > len(groups[j].Tags)
	})

	for _, group := range groups {
		if !group.Enabled {
			continue
		}
		// Check if any of the group's tags match the user's tags
		for _, gt := range group.Tags {
			if tagSet[gt] {
				channels := m.resolveChannels(group, candidates)
				if len(channels) > 0 {
					return channels, group.Name
				}
				break
			}
		}
	}

	return nil, ""
}

// resolveChannels maps a canary group's channel_ids to actual available candidates.
// Falls back to candidates if no specific channel_ids are set (uses all).
func (m *Manager) resolveChannels(group CanaryGroup, candidates []int) []int {
	if len(group.ChannelIDs) > 0 {
		result := make([]int, 0, len(group.ChannelIDs))
		candidateSet := make(map[int]bool)
		for _, cid := range candidates {
			candidateSet[cid] = true
		}
		for _, cid := range group.ChannelIDs {
			if candidateSet[cid] {
				result = append(result, cid)
			}
		}
		if len(result) > 0 {
			return result
		}
		// If none of specified IDs are in candidates, return empty (don't fall through)
		return nil
	}
	// No filter: use all candidates as-is
	return candidates
}

// GetAllConfigs returns all registered canary configurations.
func (m *Manager) GetAllConfigs() []*CanaryConfig {
	var result []*CanaryConfig
	m.configs.Range(func(key, value any) bool {
		result = append(result, value.(*CanaryConfig))
		return true
	})
	return result
}

func (m *Manager) configKey(modelName, userGroup string) string {
	return fmt.Sprintf("%s:%s", modelName, userGroup)
}

// IsCanaryChannel checks whether a channel ID belongs to any canary group.
func IsCanaryChannel(channelId int, modelName, userGroup string) bool {
	if !common.CanaryEnabled || globalManager == nil {
		return false
	}
	config := globalManager.GetConfig(modelName, userGroup)
	if config == nil {
		return false
	}
	for _, group := range config.Groups {
		if !group.Enabled {
			continue
		}
		for _, cid := range group.ChannelIDs {
			if cid == channelId {
				return true
			}
		}
	}
	return false
}

// ApplyCanaryRouting is the main integration function called from the distributor.
// It takes the full list of candidate channels and returns a potentially filtered subset.
// The second return value is the canary tag (empty string if not applied).
func ApplyCanaryRouting(modelName, userGroup string, userTags []string, candidates []model.Channel) ([]model.Channel, string) {
	if !common.CanaryEnabled || globalManager == nil || len(candidates) <= 1 {
		return candidates, ""
	}

	ids := make([]int, len(candidates))
	for i, ch := range candidates {
		ids[i] = ch.Id
	}

	selectedIDs, tagName := globalManager.SelectChannel(modelName, userGroup, userTags, ids)
	if selectedIDs == nil {
		return candidates, ""
	}

	selectedSet := make(map[int]bool)
	for _, id := range selectedIDs {
		selectedSet[id] = true
	}

	var result []model.Channel
	for _, ch := range candidates {
		if selectedSet[ch.Id] {
			result = append(result, ch)
		}
	}

	if len(result) == 0 {
		return candidates, "" // fallback to all if canary resolves empty
	}
	return result, tagName
}
