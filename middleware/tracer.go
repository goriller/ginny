package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	util "github.com/gorillazer/ginny-util"
	"github.com/gorillazer/ginny/interceptor"
	"github.com/gorillazer/ginny/logging"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var (
	grpcGatewayTag = opentracing.Tag{Key: string(ext.Component), Value: "grpc-gateway"}
)

// TracerMiddleWare
func TracerMiddleWare(t opentracing.Tracer, logger grpc_logging.Logger) MuxMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				// next
				h.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			r.Header.Set(logging.PathHeader, r.URL.Path)
			r.Header.Set(logging.MethodHeader, r.Method)
			r.Header.Set("host", r.Host)
			// 注入IP地址
			_ = util.GetIPFromHTTPRequest(r)
			if t == nil {
				t = opentracing.GlobalTracer()
			}

			ctx := interceptor.ChainContext(tags.SetInContext(r.Context(),
				tags.NewTags()))

			parentSpanContext, err := t.Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(r.Header))
			if err == nil || err == opentracing.ErrSpanContextNotFound {
				serverSpan := t.StartSpan(
					"ServeHTTP",
					// this is magical, it attaches the new span to the parent parentSpanContext, and creates an unparented one if empty.
					ext.RPCServerOption(parentSpanContext),
					grpcGatewayTag,
				)
				ctx = opentracing.ContextWithSpan(ctx, serverSpan)
				defer serverSpan.Finish()
			}

			r = r.WithContext(ctx)
			h.ServeHTTP(w, r)

			if logger == nil {
				return
			}

			// logger
			withLogger(start, logger, r, w)
		})
	}
}

// withLogger
func withLogger(start time.Time, logger grpc_logging.Logger, r *http.Request, w http.ResponseWriter) {
	fields := getLoggingFields(start, r, w)
	status := w.Header().Get(logging.ResponseStatusHeader)
	level := getLevel(status, w)
	if status != "" {
		fields = append(fields, "status", status)
	}
	logger.With(fields...).Log(level, "finished call")
}

// getLoggingFields returns all fields from tags.
func getLoggingFields(start time.Time,
	r *http.Request, w http.ResponseWriter) grpc_logging.Fields {
	var fields grpc_logging.Fields
	preTags := tags.Extract(r.Context()).Values()
	for k, v := range preTags {
		fields = append(fields, k, v)
	}
	fields = append(fields, "path", r.URL.Path)
	fields = append(fields, "host", r.Host)
	fields = append(fields, "method", r.Method)
	fields = append(fields, "protocol", r.Proto)
	fields = append(fields, "referer", r.Header.Get("referer"))
	fields = append(fields, "device_id", r.Header.Get("x-device-id"))
	used := float32(time.Since(start)) / float32(time.Millisecond)
	fields = append(fields, "time_ms", fmt.Sprintf("%3f", used))
	return fields
}

// getLevel
func getLevel(status string, w http.ResponseWriter) (logLevel grpc_logging.Level) {
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		return grpc_logging.INFO
	}

	if statusCode >= http.StatusInternalServerError {
		if statusCode == http.StatusNotImplemented {
			logLevel = grpc_logging.WARNING
		} else {
			logLevel = grpc_logging.ERROR
		}
	} else if statusCode >= http.StatusBadRequest &&
		statusCode < http.StatusInternalServerError {
		logLevel = grpc_logging.WARNING
	} else {
		logLevel = grpc_logging.INFO
	}

	return
}
