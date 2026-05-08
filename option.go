package ginny

import (
	"log/slog"

	"github.com/goriller/ginny/v2/config"
	"github.com/goriller/ginny/v2/server"
)

// options holds all configurable options for the App.
type options struct {
	name    string
	config  *config.Config
	logger  *slog.Logger
	servers []*server.Server
	hooks   []LifecycleHook
}

// Option is a functional option for configuring the App.
type Option func(*options)

func defaultOptions() *options {
	return &options{
		name: "ginny-app",
	}
}

func evalOptions(opts []Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	if o.logger == nil {
		o.logger = slog.Default()
	}
	if o.config == nil {
		o.config = &config.Config{}
	}
	return o
}

// WithName sets the application name.
func WithName(name string) Option {
	return func(o *options) { o.name = name }
}

// WithConfig sets the application configuration.
func WithConfig(cfg *config.Config) Option {
	return func(o *options) { o.config = cfg }
}

// WithLogger sets the application logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) { o.logger = logger }
}

// WithServer adds a server to the application.
func WithServer(srv *server.Server) Option {
	return func(o *options) {
		o.servers = append(o.servers, srv)
	}
}

// WithHook adds a lifecycle hook to the application.
func WithHook(hook LifecycleHook) Option {
	return func(o *options) { o.hooks = append(o.hooks, hook) }
}
