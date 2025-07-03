package limit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	limiterMap = new(sync.Map)
	// 定期清理过期的限流器
	cleanupTicker = time.NewTicker(5 * time.Minute)
)

type timerRate struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	ttl      time.Duration
}

// 启动清理协程
func init() {
	go func() {
		for range cleanupTicker.C {
			cleanupExpiredLimiters()
		}
	}()
}

// cleanupExpiredLimiters 清理过期的限流器实例
func cleanupExpiredLimiters() {
	now := time.Now()
	limiterMap.Range(func(key, value interface{}) bool {
		if tr, ok := value.(*timerRate); ok {
			// 如果超过TTL时间未使用，则删除
			if now.Sub(tr.lastSeen) > tr.ttl {
				limiterMap.Delete(key)
			}
		}
		return true
	})
}

// DefaultRateFn 优化版本，支持TTL清理
func DefaultRateFn(ctx context.Context, key string, limit int, period time.Duration,
	n int) (remaining int, reset time.Duration, allowed bool) {

	// 设置默认TTL为1小时
	ttl := time.Hour
	if period > time.Hour {
		ttl = period * 2
	}

	lm := &timerRate{
		limiter:  rate.NewLimiter(rate.Every(period), limit),
		lastSeen: time.Now(),
		ttl:      ttl,
	}

	v, loaded := limiterMap.LoadOrStore(key, lm)
	if loaded {
		lm = v.(*timerRate)
		lm.lastSeen = time.Now()
	}

	return int(lm.limiter.Tokens()),
		period, lm.limiter.AllowN(lm.lastSeen, n)
}

// GetLimiterStats 获取限流器统计信息（用于监控）
func GetLimiterStats() (count int) {
	limiterMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Cleanup 手动清理所有限流器（用于测试或关闭时）
func Cleanup() {
	limiterMap.Range(func(key, value interface{}) bool {
		limiterMap.Delete(key)
		return true
	})
	if cleanupTicker != nil {
		cleanupTicker.Stop()
	}
}
