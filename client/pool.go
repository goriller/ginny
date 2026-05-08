// Package client provides connection pooling, service discovery, and resilience
// features on top of ConnectRPC's typed clients.
//
// The client package does NOT re-wrap typed clients — those are auto-generated
// by protoc-gen-connect-go. Instead, it provides:
//   - Connection pool with service discovery
//   - Retry policies
//   - Circuit breaker
//   - Pre-configured Connect interceptors
package client

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// Pool manages HTTP clients for different backend services.
// It supports service discovery and connection pooling.
type Pool struct {
	opts       *poolOptions
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewPool creates a new client connection pool.
func NewPool(opts ...Option) *Pool {
	o := &poolOptions{
		baseClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &Pool{
		opts:       o,
		httpClient: o.baseClient,
	}
}

// HTTPClient returns the configured *http.Client for making ConnectRPC calls.
// ConnectRPC typed clients accept an *http.Client, so users should do:
//
//	client := myserviceconnect.NewMyServiceClient(pool.HTTPClient(), "https://backend:8080")
func (p *Pool) HTTPClient() *http.Client {
	return p.httpClient
}

// Close releases any resources held by the pool.
func (p *Pool) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.httpClient.CloseIdleConnections()
	return nil
}

// HealthCheck performs a health check against a service endpoint.
func (p *Pool) HealthCheck(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("client: health check request failed: %w", err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("client: health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("client: health check returned status %d", resp.StatusCode)
	}
	return nil
}
