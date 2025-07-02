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
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/timeout"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

// ClientOptions
type ClientOptions struct {
	target          string // ip+port/serviceName
	timeout         time.Duration
	retryTimes      int
	loadBalance     string
	secure          bool
	metrics         bool
	logger          *zap.Logger
	tracer          opentracing.Tracer
	grpcDialOptions []grpc.DialOption
}

// ClientOptional
type ClientOptional func(o *ClientOptions)

// WithTimeout
func WithTimeout(t time.Duration) ClientOptional {
	return func(o *ClientOptions) {
		if t > 0 {
			o.timeout = t
		}
	}
}

// WithLoadBalance
func WithLoadBalance(loadBalance string) ClientOptional {
	return func(o *ClientOptions) {
		o.loadBalance = loadBalance
	}
}

// WithSecure
func WithSecure(secure bool) ClientOptional {
	return func(o *ClientOptions) {
		o.secure = secure
	}
}

// WithMetrics
func WithMetrics(metrics bool) ClientOptional {
	return func(o *ClientOptions) {
		o.metrics = metrics
	}
}

// WithLogger
func WithLogger(logger *zap.Logger) ClientOptional {
	return func(o *ClientOptions) {
		o.logger = logger
	}
}

// WithTracer
func WithTracer(tracer opentracing.Tracer) ClientOptional {
	return func(o *ClientOptions) {
		o.tracer = tracer
	}
}

// WithResolver
func WithResolver(r resolver.Builder) ClientOptional {
	return func(o *ClientOptions) {
		resolver.Register(r)
	}
}

// WithDialOptions
func WithDialOptions(options ...grpc.DialOption) ClientOptional {
	return func(o *ClientOptions) {
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

// NewClient 参数 bNewXxxClient 对应 pb.NewXxxClient 方法
func NewClient(ctx context.Context, uri string, pbNewXxxClient interface{},
	opts ...ClientOptional) (interface{}, error) {
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
	conn, err := newClientConn(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("connect %s error for %w", u.String(), err)
	}
	ret := reflect.ValueOf(pbNewXxxClient).Call([]reflect.Value{reflect.ValueOf(conn)})
	client := ret[0].Interface()

	return client, nil
}

// newClientConn
func newClientConn(ctx context.Context, opt *ClientOptions) (*grpc.ClientConn, error) {
	var unaryInterceptor = []grpc.UnaryClientInterceptor{}
	var streamInterceptor = []grpc.StreamClientInterceptor{}
	// logger
	if opt.logger != nil {
		logger := logging.InterceptorLogger(opt.logger)
		unaryInterceptor = append(unaryInterceptor,
			grpc_logging.UnaryClientInterceptor(logger, grpc_logging.WithLogOnEvents(grpc_logging.FinishCall)))
		streamInterceptor = append(streamInterceptor,
			grpc_logging.StreamClientInterceptor(logger, grpc_logging.WithLogOnEvents(grpc_logging.FinishCall)))
	}
	// timeout
	if opt.timeout > 0 {
		unaryInterceptor = append(unaryInterceptor,
			timeout.UnaryClientInterceptor(opt.timeout))
	}
	// retry
	if opt.retryTimes > 0 {
		retryOpts := []grpc_retry.CallOption{
			grpc_retry.WithMax(uint(opt.retryTimes)),
			grpc_retry.WithCodes(codes.Unavailable),
			grpc_retry.WithBackoff(func(_ context.Context, _ uint) time.Duration {
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
		grpc.WithInitialWindowSize(InitialWindowSize),
		grpc.WithInitialConnWindowSize(InitialConnWindowSize),
		grpc.WithDefaultServiceConfig(loadBalanceConfig),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // send pings every 1 seconds if there is no activity
			Timeout:             3 * time.Second,  // wait 500 millisecond for ping ack before considering the connection dead
			PermitWithoutStream: true,             // send pings even without active streams
		}),
		grpc.WithConnectParams(
			grpc.ConnectParams{
				Backoff: backoff.Config{
					MaxDelay: BackoffMaxDelay,
				},
			},
		),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(MaxSendMsgSize),
			grpc.MaxCallRecvMsgSize(MaxRecvMsgSize),
		),

		grpc.WithChainUnaryInterceptor(unaryInterceptor...),
		grpc.WithChainStreamInterceptor(streamInterceptor...),
	)

	if opt.tracer != nil {
		opt.grpcDialOptions = append(opt.grpcDialOptions,
			grpc.WithChainUnaryInterceptor(
				interceptor.TracerUnaryClientInterceptor(opt.tracer),
			),
			grpc.WithChainStreamInterceptor(
				interceptor.TracerClientStreamInterceptor(opt.tracer),
			),
		)
	} else {
		opt.grpcDialOptions = append(opt.grpcDialOptions,
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

func evaluateOptions(ctx context.Context, u *url.URL, opts []ClientOptional) (*ClientOptions, error) {
	opt := &ClientOptions{}
	for _, o := range opts {
		o(opt)
	}
	if opt.loadBalance == "" {
		opt.loadBalance = roundrobin.Name
	}
	query := u.Query()
	if try, err := strconv.Atoi(query.Get("retry")); err == nil {
		opt.retryTimes = try
	}

	falseStr := "false"
	opt.secure = query.Get("secure") != falseStr
	opt.metrics = query.Get("metrics") != falseStr
	opt.timeout, _ = time.ParseDuration(query.Get("timeout"))
	opt.target = u.String()
	if u.Scheme == "grpc" || u.Scheme == "http" {
		opt.target = u.Host
	}

	return opt, nil
}
