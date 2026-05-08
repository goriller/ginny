// Package ratelimit provides a Connect interceptor for rate limiting.
package ratelimit

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

// Limiter is the interface for rate limiting strategies.
// Implementations can be local (in-memory) or distributed (Redis).
type Limiter interface {
	// Allow checks if a request identified by key should be allowed.
	// Returns true if allowed, false if rate-limited.
	Allow(ctx context.Context, key string) (bool, error)
}

// KeyFunc generates a rate limit key from the request context.
// Default is to use the full procedure name.
type KeyFunc func(ctx context.Context, req connect.AnyRequest) string

// DefaultKeyFunc uses the procedure name as the key.
func DefaultKeyFunc(ctx context.Context, req connect.AnyRequest) string {
	return req.Spec().Procedure
}

// NewInterceptor creates a Connect interceptor that rate-limits requests.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(limiter Limiter, opts ...Option) connect.Interceptor {
	o := &options{
		limiter: limiter,
		keyFunc: DefaultKeyFunc,
	}
	for _, opt := range opts {
		opt(o)
	}
	interceptor := &rateLimitInterceptor{
		limiter: o.limiter,
		keyFunc: o.keyFunc,
	}
	return interceptor
}

// Option configures the rate limit interceptor.
type Option func(*options)

type options struct {
	limiter Limiter
	keyFunc KeyFunc
}

// WithKeyFunc sets a custom key generation function.
func WithKeyFunc(fn KeyFunc) Option {
	return func(o *options) {
		if fn != nil {
			o.keyFunc = fn
		}
	}
}

type rateLimitInterceptor struct {
	limiter Limiter
	keyFunc KeyFunc
}

func (r *rateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		key := r.keyFunc(ctx, req)
		allowed, err := r.limiter.Allow(ctx, key)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				errors.New("ratelimit: check failed: "+err.Error()))
		}
		if !allowed {
			return nil, connect.NewError(connect.CodeResourceExhausted,
				errors.New("ratelimit: too many requests"))
		}
		return next(ctx, req)
	}
}

func (r *rateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (r *rateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		key := conn.Spec().Procedure
		allowed, err := r.limiter.Allow(ctx, key)
		if err != nil {
			return connect.NewError(connect.CodeInternal,
				errors.New("ratelimit: check failed: "+err.Error()))
		}
		if !allowed {
			return connect.NewError(connect.CodeResourceExhausted,
				errors.New("ratelimit: too many requests"))
		}
		return next(ctx, conn)
	}
}
