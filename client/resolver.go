package client

import "context"

// Resolver resolver the host
type Resolver func(ctx context.Context, host string) (addr string, err error)
