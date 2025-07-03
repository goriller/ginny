package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrCacheClosed = errors.New("cache is closed")
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
	Clear(ctx context.Context) error
	Stats() CacheStats
	Close() error
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Sets        int64   `json:"sets"`
	Deletes     int64   `json:"deletes"`
	Errors      int64   `json:"errors"`
	HitRatio    float64 `json:"hit_ratio"`
	TotalKeys   int     `json:"total_keys"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
}

// cacheItem 缓存项
type cacheItem struct {
	Value     interface{} `json:"value"`
	ExpiredAt time.Time   `json:"expired_at"`
	CreatedAt time.Time   `json:"created_at"`
	Size      int64       `json:"size"`
}

// isExpired 检查是否过期
func (item *cacheItem) isExpired() bool {
	return !item.ExpiredAt.IsZero() && time.Now().After(item.ExpiredAt)
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	stats   CacheStats
	closed  bool
	maxSize int64 // 最大内存使用量（字节）

	// 清理相关
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// MemoryCacheConfig 内存缓存配置
type MemoryCacheConfig struct {
	MaxSize         int64         // 最大内存使用量（字节）
	CleanupInterval time.Duration // 清理间隔
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(config MemoryCacheConfig) *MemoryCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 100 * 1024 * 1024 // 默认100MB
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	cache := &MemoryCache{
		items:           make(map[string]*cacheItem),
		maxSize:         config.MaxSize,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// 启动清理协程
	go cache.cleaner()

	return cache
}

// Get 获取缓存值
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrCacheClosed
	}

	item, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		return ErrKeyNotFound
	}

	if item.isExpired() {
		c.stats.Misses++
		// 延迟删除，避免在读锁中修改
		go func() {
			c.Delete(ctx, key)
		}()
		return ErrKeyNotFound
	}

	c.stats.Hits++

	// 反序列化到目标对象
	if err := c.deserialize(item.Value, dest); err != nil {
		c.stats.Errors++
		return err
	}

	return nil
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	// 序列化值
	serializedValue, size, err := c.serialize(value)
	if err != nil {
		c.stats.Errors++
		return err
	}

	// 检查内存限制
	if c.stats.MemoryUsage+size > c.maxSize {
		// 执行LRU淘汰
		c.evictLRU(size)
	}

	var expiredAt time.Time
	if ttl > 0 {
		expiredAt = time.Now().Add(ttl)
	}

	// 如果key已存在，先减去旧的大小
	if oldItem, exists := c.items[key]; exists {
		c.stats.MemoryUsage -= oldItem.Size
	}

	item := &cacheItem{
		Value:     serializedValue,
		ExpiredAt: expiredAt,
		CreatedAt: time.Now(),
		Size:      size,
	}

	c.items[key] = item
	c.stats.MemoryUsage += size
	c.stats.Sets++
	c.stats.TotalKeys = len(c.items)

	return nil
}

// Delete 删除缓存值
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	if item, exists := c.items[key]; exists {
		delete(c.items, key)
		c.stats.MemoryUsage -= item.Size
		c.stats.Deletes++
		c.stats.TotalKeys = len(c.items)
	}

	return nil
}

// Exists 检查key是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false
	}

	item, exists := c.items[key]
	if !exists {
		return false
	}

	return !item.isExpired()
}

// Clear 清空缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	c.items = make(map[string]*cacheItem)
	c.stats.MemoryUsage = 0
	c.stats.TotalKeys = 0

	return nil
}

// Stats 获取统计信息
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}
	stats.TotalKeys = len(c.items)

	return stats
}

// Close 关闭缓存
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.stopCleanup)
	c.items = nil

	return nil
}

// serialize 序列化值
func (c *MemoryCache) serialize(value interface{}) (interface{}, int64, error) {
	// 对于简单类型，直接存储
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		return v, int64(64), nil // 估算大小
	default:
		// 复杂类型使用JSON序列化
		data, err := json.Marshal(value)
		if err != nil {
			return nil, 0, err
		}
		return string(data), int64(len(data)), nil
	}
}

// deserialize 反序列化值
func (c *MemoryCache) deserialize(value interface{}, dest interface{}) error {
	switch v := value.(type) {
	case string:
		// 尝试JSON反序列化
		return json.Unmarshal([]byte(v), dest)
	default:
		// 直接赋值
		return json.Unmarshal([]byte(`"`+value.(string)+`"`), dest)
	}
}

// evictLRU LRU淘汰策略
func (c *MemoryCache) evictLRU(neededSize int64) {
	// 简单的LRU实现：找到最老的项目并删除
	for c.stats.MemoryUsage+neededSize > c.maxSize && len(c.items) > 0 {
		var oldestKey string
		var oldestTime time.Time

		for key, item := range c.items {
			if oldestTime.IsZero() || item.CreatedAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = item.CreatedAt
			}
		}

		if oldestKey != "" {
			if item := c.items[oldestKey]; item != nil {
				delete(c.items, oldestKey)
				c.stats.MemoryUsage -= item.Size
			}
		}
	}
}

// cleaner 清理过期项目
func (c *MemoryCache) cleaner() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanupExpired 清理过期项目
func (c *MemoryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	var expiredKeys []string
	for key, item := range c.items {
		if item.isExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		if item := c.items[key]; item != nil {
			delete(c.items, key)
			c.stats.MemoryUsage -= item.Size
		}
	}

	c.stats.TotalKeys = len(c.items)
}
