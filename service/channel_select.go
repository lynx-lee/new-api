package service

import (
	"errors"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/constant"
	"github.com/QuantumNous/ai-bridge/logger"
	"github.com/QuantumNous/ai-bridge/model"
	"github.com/QuantumNous/ai-bridge/pkg/canary"
	"github.com/QuantumNous/ai-bridge/setting"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)

	if param.TokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			channel, _ = model.GetRandomSatisfiedChannel(autoGroup, param.ModelName, priorityRetry)
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group", autoGroup, param.ModelName, priorityRetry)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, param.GetRetry())
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}

	// Apply canary release routing if enabled and multiple channels exist
	if channel != nil && common.CanaryEnabled {
		if selected, tag := applyCanarySelection(param.Ctx, channel, selectGroup, param.ModelName, userGroup); selected != nil {
			channel = selected
			if len(tag) > 0 {
				param.Ctx.Set(canary.CanaryTagKey, tag)
			}
		}
	}

	return channel, selectGroup, nil
}

// applyCanarySelection checks if canary rules override the selected channel.
// Returns (possibly replaced channel, canary tag). Returns original if no canary rule matches.
func applyCanarySelection(c *gin.Context, selected *model.Channel, groupName, modelName, userGroup string) (*model.Channel, string) {
	mgr := canary.GetManager()
	if mgr == nil {
		return selected, ""
	}
	config := mgr.GetConfig(modelName, userGroup)
	if config == nil || !config.Enabled {
		return selected, ""
	}
	// Get all candidate IDs for this group+model
	allIDs := model.GetAllSatisfiedChannelIDs(groupName, modelName)
	if len(allIDs) <= 1 {
		return selected, "" // no point filtering single channel
	}
	userTags := GetCanaryUserTags(c)
	filteredIDs, tag := mgr.SelectChannel(modelName, userGroup, userTags, allIDs)
	if filteredIDs == nil {
		return selected, "" // no canary rule matched, keep original selection
	}
	// Build allowed ID set for quick lookup
	allowed := make(map[int]bool)
	for _, id := range filteredIDs {
		allowed[id] = true
	}
	// If already in allowed set, keep it
	if allowed[selected.Id] {
		return selected, tag
	}
	// Otherwise pick a random channel from the canary-filtered set
	for _, id := range filteredIDs {
		ch, err := model.CacheGetChannel(id)
		if err == nil && ch != nil && ch.Status == common.ChannelStatusEnabled {
			logger.LogDebug(c, "canary routing: replaced channel %d -> %d (tag=%s)", selected.Id, ch.Id, tag)
			return ch, tag
		}
	}
	return selected, "" // fallback if all filtered channels unavailable
}

// GetCanaryUserTags extracts user tags from gin context for canary matching.
// Tags come from custom headers or user attributes.
func GetCanaryUserTags(c *gin.Context) []string {
	var tags []string
	if tag := c.GetHeader("X-Canary-Tag"); tag != "" {
		tags = append(tags, tag)
	}
	if tag := c.GetHeader("X-Beta-User"); tag != "" {
		tags = append(tags, tag)
	}
	// Add user group as implicit tag
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup != "" && userGroup != "default" && userGroup != "auto" {
		tags = append(tags, "group:"+userGroup)
	}
	return tags
}
