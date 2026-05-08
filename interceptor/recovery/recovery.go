// Package recovery provides a Connect interceptor for panic recovery.
package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"connectrpc.com/connect"
)

// NewInterceptor creates a Connect interceptor that recovers from panics
// in RPC handlers and converts them to Internal errors.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(logger *slog.Logger, opts ...Option) connect.Interceptor {
	o := &options{
		logger: logger,
	}
	for _, opt := range opts {
		opt(o)
	}
	interceptor := &recoveryInterceptor{
		logger: o.logger,
	}
	return interceptor
}

// Option configures the recovery interceptor.
type Option func(*options)

type options struct {
	logger *slog.Logger
}

// WithLogger sets a custom logger for the recovery interceptor.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

type recoveryInterceptor struct {
	logger *slog.Logger
}

func (r *recoveryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, retErr error) {
		defer func() {
			if p := recover(); p != nil {
				stack := debug.Stack()
				if r.logger != nil {
					r.logger.ErrorContext(ctx, "panic recovered in RPC handler",
						slog.Any("panic", p),
						slog.String("stack", string(stack)),
						slog.String("procedure", req.Spec().Procedure),
					)
				}
				retErr = connect.NewError(
					connect.CodeInternal,
					fmt.Errorf("internal error: panic recovered: %v", p),
				)
			}
		}()
		resp, retErr = next(ctx, req)
		return
	}
}

func (r *recoveryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (r *recoveryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) (retErr error) {
		defer func() {
			if p := recover(); p != nil {
				stack := debug.Stack()
				if r.logger != nil {
					r.logger.ErrorContext(ctx, "panic recovered in streaming handler",
						slog.Any("panic", p),
						slog.String("stack", string(stack)),
					)
				}
				retErr = connect.NewError(
					connect.CodeInternal,
					fmt.Errorf("internal error: panic recovered: %v", p),
				)
			}
		}()
		retErr = next(ctx, conn)
		return
	}
}
