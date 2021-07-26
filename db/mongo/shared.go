package mongo

import (
	"log"

	"github.com/google/wire"
)

// Provider
var Provider = wire.NewSet(New)

// New
func New(conf *Config) *Manager {
	mgr, err := NewManager(conf)
	if err != nil {
		log.Fatalf("mongodb manager error: %s", err.Error())
	}
	return mgr
}
