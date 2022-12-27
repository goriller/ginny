package limit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_DefaultRate(t *testing.T) {
	tests := []struct {
		ctx    context.Context
		key    string
		limit  int
		period time.Duration
		n      int
	}{
		// TODO: Add test cases.
		{
			ctx:    context.Background(),
			key:    "test1",
			limit:  100,
			period: time.Second * 1,
			n:      1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// x := 100
			for i := 0; i < 300; i++ {
				// if i%2 == 0 {
				// 	go func() {
				// 		remaining, reset, allowed := DefaultRate(tt.ctx, tt.key, tt.limit, tt.period, tt.n)
				// 		fmt.Printf("key: %v, limit: %v,remaining: %v, reset: %v, allowed: %v\n", tt.key, tt.limit, remaining, reset, allowed)
				// 	}()
				// } else {
				if i == 200 {
					time.Sleep(time.Second * 2)
				}
				remaining, reset, allowed := DefaultRateFn(tt.ctx, tt.key, tt.limit, tt.period, tt.n)
				fmt.Printf("i: %v, limit: %v,remaining: %v, reset: %v, allowed: %v\n", i, tt.limit, remaining, reset, allowed)
				// if !allowed {
				// 	time.Sleep(reset)
				// }
				// }
			}

		})
	}
}
