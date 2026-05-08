// Package log provides a slog-based logging interface for the Ginny framework.
//
// The framework uses *slog.Logger as its logging interface, allowing users
// to inject custom slog.Handler implementations (e.g., Zap backend, Loki push)
// without changing any framework code.
package log

import (
	"context"
	"log/slog"
	"os"
)

// Logger is the standard slog logger type used throughout Ginny.
// Users can inject any *slog.Logger, including ones backed by custom handlers.
type Logger = *slog.Logger

// Option configures the log package.
type Option func(*options)

type options struct {
	handler slog.Handler
	level   slog.Level
	output  string // file path for output (empty = stderr)
	addSource bool
}

// WithHandler sets a custom slog.Handler (e.g., a Zap adapter, Loki handler, etc.).
func WithHandler(h slog.Handler) Option {
	return func(o *options) {
		if h != nil {
			o.handler = h
		}
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithSource enables adding source file/line info to log entries.
func WithSource(addSource bool) Option {
	return func(o *options) {
		o.addSource = addSource
	}
}

// New creates a new *slog.Logger with the given options.
// By default, it writes JSON to stderr at Info level.
func New(opts ...Option) *slog.Logger {
	o := &options{
		level:  slog.LevelInfo,
		output: "",
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.handler == nil {
		w := os.Stderr
		if o.output != "" {
			f, err := os.OpenFile(o.output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// Fall back to stderr
				w = os.Stderr
			} else {
				w = f
			}
		}
		ho := &slog.HandlerOptions{
			Level:     o.level,
			AddSource: o.addSource,
		}
		o.handler = slog.NewJSONHandler(w, ho)
	}
	return slog.New(o.handler)
}

// Default returns a default logger (JSON to stderr, Info level).
func Default() *slog.Logger {
	return New()
}

// WithContext returns a logger with context values extracted.
// Extracts values from context using the standard slog context keys.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	// slog already supports context extraction via slog.LogAttrs etc.
	// For advanced context extraction, users can use a custom handler.
	return logger
}

// contextKey for storing logger in context.
type loggerKey struct{}

// SetContextLogger stores a logger in the context.
func SetContextLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetContextLogger retrieves a logger from the context, or returns default.
func GetContextLogger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return Default()
}
