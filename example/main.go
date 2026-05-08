// Package main demonstrates a minimal Ginny v2 application.
//
// Usage:
//
//	go run example/main.go
//
// This starts an app server on :8080 and an admin server on :8081.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/goriller/ginny/v2"
	"github.com/goriller/ginny/v2/config"
	"github.com/goriller/ginny/v2/interceptor/logging"
	"github.com/goriller/ginny/v2/interceptor/recovery"
	"github.com/goriller/ginny/v2/log"
	"github.com/goriller/ginny/v2/server"
)

func main() {
	// Load configuration (from file or defaults)
	cfg := config.MustLoad()

	// Create a structured logger
	logger := log.New(
		log.WithLevel(slog.LevelInfo),
		log.WithSource(true),
	)

	// Create the app server with interceptors
	srv := server.New(
		server.WithAppAddr(cfg.Server.Addr),
		server.WithAdminAddr(cfg.Admin.Addr),
		server.WithLogger(logger),
		server.WithInterceptor(recovery.NewInterceptor(logger)),
		server.WithInterceptor(logging.NewInterceptor(logger,
			logging.WithLevel(slog.LevelInfo),
		)),
		server.WithReflection(true),
		server.WithReflectionServiceNames(
			// Add your service names here, e.g.:
			// "myapp.v1.MyService",
		),
		server.WithDebug(true), // enable pprof in dev
	)

	// Create the application
	app, err := ginny.New(
		ginny.WithName(cfg.App.Name),
		ginny.WithConfig(cfg),
		ginny.WithLogger(logger),
		ginny.WithServer(srv),
	)
	if err != nil {
		logger.Error("failed to create app", slog.Any("error", err))
		os.Exit(1)
	}

	// Start the application (blocks until shutdown)
	ctx := context.Background()
	logger.Info("starting application",
		slog.String("name", app.Name()),
		slog.String("app_addr", cfg.Server.Addr),
		slog.String("admin_addr", cfg.Admin.Addr),
	)
	if err := app.Start(ctx); err != nil {
		logger.Error("application stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("application stopped gracefully")
}
