// Package logging provides a Connect interceptor for RPC request logging.
// It logs method, duration, and status code using the standard slog logger.
package logging

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// NewInterceptor creates a Connect interceptor that logs every RPC request.
//
// Logs include: procedure, duration, status code, and error message if any.
// Level can be switched between debug and info via options.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(logger *slog.Logger, opts ...Option) connect.Interceptor {
	o := &options{
		logger: logger,
		level:  slog.LevelInfo,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = slog.Default()
	}
	interceptor := &loggingInterceptor{
		logger: o.logger,
		level:  o.level,
	}
	return interceptor
}

// Option configures the logging interceptor.
type Option func(*options)

type options struct {
	logger *slog.Logger
	level  slog.Level
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// WithLevel sets the log level for successful requests.
// Default is slog.LevelInfo. Use slog.LevelDebug for verbose logging.
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

type loggingInterceptor struct {
	logger *slog.Logger
	level  slog.Level
}

func (l *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		duration := time.Since(start)

		var code connect.Code
		if err != nil {
			code = connect.CodeOf(err)
		}

		attrs := []slog.Attr{
			slog.String("procedure", req.Spec().Procedure),
			slog.String("duration", duration.String()),
			slog.String("code", code.String()),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		l.logger.LogAttrs(ctx, l.level, "unary RPC completed", attrs...)
		return resp, err
	}
}

func (l *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (l *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		err := next(ctx, conn)
		duration := time.Since(start)

		var code connect.Code
		if err != nil {
			code = connect.CodeOf(err)
		}

		attrs := []slog.Attr{
			slog.String("procedure", conn.Spec().Procedure),
			slog.String("duration", duration.String()),
			slog.String("code", code.String()),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		l.logger.LogAttrs(ctx, l.level, "streaming RPC completed", attrs...)
		return err
	}
}
