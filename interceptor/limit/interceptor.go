package limit

import (
	"context"

	"github.com/goriller/ginny/interceptor/tags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptors that performs request rate limiting.
func UnaryServerInterceptor(limiter *Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		err := limit(ctx, info.FullMethod, limiter)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new stream server interceptor that performs rate limiting on the request.
func StreamServerInterceptor(limiter *Limiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := limit(stream.Context(), info.FullMethod, limiter)
		if err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func limit(ctx context.Context, fullMethod string, limiter *Limiter) error {
	ctxTagValues := tags.Extract(ctx).Values()
	lv := limiter.Config.MatchMap(fullMethod, ctxTagValues)
	if lv.Quota == Block {
		return status.Errorf(codes.Aborted, "%s is aborted for %s", fullMethod, lv.Message)
	}
	_, resetIn, allowed := limiter.RateFn(ctx, lv.Key, lv.Quota, lv.Duration, 1)
	if !allowed {
		return status.Errorf(codes.ResourceExhausted, "method is rejected for %s, "+
			"please try in %s. ", lv.Message, resetIn)
	}
	return nil
}
