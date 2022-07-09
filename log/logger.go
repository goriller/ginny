package log

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"go.uber.org/zap"
)

const actionKey = "action"

type loggerKey struct{}

// Default the default log instance
func Default() *zap.Logger {
	return std
}

// Action set action filed for logger
func Action(action string) *zap.Logger {
	return std.With(zap.String(actionKey, action))
}

// With with any map data， the value of key must be string, int ... basic value
func With(m ...zap.Field) *zap.Logger {
	return std.With(m...)
}

// WithTags with map string string tags
// more effective
func WithTags(m map[string]string) *zap.Logger {
	fields := make([]zap.Field, 0)
	for k, v := range m {
		fields = append(fields, zap.String(k, v))
	}
	return std.With(fields...)
}

// WithContext
func WithContext(ctx context.Context) *zap.Logger {
	fields := make([]zap.Field, 0)
	ts := tags.Extract(ctx).Values()
	for k, v := range ts {
		fields = append(fields, zap.String(k, v))
	}
	return std.With(fields...)
}

// SetContextLogger context with tags logger
func SetContextLogger(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, log)
}

// GetLogger extract logger from context
func GetContextLogger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return std
	}
	if ctxLogger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return ctxLogger
	}
	return std
}
