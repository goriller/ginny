package interceptor

import (
	"context"
	"strings"

	"github.com/google/uuid"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MDCarrier custome carrier
type MDCarrier struct {
	metadata.MD
}

// ForeachKey conforms to the TextMapReader interface.
// TextMapReader is the Extract() carrier for the TextMap builtin format. With it,
// the caller can decode a propagated SpanContext as entries in a map of
// unicode strings.
//
//	type TextMapReader interface {
//		// ForeachKey returns TextMap contents via repeated calls to the `handler`
//		// function. If any call to `handler` returns a non-nil error, ForeachKey
//		// terminates and returns that error.
//		//
//		// NOTE: The backing store for the TextMapReader may contain data unrelated
//		// to SpanContext. As such, Inject() and Extract() implementations that
//		// call the TextMapWriter and TextMapReader interfaces must agree on a
//		// prefix or other convention to distinguish their own key:value pairs.
//		//
//		// The "foreach" callback pattern reduces unnecessary copying in some cases
//		// and also allows implementations to hold locks while the map is read.
//		ForeachKey(handler func(key, val string) error) error
//	}
func (m MDCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, strs := range m.MD {
		for _, v := range strs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
// TextMapWriter is the Inject() carrier for the TextMap builtin format. With
// it, the caller can encode a SpanContext for propagation as entries in a map
// of unicode strings.
//
//	type TextMapWriter interface {
//		// Set a key:value pair to the carrier. Multiple calls to Set() for the
//		// same key leads to undefined behavior.
//		//
//		// NOTE: The backing store for the TextMapWriter may contain data unrelated
//		// to SpanContext. As such, Inject() and Extract() implementations that
//		// call the TextMapWriter and TextMapReader interfaces must agree on a
//		// prefix or other convention to distinguish their own key:value pairs.
//		Set(key, val string)
//	}
func (m MDCarrier) Set(key, val string) {
	m.MD[key] = append(m.MD[key], val)
}

// TracerClientInterceptor
// https://godoc.org/google.golang.org/grpc#UnaryClientInterceptor
func TracerUnaryClientInterceptor(tracer opentracing.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if tracer == nil {
			tracer = opentracing.GlobalTracer()
		}
		//一个RPC调用的服务端的span，和RPC服务客户端的span构成ChildOf关系
		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		span := tracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC Client"},
			ext.SpanKindRPCClient,
		)
		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		if err := tracer.Inject(span.Context(), opentracing.TextMap,
			MDCarrier{md}, // 自定义 carrier
		); err != nil {
			return err
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		if err := invoker(newCtx, method, request, reply, cc, opts...); err != nil {
			return err
		}
		return nil
	}
}

// TracerClientStreamInterceptor
func TracerClientStreamInterceptor(tracer opentracing.Tracer) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
		streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if tracer == nil {
			tracer = opentracing.GlobalTracer()
		}
		//一个RPC调用的服务端的span，和RPC服务客户端的span构成ChildOf关系
		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		span := tracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC Client"},
			ext.SpanKindRPCClient,
		)
		defer span.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		if err := tracer.Inject(span.Context(), opentracing.TextMap,
			MDCarrier{md}, // 自定义 carrier
		); err != nil {
			return nil, err
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		return streamer(newCtx, desc, cc, method, opts...)
	}
}

// TracerServerUnaryInterceptor
func TracerServerUnaryInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		if tracer == nil {
			tracer = opentracing.GlobalTracer()
		}
		spanContext, err := tracer.Extract(
			opentracing.TextMap,
			MDCarrier{md},
		)

		if err != nil && err != opentracing.ErrSpanContextNotFound {
			return nil, err
		} else {
			span := tracer.StartSpan(
				info.FullMethod,
				ext.RPCServerOption(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "gRPC Client"},
				ext.SpanKindRPCServer,
			)
			defer span.Finish()

			ctx = ChainContext(opentracing.ContextWithSpan(ctx, span), md)
		}

		return handler(ctx, req)
	}
}

// TracerServerStreamInterceptor
func TracerServerStreamInterceptor(tracer opentracing.Tracer) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		if tracer == nil {
			tracer = opentracing.GlobalTracer()
		}
		spanContext, err := tracer.Extract(
			opentracing.TextMap,
			MDCarrier{md},
		)

		if err != nil && err != opentracing.ErrSpanContextNotFound {
			return err
		} else {
			span := tracer.StartSpan(
				info.FullMethod,
				ext.RPCServerOption(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "gRPC Client"},
				ext.SpanKindRPCServer,
			)
			defer span.Finish()

			ctx = opentracing.ContextWithSpan(ctx, span)
			wrapped := middleware.WrapServerStream(ss)
			wrapped.WrappedContext = ChainContext(ctx, md)
			return handler(srv, wrapped)
		}
	}
}

// ChainContext
func ChainContext(ctx context.Context, md metadata.MD) context.Context {
	var (
		reqId     string
		otHeaders = HeaderMap
	)
	preTags := tags.Extract(ctx)

	for _, v := range otHeaders {
		val := md.Get(v)
		if len(val) > 0 {
			if v == RequestId {
				reqId = val[0]
			}
			preTags.Set(v, strings.Join(val, ","))
		}
	}
	if !preTags.Has(RequestId) || reqId == "" {
		reqId = uuid.New().String()
	}
	preTags.Set(RequestId, reqId)
	md.Set(RequestIDHeader, reqId)
	context := metadata.NewOutgoingContext(ctx, md)
	return context
}
