package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goriller/ginny/interceptor/limit"
	"github.com/goriller/ginny/interceptor/tags"
	"github.com/goriller/ginny/server/mux/rewriter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LimitMiddleWare
func LimitMiddleWare(limiter *limit.Limiter) MuxMiddleware {
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

			ctx := r.Context()
			var lv *limit.LimitValue
			ctxTags := tags.Extract(ctx)
			ctxTagsValues := ctxTags.Values()
			if len(ctxTagsValues) > 0 {
				// Convert map to logging.Fields
				fields := ctxTags.ToLoggingFields()
				lv = limiter.Config.MatchMap(r.URL.Path, fields)
				ctxTags.Set("rate_limit", lv.Key)
			} else {
				lv = limiter.Config.MatchHeader(r.URL.Path, r.Header)
			}
			if lv.Quota == limit.NoLimit {
				h.ServeHTTP(w, r)
				return
			}

			if lv.Quota == limit.Block {
				rewriter.WriteHTTPErrorResponse(w, r, status.Errorf(codes.Aborted, "rate limit aborted, %s", lv.Message))
				return
			}
			if limiter.RateFn == nil {
				limiter.RateFn = limit.DefaultRateFn
			}
			remaining, reset, allowed := limiter.RateFn(r.Context(), lv.Key, lv.Quota, lv.Duration, 1)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(lv.Quota))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(reset/time.Second)))
			w.Header().Set("X-RateLimit-Resource", resourceKey(lv.Key))
			if !allowed {
				rewriter.WriteHTTPErrorResponse(w, r, status.Errorf(codes.ResourceExhausted, "rate limit exhausted, %s", lv.Message))
				return
			}
			h.ServeHTTP(w, r)
		}
	}
}

func resourceKey(src string) string {
	src = strings.Replace(src, "/", "", -1)
	src = strings.Replace(src, ".", "", -1)
	src = strings.Replace(src, "-", "", 1)
	return src
}
