package limit

import (
	"context"
	"time"
)

// Limiter the limiter struct for hold fn and config
type Limiter struct {
	PersistenceFn PersistenceFn
	Config        *RouterLimit
}

// PersistenceFn the limit persistence fn for store limit status
type PersistenceFn func(ctx context.Context, key string,
	limit int, period time.Duration, n int) (remaining int, reset time.Duration, allowed bool)
