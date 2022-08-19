package logging

import (
	"context"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// UnaryServerInterceptor returns a new unary server interceptors that optionally logs endpoint handling.
// Logger will use all tags (from tags package) available in current context as fields.
func UnaryServerInterceptor(logger logging.Logger, opts ...Option) grpc.UnaryServerInterceptor {
	o := evaluateOpt(opts)
	return interceptors.UnaryServerInterceptor(&reportable{logger: logger, opts: o})
}

// StreamServerInterceptor returns a new stream server interceptors that optionally logs endpoint handling.
// Logger will use all tags (from tags package) available in current context as fields.
func StreamServerInterceptor(logger logging.Logger, opts ...Option) grpc.StreamServerInterceptor {
	o := evaluateOpt(opts)
	return interceptors.StreamServerInterceptor(&reportable{logger: logger, opts: o})
}

// ServerReporter implement the ServerReporter.
func (r *reportable) ServerReporter(ctx context.Context, _ interface{}, typ interceptors.GRPCType,
	service string, method string,
) (interceptors.Reporter, context.Context) {
	newCtx := newTagsForCtx(ctx)
	return r.reporter(newCtx, typ, service, method, logging.KindServerFieldValue)
}

func newTagsForCtx(ctx context.Context) context.Context {
	t := tags.NewTags()
	if peer, ok := peer.FromContext(ctx); ok {
		addrHost, _, err := net.SplitHostPort(peer.Addr.String())
		if err == nil {
			t.Set("ip", addrHost)
		}
	}
	return tags.SetInContext(ctx, t)
}
