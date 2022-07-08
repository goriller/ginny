package mux

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	util "github.com/gorillazer/ginny-util"
	"github.com/gorillazer/ginny/logging"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	RequestIDHeader    = "x-request-id"
	CaptchaTokenHeader = "x-captcha-token"
	PathHeader         = "x-request-path"
	MethodHeader       = "x-request-method"
	TraceidHeader      = "x-b3-traceid"
	SpanidHeader       = "x-b3-spanid"
	ParentspanidHeader = "x-b3-parentspanid"
	SampledHeader      = "x-b3-sampled"
	FlagsHeader        = "x-b3-flags"
	SpanContextHeader  = "x-ot-span-context"
)

var recoverPanic = true

type annotator func(context.Context, *http.Request) metadata.MD

// DisableRecover disable panic recover
func DisableRecover() {
	recoverPanic = false
}

// extractFields returns all fields from tags.
func extractFields(tagsData map[string]string) grpc_logging.Fields {
	var fields grpc_logging.Fields
	for k, v := range tagsData {
		fields = append(fields, k, v)
	}
	return fields
}

// TraceMiddleWare add logger
func TraceMiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if r.URL.Path == "/healthz" {
			// next
			h.ServeHTTP(w, r)
			return
		}
		r.Header.Set(PathHeader, r.URL.Path)
		r.Header.Set(MethodHeader, r.Method)
		r.Header.Set("host", r.Host)
		// 注入IP地址
		_ = util.GetIPFromHTTPRequest(r)
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		t := tags.NewTags()
		ctx := tags.SetInContext(r.Context(), t)
		r = r.WithContext(ctx)

		r.Header.Set(RequestIDHeader, requestID)
		w.Header().Set(RequestIDHeader, requestID)

		defer func() {
			if recoverPanic {
				if rec := recover(); rec != nil {
					stack := zap.StackSkip("", 2).String
					defaultMuxOption.logger.Log(grpc_logging.ERROR, stack)
					tags.Extract(ctx).Set("stacktrace", stack)
					err := status.Errorf(codes.Internal, "%s", rec)
					WriteHTTPErrorResponse(w, r, err)
					return
				}
			}
		}()
		// captcha_token
		captchaToken := r.URL.Query().Get(CaptchaTokenHeader)
		if captchaToken != "" {
			r.Header.Set(CaptchaTokenHeader, captchaToken)
		}
		writer := &responseWriter{
			w:                 w,
			header:            200,
			withoutHTTPStatus: defaultMuxOption.withoutHTTPStatus,
		}

		if defaultMuxOption.logger == nil {
			h.ServeHTTP(writer, r)
			return
		}

		// next
		h.ServeHTTP(writer, r)
		// logger
		withLogger(start, r, writer)
	})
}

// withLogger
func withLogger(start time.Time, r *http.Request, w http.ResponseWriter) {
	preTags := tags.Extract(r.Context())
	logData := preTags.Values()
	logData["action"] = r.URL.Path
	logData["host"] = r.Host
	logData["protocol"] = r.Proto
	logData["referer"] = r.Header.Get("referer")
	logData[logging.RequestId] = r.Header.Get(RequestIDHeader)
	logData["device_id"] = r.Header.Get("x-device-id")
	used := float32(time.Since(start)) / float32(time.Millisecond)
	logData["time_ms"] = fmt.Sprintf("%3f", used)
	var logLevel grpc_logging.Level
	if writer, ok := w.(*responseWriter); ok {
		logData["status"] = strconv.Itoa(writer.header)
		switch statusCode := writer.header; {
		case statusCode >= http.StatusInternalServerError:
			if statusCode == http.StatusNotImplemented {
				logLevel = grpc_logging.WARNING
			} else {
				logLevel = grpc_logging.ERROR
			}
		case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
			logLevel = grpc_logging.WARNING
		default:
			logLevel = grpc_logging.INFO
		}
	}
	defaultMuxOption.logger.With(extractFields(logData)...).Log(logLevel, r.Method)
}

// chainGrpcAnnotators
func chainGrpcAnnotators(ctx context.Context, req *http.Request) metadata.MD {
	var (
		pairs     []string
		otHeaders = []string{
			RequestIDHeader,
			TraceidHeader,
			SpanidHeader,
			ParentspanidHeader,
			SampledHeader,
			FlagsHeader,
			SpanContextHeader}
	)

	for _, h := range otHeaders {
		if v := req.Header.Get(h); len(v) > 0 {
			pairs = append(pairs, h, v)
		}
	}
	return metadata.Pairs(pairs...)
}

// chainGrpcMetadata
func chainGrpcMetadata() annotator {
	var (
		mds []metadata.MD
	)
	return func(c context.Context, r *http.Request) metadata.MD {
		annotators := []annotator{chainGrpcAnnotators}
		for _, a := range annotators {
			mds = append(mds, a(c, r))
		}
		return metadata.Join(mds...)
	}
}

// chainGrpcContext
func chainGrpcContext(ctx context.Context, req interface{}) context.Context {
	var (
		otHeaders = []string{
			RequestIDHeader,
			TraceidHeader,
			SpanidHeader,
			ParentspanidHeader,
			SampledHeader,
			FlagsHeader,
			SpanContextHeader}
	)
	preTags := tags.Extract(ctx)
	headersIn, _ := metadata.FromIncomingContext(ctx)
	var reqId string
	for _, v := range otHeaders {
		val := headersIn.Get(v)
		if len(val) > 0 {
			if v == RequestIDHeader {
				reqId = val[0]
			}
			preTags.Set(v, strings.Join(val, ","))
		}
	}
	if !preTags.Has(logging.RequestId) && reqId == "" {
		reqId = uuid.New().String()
	}
	preTags.Set(logging.RequestId, reqId)
	headersIn.Set(logging.RequestId, reqId)
	context := metadata.NewOutgoingContext(ctx, headersIn)
	return context
}

// TraceUnaryServerInterceptor returns a new unary server interceptors that performs per-request.
func TraceUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		newCtx := chainGrpcContext(ctx, req)
		return handler(newCtx, req)
	}
}

// TraceStreamServerInterceptor returns a new unary server interceptors that performs per-request.
func TraceStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		newCtx := chainGrpcContext(stream.Context(), nil)
		wrapped := middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}
