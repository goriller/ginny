package mysql

import (
	"github.com/google/wire"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Provider
var Provider = wire.NewSet(NewConfig, NewSqlBuilder)

// NewConfig
func NewConfig(v *viper.Viper, logger *zap.Logger) (*Config, error) {
	var err error
	o := new(Config)
	if err = v.UnmarshalKey("mysql", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal app option error")
	}

	if o.RDB.Host == "" {
		o.RDB.Host = o.WDB.Host
		o.RDB.User = o.WDB.User
		o.RDB.Pass = o.WDB.Pass
	}

	logger.Info("load options success")

	return o, err
}
