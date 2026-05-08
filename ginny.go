// Package ginny is the entry point for the Ginny v2 framework.
// It provides App lifecycle management with ordered Lifecycle Hooks,
// dual-port server (App + Admin), and unified interceptor chains.
package ginny

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/goriller/ginny/v2/config"
	"github.com/goriller/ginny/v2/server"
	"go.uber.org/automaxprocs/maxprocs"
)

// App is the main application instance.
type App struct {
	name    string
	config  *config.Config
	logger  *slog.Logger
	servers []*server.Server
	hooks   []LifecycleHook
}

// New creates a new App with the given options.
func New(opts ...Option) (*App, error) {
	o := evalOptions(opts)

	// Auto-configure GOMAXPROCS for container environments
	if _, err := maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
		if o.logger != nil {
			o.logger.Debug(fmt.Sprintf(format, args...))
		}
	})); err != nil {
		return nil, fmt.Errorf("automaxprocs: %w", err)
	}

	app := &App{
		name:    o.name,
		config:  o.config,
		logger:  o.logger,
		servers: o.servers,
		hooks:   o.hooks,
	}

	// Sort hooks by priority (lower = earlier start, later stop)
	sort.Slice(app.hooks, func(i, j int) bool {
		return app.hooks[i].Priority() < app.hooks[j].Priority()
	})

	return app, nil
}

// Start runs the application. It calls OnStart for each hook in priority order,
// then blocks until a signal (SIGINT/SIGTERM) is received, then gracefully stops.
func (a *App) Start(ctx context.Context) error {
	// OnStart hooks (priority order: lower first)
	for _, hook := range a.hooks {
		if err := hook.OnStart(ctx); err != nil {
			return fmt.Errorf("hook %q start: %w", hook.Name(), err)
		}
		a.logInfo(ctx, "hook started", "hook", hook.Name(), "priority", hook.Priority())
	}

	// Start all servers
	for i, srv := range a.servers {
		go func(idx int, s *server.Server) {
			if err := s.Start(ctx); err != nil {
				a.logError(ctx, "server stopped", "index", idx, "error", err)
			}
		}(i, srv)
	}

	a.logInfo(ctx, "application started", "name", a.name)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	a.logInfo(ctx, "received signal, shutting down", "signal", sig.String())

	return a.Stop(ctx)
}

// Stop gracefully stops the application. It calls OnStop for each hook
// in reverse priority order (higher first).
func (a *App) Stop(ctx context.Context) error {
	// OnStop hooks (reverse priority order: higher first)
	for i := len(a.hooks) - 1; i >= 0; i-- {
		hook := a.hooks[i]
		if err := hook.OnStop(ctx); err != nil {
			a.logError(ctx, "hook stop error", "hook", hook.Name(), "error", err)
		}
		a.logInfo(ctx, "hook stopped", "hook", hook.Name())
	}

	// Stop servers
	for i := len(a.servers) - 1; i >= 0; i-- {
		if err := a.servers[i].Stop(ctx); err != nil {
			a.logError(ctx, "server stop error", "index", i, "error", err)
		}
	}

	a.logInfo(ctx, "application stopped", "name", a.name)
	return nil
}

// Name returns the application name.
func (a *App) Name() string { return a.name }

// Config returns the application configuration.
func (a *App) Config() *config.Config { return a.config }

// Logger returns the application logger.
func (a *App) Logger() *slog.Logger { return a.logger }

// logInfo is a helper for info-level logging.
func (a *App) logInfo(ctx context.Context, msg string, args ...any) {
	if a.logger != nil {
		a.logger.InfoContext(ctx, msg, args...)
	}
}

// logError is a helper for error-level logging.
func (a *App) logError(ctx context.Context, msg string, args ...any) {
	if a.logger != nil {
		a.logger.ErrorContext(ctx, msg, args...)
	}
}

// LifecycleHook defines a component that needs ordered start/stop.
// Lower Priority() = starts earlier, stops later.
type LifecycleHook interface {
	Name() string
	OnStart(ctx context.Context) error
	OnStop(ctx context.Context) error
	Priority() int
}
