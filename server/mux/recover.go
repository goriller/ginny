package mux

import (
	"net/http"

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

		// next
		h.ServeHTTP(writer, r)
	})
}
