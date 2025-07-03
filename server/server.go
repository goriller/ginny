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
