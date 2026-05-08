// Package validation provides a Connect interceptor for request validation.
// It validates request messages that implement a Validate() interface.
package validation

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
)

// Validator is the interface that request messages must implement
// for the validation interceptor to validate them.
//
// Generated proto messages can implement this via protoc-gen-validate
// or custom Validate methods.
type Validator interface {
	Validate() error
}

// NewInterceptor creates a Connect interceptor that validates request messages.
//
// If the request message implements the Validator interface, its Validate()
// method is called. Validation failures return connect.CodeInvalidArgument.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(opts ...Option) connect.Interceptor {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	interceptor := &validationInterceptor{
		skipProcedures: o.skipProcedures,
	}
	return interceptor
}

// Option configures the validation interceptor.
type Option func(*options)

type options struct {
	skipProcedures map[string]bool
}

// SkipProcedures specifies RPC procedures to skip validation for.
func SkipProcedures(procedures ...string) Option {
	return func(o *options) {
		if o.skipProcedures == nil {
			o.skipProcedures = make(map[string]bool)
		}
		for _, p := range procedures {
			o.skipProcedures[p] = true
		}
	}
}

type validationInterceptor struct {
	skipProcedures map[string]bool
}

func (v *validationInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if v.skipProcedures != nil && v.skipProcedures[req.Spec().Procedure] {
			return next(ctx, req)
		}

		if validator, ok := req.Any().(Validator); ok {
			if err := validator.Validate(); err != nil {
				return nil, connect.NewError(
					connect.CodeInvalidArgument,
					fmt.Errorf("validation: %w", err),
				)
			}
		}
		return next(ctx, req)
	}
}

func (v *validationInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (v *validationInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if v.skipProcedures != nil && v.skipProcedures[conn.Spec().Procedure] {
			return next(ctx, conn)
		}

		// For streaming, validate the initial request message if possible
		// Note: StreamingHandlerConn doesn't expose the request message directly,
		// so we validate at the unary level. Streaming validation would need
		// to be done in the stream's Recv loop.
		return next(ctx, conn)
	}
}
