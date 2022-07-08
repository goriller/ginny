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
			r = r.WithContext(ctx)

			parentSpanContext, err := t.Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(r.Header))
			if err == nil || err == opentracing.ErrSpanContextNotFound {
				serverSpan := opentracing.GlobalTracer().StartSpan(
					"ServeHTTP",
					// this is magical, it attaches the new span to the parent parentSpanContext, and creates an unparented one if empty.
					ext.RPCServerOption(parentSpanContext),
					grpcGatewayTag,
				)
				r = r.WithContext(opentracing.ContextWithSpan(r.Context(), serverSpan))
				defer serverSpan.Finish()
			}
			h.ServeHTTP(w, r)
		})
	}
}

// chainAnnotators
func chainAnnotators(ctx context.Context, req *http.Request) metadata.MD {
	var (
		pairs     []string
		otHeaders = []string{
			logging.RequestIDHeader,
			logging.TraceidHeader,
			logging.SpanidHeader,
			logging.ParentspanidHeader,
			logging.SampledHeader,
			logging.FlagsHeader,
			logging.SpanContextHeader}
	)

	for _, h := range otHeaders {
		if v := req.Header.Get(h); v != "" {
			pairs = append(pairs, h, v)
		}
	}
	return metadata.Pairs(pairs...)
}

// ChainMetadata
func ChainMetadata() annotator {
	var (
		mds []metadata.MD
	)
	return func(c context.Context, r *http.Request) metadata.MD {
		annotators := []annotator{chainAnnotators}
		for _, a := range annotators {
			mds = append(mds, a(c, r))
		}
		return metadata.Join(mds...)
	}
}
