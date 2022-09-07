package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/goriller/ginny/server/mux/rewriter"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var recoverPanic = true

// DisableRecover disable panic recover
func DisableRecover() {
	recoverPanic = false
}

// RecoverMiddleWare revover add logger
func RecoverMiddleWare(logger grpc_logging.Logger, bodyMarshaler,
	errorMarshaler runtime.Marshaler, withoutHTTPStatus bool) MuxMiddleware {
	return func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			// writer
			writer := &rewriter.ResponseWriter{
				Writer:            w,
				HeaderStatus:      200,
				WithoutHTTPStatus: withoutHTTPStatus,
				Status:            status.New(codes.OK, "success"),
				BodyWriter: rewriter.DefaultBodyWriter(bodyMarshaler, errorMarshaler,
					withoutHTTPStatus),
			}
			ctx := tags.SetInContext(r.Context(), tags.NewTags())
			r = r.WithContext(ctx)
			defer func(wt http.ResponseWriter, req *http.Request) {
				if recoverPanic {
					if rec := recover(); rec != nil {
						stack := zap.StackSkip("", 2).String
						tags.Extract(r.Context()).Set("stacktrace", stack)
						err := status.Errorf(codes.Internal, "%s", rec)
						rewriter.WriteHTTPErrorResponse(wt, req, err)
						withLogger(start, logger, req, wt)
					}
				}
			}(writer, r)

			if r.URL.Path == "/healthz" {
				h.ServeHTTP(w, r)
				return
			}

			// next
			h.ServeHTTP(writer, r)

			// logger
			withLogger(start, logger, r, writer)
		}
	}
}

// withLogger
func withLogger(start time.Time, logger grpc_logging.Logger, r *http.Request, w http.ResponseWriter) {
	if logger == nil {
		return
	}
	fields := getLoggingFields(start, r, w)
	status := w.Header().Get(ResponseStatusHeader)
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
	fields = append(fields, "device_id", r.Header.Get(DeviceIDHeader))
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
