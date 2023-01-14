package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goriller/ginny-util/ip"
	"github.com/goriller/ginny/interceptor"
	"github.com/goriller/ginny/interceptor/limit"
	"github.com/goriller/ginny/interceptor/logging"
	"github.com/goriller/ginny/interceptor/tags"
	"github.com/goriller/ginny/server/mux"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

const (
	// InitialWindowSize we set it 1GB is to provide system's throughput.
	InitialWindowSize = 1 << 30

	// InitialConnWindowSize we set it 1GB is to provide system's throughput.
	InitialConnWindowSize = 1 << 30

	// MaxSendMsgSize set max gRPC request message size sent to server.
	// If any request message size is larger than current value, an error will be reported from gRPC.
	MaxSendMsgSize = 4 << 30

	// MaxRecvMsgSize set max gRPC receive message size received from server.
	// If any message size is larger than current value, an error will be reported from gRPC.
	MaxRecvMsgSize = 4 << 30
)

type options struct {
	grpcAddr    string
	grpcSevAddr string
	httpAddr    string
	httpSevAddr string
	metricsAddr string
	tags        []string // for register service

	discover Discover
	tracer   opentracing.Tracer

	authFunc                   interceptor.Authorize
	logger                     grpc_logging.Logger
	loggingDecider             logging.Decider
	limiter                    *limit.Limiter
	grpcServerOpts             []grpc.ServerOption
	withOutKeepAliveOpts       bool
	muxOptions                 []mux.Optional
	streamServerInterceptors   []grpc.StreamServerInterceptor
	unaryServerInterceptors    []grpc.UnaryServerInterceptor
	requestFieldExtractorFunc  logging.RequestFieldExtractorFunc
	responseFieldExtractorFunc logging.ResponseFieldExtractorFunc
}

// Discover service discovery
type Discover interface {
	ServiceRegister(ctx context.Context, name, addr string, tags []string, meta map[string]string) error
	ServiceDeregister(ctx context.Context, name string) error
}

var defaultOptions = &options{
	grpcAddr:                 ":9000",
	httpAddr:                 ":8080",
	metricsAddr:              ":8081",
	tags:                     []string{},
	muxOptions:               []mux.Optional{},
	grpcServerOpts:           []grpc.ServerOption{},
	streamServerInterceptors: []grpc.StreamServerInterceptor{},
	unaryServerInterceptors:  []grpc.UnaryServerInterceptor{},
}

// Option the option for this module
type Option func(*options)

func evaluateOptions(opts []Option) *options {

	optCopy := &options{}
	*optCopy = *defaultOptions
	for _, o := range opts {
		o(optCopy)
	}

	t := os.Getenv("SERVICE_TAG")
	tags := strings.Split(t, ",")
	optCopy.tags = append(optCopy.tags, tags...)

	localIp := ip.GetLocalIP4()
	httpAddrs := strings.Split(optCopy.httpAddr, ":")
	if len(httpAddrs) == 2 {
		host := httpAddrs[0]
		if host == "" {
			host = localIp
		}
		optCopy.httpSevAddr = fmt.Sprintf("http://%s:%s", host, httpAddrs[1])
	}

	grpcAddrs := strings.Split(optCopy.grpcAddr, ":")
	if len(grpcAddrs) == 2 {
		host := grpcAddrs[0]
		if host == "" {
			host = localIp
		}
		optCopy.grpcSevAddr = fmt.Sprintf("grpc://%s:%s", host, grpcAddrs[1])
	}

	return optCopy
}

// WithLogger
func WithLogger(logger grpc_logging.Logger) Option {
	return func(o *options) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// WithTags
func WithTags(tags []string) Option {
	return func(o *options) {
		if len(tags) > 0 {
			o.tags = tags
		}
	}
}

// WithGrpcAddr
func WithGrpcAddr(addr string) Option {
	return func(o *options) {
		if addr != "" {
			o.grpcAddr = addr
		}
	}
}

// WithHttpAddr
func WithHttpAddr(addr string) Option {
	return func(o *options) {
		if addr != "" {
			o.httpAddr = addr
		}
	}
}

// WithMetricsAddr
func WithMetricsAddr(addr string) Option {
	return func(o *options) {
		if addr != "" {
			o.metricsAddr = addr
		}
	}
}

// WithDiscover
func WithDiscover(d Discover, tags ...string) Option {
	return func(o *options) {
		if d != nil {
			o.discover = d
		}
		if len(tags) > 0 {
			o.tags = append(o.tags, tags...)
		}
	}
}

// WithTracer
func WithTracer(tracer opentracing.Tracer) Option {
	return func(o *options) {
		if tracer != nil {
			o.tracer = tracer
		}
	}
}

// WithLoggingDecider Decider how log output.
func WithLoggingDecider(decider logging.Decider) Option {
	return func(o *options) {
		if decider != nil {
			o.loggingDecider = decider
		}
	}
}

// WithLimiter performs rate limiting on the request.
func WithLimiter(l *limit.Limiter) Option {
	return func(o *options) {
		o.limiter = l
	}
}

// WithAuthFunc
func WithAuthFunc(a interceptor.Authorize) Option {
	return func(o *options) {
		o.authFunc = a
	}
}

// WithStreamServerInterceptor
func WithStreamServerInterceptor(f grpc.StreamServerInterceptor) Option {
	return func(o *options) {
		if f != nil {
			o.streamServerInterceptors = append(o.streamServerInterceptors, f)
		}
	}
}

// WithUnaryServerInterceptor
func WithUnaryServerInterceptor(f grpc.UnaryServerInterceptor) Option {
	return func(o *options) {
		if f != nil {
			o.unaryServerInterceptors = append(o.unaryServerInterceptors, f)
		}
	}
}

// WithRequestFieldExtractor customizes the function for extracting log fields from protobuf messages, for
// unary and server-streamed methods only.
func WithRequestFieldExtractor(f logging.RequestFieldExtractorFunc) Option {
	return func(o *options) {
		if f != nil {
			o.requestFieldExtractorFunc = f
		}
	}
}

// WithResponseFieldExtractor customizes the function for extracting log fields from protobuf messages, for
// unary and server-streamed methods only.
func WithResponseFieldExtractor(f logging.ResponseFieldExtractorFunc) Option {
	return func(o *options) {
		if f != nil {
			o.responseFieldExtractorFunc = f
		}
	}
}

// WithGRPCServerOption with other grpc options
func WithGrpcServerOption(opts ...grpc.ServerOption) Option {
	return func(o *options) {
		if len(opts) > 0 {
			o.grpcServerOpts = append(o.grpcServerOpts, opts...)
		}
	}
}

// WithHTTPServerOption with http server options
func WithHttpServerOption(opts ...mux.Optional) Option {
	return func(o *options) {
		if len(opts) > 0 {
			o.muxOptions = append(o.muxOptions, opts...)
		}
	}
}

// fullOptions
func fullOptions(logger *zap.Logger,
	opts ...Option) (opt *options) {
	opt = evaluateOptions(opts)
	if opt.logger == nil {
		opt.logger = grpc_zap.InterceptorLogger(logger)
	}

	muxLoggingOpts := []logging.Option{}
	if opt.loggingDecider != nil {
		muxLoggingOpts = append(muxLoggingOpts,
			logging.WithDecider(opt.loggingDecider))
	}
	if opt.requestFieldExtractorFunc != nil {
		muxLoggingOpts = append(muxLoggingOpts,
			logging.WithRequestFieldExtractorFunc(opt.requestFieldExtractorFunc))
	}
	if opt.responseFieldExtractorFunc != nil {
		muxLoggingOpts = append(muxLoggingOpts,
			logging.WithResponseFieldExtractorFunc(opt.responseFieldExtractorFunc))
	}

	unaryServerInterceptors := []grpc.UnaryServerInterceptor{
		tags.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		logging.UnaryServerInterceptor(
			opt.logger,
			muxLoggingOpts...,
		),
		validator.UnaryServerInterceptor(false),
	}

	streamServerInterceptors := []grpc.StreamServerInterceptor{
		tags.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		logging.StreamServerInterceptor(
			opt.logger,
			muxLoggingOpts...,
		),
		validator.StreamServerInterceptor(false),
	}

	// tracer
	if opt.tracer != nil {
		opt.muxOptions = append(opt.muxOptions, mux.WithTracer(opt.tracer))
		unaryServerInterceptors = append(unaryServerInterceptors,
			interceptor.TracerServerUnaryInterceptor(opt.tracer))
		streamServerInterceptors = append(streamServerInterceptors,
			interceptor.TracerServerStreamInterceptor(opt.tracer))
	}
	// limiter
	if opt.limiter != nil {
		opt.muxOptions = append(opt.muxOptions, mux.WithLimiter(opt.limiter))
		unaryServerInterceptors = append(unaryServerInterceptors,
			limit.UnaryServerInterceptor(opt.limiter))
		streamServerInterceptors = append(streamServerInterceptors,
			limit.StreamServerInterceptor(opt.limiter))
	}

	// auth
	if opt.authFunc != nil {
		opt.muxOptions = append(opt.muxOptions, mux.WithAuthFunc(opt.authFunc))
		unaryServerInterceptors = append(unaryServerInterceptors,
			interceptor.AuthUnaryServerInterceptor(opt.authFunc))
		streamServerInterceptors = append(streamServerInterceptors,
			interceptor.AuthStreamServerInterceptor(opt.authFunc))
	}

	if len(opt.unaryServerInterceptors) > 0 {
		unaryServerInterceptors = append(unaryServerInterceptors, opt.unaryServerInterceptors...)
	}
	if len(opt.streamServerInterceptors) > 0 {
		streamServerInterceptors = append(streamServerInterceptors, opt.streamServerInterceptors...)
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	recoverFunc := func(ctx context.Context, p interface{}) (err error) {
		tags.Extract(ctx).Set("stacktrace", zap.StackSkip("", 4).String)
		return status.Errorf(codes.Internal, "%s", p)
	}

	unaryServerInterceptors = append(unaryServerInterceptors,
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(recoverFunc)))
	streamServerInterceptors = append(streamServerInterceptors,
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandlerContext(recoverFunc)))

	opt.grpcServerOpts = append(opt.grpcServerOpts,
		grpc.ChainStreamInterceptor(streamServerInterceptors...),
		grpc.ChainUnaryInterceptor(unaryServerInterceptors...),
		grpc.MaxSendMsgSize(MaxSendMsgSize),
		grpc.MaxRecvMsgSize(MaxRecvMsgSize),
		grpc.InitialWindowSize(InitialWindowSize),
		grpc.InitialConnWindowSize(InitialConnWindowSize),
	)

	if !opt.withOutKeepAliveOpts {
		opt.grpcServerOpts = append(opt.grpcServerOpts,
			grpc.KeepaliveParams(
				keepalive.ServerParameters{
					MaxConnectionAge: time.Minute,
					Time:             time.Second * 10,
					Timeout:          time.Second * 3,
				}),
			grpc.KeepaliveEnforcementPolicy(
				keepalive.EnforcementPolicy{
					PermitWithoutStream: true,
				},
			),
		)
	}

	return
}
