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

	"github.com/gorillazer/ginny-serve/health"
	"github.com/gorillazer/ginny-serve/mux"
	"github.com/gorillazer/ginny-util/graceful"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	consulApi "github.com/hashicorp/consul/api"
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

// RegistrarFunc
type RegistrarFunc func(s *grpc.Server)

// NewServer new grpc server with all common middleware.
func NewServer(logger *zap.Logger, regFunc RegistrarFunc, opts ...Option) *Server {
	var gs *grpc.Server
	opt := fullOptions(logger, opts...)
	{
		gs = grpc.NewServer(opt.grpcServerOpts...)
		regFunc(gs)
	}
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
	if s.options.consul != nil {
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
	err = s.grpcServer.Serve(lis)
	if err != nil {
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
	err := s.httpServer.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("Start http failed for " + err.Error())
	}
	return nil
}

// RegisterService 注册函数
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
// 默认超时1分钟再关闭，K8S service 关闭策略
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

// Handle 注册自定义HTTP路由
func (s *Server) Handle(method, path string, h runtime.HandlerFunc) {
	s.mux.Handle(method, path, h)
}

// ServeMux 返回grpc gateay 原生的 server mux
func (s *Server) ServeMux() *runtime.ServeMux {
	return s.mux.ServeMux()
}

// register
func (s *Server) register() error {
	if s.options.consul == nil {
		return nil
	}
	i := strings.LastIndex(":", s.options.grpcAddr)
	host := string([]byte(s.options.grpcAddr)[:i])
	port, err := strconv.Atoi(string([]byte(s.options.grpcAddr)[i:]))
	if err != nil {
		return err
	}

	for key, _ := range s.grpcServer.GetServiceInfo() {
		check := &consulApi.AgentServiceCheck{
			Interval:                       "10s",
			DeregisterCriticalServiceAfter: "60m",
			TCP:                            s.options.grpcAddr,
		}

		id := fmt.Sprintf("%s[%s]", key, s.options.grpcAddr)

		svcReg := &consulApi.AgentServiceRegistration{
			ID:                id,
			Name:              key,
			Tags:              []string{"grpc"},
			Port:              port,
			Address:           host,
			EnableTagOverride: true,
			Check:             check,
			Checks:            nil,
		}

		err := s.options.consul.Agent().ServiceRegister(svcReg)
		if err != nil {
			return errors.Wrap(err, "register service error")
		}
		s.logger.Log(logging.INFO, "register grpc service success: "+id)
	}

	return nil
}

// deRegister
func (s *Server) deRegister() error {
	if s.options.consul == nil {
		return nil
	}
	for key, _ := range s.grpcServer.GetServiceInfo() {
		id := fmt.Sprintf("%s[%s]", key, s.options.grpcAddr)

		err := s.options.consul.Agent().ServiceDeregister(id)
		if err != nil {
			return errors.Wrapf(err, "deregister service error[id=%s]", id)
		}
		s.logger.Log(logging.INFO, "deregister service success: "+id)
	}

	return nil
}
