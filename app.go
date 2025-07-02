package ginny

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/goriller/ginny/config"
	"github.com/goriller/ginny/logger"
	"github.com/goriller/ginny/server"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	// AppProviderSet
	AppProviderSet = wire.NewSet(
		logger.Default,
		config.ConfigProviderSet,
		NewOption, NewApp,
	)
	Logo = `

	┌─┐┬┌┐┌┌┐┌┬ ┬
	│ ┬│││││││└┬┘
	└─┘┴┘└┘┘└┘ ┴ 
	https://github.com/goriller/ginny
	
	`
)

func init() {
	fmt.Printf("\x1b[35;1m%s\x1b[0m\n", Logo)
}

// RegistrarFunc
type RegistrarFunc func(app *Application) error

// Application
type Application struct {
	Name    string
	Version string
	Option  *Option
	Logger  *zap.Logger
	Server  *server.Server
	Ctx     context.Context

	regFunc RegistrarFunc
}

// Option
type Option struct {
	Name        string
	Version     string
	GrpcAddr    string
	HttpAddr    string
	MetricsAddr string
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
func NewApp(
	ctx context.Context,
	option *Option,
	logger *zap.Logger,
	regFunc RegistrarFunc,
	opts ...server.Option,
) (*Application, error) {
	app := &Application{
		Name:    option.Name,
		Version: option.Version,
		Option:  option,
		regFunc: regFunc,
		Logger:  logger.With(zap.String("action", "App")),
	}
	opt := []server.Option{
		server.WithGrpcAddr(option.GrpcAddr),
	}
	if option.HttpAddr != "" {
		opts = append(opts,
			server.WithHttpAddr(option.HttpAddr),
		)
	}
	if option.MetricsAddr != "" {
		opts = append(opts,
			server.WithMetricsAddr(option.MetricsAddr),
		)
	}

	opts = append(opts, opt...)
	app.Server = server.NewServer(ctx, logger, opts...)
	return app, nil
}

// Start
func (a *Application) Start(ctx context.Context) error {
	if err := a.regFunc(a); err != nil {
		return err
	}
	a.Server.Start(ctx)
	return nil
}

// Stop
func (a *Application) Stop(ctx context.Context) error {
	return a.Server.Close(ctx)
}
