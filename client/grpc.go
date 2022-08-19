package client

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/goriller/ginny-util/graceful"
	"github.com/goriller/ginny/interceptor"
	"github.com/goriller/ginny/interceptor/logging"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GrpcClientOptions
type GrpcClientOptions struct {
	target          string // ip+port/serviceName
	timeout         time.Duration
	retry           int
	loadBalance     string
	secure          bool
	metrics         bool
	resolver        Resolver
	logger          *zap.Logger
	tracer          opentracing.Tracer
	grpcDialOptions []grpc.DialOption
}

// GrpcClientOptional
type GrpcClientOptional func(o *GrpcClientOptions)

// WithTimeout
func WithTimeout(d time.Duration) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.timeout = d
	}
}

// WithLoadBalance
func WithLoadBalance(loadBalance string) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.loadBalance = loadBalance
	}
}

// WithSecure
func WithSecure(secure bool) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.secure = secure
	}
}

// WithMetrics
func WithMetrics(metrics bool) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.metrics = metrics
	}
}

// WithGrpcLogger
func WithGrpcLogger(logger *zap.Logger) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.logger = logger
	}
}

// WithGrpcTracer
func WithGrpcTracer(tracer opentracing.Tracer) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.tracer = tracer
	}
}

// WithGrpcResolver
func WithGrpcResolver(resolver Resolver) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.resolver = resolver
	}
}

// WithGrpcDialOptions
func WithGrpcDialOptions(options ...grpc.DialOption) GrpcClientOptional {
	return func(o *GrpcClientOptions) {
		o.grpcDialOptions = append(o.grpcDialOptions, options...)
	}
}

// BindMetadataForContext
func BindMetadataForContext(ctx context.Context, data map[string]string) context.Context {
	headersIn, _ := metadata.FromIncomingContext(ctx)
	for k, v := range data {
		headersIn.Set(k, v)
	}
	cc := metadata.NewOutgoingContext(ctx, headersIn)
	return cc
}

// NewGrpcClient 参数 bNewXxxClient 对应 pb.NewXxxClient 方法
func NewGrpcClient(ctx context.Context, uri string, pbNewXxxClient interface{},
	opts ...GrpcClientOptional) (interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("parse grpc uri %s error for %w", uri, err)
	}
	opt, err := evaluateOptions(ctx, u, opts)
	if err != nil {
		return nil, err
	}

	t := reflect.TypeOf(pbNewXxxClient)
	var isGRPCCreator bool
	if t.NumOut() == 1 && t.NumIn() == 1 {
		if t.In(0) == reflect.TypeOf((*grpc.ClientConnInterface)(nil)).Elem() {
			isGRPCCreator = true
		}
	}
	if !isGRPCCreator {
		return nil, fmt.Errorf("function %s is not grpc creator", pbNewXxxClient)
	}
	conn, err := newGrpcClientConn(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("connect %s error for %w", u.String(), err)
	}
	ret := reflect.ValueOf(pbNewXxxClient).Call([]reflect.Value{reflect.ValueOf(conn)})
	client := ret[0].Interface()

	return client, nil
}

// newGrpcClientConn
func newGrpcClientConn(ctx context.Context, opt *GrpcClientOptions) (*grpc.ClientConn, error) {
	var unaryInterceptor = []grpc.UnaryClientInterceptor{}
	var streamInterceptor = []grpc.StreamClientInterceptor{}
	// logger
	if opt.logger != nil {
		logger := grpc_zap.InterceptorLogger(opt.logger)
		unaryInterceptor = append(unaryInterceptor,
			logging.UnaryClientInterceptor(logger))
		streamInterceptor = append(streamInterceptor,
			logging.StreamClientInterceptor(logger))
	}
	// timeout
	if opt.timeout > 0 {
		unaryInterceptor = append(unaryInterceptor,
			timeout.TimeoutUnaryClientInterceptor(opt.timeout))
	}
	// retry
	if opt.retry > 0 {
		retryOpts := []grpc_retry.CallOption{
			grpc_retry.WithMax(uint(opt.retry)),
			grpc_retry.WithCodes(codes.Unavailable),
			grpc_retry.WithBackoff(func(_ uint) time.Duration {
				return time.Second
			}),
		}
		unaryInterceptor = append(unaryInterceptor, grpc_retry.UnaryClientInterceptor(retryOpts...))
		streamInterceptor = append(streamInterceptor, grpc_retry.StreamClientInterceptor(retryOpts...))
	}
	// metrics
	if opt.metrics {
		grpc_prometheus.EnableClientHandlingTimeHistogram()
		grpc_prometheus.EnableClientStreamReceiveTimeHistogram()
		grpc_prometheus.EnableClientStreamSendTimeHistogram()
		unaryInterceptor = append(unaryInterceptor, grpc_prometheus.UnaryClientInterceptor)
		streamInterceptor = append(streamInterceptor, grpc_prometheus.StreamClientInterceptor)
	}
	// secure
	if opt.secure {
		opt.grpcDialOptions = append(opt.grpcDialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	loadBalanceConfig := fmt.Sprintf(`{"LoadBalancingPolicy":"%s"}`, opt.loadBalance)
	opt.grpcDialOptions = append(opt.grpcDialOptions,
		grpc.WithDefaultServiceConfig(loadBalanceConfig),
		grpc.WithChainUnaryInterceptor(unaryInterceptor...),
		grpc.WithChainStreamInterceptor(streamInterceptor...),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Minute,
			Timeout: time.Second * 30,
		}),
	)

	if opt.tracer != nil {
		opt.grpcDialOptions = append(opt.grpcDialOptions,
			// grpc.WithChainUnaryInterceptor(
			// 	tracing.UnaryClientInterceptor(
			// 		tracing.WithTracer(opt.tracer),
			// 		tracing.WithTraceHeaderName(logging.RequestIDHeader),
			// 	),
			// ),
			// grpc.WithChainStreamInterceptor(
			// 	tracing.StreamClientInterceptor(
			// 		tracing.WithTracer(opt.tracer),
			// 		tracing.WithTraceHeaderName(logging.RequestIDHeader),
			// 	),
			// ),
			grpc.WithChainUnaryInterceptor(
				interceptor.TracerUnaryClientInterceptor(opt.tracer),
			),
			grpc.WithChainStreamInterceptor(
				interceptor.TracerClientStreamInterceptor(opt.tracer),
			),
		)
	} else {
		opt.grpcDialOptions = append(opt.grpcDialOptions,
			// grpc.WithChainUnaryInterceptor(
			// 	tracing.UnaryClientInterceptor(
			// 		tracing.WithTracer(opentracing.GlobalTracer()),
			// 		tracing.WithTraceHeaderName(logging.RequestIDHeader),
			// 	),
			// ),
			// grpc.WithChainStreamInterceptor(
			// 	tracing.StreamClientInterceptor(
			// 		tracing.WithTracer(opentracing.GlobalTracer()),
			// 		tracing.WithTraceHeaderName(logging.RequestIDHeader),
			// 	),
			// ),
			grpc.WithChainUnaryInterceptor(
				interceptor.TracerUnaryClientInterceptor(opentracing.GlobalTracer()),
			),
			grpc.WithChainStreamInterceptor(
				interceptor.TracerClientStreamInterceptor(opentracing.GlobalTracer()),
			),
		)
	}
	if opt.timeout == 0 {
		opt.timeout = time.Second * 10
	}

	conn, err := grpc.DialContext(ctx, opt.target, opt.grpcDialOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "grpc dial error")
	}

	// 添加全局退出时的链接关闭
	graceful.AddCloser(func(ctx context.Context) error {
		return conn.Close()
	})

	return conn, nil
}

func evaluateOptions(ctx context.Context, u *url.URL, opts []GrpcClientOptional) (*GrpcClientOptions, error) {
	opt := &GrpcClientOptions{}
	for _, o := range opts {
		o(opt)
	}
	if opt.loadBalance == "" {
		opt.loadBalance = roundrobin.Name
	}
	query := u.Query()
	try, _ := strconv.Atoi(query.Get("retry"))
	opt.retry = try
	falseStr := "false"
	opt.secure = query.Get("secure") != falseStr
	opt.metrics = query.Get("metrics") != falseStr
	opt.target = u.String()
	if u.Scheme == "grpc" || u.Scheme == "http" {
		opt.target = u.Host
	}

	if opt.resolver != nil {
		addr, err := opt.resolver(ctx, u.String(), "grpc")
		if err != nil {
			return nil, err
		}
		opt.target = addr
	}

	return opt, nil
}
