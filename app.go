package ginny

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/google/wire"
	"github.com/gorillazer/ginny-serve/grpc"
	"github.com/gorillazer/ginny-serve/http"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Application
type Application struct {
	name       string
	version    string
	logger     *zap.Logger
	httpServer *http.Server
	grpcServer *grpc.Server
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
		name:    option.Name,
		version: option.Version,
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
func (a *Application) Start() error {

	if a.httpServer != nil {
		if err := a.httpServer.Start(); err != nil {
			return errors.Wrap(err, "http server start error")
		}
	}

	if a.grpcServer != nil {
		if err := a.grpcServer.Start(); err != nil {
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
		if a.httpServer != nil {
			if err := a.httpServer.Stop(); err != nil {
				a.logger.Warn("stop http server error", zap.Error(err))
			}
		}

		if a.grpcServer != nil {
			if err := a.grpcServer.Stop(); err != nil {
				a.logger.Warn("stop grpc server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var AppProviderSet = wire.NewSet(NewOption, NewApp)
