package ginny

import (
	"context"
	"time"

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
	config.ConfigProviderSet,
	NewOption, NewApp)

// RegistrarFunc
type RegistrarFunc func(app *Application) error

// Application
type Application struct {
	Name    string
	Version string
	Option  *Option
	Logger  logger.StdLogger
	Ctx     context.Context
	regFunc RegistrarFunc
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
	regFunc RegistrarFunc, opts ...server.Option,
) (*Application, error) {
	ctx, cc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cc()

	app := &Application{
		Name:    option.Name,
		Version: option.Version,
		Option:  option,
		Ctx:     ctx,
		regFunc: regFunc,
		Logger:  logger.Action("App"),
	}
	opt := []server.Option{
		server.WithGrpcAddr(option.GrpcAddr),
	}
	if option.HttpAddr != "" {
		opts = append(opts,
			server.WithHttp(true),
			server.WithHttpAddr(option.HttpAddr),
		)
	}

	opts = append(opts, opt...)
	app.Server = server.NewServer(
		logger.Default().With(zap.String("action", "App")), opts...)
	return app, nil
}

// Start
func (a *Application) Start() error {
	if err := a.regFunc(a); err != nil {
		return err
	}
	a.Server.Start()
	return nil
}
