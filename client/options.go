package client

import "context"

// Resolver resolver the host
type Resolver func(ctx context.Context, service, tag string) (addr string, err error)
