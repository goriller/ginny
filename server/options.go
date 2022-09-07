package server

import (
	"context"
	"time"

	"github.com/goriller/ginny/interceptor"
	"github.com/goriller/ginny/interceptor/limit"
	"github.com/goriller/ginny/interceptor/logging"
	"github.com/goriller/ginny/server/mux"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tracing"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type options struct {
	autoHttp bool // 开启http服务
	grpcAddr string
	httpAddr string

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
	ServiceRegister(name, addr string, tags []string, meta map[string]string) error
	ServiceDeregister(name string) error
}

var defaultOptions = &options{
	grpcAddr:                 ":9000",
	httpAddr:                 ":8080",
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
			o.autoHttp = true
		}
	}
}

// WithDiscover
func WithDiscover(d Discover) Option {
	return func(o *options) {
		if d != nil {
			o.discover = d
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
		interceptor.TracerServerUnaryInterceptor(opt.tracer),
	}

	streamServerInterceptors := []grpc.StreamServerInterceptor{
		tags.StreamServerInterceptor(),
		tracing.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		logging.StreamServerInterceptor(
			opt.logger,
			muxLoggingOpts...,
		),
		validator.StreamServerInterceptor(false),
		interceptor.TracerServerStreamInterceptor(opt.tracer),
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
	// if opt.tracer != nil {
	// 	unaryServerInterceptors = append(unaryServerInterceptors,
	// 		tracing.UnaryServerInterceptor(tracing.WithTracer(opt.tracer)))
	// 	streamServerInterceptors = append(streamServerInterceptors,
	// 		tracing.StreamServerInterceptor(tracing.WithTracer(opt.tracer)))
	// } else {
	// 	unaryServerInterceptors = append(unaryServerInterceptors,
	// 		tracing.UnaryServerInterceptor(tracing.WithTracer(opentracing.GlobalTracer())))
	// 	streamServerInterceptors = append(streamServerInterceptors,
	// 		tracing.StreamServerInterceptor(tracing.WithTracer(opentracing.GlobalTracer())))
	// }

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
	)

	if !opt.withOutKeepAliveOpts {
		opt.grpcServerOpts = append(opt.grpcServerOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: time.Minute,
		}))
	}

	return
}
