package gorm

import (
	"github.com/google/wire"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewOptions(v *viper.Viper, logger *zap.Logger) (*gorm.Config, error) {
	var err error
	o := new(gorm.Config)
	if err = v.UnmarshalKey("db", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal db option error")
	}

	logger.Info("load database options success", zap.Any("gorm.Config", o))

	return o, err
}

// Init 初始化数据库
func New(dialector gorm.Dialector, conf *gorm.Config, models ...interface{}) (*gorm.DB, error) {
	var err error
	db, err := gorm.Open(dialector, conf)
	if err != nil {
		return nil, errors.Wrap(err, "gorm open database connection error")
	}

	// AutoMigrate
	if len(models) > 0 {
		for _, v := range models {
			db.AutoMigrate(v)
		}
	}

	return db, nil
}

var ProviderSet = wire.NewSet(New, NewOptions)
