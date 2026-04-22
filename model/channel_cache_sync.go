package model

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/ai-bridge/common"
	"github.com/QuantumNous/ai-bridge/metrics"

	"github.com/go-redis/redis/v8"
)

// ChannelCacheSyncPubSub 渠道缓存实时同步的 Redis Pub/Sub 实现。
// 解决 K8s 多 Pod 部署场景下，各节点间渠道缓存延迟不一致的问题。
//
// 架构：
//   - 任一 Pod 执行渠道写操作 → 发布消息到 Redis channel:aibridge-channel-sync
//   - 其他 Pod 订阅该 channel → 收到消息后立即刷新本地缓存
//   - 保留定时全量同步 (SyncChannelCache) 作为兜底

const (
	// ChannelCacheSyncTopic Redis Pub/Sub topic 名称
	ChannelCacheSyncTopic = "aibridge:channel-cache-sync"

	// ChannelCacheSyncAction 全量刷新（如新建/删除渠道，需要重建索引）
	ChannelCacheSyncActionFullRefresh = "full_refresh"

	// ChannelCacheSyncActionStatusUpdate 单条渠道状态变更（禁用/启用）
	ChannelCacheSyncActionStatusUpdate = "status_update"

	// ChannelCacheSyncActionInfoUpdate 单条渠道信息变更（key 轮换等）
	ChannelCacheSyncActionInfoUpdate = "info_update"

	// ChannelCacheSyncDebounceMinMs 最小防抖间隔（毫秒），防止短时间内大量消息导致频繁刷新
	ChannelCacheSyncDebounceMinMs = 500
)

// ChannelCacheSyncMessage 发布的同步消息结构
type ChannelCacheSyncMessage struct {
	Action    string `json:"action"`              // full_refresh, status_update, info_update
	ChannelID int    `json:"channel_id,omitempty"` // 单条更新时的渠道 ID
	NodeName string `json:"node_name,omitempty"`   // 发送方节点名，用于调试
}

var (
	channelSyncOnce      sync.Once
	channelSyncCtx       context.Context
	channelSyncCancel    context.CancelFunc
	lastSyncMsgTime      time.Time
	channelSyncMsgMu     sync.Mutex
)

// InitChannelCacheSync 初始化 Redis Pub/Sub 渠道缓存同步订阅者。
// 必须在 InitRedisClient() 之后调用。
func InitChannelCacheSync() {
	if !common.RedisEnabled || !common.MemoryCacheEnabled {
		common.SysLog("Redis or memory cache not enabled, skipping channel cache Pub/Sub sync")
		return
	}

	channelSyncOnce.Do(func() {
		channelSyncCtx, channelSyncCancel = context.WithCancel(context.Background())
		go subscribeChannelCacheSync(channelSyncCtx)
		common.SysLog("channel cache Pub/Sub sync initialized")
	})
}

// StopChannelCacheSync 停止 Pub/Sub 订阅
func StopChannelCacheSync() {
	if channelSyncCancel != nil {
		channelSyncCancel()
	}
}

// PublishChannelCacheChange 发布渠道变更通知到 Redis Pub/Sub。
// action: full_refresh / status_update / info_update
func PublishChannelCacheChange(action string, channelID int) {
	if !common.RedisEnabled || !common.MemoryCacheEnabled {
		return
	}

	msg := ChannelCacheSyncMessage{
		Action:    action,
		ChannelID: channelID,
		NodeName:  common.NodeName,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to marshal channel cache sync message: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = common.RDB.Publish(ctx, ChannelCacheSyncTopic, data).Err()
	if err != nil {
		common.SysError(fmt.Sprintf("failed to publish channel cache sync message: %v", err))
		return
	}

	if common.DebugEnabled {
		common.SysLog(fmt.Sprintf("published channel cache sync: action=%s, channelID=%d", action, channelID))
	}
}

// subscribeChannelCacheSync 启动后台协程订阅 Redis channel 变更通知
func subscribeChannelCacheSync(ctx context.Context) {
	pubsub := common.RDB.Subscribe(ctx, ChannelCacheSyncTopic)
	defer func() {
		_ = pubsub.Close()
	}()

	ch := pubsub.Channel(redis.WithChannelSize(256))

	for {
		select {
		case <-ctx.Done():
			common.SysLog("channel cache Pub/Sub subscriber stopped")
			return
		case msg, ok := <-ch:
			if !ok || msg == nil {
				// 连接断开，重连
				common.SysError("channel cache Pub/Sub channel closed, reconnecting...")
				time.Sleep(1 * time.Second)
				newPubsub := common.RDB.Subscribe(ctx, ChannelCacheSyncTopic)
				if newPubsub != nil {
					_ = pubsub.Close()
					pubsub = newPubsub
					ch = pubsub.Channel(redis.WithChannelSize(256))
				}
				continue
			}
			handleChannelCacheSyncMessage(msg.Payload)
		}
	}
}

// handleChannelCacheSyncMessage 处理收到的渠道缓存同步消息
func handleChannelCacheSyncMessage(payload string) {
	var msg ChannelCacheSyncMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		common.SysError(fmt.Sprintf("failed to unmarshal channel cache sync message: %v, payload=%s", err, payload))
		return
	}

	// 忽略自己发出的消息（避免循环刷新）
	if msg.NodeName == common.NodeName && msg.NodeName != "" {
		if common.DebugEnabled {
			common.SysLog(fmt.Sprintf("ignored self-originated channel cache sync: action=%s", msg.Action))
		}
		return
	}

	// 防抖：短时间内多条合并为一次全量刷新
	now := time.Now()
	shouldRefresh := false

	channelSyncMsgMu.Lock()
	elapsed := now.Sub(lastSyncMsgTime)
	if elapsed < time.Duration(ChannelCacheSyncDebounceMinMs)*time.Millisecond {
		// 短时间内已有消息触发过，跳过本次（下一次定时同步会兜底）
		channelSyncMsgMu.Unlock()
		return
	}
	lastSyncMsgTime = now
	channelSyncMsgMu.Unlock()

	switch msg.Action {
	case ChannelCacheSyncActionFullRefresh:
		shouldRefresh = true
	case ChannelCacheSyncActionStatusUpdate:
		// 对于单条状态更新，可以只更新该条记录而不做全量刷新
		// 但为保证 group2model2channels 一致性，仍做全量刷新
		shouldRefresh = true
	case ChannelCacheSyncActionInfoUpdate:
		shouldRefresh = true
	default:
		common.SysError(fmt.Sprintf("unknown channel cache sync action: %s", msg.Action))
		return
	}

	if shouldRefresh {
		common.SysLog(fmt.Sprintf("received channel cache sync from node '%s', action=%s, refreshing...", msg.NodeName, msg.Action))
		metrics.PubSubMessagesReceived.WithLabelValues(msg.Action).Inc()
		InitChannelCache()
	}
}

// ---- 增强的缓存更新方法（带 Pub/Sub 发布） ----

// CacheUpdateChannelStatusWithSync 更新渠道状态并广播变更（替代 CacheUpdateChannelStatus）
func CacheUpdateChannelStatusWithSync(id int, status int) {
	CacheUpdateChannelStatus(id, status)
	PublishChannelCacheChange(ChannelCacheSyncActionStatusUpdate, id)
}

// CacheUpdateChannelWithSync 更新渠道信息并广播变更（替代 CacheUpdateChannel）
func CacheUpdateChannelWithSync(channel *Channel) {
	CacheUpdateChannel(channel)
	if channel != nil {
		PublishChannelCacheChange(ChannelCacheSyncActionInfoUpdate, channel.Id)
	}
}

// PublishFullCacheRefresh 发布全量刷新通知（如批量导入渠道、数据库恢复后）
func PublishFullCacheRefresh() {
	PublishChannelCacheChange(ChannelCacheSyncActionFullRefresh, 0)
}
