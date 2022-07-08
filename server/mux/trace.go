package mux

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	util "github.com/gorillazer/ginny-util"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	RequestIDHeader    = "x-request-id"
	CaptchaTokenHeader = "x-captcha-token"
)

var recoverPanic = true

// DisableRecover disable panic recover
func DisableRecover() {
	recoverPanic = false
}

// ExtractFields returns all fields from tags.
func ExtractFields(tagsData map[string]string) logging.Fields {
	var fields logging.Fields
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
		r.Header.Set("x-request-path", r.URL.Path)
		r.Header.Set("x-request-method", r.Method)
		r.Header.Set("host", r.Host)
		// 注入IP地址
		_ = util.GetIPFromHTTPRequest(r)

		ctx := tags.SetInContext(r.Context(), tags.NewTags())
		r = r.WithContext(ctx)

		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		r.Header.Set(RequestIDHeader, requestID)
		w.Header().Set(RequestIDHeader, requestID)

		defer func() {
			if recoverPanic {
				if rec := recover(); rec != nil {
					stack := zap.StackSkip("", 2).String
					defaultMuxOption.logger.Log(logging.ERROR, stack)
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
	logData["action"] = r.Method + ":" + r.URL.Path
	logData["host"] = r.Host
	logData["protocol"] = r.Proto
	logData["referer"] = r.Header.Get("referer")
	logData["request_id"] = r.Header.Get(RequestIDHeader)
	logData["device_id"] = r.Header.Get("x-device-id")
	used := float32(time.Since(start)) / float32(time.Millisecond)
	logData["time_ms"] = fmt.Sprintf("%3f", used)
	var logLevel logging.Level
	if writer, ok := w.(*responseWriter); ok {
		logData["status"] = strconv.Itoa(writer.header)
		switch statusCode := writer.header; {
		case statusCode >= http.StatusInternalServerError:
			if statusCode == http.StatusNotImplemented {
				logLevel = logging.WARNING
			} else {
				logLevel = logging.ERROR
			}
		case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
			logLevel = logging.WARNING
		default:
			logLevel = logging.INFO
		}
	}
	defaultMuxOption.logger.With(ExtractFields(logData)...).Log(logLevel, r.Method)
}
