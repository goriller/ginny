package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/goriller/ginny-util/ip"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var (
	grpcGatewayTag = opentracing.Tag{Key: string(ext.Component), Value: "grpc-gateway"}
)

// TracerMiddleWare
func TracerMiddleWare(t opentracing.Tracer) MuxMiddleware {
	return func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				// next
				h.ServeHTTP(w, r)
				return
			}

			r.Header.Set(PathHeader, r.URL.Path)
			r.Header.Set(MethodHeader, r.Method)
			r.Header.Set("host", r.Host)
			// 注入IP地址
			_ = ip.GetIPFromHTTPRequest(r)
			if t == nil {
				t = opentracing.GlobalTracer()
			}

			ctx := ChainHeader(w, r)

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

		}
	}
}

// ChainHeader
func ChainHeader(w http.ResponseWriter, r *http.Request) context.Context {
	var (
		reqId     string
		otHeaders = HeaderMap
	)
	ctx := r.Context()
	preTags := tags.Extract(ctx)

	for k, v := range otHeaders {
		val := r.Header.Get(k)
		if len(val) > 0 {
			if v == RequestId {
				reqId = val
			}
			preTags.Set(v, val)
		}
	}
	if !preTags.Has(RequestId) || reqId == "" {
		reqId = uuid.New().String()
	}
	preTags.Set(RequestId, reqId)
	w.Header().Set(RequestIDHeader, reqId)
	context := tags.SetInContext(ctx, preTags)
	return context
}
