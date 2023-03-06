package mux

import (
	"strings"

	"github.com/goriller/ginny/interceptor"
	"github.com/goriller/ginny/interceptor/limit"
	"github.com/goriller/ginny/middleware"
	"github.com/goriller/ginny/server/mux/rewriter"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// MuxOption
type MuxOption struct {
	authFunc          interceptor.Authorize
	logger            logging.Logger
	tracer            opentracing.Tracer
	limiter           *limit.Limiter
	bodyMarshaler     runtime.Marshaler
	bodyWriter        rewriter.BodyReWriterFunc
	errorMarshaler    runtime.Marshaler
	errorHandler      runtime.ErrorHandlerFunc
	runTimeOpts       []runtime.ServeMuxOption
	withoutHTTPStatus bool
	middleWares       []middleware.MuxMiddleware
}

var (
	defaultMarshalOptions = protojson.MarshalOptions{
		Multiline:       false,
		Indent:          "",
		AllowPartial:    false,
		UseProtoNames:   true,
		UseEnumNumbers:  false,
		EmitUnpopulated: true,
	}

	defaultMarshaler = &runtime.HTTPBodyMarshaler{
		Marshaler: &runtime.JSONPb{
			MarshalOptions: defaultMarshalOptions,
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		},
	}

	defaultMuxOption = &MuxOption{
		bodyMarshaler:     defaultMarshaler,
		errorMarshaler:    defaultMarshaler,
		withoutHTTPStatus: true,
		middleWares:       []middleware.MuxMiddleware{},
		runTimeOpts:       []runtime.ServeMuxOption{},
	}
)

// Optional the Options for this module
type Optional func(*MuxOption)

// WithErrorHandler
func WithErrorHandler(fn runtime.ErrorHandlerFunc) Optional {
	return func(o *MuxOption) {
		if fn != nil {
			o.errorHandler = fn
		}
	}
}

// WithBodyWriter
func WithBodyWriter(b rewriter.BodyReWriterFunc) Optional {
	return func(o *MuxOption) {
		if b != nil {
			o.bodyWriter = b
		}
	}
}

// WithBodyMarshaler
func WithBodyMarshaler(ms runtime.Marshaler) Optional {
	return func(o *MuxOption) {
		if ms != nil {
			o.bodyMarshaler = ms
		}
	}
}

// WithErrorMarshaler
func WithErrorMarshaler(ms runtime.Marshaler) Optional {
	return func(o *MuxOption) {
		if ms != nil {
			o.errorMarshaler = ms
		}
	}
}

// WithRunTimeOpts with runtime MuxOption
func WithRunTimeOpts(opts runtime.ServeMuxOption) Optional {
	return func(o *MuxOption) {
		if opts != nil {
			o.runTimeOpts = append(o.runTimeOpts, opts)
		}
	}
}

// WithoutHTTPStatus pluggable function that performs if use http status on response.
func WithoutHTTPStatus() Optional {
	return func(o *MuxOption) {
		o.withoutHTTPStatus = true
	}
}

// WithTracer
func WithTracer(tracer opentracing.Tracer) Optional {
	return func(o *MuxOption) {
		if tracer != nil {
			o.tracer = tracer
		}
	}
}

// WithLimiter performs rate limiting on the request.
func WithLimiter(l *limit.Limiter) Optional {
	return func(o *MuxOption) {
		o.limiter = l
	}
}

// WithAuthFunc
func WithAuthFunc(a interceptor.Authorize) Optional {
	return func(o *MuxOption) {
		o.authFunc = a
	}
}

// WithMiddleWares pluggable function that performs middle wares.
func WithMiddleWares(middleWares ...middleware.MuxMiddleware) Optional {
	return func(o *MuxOption) {
		if len(middleWares) > 0 {
			o.middleWares = append(o.middleWares, middleWares...)
		}
	}
}

// fullOptions
func fullOptions(logger *zap.Logger,
	opts ...Optional) (opt *MuxOption) {
	o := evaluateOptions(opts)
	o.logger = grpc_zap.InterceptorLogger(logger)
	if o.errorHandler == nil {
		o.errorHandler = defaultErrorHandler
	}
	if o.bodyWriter == nil {
		o.bodyWriter = rewriter.DefaultBodyWriter(o.bodyMarshaler, o.bodyMarshaler, o.withoutHTTPStatus)
	}

	// tracer
	// if o.tracer != nil {
	o.middleWares = append(o.middleWares,
		middleware.TracerMiddleWare(o.tracer))
	// }

	// limiter
	if o.limiter != nil {
		o.middleWares = append(o.middleWares,
			middleware.LimitMiddleWare(o.limiter))
	}

	// auth
	if o.authFunc != nil {
		o.middleWares = append(o.middleWares,
			middleware.AuthMiddleWare(o.authFunc))
	}

	runtimeOpt := []runtime.ServeMuxOption{
		runtime.WithIncomingHeaderMatcher(func(s string) (string, bool) {
			return strings.ToLower(s), true
		}),
		runtime.WithOutgoingHeaderMatcher(func(s string) (string, bool) {
			return "", false
		}),
		runtime.WithErrorHandler(o.errorHandler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, o.bodyMarshaler),
		runtime.WithForwardResponseOption(forwardResponseOptionFunc),
	}
	o.runTimeOpts = append(o.runTimeOpts, runtimeOpt...)

	defaultMuxOption = o
	return o
}

func evaluateOptions(opts []Optional) *MuxOption {
	optCopy := &MuxOption{}
	*optCopy = *defaultMuxOption
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}
