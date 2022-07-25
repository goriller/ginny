package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goriller/ginny-util/graceful"
	"github.com/goriller/ginny/health"
	"github.com/goriller/ginny/server/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
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

	grpcServer   *grpc.Server
	httpServer   *http.Server
	healthServer *health.HealthServer
}

// NewServer new grpc server with all common middleware.
func NewServer(logger *zap.Logger, opts ...Option) *Server {
	opt := fullOptions(logger, opts...)

	svc := &Server{
		logger:  opt.logger,
		options: opt,
		locker:  &sync.Mutex{},
	}
	svc.grpcServer = grpc.NewServer(opt.grpcServerOpts...)
	if opt.autoHttp {
		svc.mux = mux.NewMuxServe(logger, opt.muxOptions...)
		svc.httpServer = &http.Server{Addr: opt.httpAddr, Handler: svc.mux}
	}
	svc.healthServer = health.NewHealthServer()

	return svc
}

// Start
func (s *Server) Start() {
	graceful.AddCloser(s.Close)
	fns := []graceful.Fn{s.startGRPC}
	if s.options.autoHttp {
		fns = append(fns, s.startHTTP)
	}
	if s.options.discover != nil {
		fns = append(fns, s.register)
	}

	graceful.Start(fns...)
}

// startGRPC
func (s *Server) startGRPC() error {
	lis, err := net.Listen("tcp", s.options.grpcAddr)
	if err != nil {
		s.options.logger.Log(logging.ERROR, "Listen grpc "+s.options.grpcAddr+" error for "+err.Error())
		return err
	}
	s.healthServer.Start(s.grpcServer)

	s.logger.Log(logging.INFO, "Start grpc at "+s.options.grpcAddr)
	if err := s.grpcServer.Serve(lis); err != nil {
		return errors.New("Start grpc failed for " + err.Error())
	}
	return nil
}

// startHTTP
func (s *Server) startHTTP() error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Log(logging.INFO, "Start http at "+s.options.httpAddr)
	if err := s.httpServer.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("Start http failed for " + err.Error())
	}
	return nil
}

// RegisterService registering gRPC service
func (s *Server) RegisterService(desc *grpc.ServiceDesc, serviceImpl interface{}) {
	s.grpcServer.RegisterService(desc, serviceImpl)
	// auto bind http handler
	if s.options.autoHttp {
		for _, v := range desc.Methods {
			path := "/" + desc.ServiceName + "/" + v.MethodName
			s.logger.With("path", path).Log(logging.DEBUG, "handled")
			s.mux.Handle(http.MethodPost, path, mux.HandlerGRPCService(s.mux.ServeMux(), serviceImpl, v))
		}
	}
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
	// deRegister
	err := s.deRegister()
	if err != nil {
		s.logger.Log(logging.WARNING, "DeRegister service failed: "+err.Error())
	}
	// prestop
	preStop := os.Getenv("PRE_STOP")
	if preStop != "" {
		preStopDuration, _ := time.ParseDuration(preStop)
		if preStopDuration == 0 || preStopDuration > 5*time.Minute {
			preStopDuration = time.Minute
		}
		s.logger.Log(logging.DEBUG, fmt.Sprintf("wait %s to stop the service", preStopDuration))
		time.Sleep(preStopDuration)
	}
	if s.httpServer != nil {
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			s.logger.Log(logging.WARNING, "Shutdown http failed for "+err.Error())
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
func (s *Server) register() error {
	if s.options.discover == nil {
		return nil
	}

	i := strings.LastIndex(":", s.options.grpcAddr)
	host := string([]byte(s.options.grpcAddr)[:i])
	port, err := strconv.Atoi(string([]byte(s.options.grpcAddr)[i:]))
	if err != nil {
		return err
	}
	// gRPC
	for key := range s.grpcServer.GetServiceInfo() {
		id := fmt.Sprintf("%s[%s]", key, s.options.grpcAddr)
		err := s.options.discover.ServiceRegister(id, key, host, port, []string{"grpc"}, nil)
		if err != nil {
			return errors.Wrap(err, "register service error")
		}
		s.logger.Log(logging.INFO, "register grpc service success: "+id)
	}
	// HTTP server
	if s.options.autoHttp {
		id := fmt.Sprintf("%s[%s]", "http", s.options.httpAddr)
		i := strings.LastIndex(":", s.options.grpcAddr)
		host := string([]byte(s.options.httpAddr)[:i])
		port, err := strconv.Atoi(string([]byte(s.options.httpAddr)[i:]))
		if err != nil {
			return err
		}
		err = s.options.discover.ServiceRegister(id, s.options.httpAddr, host, port, []string{"http"}, nil)
		if err != nil {
			return errors.Wrap(err, "register http server error")
		}
		s.logger.Log(logging.INFO, "register http server success: "+id)
	}

	return nil
}

// deRegister deregistering from service discovery
func (s *Server) deRegister() error {
	if s.options.discover == nil {
		return nil
	}
	// gRPC
	for key := range s.grpcServer.GetServiceInfo() {
		id := fmt.Sprintf("%s[%s]", key, s.options.grpcAddr)

		err := s.options.discover.ServiceDeregister(id)
		if err != nil {
			return errors.Wrapf(err, "deregister service error[id=%s]", id)
		}
		s.logger.Log(logging.INFO, "deregister service success: "+id)
	}

	// HTTP server
	if s.options.autoHttp {
		id := fmt.Sprintf("%s[%s]", "http", s.options.httpAddr)
		err := s.options.discover.ServiceDeregister(id)
		if err != nil {
			return errors.Wrapf(err, "deregister http server error[id=%s]", id)
		}
		s.logger.Log(logging.INFO, "deregister http server success: "+id)
	}

	return nil
}
