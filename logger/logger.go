package logger

import (
	"context"
	"sync"

	"github.com/goriller/ginny/interceptor/tags"
	"go.uber.org/zap"
)

const actionKey = "action"

type loggerKey struct{}

// StdLogger the standard logger
type StdLogger interface {
	// With with any map data， the value of key must be string, int ... basic value
	With(m ...zap.Field) *zap.Logger
	// Debug print the debug log if the len(args) is 0, the args will be ignored
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
}

// 对象池优化，减少内存分配
var (
	fieldsPool = sync.Pool{
		New: func() interface{} {
			// 预分配容量，减少slice扩容
			return make([]zap.Field, 0, 16)
		},
	}
)

// getFields 从对象池获取字段slice
func getFields() []zap.Field {
	return fieldsPool.Get().([]zap.Field)
}

// putFields 将字段slice放回对象池
func putFields(fields []zap.Field) {
	// 重置slice但保留容量
	fields = fields[:0]
	fieldsPool.Put(fields)
}

// Default the default log instance
func Default() *zap.Logger {
	return std
}

// Action set action filed for logger
func Action(action string) StdLogger {
	return std.With(zap.String(actionKey, action))
}

// With with any map data， the value of key must be string, int ... basic value
func With(m ...zap.Field) StdLogger {
	return std.With(m...)
}

// WithTags with map string string tags - 优化版本
func WithTags(m map[string]string) StdLogger {
	if len(m) == 0 {
		return std
	}

	fields := getFields()
	defer putFields(fields)

	for k, v := range m {
		fields = append(fields, zap.String(k, v))
	}

	// 复制fields，避免池化对象被修改
	copyFields := make([]zap.Field, len(fields))
	copy(copyFields, fields)

	return std.With(copyFields...)
}

// WithContext - 优化版本，使用对象池
func WithContext(ctx context.Context) StdLogger {
	ts := tags.Extract(ctx).Values()
	if len(ts) == 0 {
		return std
	}

	fields := getFields()
	defer putFields(fields)

	for k, v := range ts {
		if strVal, ok := v.(string); ok {
			fields = append(fields, zap.String(k, strVal))
		}
	}

	if len(fields) == 0 {
		return std
	}

	// 复制fields，避免池化对象被修改
	copyFields := make([]zap.Field, len(fields))
	copy(copyFields, fields)

	return std.With(copyFields...)
}

// SetContextLogger context with tags logger
func SetContextLogger(ctx context.Context, log StdLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, log)
}

// GetContextLogger extract logger from context
func GetContextLogger(ctx context.Context) StdLogger {
	if ctx == nil {
		return std
	}
	if ctxLogger, ok := ctx.Value(loggerKey{}).(StdLogger); ok {
		return ctxLogger
	}
	return std
}
