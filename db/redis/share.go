package redis

import (
	"log"

	"github.com/google/wire"
)

// Provider
var Provider = wire.NewSet(New)

func New(config *Config) *Manager {
	mgr, err := NewManager(config)
	if err != nil {
		log.Fatalf("redis manager error: %s", err.Error())
	}
	return mgr
}
