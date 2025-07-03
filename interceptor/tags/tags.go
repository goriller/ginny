package tags

import (
	"context"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

// Tags represents a collection of tags that can be added to the context
type Tags struct {
	values map[string]interface{}
	mu     sync.RWMutex
}

// NewTags creates a new Tags instance
func NewTags() *Tags {
	return &Tags{
		values: make(map[string]interface{}),
	}
}

// Set adds a key-value pair to the tags
func (t *Tags) Set(key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values[key] = value
}

// Has checks if a key exists in the tags
func (t *Tags) Has(key string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.values[key]
	return exists
}

// Values returns all the tag values as a map
func (t *Tags) Values() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range t.values {
		result[k] = v
	}
	return result
}

// contextKey is used to store tags in context
type contextKey struct{}

var tagsKey = &contextKey{}

// Extract retrieves tags from the context, creating new ones if they don't exist
func Extract(ctx context.Context) *Tags {
	if tags, ok := ctx.Value(tagsKey).(*Tags); ok {
		return tags
	}
	return NewTags()
}

// InjectIntoContext injects tags into the context
func InjectIntoContext(ctx context.Context, tags *Tags) context.Context {
	return context.WithValue(ctx, tagsKey, tags)
}

// ToLoggingFields converts tags to logging fields for grpc-middleware v2
func (t *Tags) ToLoggingFields() logging.Fields {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fields := logging.Fields{}
	for k, v := range t.values {
		fields = append(fields, k, v)
	}
	return fields
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that injects tags into the context
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		tags := NewTags()
		newCtx := InjectIntoContext(ctx, tags)
		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor that injects tags into the context
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tags := NewTags()
		newCtx := InjectIntoContext(stream.Context(), tags)

		// Create a wrapper stream with the new context
		wrapped := &wrappedServerStream{
			ServerStream: stream,
			ctx:          newCtx,
		}

		return handler(srv, wrapped)
	}
}

// wrappedServerStream wraps grpc.ServerStream to provide a new context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
