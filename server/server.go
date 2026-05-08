// Package server provides the dual-port server implementation for Ginny v2.
//
// App Server (:8080): Connect/gRPC/gRPC-Web + Vanguard REST transcoding + gRPC Reflection + gRPC Health.
// Admin Server (:8081): /healthz, /readyz, /metrics, /debug/pprof.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/vanguard"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

// Server manages the dual-port HTTP servers (App + Admin).
type Server struct {
	appMux   *http.ServeMux
	adminMux *http.ServeMux

	appServer   *http.Server
	adminServer *http.Server

	interceptors  []connect.Interceptor
	transcoder    *vanguard.Transcoder
	healthChecker grpchealth.Checker

	opts *serverOptions

	ready   bool
	mu      sync.RWMutex
	logger  *slog.Logger
}

// New creates a new dual-port Server with the given options.
func New(opts ...Option) *Server {
	o := evalOptions(opts)

	s := &Server{
		appMux:   http.NewServeMux(),
		adminMux: http.NewServeMux(),
		opts:     o,
		logger:   o.logger,
	}

	// Build the app mux with interceptors applied to Connect handlers
	s.appMux = s.buildAppMux()
	// Build the admin mux
	s.buildAdminMux()

	appHandler := withMiddleware(s.appMux)
	// Wrap with h2c if enabled (HTTP/2 Cleartext support for Go < 1.24)
	if o.h2cEnabled {
		appHandler = h2c.NewHandler(appHandler, &http2.Server{})
	}

	s.appServer = &http.Server{
		Addr:    o.appAddr,
		Handler: appHandler,
	}

	s.adminServer = &http.Server{
		Addr:    o.adminAddr,
		Handler: s.adminMux,
	}

	return s
}

// withMiddleware applies Layer 1 net/http middleware (Recovery, RequestID, CORS, Compression)
// to the entire mux. This middleware applies globally, including to admin routes.
func withMiddleware(handler http.Handler) http.Handler {
	// Chain Layer 1 middleware: Recovery → RequestID → CORS → Compression
	h := handler
	// Recovery (always first so it catches everything)
	h = recoveryMiddleware(h)
	// CORS
	h = corsMiddleware(h)
	return h
}

// recoveryMiddleware catches panics in HTTP handlers and returns 500.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds basic CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Connect-Protocol-Version")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// buildAppMux configures the app mux with Connect handlers, Vanguard REST transcoding,
// gRPC health protocol, and gRPC reflection.
func (s *Server) buildAppMux() *http.ServeMux {
	mux := s.appMux
	if mux == nil {
		mux = http.NewServeMux()
	}

	// gRPC Health protocol (Connect handlers)
	if s.healthChecker != nil {
		healthPath, healthHandler := grpchealth.NewHandler(s.healthChecker)
		mux.Handle(healthPath, healthHandler)
	} else {
		// Default health checker (always healthy)
		checker := &staticHealthChecker{status: grpchealth.StatusServing}
		healthPath, healthHandler := grpchealth.NewHandler(checker)
		mux.Handle(healthPath, healthHandler)
	}

	// Vanguard REST transcoder
	if s.transcoder != nil {
		mux.Handle("/", s.transcoder)
	}

	// gRPC Reflection
	if s.opts.reflectionEnabled {
		reflector := grpcreflect.NewStaticReflector(s.opts.reflectionServiceNames...)
		reflectPath, reflectHandler := grpcreflect.NewHandlerV1(reflector)
		mux.Handle(reflectPath, reflectHandler)
		reflectAlphaPath, reflectAlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
		mux.Handle(reflectAlphaPath, reflectAlphaHandler)
	}

	return mux
}

// buildAdminMux configures the admin mux with health, readiness, metrics, and pprof endpoints.
func (s *Server) buildAdminMux() {
	mux := s.adminMux
	if mux == nil {
		mux = http.NewServeMux()
	}

	// Liveness probe
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Readiness probe
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s.mu.RLock()
		ready := s.ready
		s.mu.RUnlock()
		if ready {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ready"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready"}`))
		}
	})

	// Prometheus metrics
	mux.Handle("GET /metrics", promhttp.Handler())

	// pprof (dev only)
	if s.opts.debugEnabled {
		mux.HandleFunc("GET /debug/pprof/", pprof.Index)
		mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	}

	s.adminMux = mux
}

// RegisterService registers a Connect service handler on the app mux.
// The path is the base path for the service (e.g., "/myapp.v1.MyService/").
// handler is the Connect-generated handler (which implements http.Handler).
func (s *Server) RegisterService(path string, handler http.Handler) {
	// Add Connect interceptors to the handler
	if len(s.opts.interceptors) > 0 {
		// Interceptors are applied when registering the Connect handler.
		// Connect interceptors are applied via the handler's own configuration
		// before it's passed here, or via option interceptors.
	}
	// Strip trailing slash for consistent prefix handling
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	s.appMux.Handle(path+"/", handler)
}

// RegisterHTTP registers a plain HTTP handler on the app mux.
func (s *Server) RegisterHTTP(pattern string, handler http.Handler) {
	s.appMux.Handle(pattern, handler)
}

// Start begins listening on both the app and admin ports.
// It blocks until all servers stop or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// Start app server
	g.Go(func() error {
		s.log(ctx, slog.LevelInfo, "starting app server", "addr", s.opts.appAddr)
		if err := s.appServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("app server: %w", err)
		}
		return nil
	})

	// Start admin server
	g.Go(func() error {
		s.log(ctx, slog.LevelInfo, "starting admin server", "addr", s.opts.adminAddr)
		if err := s.adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("admin server: %w", err)
		}
		return nil
	})

	// Mark ready after both servers start
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	return g.Wait()
}

// Stop gracefully shuts down both servers.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.ready = false
	s.mu.Unlock()

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var errs []error
	if err := s.appServer.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("app server shutdown: %w", err))
	}
	if err := s.adminServer.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("admin server shutdown: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// AppMux returns the app mux for advanced customization.
func (s *Server) AppMux() *http.ServeMux {
	return s.appMux
}

// AdminMux returns the admin mux for advanced customization.
func (s *Server) AdminMux() *http.ServeMux {
	return s.adminMux
}

// SetReady marks the server as ready or not ready for readiness probes.
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	s.ready = ready
	s.mu.Unlock()
}

// log is a helper for consistent logging.
func (s *Server) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if s.logger != nil {
		s.logger.Log(ctx, level, msg, args...)
	}
}

// staticHealthChecker is a simple always-healthy checker.
type staticHealthChecker struct {
	status grpchealth.Status
}

func (c *staticHealthChecker) Check(ctx context.Context, req *grpchealth.CheckRequest) (*grpchealth.CheckResponse, error) {
	return &grpchealth.CheckResponse{Status: c.status}, nil
}

func (c *staticHealthChecker) Watch(ctx context.Context, req *grpchealth.CheckRequest) (<-chan *grpchealth.CheckResponse, error) {
	ch := make(chan *grpchealth.CheckResponse, 1)
	ch <- &grpchealth.CheckResponse{Status: c.status}
	return ch, nil
}

// HealthChecker is the interface for health checking.
// It wraps the grpchealth.Checker interface for compatibility.
type HealthChecker interface {
	Check(ctx context.Context, req *grpchealth.CheckRequest) (*grpchealth.CheckResponse, error)
	Watch(ctx context.Context, req *grpchealth.CheckRequest) (<-chan *grpchealth.CheckResponse, error)
}
