package redis

import (
	"log"

	"github.com/google/wire"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Provider
var Provider = wire.NewSet(NewConfig, New)

// NewConfig
func NewConfig(v *viper.Viper, logger *zap.Logger) (*Config, error) {
	var err error
	o := new(Config)
	if err = v.UnmarshalKey("redis", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal app option error")
	}

	logger.Info("load options success")

	return o, err
}

func New(config *Config) *Manager {
	mgr, err := NewManager(config)
	if err != nil {
		log.Fatalf("redis manager error: %s", err.Error())
	}
	return mgr
}
