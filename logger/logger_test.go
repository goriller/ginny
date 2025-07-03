package logger

import (
	"context"
	"testing"

	"github.com/goriller/ginny/interceptor/tags"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestWithTags(t *testing.T) {
	// 使用测试logger
	std = zaptest.NewLogger(t)
	defer func() { std = zap.NewNop() }()

	tests := []struct {
		name string
		tags map[string]string
		want int // 期望的字段数量
	}{
		{
			name: "empty tags",
			tags: map[string]string{},
			want: 0,
		},
		{
			name: "single tag",
			tags: map[string]string{"key1": "value1"},
			want: 1,
		},
		{
			name: "multiple tags",
			tags: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := WithTags(tt.tags)
			if logger == nil {
				t.Error("WithTags returned nil logger")
			}

			// 测试logger是否可用
			logger.Info("test message")
		})
	}
}

func TestWithContext(t *testing.T) {
	// 使用测试logger
	std = zaptest.NewLogger(t)
	defer func() { std = zap.NewNop() }()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "empty context",
			ctx:  context.Background(),
		},
		{
			name: "context with tags",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = tags.InjectIntoContext(ctx, tags.NewTags())
				ts := tags.Extract(ctx)
				ts.Set("request_id", "12345")
				ts.Set("user_id", "user123")
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := WithContext(tt.ctx)
			if logger == nil {
				t.Error("WithContext returned nil logger")
			}

			// 测试logger是否可用
			logger.Info("test message with context")
		})
	}
}

func TestContextLogger(t *testing.T) {
	// 使用测试logger
	std = zaptest.NewLogger(t)
	defer func() { std = zap.NewNop() }()

	// 创建一个logger
	testLogger := With(zap.String("test", "value"))

	// 设置到context中
	ctx := SetContextLogger(context.Background(), testLogger)

	// 从context中获取logger
	retrievedLogger := GetContextLogger(ctx)
	if retrievedLogger == nil {
		t.Error("GetContextLogger returned nil")
	}

	// 测试nil context
	nilLogger := GetContextLogger(nil)
	if nilLogger == nil {
		t.Error("GetContextLogger should return default logger for nil context")
	}
}

func TestObjectPoolEfficiency(t *testing.T) {
	// 测试对象池是否正确复用
	field1 := getFields()
	field1 = append(field1, zap.String("key1", "value1"))
	putFields(field1)

	field2 := getFields()
	// 验证pool复用了对象，长度应该为0
	if len(field2) != 0 {
		t.Errorf("Expected empty slice from pool, got length %d", len(field2))
	}

	// 验证容量被保持
	if cap(field2) < 16 {
		t.Errorf("Expected capacity >= 16, got %d", cap(field2))
	}

	putFields(field2)
}

// 性能基准测试 - WithTags优化前后对比
func BenchmarkWithTags(b *testing.B) {
	std = zap.NewNop() // 使用空logger避免I/O开销

	tags := map[string]string{
		"request_id": "12345",
		"user_id":    "user123",
		"service":    "api-server",
		"method":     "POST",
		"path":       "/api/v1/users",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger := WithTags(tags)
			_ = logger
		}
	})
}

// 性能基准测试 - WithContext优化前后对比
func BenchmarkWithContext(b *testing.B) {
	std = zap.NewNop() // 使用空logger避免I/O开销

	// 准备包含tags的context
	ctx := context.Background()
	ctx = tags.InjectIntoContext(ctx, tags.NewTags())
	ts := tags.Extract(ctx)
	ts.Set("request_id", "12345")
	ts.Set("user_id", "user123")
	ts.Set("service", "api-server")
	ts.Set("method", "POST")
	ts.Set("path", "/api/v1/users")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger := WithContext(ctx)
			_ = logger
		}
	})
}

// 内存分配基准测试
func BenchmarkFieldsPool(b *testing.B) {
	b.Run("with pool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fields := getFields()
			fields = append(fields, zap.String("key", "value"))
			putFields(fields)
		}
	})

	b.Run("without pool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fields := make([]zap.Field, 0, 16)
			fields = append(fields, zap.String("key", "value"))
			_ = fields
		}
	})
}
