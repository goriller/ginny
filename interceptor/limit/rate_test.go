package limit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_DefaultRate(t *testing.T) {
	tests := []struct {
		ctx    context.Context
		key    string
		limit  int
		period time.Duration
		n      int
	}{
		// TODO: Add test cases.
		{
			ctx:    context.Background(),
			key:    "test1",
			limit:  100,
			period: time.Second * 1,
			n:      1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// x := 100
			for i := 0; i < 300; i++ {
				// if i%2 == 0 {
				// 	go func() {
				// 		remaining, reset, allowed := DefaultRate(tt.ctx, tt.key, tt.limit, tt.period, tt.n)
				// 		fmt.Printf("key: %v, limit: %v,remaining: %v, reset: %v, allowed: %v\n", tt.key, tt.limit, remaining, reset, allowed)
				// 	}()
				// } else {
				if i == 200 {
					time.Sleep(time.Second * 2)
				}
				remaining, reset, allowed := DefaultRateFn(tt.ctx, tt.key, tt.limit, tt.period, tt.n)
				fmt.Printf("i: %v, limit: %v,remaining: %v, reset: %v, allowed: %v\n", i, tt.limit, remaining, reset, allowed)
				// if !allowed {
				// 	time.Sleep(reset)
				// }
				// }
			}

		})
	}
}

// 测试TTL清理功能
func TestLimiterCleanup(t *testing.T) {
	// 清理现有的限流器
	Cleanup()

	ctx := context.Background()
	key := "test-cleanup"
	limit := 10
	period := time.Millisecond * 100

	// 创建一个短TTL的限流器
	_, _, _ = DefaultRateFn(ctx, key, limit, period, 1)

	// 验证限流器已创建
	if count := GetLimiterStats(); count != 1 {
		t.Errorf("Expected 1 limiter, got %d", count)
	}

	// 等待TTL过期并触发清理
	time.Sleep(time.Millisecond * 200)
	cleanupExpiredLimiters()

	// 由于TTL较短，限流器应该被清理
	// 注意：由于TTL默认为1小时，这里我们手动设置过期时间
	limiterMap.Range(func(key, value interface{}) bool {
		if tr, ok := value.(*timerRate); ok {
			// 手动设置为过期
			tr.lastSeen = time.Now().Add(-2 * time.Hour)
		}
		return true
	})

	cleanupExpiredLimiters()

	if count := GetLimiterStats(); count != 0 {
		t.Errorf("Expected 0 limiters after cleanup, got %d", count)
	}
}

// 测试并发安全性
func TestLimiterConcurrency(t *testing.T) {
	Cleanup()

	ctx := context.Background()
	key := "test-concurrent"
	limit := 100
	period := time.Second

	// 并发调用限流器
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _, _ = DefaultRateFn(ctx, key, limit, period, 1)
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证只创建了一个限流器实例
	if count := GetLimiterStats(); count != 1 {
		t.Errorf("Expected 1 limiter after concurrent access, got %d", count)
	}
}

// 性能基准测试
func BenchmarkDefaultRateFn(b *testing.B) {
	ctx := context.Background()
	key := "benchmark-test"
	limit := 1000
	period := time.Second

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = DefaultRateFn(ctx, key, limit, period, 1)
		}
	})
}

// 测试内存使用情况
func BenchmarkLimiterMemory(b *testing.B) {
	ctx := context.Background()
	limit := 100
	period := time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "test-" + string(rune(i%1000)) // 限制key的数量，避免无限增长
		_, _, _ = DefaultRateFn(ctx, key, limit, period, 1)
	}
}
