package mux

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var recoverPanic = true

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

// RecoverMiddleWare revover add logger
func RecoverMiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recoverPanic {
				if rec := recover(); rec != nil {
					stack := zap.StackSkip("", 2).String
					defaultMuxOption.logger.Log(grpc_logging.ERROR, stack)
					tags.Extract(r.Context()).Set("stacktrace", stack)
					err := status.Errorf(codes.Internal, "%s", rec)
					WriteHTTPErrorResponse(w, r, err)
					return
				}
			}
		}()
		if r.URL.Path == "/healthz" {
			h.ServeHTTP(w, r)
			return
		}
		// writer
		writer := &responseWriter{
			w:                 w,
			header:            200,
			withoutHTTPStatus: defaultMuxOption.withoutHTTPStatus,
		}

		if defaultMuxOption.logger == nil {
			h.ServeHTTP(writer, r)
			return
		}
		start := time.Now()
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
	// logData[logging.RequestId] = r.Header.Get(RequestIDHeader)
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
