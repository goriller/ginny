package ginny

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/google/wire"
	"github.com/gorillazer/ginny-serve/options"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Application
type Application struct {
	Name    string
	Version string
	logger  *zap.Logger
	Server  *Server
}

// Option
type Option struct {
	Name    string
	Version string
}

// NewOption
func NewOption(v *viper.Viper, logger *zap.Logger) (*Option, error) {
	var err error
	o := new(Option)
	if err = v.UnmarshalKey("app", o); err != nil {
		return nil, errors.Wrap(err, "unmarshal app option error")
	}

	logger.Info("load application options success")

	return o, err
}

// NewApp
func NewApp(option *Option, logger *zap.Logger, serves ...Serve) (*Application, error) {
	app := &Application{
		Name:    option.Name,
		Version: option.Version,
		logger:  logger.With(zap.String("type", "Application")),
	}

	for _, o := range serves {
		if err := o(app); err != nil {
			return nil, err
		}
	}

	return app, nil
}

// Start
func (a *Application) Start(opts ...options.ServerOptional) error {
	if a.Server.HttpServer == nil && a.Server.GrpcServer == nil {
		return errors.New("no server provider")
	}

	if a.Server.HttpServer != nil {
		if err := a.Server.HttpServer.Start(opts...); err != nil {
			return errors.Wrap(err, "http server start error")
		}
	}

	if a.Server.GrpcServer != nil {
		if err := a.Server.GrpcServer.Start(opts...); err != nil {
			return errors.Wrap(err, "grpc server start error")
		}
	}

	return nil
}

// AwaitSignal
func (a *Application) AwaitSignal() {
	c := make(chan os.Signal, 1)
	signal.Reset(syscall.SIGTERM, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	select {
	case s := <-c:
		a.logger.Info("receive a signal", zap.String("signal", s.String()))
		if a.Server.HttpServer != nil {
			if err := a.Server.HttpServer.Stop(); err != nil {
				a.logger.Error("stop http server error", zap.Error(err))
			}
		}

		if a.Server.GrpcServer != nil {
			if err := a.Server.GrpcServer.Stop(); err != nil {
				a.logger.Error("stop grpc server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var AppProviderSet = wire.NewSet(NewOption, NewApp)
