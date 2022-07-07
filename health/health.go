package health

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const AllServices = "*"

// HealthServer 简化版健康检查
type HealthServer struct {
	Server *grpc_health.Server
}

func NewHealthServer() *HealthServer {
	return &HealthServer{
		Server: grpc_health.NewServer(),
	}
}

// Start
func (s *HealthServer) Start(gs *grpc.Server) {
	healthpb.RegisterHealthServer(gs, s)
	s.Server.SetServingStatus(AllServices, healthpb.HealthCheckResponse_SERVING)
}

// Close
func (s *HealthServer) Close() {
	s.Server.SetServingStatus(AllServices, healthpb.HealthCheckResponse_NOT_SERVING)
}

// Check implement check
func (s *HealthServer) Check(ctx context.Context,
	in *healthpb.HealthCheckRequest,
) (*healthpb.HealthCheckResponse, error) {
	in.Service = AllServices
	return s.Server.Check(ctx, in)
}

// Watch implement watch
func (s *HealthServer) Watch(in *healthpb.HealthCheckRequest,
	server healthpb.Health_WatchServer,
) error {
	in.Service = AllServices
	return s.Server.Watch(in, server)
}

// AuthFuncOverride health check without grpc auth middleware.
func (s *HealthServer) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	return ctx, nil
}

// HealthMiddleware HTTP健康检查中间件
func HealthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			_, _ = w.Write([]byte(healthpb.HealthCheckResponse_SERVING.String()))
			return
		}
		h.ServeHTTP(w, r)
	})
}
