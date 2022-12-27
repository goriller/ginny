package limit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var limiterMap = new(sync.Map)

type timerRate struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// DefaultRateFn
func DefaultRateFn(ctx context.Context, key string, limit int, period time.Duration,
	n int) (remaining int, reset time.Duration, allowed bool) {
	lm := &timerRate{
		limiter:  rate.NewLimiter(rate.Every(period), limit),
		lastSeen: time.Now(),
	}
	v, loaded := limiterMap.LoadOrStore(key, lm)
	if loaded {
		lm = v.(*timerRate)
		lm.lastSeen = time.Now()
	}
	return int(lm.limiter.Tokens()),
		period, lm.limiter.AllowN(lm.lastSeen, n)
}
