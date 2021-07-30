package mongo

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
	if err = v.UnmarshalKey("mongo", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal app option error")
	}

	logger.Info("load options success")

	return o, err
}

// New
func New(conf *Config) *Manager {
	mgr, err := NewManager(conf)
	if err != nil {
		log.Fatalf("mongodb manager error: %s", err.Error())
	}
	return mgr
}
