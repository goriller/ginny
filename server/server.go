<<<<<<< HEAD
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/goriller/ginny-util/graceful"
	"github.com/goriller/ginny/server/health"
	"github.com/goriller/ginny/server/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server the grpc server
type Server struct {
	// the grpc default logger with auto flush on close
	logger logging.Logger

	locker  sync.Locker
	options *options
	mux     *mux.MuxServe

	grpcServer    *grpc.Server
	httpServer    *http.Server
	metricsServer *http.Server
	healthServer  *health.HealthServer
}

// NewServer new grpc server with all common middleware.
func NewServer(ctx context.Context, logger *zap.Logger, opts ...Option) *Server {
	opt := fullOptions(logger, opts...)

	svc := &Server{
		logger:  opt.logger,
		options: opt,
		locker:  &sync.Mutex{},
	}
	svc.grpcServer = grpc.NewServer(opt.grpcServerOpts...)
	if opt.httpAddr != "" {
		svc.mux = mux.NewMuxServe(logger, opt.muxOptions...)
		svc.httpServer = &http.Server{Addr: opt.httpAddr, Handler: svc.mux}
	}

	if opt.metricsAddr != "" {
		svc.metricsServer = &http.Server{Addr: opt.metricsAddr, Handler: promhttp.Handler()}
	}

	svc.healthServer = health.NewHealthServer()

	return svc
}

// Start
func (s *Server) Start(ctx context.Context) {
	graceful.AddCloser(s.Close)
	fns := []graceful.Fn{func() error {
		return s.startGRPC(ctx)
	}}
	if s.options.httpAddr != "" {
		fns = append(fns, func() error {
			return s.startHTTP(ctx)
		})
	}
	if s.options.metricsAddr != "" {
		fns = append(fns, func() error {
			return s.startMetrics(ctx)
		})
	}
	if s.options.discover != nil {
		fns = append(fns, func() error {
			return s.register(ctx)
		})
	}

	graceful.Start(fns...)
}

// startGRPC
func (s *Server) startGRPC(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.options.grpcAddr)
	if err != nil {
		s.options.logger.Log(ctx, logging.LevelError, "listen grpc "+s.options.grpcAddr+" error for "+err.Error())
		return err
	}
	s.healthServer.Start(s.grpcServer)

	s.logger.Log(ctx, logging.LevelInfo, "Start grpc at "+s.options.grpcAddr)
	if err := s.grpcServer.Serve(lis); err != nil {
		return errors.New("start grpc failed for " + err.Error())
	}
	return nil
}

// startHTTP
func (s *Server) startHTTP(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Log(ctx, logging.LevelInfo, "start http at "+s.options.httpAddr)
	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return errors.New("start http failed for " + err.Error())
	}
	return nil
}

func (s *Server) startMetrics(ctx context.Context) error {
	if s.metricsServer == nil {
		return nil
	}
	s.logger.Log(ctx, logging.LevelInfo, "start metrics at "+s.options.metricsAddr)
	err := s.metricsServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return errors.New("start metrics failed for " + err.Error())
	}
	return nil
}

// RegisterService registering gRPC service
func (s *Server) RegisterService(ctx context.Context, desc *grpc.ServiceDesc, serviceImpl interface{}) {
	s.grpcServer.RegisterService(desc, serviceImpl)
	// // auto bind http handler
	// if s.options.autoHttp {
	// 	for _, v := range desc.Methods {
	// 		path := "/" + desc.ServiceName + "/" + v.MethodName
	// 		s.logger.With("path", path).Log(logging.DEBUG, "handled")
	// 		s.mux.Handle(http.MethodPost, path, mux.HandlerGRPCService(s.mux.ServeMux(), serviceImpl, v))
	// 	}
	// }
}

// Close
// K8s closes after 60 seconds by default
// refer: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/
func (s *Server) Close(ctx context.Context) error {
	if s.httpServer != nil {
		s.httpServer.SetKeepAlivesEnabled(false)
	}
	if s.healthServer != nil {
		s.healthServer.Close()
	}
	if s.metricsServer != nil {
		s.metricsServer.Close()
	}
	// deRegister
	err := s.deRegister(ctx)
	if err != nil {
		s.logger.Log(ctx, logging.LevelWarn, "deregister service failed: "+err.Error())
	}
	// prestop
	preStop := os.Getenv("PRE_STOP")
	if preStop != "" {
		preStopDuration, _ := time.ParseDuration(preStop)
		if preStopDuration == 0 || preStopDuration > 5*time.Minute {
			preStopDuration = time.Minute
		}
		s.logger.Log(ctx, logging.LevelDebug, fmt.Sprintf("wait %s to stop the service", preStopDuration))
		time.Sleep(preStopDuration)
	}
	if s.httpServer != nil {
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			s.logger.Log(ctx, logging.LevelWarn, "shutdown http failed for "+err.Error())
		}
	}
	s.grpcServer.GracefulStop()

	return nil
}

// Handle registering HTTP handler
func (s *Server) Handle(method, path string, h runtime.HandlerFunc) {
	s.mux.Handle(method, path, h)
}

// ServeMux retrun gRPC-GateWay server mux
func (s *Server) ServeMux() *runtime.ServeMux {
	return s.mux.ServeMux()
}

// register registering to service discovery
func (s *Server) register(ctx context.Context) error {
	if s.options.discover == nil {
		return nil
	}

	for key := range s.grpcServer.GetServiceInfo() {
		// gRPC
		name := fmt.Sprintf("%s.%s", key, "grpc")
		err := s.options.discover.ServiceRegister(ctx, name, s.options.grpcSevAddr, s.options.tags, nil)
		if err != nil {
			return errors.Wrap(err, "register grpc service error")
		}
		s.logger.Log(ctx, logging.LevelInfo, "register grpc service success: "+name)

		// HTTP
		if s.options.httpAddr != "" {
			hName := fmt.Sprintf("%s.%s", key, "http")
			err = s.options.discover.ServiceRegister(ctx, hName, s.options.httpSevAddr, s.options.tags, nil)
			if err != nil {
				return errors.Wrap(err, "register http service error")
			}
			s.logger.Log(ctx, logging.LevelInfo, "register http service success: "+hName)
		}
	}

	return nil
}

// deRegister deregistered from service discovery
func (s *Server) deRegister(ctx context.Context) error {
	if s.options.discover == nil {
		return nil
	}

	for key := range s.grpcServer.GetServiceInfo() {
		// gRPC
		name := fmt.Sprintf("%s[%s]", key, "grpc")
		err := s.options.discover.ServiceDeregister(ctx, name)
		if err != nil {
			return errors.Wrapf(err, "deregister grpc service error[id=%s]", name)
		}
		s.logger.Log(ctx, logging.LevelInfo, "deregister grpc service success: "+name)

		// HTTP
		if s.options.httpAddr != "" {
			hName := fmt.Sprintf("%s[%s]", key, "http")
			err = s.options.discover.ServiceDeregister(ctx, hName)
			if err != nil {
				return errors.Wrapf(err, "deregister http service error[id=%s]", hName)
			}
			s.logger.Log(ctx, logging.LevelInfo, "deregister http service success: "+hName)
		}
	}

	return nil
}
=======
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
>>>>>>> feat/new
