package limit

import (
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	routerLimit := &RouterLimit{
		Limit: []Limit{
			{
				Prefix:   "/v1/prefix/",
				Headers:  []string{"user-id", "client-id"},
				Quota:    200,
				Duration: 20 * time.Minute,
			},
		},
		Block: []KV{
			{
				Key:   "client-id",
				Value: "block",
			},
		},
		Default: Default{
			Headers:  []string{"user-id", "client-id"},
			Quota:    100,
			Duration: 10 * time.Minute,
		},
	}
	data := routerLimit.MatchHeader("/v1/not/match", map[string][]string{
		"user-id":   {"user"},
		"client-id": {"client"},
	})
	if data.Key != ".user.client" || data.Quota != 100 {
		t.Errorf("test /v1/not/match failed expect %s but got %s quota %d", ".user.client", data.Key, data.Quota)
		return
	}
	data = routerLimit.MatchHeader("/v1/prefix/match", map[string][]string{
		"user-id":   {"user1"},
		"client-id": {"client1"},
	})

	if data.Key != ".user1.client1" || data.Quota != 200 {
		t.Errorf("test /v1/prefix/match failed expect got %s quota %d", data.Key, data.Quota)
		return
	}

	data = routerLimit.MatchHeader("/v1/prefix/block", map[string][]string{
		"client-id": {"block"},
	})

	if data.Quota != Block {
		t.Errorf("test /v1/prefix/block failed expect got %s quota %d", data.Key, data.Quota)
		return
	}

}
