package logger

import (
	"context"

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

// WithTags with map string string tags
// more effective
func WithTags(m map[string]string) StdLogger {
	fields := make([]zap.Field, 0)
	for k, v := range m {
		fields = append(fields, zap.String(k, v))
	}
	return std.With(fields...)
}

// WithContext
func WithContext(ctx context.Context) StdLogger {
	fields := make([]zap.Field, 0)
	ts := tags.Extract(ctx).Values()
	for k, v := range ts {
		fields = append(fields, zap.String(k, v))
	}
	return std.With(fields...)
}

// SetContextLogger context with tags logger
func SetContextLogger(ctx context.Context, log StdLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, log)
}

// GetLogger extract logger from context
func GetContextLogger(ctx context.Context) StdLogger {
	if ctx == nil {
		return std
	}
	if ctxLogger, ok := ctx.Value(loggerKey{}).(StdLogger); ok {
		return ctxLogger
	}
	return std
}
