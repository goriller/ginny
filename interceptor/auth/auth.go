// Package auth provides a Connect interceptor for authentication and authorization.
package auth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

// TokenValidator validates authentication tokens and returns identity information.
type TokenValidator interface {
	// Validate validates a token and returns the associated identity.
	// Returns an error if the token is invalid or expired.
	Validate(ctx context.Context, token string) (Identity, error)
}

// Identity represents an authenticated entity.
type Identity interface {
	// Subject returns the unique identifier of the identity (e.g., user ID).
	Subject() string
	// Claims returns additional claims associated with the identity.
	Claims() map[string]any
}

// TokenExtractor extracts an authentication token from a request.
type TokenExtractor func(ctx context.Context, req connect.AnyRequest) (string, error)

// BearerTokenExtractor extracts a Bearer token from the Authorization header.
func BearerTokenExtractor(ctx context.Context, req connect.AnyRequest) (string, error) {
	auth := req.Header().Get("Authorization")
	if len(auth) < 8 || auth[:7] != "Bearer " {
		return "", errors.New("auth: missing or invalid Authorization header")
	}
	return auth[7:], nil
}

// NewInterceptor creates a Connect interceptor that authenticates requests.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(validator TokenValidator, opts ...Option) connect.Interceptor {
	o := &options{
		validator:      validator,
		tokenExtractor: BearerTokenExtractor,
	}
	for _, opt := range opts {
		opt(o)
	}
	interceptor := &authInterceptor{
		validator:      o.validator,
		tokenExtractor: o.tokenExtractor,
		skippedProcs:   o.skippedProcs,
	}
	return interceptor
}

// Option configures the auth interceptor.
type Option func(*options)

type options struct {
	validator      TokenValidator
	tokenExtractor TokenExtractor
	skippedProcs   map[string]bool
}

// WithTokenExtractor sets a custom token extraction function.
func WithTokenExtractor(extractor TokenExtractor) Option {
	return func(o *options) {
		if extractor != nil {
			o.tokenExtractor = extractor
		}
	}
}

// SkipProcedures specifies RPC procedures to skip authentication for.
func SkipProcedures(procedures ...string) Option {
	return func(o *options) {
		if o.skippedProcs == nil {
			o.skippedProcs = make(map[string]bool)
		}
		for _, p := range procedures {
			o.skippedProcs[p] = true
		}
	}
}

type authInterceptor struct {
	validator      TokenValidator
	tokenExtractor TokenExtractor
	skippedProcs   map[string]bool
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if a.skippedProcs != nil && a.skippedProcs[req.Spec().Procedure] {
			return next(ctx, req)
		}
		token, err := a.tokenExtractor(ctx, req)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		identity, err := a.validator.Validate(ctx, token)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated,
				errors.New("auth: invalid token: "+err.Error()))
		}
		ctx = WithIdentity(ctx, identity)
		return next(ctx, req)
	}
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if a.skippedProcs != nil && a.skippedProcs[conn.Spec().Procedure] {
			return next(ctx, conn)
		}
		// For streaming, extract token from request headers
		token := conn.RequestHeader().Get("Authorization")
		if len(token) < 8 || token[:7] != "Bearer " {
			return connect.NewError(connect.CodeUnauthenticated,
				errors.New("auth: missing or invalid Authorization header"))
		}
		token = token[7:]
		identity, err := a.validator.Validate(ctx, token)
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated,
				errors.New("auth: invalid token: "+err.Error()))
		}
		ctx = WithIdentity(ctx, identity)
		return next(ctx, conn)
	}
}

// Context keys for storing identity.
type identityKey struct{}

// WithIdentity stores an identity in the context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// GetIdentity retrieves the identity from the context.
func GetIdentity(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}
