package logging

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

// UnaryClientInterceptor returns a new unary client interceptor that optionally logs the
// execution of external gRPC calls.
// Logger will use all tags (from tags package) available in current context as fields.
func UnaryClientInterceptor(logger logging.Logger, opts ...Option) grpc.UnaryClientInterceptor {
	o := evaluateOpt(opts)
	return interceptors.UnaryClientInterceptor(&reportable{logger: logger, opts: o})
}

// StreamClientInterceptor returns a new streaming client interceptor that optionally logs the
// execution of external gRPC calls.
// Logger will use all tags (from tags package) available in current context as fields.
func StreamClientInterceptor(logger logging.Logger, opts ...Option) grpc.StreamClientInterceptor {
	o := evaluateOpt(opts)
	return interceptors.StreamClientInterceptor(&reportable{logger: logger, opts: o})
}

// ClientReporter the client reporter
func (r *reportable) ClientReporter(ctx context.Context, _ interface{}, typ interceptors.GRPCType, service string,
	method string,
) (interceptors.Reporter, context.Context) {
	return r.reporter(ctx, typ, service, method, logging.KindClientFieldValue)
}
