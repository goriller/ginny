package middleware

import (
	"context"
	"net/http"

	util "github.com/gorillazer/ginny-util"
	"github.com/gorillazer/ginny/interceptor"
	"github.com/gorillazer/ginny/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc/metadata"
)

var (
	grpcGatewayTag = opentracing.Tag{Key: string(ext.Component), Value: "grpc-gateway"}
)

type annotator func(context.Context, *http.Request) metadata.MD

// TracerMiddleWare
func TracerMiddleWare(t opentracing.Tracer) MuxMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				// next
				h.ServeHTTP(w, r)
				return
			}
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
		})
	}
}
