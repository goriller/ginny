package ginny

import (
	"github.com/google/wire"
	"github.com/gorillazer/ginny/config"
	"github.com/gorillazer/ginny/logger"
	"github.com/gorillazer/ginny/server"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AppProviderSet
var AppProviderSet = wire.NewSet(
	logger.LoggerProviderSet,
	config.ConfigProviderSet,
	NewOption, NewApp)

// Application
type Application struct {
	Name    string
	Version string
	Option  *Option
	Logger  *zap.Logger
	Server  *server.Server
}

// Option
type Option struct {
	Name     string
	Version  string
	GrpcAddr string
	HttpAddr string
}

// NewOption
func NewOption(v *viper.Viper) (*Option, error) {
	var err error
	o := new(Option)
	if err = v.UnmarshalKey("app", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal app option error")
	}

	return o, nil
}

// NewApp
func NewApp(option *Option,
	logger *zap.Logger,
	regFunc server.RegistrarFunc, opts ...server.Option,
) (*Application, error) {
	app := &Application{
		Name:    option.Name,
		Version: option.Version,
		Option:  option,
		Logger:  logger.With(zap.String("type", "Application")),
	}
	opt := []server.Option{
		server.WithGrpcAddr(option.GrpcAddr),
		server.WithHttpAddr(option.HttpAddr),
	}

	opts = append(opts, opt...)
	app.Server = server.NewServer(app.Logger, regFunc, opts...)
	return app, nil
}

// Start
func (a *Application) Start() error {
	a.Server.Start()
	return nil
}
