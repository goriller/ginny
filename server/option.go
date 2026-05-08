package server

import (
	"log/slog"

	"connectrpc.com/connect"
)

// serverOptions holds all configuration for the Server.
type serverOptions struct {
	appAddr   string
	adminAddr string

	interceptors []connect.Interceptor

	// Vanguard REST transcoding
	restTranscodingEnabled bool

	// gRPC Reflection
	reflectionEnabled      bool
	reflectionServiceNames []string

	// h2c (HTTP/2 Cleartext) for Go < 1.24
	h2cEnabled bool

	// Debug endpoints (pprof on admin port)
	debugEnabled bool

	// Health
	healthChecker HealthChecker

	// Logger
	logger *slog.Logger
}

// defaultOptions returns the default server configuration.
func defaultOptions() *serverOptions {
	return &serverOptions{
		appAddr:                ":8080",
		adminAddr:              ":8081",
		restTranscodingEnabled: true,
		reflectionEnabled:      true,
		debugEnabled:           false,
		interceptors:           []connect.Interceptor{},
		reflectionServiceNames: []string{},
	}
}

// Option is a functional option for configuring the Server.
type Option func(*serverOptions)

// WithAppAddr sets the address for the app (RPC) server. Default ":8080".
func WithAppAddr(addr string) Option {
	return func(o *serverOptions) {
		if addr != "" {
			o.appAddr = addr
		}
	}
}

// WithAdminAddr sets the address for the admin server. Default ":8081".
func WithAdminAddr(addr string) Option {
	return func(o *serverOptions) {
		if addr != "" {
			o.adminAddr = addr
		}
	}
}

// WithInterceptor adds a Connect interceptor to the app server.
func WithInterceptor(i connect.Interceptor) Option {
	return func(o *serverOptions) {
		if i != nil {
			o.interceptors = append(o.interceptors, i)
		}
	}
}

// WithRESTTranscoding enables or disables Vanguard REST transcoding.
// Enabled by default.
func WithRESTTranscoding(enabled bool) Option {
	return func(o *serverOptions) {
		o.restTranscodingEnabled = enabled
	}
}

// WithReflection enables or disables gRPC Reflection.
// Enabled by default in development.
func WithReflection(enabled bool) Option {
	return func(o *serverOptions) {
		o.reflectionEnabled = enabled
	}
}

// WithReflectionServiceNames sets the service names to expose via reflection.
func WithReflectionServiceNames(names ...string) Option {
	return func(o *serverOptions) {
		o.reflectionServiceNames = append(o.reflectionServiceNames, names...)
	}
}

// WithDebug enables pprof debug endpoints on the admin port.
func WithDebug(enabled bool) Option {
	return func(o *serverOptions) {
		o.debugEnabled = enabled
	}
}

// WithH2C enables HTTP/2 Cleartext (h2c) support.
// This is useful for Go versions before 1.24 where h2c is not natively supported
// in net/http. When enabled, the app server wraps the handler with
// golang.org/x/net/http2/h2c.NewHandler.
func WithH2C(enabled bool) Option {
	return func(o *serverOptions) {
		o.h2cEnabled = enabled
	}
}

// WithHealthChecker sets a custom health checker.
func WithHealthChecker(checker HealthChecker) Option {
	return func(o *serverOptions) {
		if checker != nil {
			o.healthChecker = checker
		}
	}
}

// WithLogger sets a custom logger for the server.
func WithLogger(logger *slog.Logger) Option {
	return func(o *serverOptions) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// evalOptions evaluates all options and returns the final configuration.
func evalOptions(opts []Option) *serverOptions {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	// Ensure we have a logger
	if o.logger == nil {
		o.logger = slog.Default()
	}
	return o
}
