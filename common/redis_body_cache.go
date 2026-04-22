package common

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisBodyCachePrefix Redis 中请求体缓存的 key 前缀
const RedisBodyCachePrefix = "aibridge:bodycache:"

// DefaultRedisBodyTTL Redis 缓存默认 TTL，5 分钟足够覆盖一个请求的生命周期
const DefaultRedisBodyTTL = 300 // seconds

// RedisBodyStorage 基于 Redis 的请求体存储实现，用于 K8s 集群部署场景。
// 消除了对本地文件系统的依赖，使多 Pod 共享缓存状态成为可能。
type RedisBodyStorage struct {
	key     string
	size    int64
	closed  int32
	mu      sync.Mutex
	client  *redis.Client
}

// NewRedisBodyStorage 创建基于 Redis 的请求体存储
func NewRedisBodyStorage(data []byte) (*RedisBodyStorage, error) {
	if !RedisEnabled {
		return nil, fmt.Errorf("Redis is not enabled, cannot use RedisBodyStorage")
	}
	key := generateRedisCacheKey()

	ctx := context.Background()
	err := RDB.Set(ctx, key, data, DefaultRedisBodyTTL).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to write to Redis: %w", err)
	}

	size := int64(len(data))
	IncrementMemoryBuffers(size) // 复用内存统计

	return &RedisBodyStorage{
		key:    key,
		size:   size,
		client: RDB,
	}, nil
}

// NewRedisBodyStorageFromReader 从 io.Reader 创建 Redis 存储（用于流式读取大请求体）
func NewRedisBodyStorageFromReader(reader io.Reader, maxBytes int64) (*RedisBodyStorage, error) {
	if !RedisEnabled {
		return nil, fmt.Errorf("Redis is not enabled, cannot use RedisBodyStorage")
	}

	data := make([]byte, maxBytes)
	n, err := io.ReadFull(io.LimitReader(reader, maxBytes+1), data)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	if int64(n) > maxBytes {
		return nil, ErrRequestBodyTooLarge
	}
	data = data[:n]

	return NewRedisBodyStorage(data)
}

func (r *RedisBodyStorage) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if atomic.LoadInt32(&r.closed) == 1 {
		return 0, ErrStorageClosed
	}

	ctx := context.Background()
	val, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("body cache expired or not found in Redis: key=%s", r.key)
		}
		return 0, fmt.Errorf("failed to read from Redis: %w", err)
	}

	copy(p, val)
	if len(val) > len(p) {
		return len(p), nil
	}
	return len(val), nil
}

func (r *RedisBodyStorage) Seek(offset int64, whence int) (int64, error) {
	// Redis string 不支持真正的 seek，但需要满足接口契约
	// 返回 offset 表示"已定位"，实际 Read 时会从开头读
	switch whence {
	case io.SeekStart:
		return offset, nil
	case io.SeekCurrent:
		return offset, nil
	case io.SeekEnd:
		return r.size + offset, nil
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
}

func (r *RedisBodyStorage) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
		ctx := context.Background()
		_ = r.client.Del(ctx, r.key) // 异步清理，不阻塞
		DecrementMemoryBuffers(r.size)
	}
	return nil
}

func (r *RedisBodyStorage) Bytes() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if atomic.LoadInt32(&r.closed) == 1 {
		return nil, ErrStorageClosed
	}

	ctx := context.Background()
	val, err := r.client.Get(ctx, r.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("body cache expired or not found in Redis")
		}
		return nil, fmt.Errorf("failed to get bytes from Redis: %w", err)
	}
	return val, nil
}

func (r *RedisBodyStorage) Size() int64 {
	return r.size
}

func (r *RedisBodyStorage) IsDisk() bool {
	return false
}

// Key 返回 Redis 中的缓存 key（用于调试）
func (r *RedisBodyStorage) Key() string {
	return r.key
}

// ---- 辅助函数 ----

var redisCacheCounter uint32

func generateRedisCacheKey() string {
	cnt := atomic.AddUint32(&redisCacheCounter, 1)
	return fmt.Sprintf("%s%d-%d", RedisBodyCachePrefix, cnt, time.Now().UnixNano())
}

// ShouldUseRedisBodyCache 判断是否应该使用 Redis 作为请求体存储后端
// 条件：Redis 已启用 && 配置允许使用 Redis 后端
func ShouldUseRedisBodyCache() bool {
	if !RedisEnabled {
		return false
	}
	return GetEnvOrDefaultBool("REDIS_BODY_CACHE_ENABLED", false)
}
