// Package tracing provides a Connect interceptor for OpenTelemetry tracing.
package tracing

import (
	"context"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewInterceptor creates a Connect interceptor that adds OpenTelemetry tracing spans.
//
// This is Layer 2: Connect interceptor (only applied to RPC handlers).
func NewInterceptor(opts ...Option) connect.Interceptor {
	o := &options{
		tracerProvider: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt(o)
	}
	tracer := o.tracerProvider.Tracer("github.com/goriller/ginny/v2/interceptor/tracing")
	interceptor := &tracingInterceptor{
		tracer: tracer,
	}
	return interceptor
}

// Option configures the tracing interceptor.
type Option func(*options)

type options struct {
	tracerProvider trace.TracerProvider
}

// WithTracerProvider sets a custom OpenTelemetry TracerProvider.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(o *options) {
		if tp != nil {
			o.tracerProvider = tp
		}
	}
}

type tracingInterceptor struct {
	tracer trace.Tracer
}

func (t *tracingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, span := t.tracer.Start(ctx, req.Spec().Procedure,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "connect"),
				attribute.String("rpc.service", req.Spec().Procedure),
				attribute.String("rpc.peer", req.Peer().Addr),
			),
		)
		defer span.End()

		resp, err := next(ctx, req)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		return resp, err
	}
}

func (t *tracingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (t *tracingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, span := t.tracer.Start(ctx, conn.Spec().Procedure,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "connect"),
				attribute.String("rpc.service", conn.Spec().Procedure),
				attribute.String("rpc.peer", conn.Peer().Addr),
			),
		)
		defer span.End()

		err := next(ctx, conn)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
		return err
	}
}
