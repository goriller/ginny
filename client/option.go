package client

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
)

// Option configures the client Pool.
type Option func(*poolOptions)

// poolOptions holds all client configuration.
type poolOptions struct {
	baseClient *http.Client
	interceptors []connect.Interceptor
	retryMax int
	circuitBreakerEnabled bool
}

// WithHTTPClient sets a custom base HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *poolOptions) {
		if client != nil {
			o.baseClient = client
		}
	}
}

// WithTimeout sets the timeout for HTTP requests.
func WithTimeout(timeout time.Duration) Option {
	return func(o *poolOptions) {
		if o.baseClient == nil {
			o.baseClient = &http.Client{}
		}
		o.baseClient.Timeout = timeout
	}
}

// WithInterceptor adds a Connect interceptor to the client.
func WithInterceptor(i connect.Interceptor) Option {
	return func(o *poolOptions) {
		if i != nil {
			o.interceptors = append(o.interceptors, i)
		}
	}
}

// WithRetry sets the maximum number of retries for failed requests.
func WithRetry(maxRetries int) Option {
	return func(o *poolOptions) {
		if maxRetries > 0 {
			o.retryMax = maxRetries
		}
	}
}

// WithCircuitBreaker enables circuit breaker protection.
func WithCircuitBreaker(enabled bool) Option {
	return func(o *poolOptions) {
		o.circuitBreakerEnabled = enabled
	}
}
