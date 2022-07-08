package mux

import (
	"context"
	"net/http"
	"strings"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MuxMiddleware
type MuxMiddleware func(http.Handler) http.Handler

// MuxOption
type MuxOption struct {
	logger            logging.Logger
	bodyMarshaler     runtime.Marshaler
	bodyWriter        bodyReWriterFunc
	errorMarshaler    runtime.Marshaler
	errorHandler      runtime.ErrorHandlerFunc
	runTimeOpts       []runtime.ServeMuxOption
	withoutHTTPStatus bool
	middleWares       []MuxMiddleware
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
		middleWares:       []MuxMiddleware{},
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
func WithBodyWriter(b bodyReWriterFunc) Optional {
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

// WithMiddleWares pluggable function that performs middle wares.
func WithMiddleWares(middleWares ...MuxMiddleware) Optional {
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
		o.bodyWriter = defaultBodyWriter
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
		runtime.WithMetadata(chainGrpcMetadata()),
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

func forwardResponseOptionFunc(ctx context.Context, w http.ResponseWriter, message proto.Message) error {
	if body, ok := message.(*httpbody.HttpBody); ok {
		if body.ContentType == typeLocation {
			location := string(body.Data)
			w.Header().Set(typeLocation, location)
			body.ContentType = "text/html; charset=utf-8"
			w.Header().Set("Content-Type", body.ContentType)
			w.WriteHeader(http.StatusFound)
			body.Data = []byte("<a href=\"" + htmlReplacer.Replace(location) + "\">Found</a>.\n")
		}
	}
	return nil
}
