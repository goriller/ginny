package logger

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestBatchLogger(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     5,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	// 测试基本日志记录
	batchLogger.Info("test message 1")
	batchLogger.Debug("test message 2")
	batchLogger.Warn("test message 3")
	batchLogger.Error("test message 4")

	// 手动刷新
	batchLogger.Flush()

	stats := batchLogger.Stats()
	if stats.TotalLogs != 4 {
		t.Errorf("Expected 4 total logs, got %d", stats.TotalLogs)
	}
}

func TestBatchLoggerFlushSize(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     3,
		FlushInterval: time.Hour, // 长间隔，强制通过大小触发
		BufferSize:    100,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	// 发送3条日志，应该触发自动刷新
	batchLogger.Info("message 1")
	batchLogger.Info("message 2")
	batchLogger.Info("message 3")

	// 稍等片刻让批处理完成
	time.Sleep(10 * time.Millisecond)

	stats := batchLogger.Stats()
	if stats.FlushCount == 0 {
		t.Error("Expected at least one flush")
	}
}

func TestBatchLoggerFlushInterval(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     100, // 大批量，强制通过时间触发
		FlushInterval: 50 * time.Millisecond,
		BufferSize:    100,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	// 发送少量日志
	batchLogger.Info("message 1")
	batchLogger.Info("message 2")

	// 等待刷新间隔
	time.Sleep(100 * time.Millisecond)

	stats := batchLogger.Stats()
	if stats.FlushCount == 0 {
		t.Error("Expected at least one flush due to time interval")
	}
}

func TestBatchLoggerConcurrency(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     10,
		FlushInterval: 50 * time.Millisecond,
		BufferSize:    1000,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	var wg sync.WaitGroup
	concurrency := 10
	logsPerGoroutine := 100

	// 并发写入日志
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				batchLogger.Info("concurrent log", zap.Int("goroutine", id), zap.Int("message", j))
			}
		}(i)
	}

	wg.Wait()
	batchLogger.Flush()

	stats := batchLogger.Stats()
	expectedLogs := int64(concurrency * logsPerGoroutine)
	if stats.TotalLogs != expectedLogs {
		t.Errorf("Expected %d total logs, got %d", expectedLogs, stats.TotalLogs)
	}
}

func TestBatchLoggerBufferOverflow(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     1000,      // 大批量，不触发刷新
		FlushInterval: time.Hour, // 长间隔
		BufferSize:    5,         // 小缓冲区
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	// 发送超过缓冲区大小的日志
	for i := 0; i < 10; i++ {
		batchLogger.Info("overflow test", zap.Int("index", i))
	}

	stats := batchLogger.Stats()
	if stats.DroppedLogs == 0 {
		t.Error("Expected some logs to be dropped due to buffer overflow")
	}
}

func TestBatchWrapper(t *testing.T) {
	baseLogger := zaptest.NewLogger(t)
	config := BatchConfig{
		FlushSize:     5,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}

	wrapper := NewBatchWrapper(baseLogger, config)
	defer wrapper.Close()

	// 测试批处理功能
	wrapper.Info("wrapper test")

	// 测试With方法
	childLogger := wrapper.With(zap.String("component", "test"))
	if childLogger == nil {
		t.Error("With method should return a logger")
	}
}

func TestGlobalBatchLogging(t *testing.T) {
	config := BatchConfig{
		FlushSize:     5,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}

	// 启用全局批处理
	EnableBatchLogging(config)
	defer func() {
		if globalBatchLogger != nil {
			globalBatchLogger.Close()
			globalBatchLogger = nil
			batchOnce = sync.Once{}
		}
	}()

	// 测试全局批处理函数
	BatchInfo("global batch info")
	BatchDebug("global batch debug")
	BatchWarn("global batch warn")
	BatchError("global batch error")

	FlushBatchLogs()

	if globalBatchLogger == nil {
		t.Error("Global batch logger should be initialized")
	}
}

func BenchmarkBatchLogger(b *testing.B) {
	baseLogger := zap.NewNop()
	config := BatchConfig{
		FlushSize:     100,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    10000,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			batchLogger.Info("benchmark message",
				zap.String("key1", "value1"),
				zap.Int("key2", 123),
			)
		}
	})

	batchLogger.Flush()
}

func BenchmarkDirectLogger(b *testing.B) {
	logger := zap.NewNop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message",
				zap.String("key1", "value1"),
				zap.Int("key2", 123),
			)
		}
	})
}

func BenchmarkBatchLoggerMemory(b *testing.B) {
	baseLogger := zap.NewNop()
	config := BatchConfig{
		FlushSize:     1000,
		FlushInterval: time.Hour, // 不自动刷新
		BufferSize:    10000,
	}

	batchLogger := NewBatchLogger(baseLogger, config)
	defer batchLogger.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		batchLogger.Info("memory test message",
			zap.String("key", "value"),
			zap.Int("index", i),
		)
	}
}
