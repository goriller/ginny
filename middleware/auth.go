package middleware

import (
	"net/http"

	"github.com/goriller/ginny/interceptor"
	"github.com/goriller/ginny/server/mux/rewriter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthMiddleWare
func AuthMiddleWare(authFunc interceptor.Authorize) MuxMiddleware {
	return func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				h.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/healthz" {
				// next
				h.ServeHTTP(w, r)
				return
			}
			newCtx, err := authFunc(r.Context(), r)
			if err != nil {
				rewriter.WriteHTTPErrorResponse(w, r, status.Errorf(codes.PermissionDenied, "permission denied."))
				return
			}
			h.ServeHTTP(w, r.WithContext(newCtx))
		}
	}
}
