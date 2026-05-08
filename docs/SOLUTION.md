# Ginny v2 — Architecture & Design

> 架构设计、技术细节、迁移指南与最佳实践

---

## Architecture | 架构

```
┌──────────────────────────────────────────────────────────┐
│                      ginny.App                            │
│  Lifecycle Hooks (OnStart/OnStop, priority-ordered)      │
│  Config: koanf (file + env + remote)                     │
│  Logger: slog interface (pluggable handler)              │
└──────────────┬───────────────────────────────────────────┘
               │
    ┌──────────▼──────────────────────────────────┐
    │        App Server (:8080)                    │
    │  ┌─────────────────────────────────────────┐ │
    │  │  ConnectRPC Handlers                    │ │
    │  │  ├─ gRPC    (HTTP/2 h2c)                │ │
    │  │  ├─ Connect (HTTP/1.1 + HTTP/2)         │ │
    │  │  └─ gRPC-Web(HTTP/1.1)                  │ │
    │  │                                         │ │
    │  │  Interceptors (Layer 2):                │ │
    │  │    OTel Tracing · Auth · RateLimit      │ │
    │  │    Validation · Recovery · Logging      │ │
    │  │                                         │ │
    │  │  Vanguard REST Transcoder (optional)     │ │
    │  │  gRPC Reflection · gRPC Health          │ │
    │  └─────────────────────────────────────────┘ │
    └──────────────────────────────────────────────┘
    ┌──────────────────────────────────────────────┐
    │        Admin Server (:8081)                   │
    │  GET /healthz     ·  Liveness probe           │
    │  GET /readyz      ·  Readiness probe          │
    │  GET /metrics     ·  Prometheus metrics       │
    │  GET /debug/pprof ·  Runtime profiling        │
    └──────────────────────────────────────────────┘
```

## Package Layout | 包结构

```
ginny/              # App entry point + LifecycleHook
├── server/         # Dual-port Server (App + Admin)
├── interceptor/    # auth, tracing, recovery, ratelimit, logging, validation
├── config/         # koanf configuration
├── log/            # slog interface + helpers
├── errs/           # Connect error + biz code
├── client/         # Pool + retry + circuit breaker
├── ginnytest/      # Test helpers
├── example/        # Minimal example
└── buf/            # Code generation templates
```

---

## Full Example with Interceptors | 完整拦截器示例

```go
package main

import (
    "context"
    "log/slog"

    "github.com/goriller/ginny/v2"
    "github.com/goriller/ginny/v2/config"
    "github.com/goriller/ginny/v2/interceptor/auth"
    "github.com/goriller/ginny/v2/interceptor/logging"
    "github.com/goriller/ginny/v2/interceptor/ratelimit"
    "github.com/goriller/ginny/v2/interceptor/recovery"
    "github.com/goriller/ginny/v2/interceptor/tracing"
    "github.com/goriller/ginny/v2/interceptor/validation"
    "github.com/goriller/ginny/v2/log"
    "github.com/goriller/ginny/v2/server"
)

func main() {
    cfg := config.MustLoad()
    logger := log.New(log.WithLevel(slog.LevelInfo), log.WithSource(true))

    srv := server.New(
        server.WithAppAddr(cfg.Server.Addr),
        server.WithAdminAddr(cfg.Admin.Addr),
        server.WithLogger(logger),
        server.WithInterceptor(recovery.NewInterceptor(logger)),
        server.WithInterceptor(tracing.NewInterceptor()),
        server.WithInterceptor(logging.NewInterceptor(logger, logging.WithLevel(slog.LevelInfo))),
        server.WithInterceptor(validation.NewInterceptor()),
        server.WithReflection(true),
        server.WithReflectionServiceNames("myapp.v1.MyService"),
    )

    app, _ := ginny.New(
        ginny.WithName(cfg.App.Name),
        ginny.WithConfig(cfg),
        ginny.WithLogger(logger),
        ginny.WithServer(srv),
    )
    app.Start(context.Background())
}
```

## Interceptor Model | 三层拦截器

| Layer | Scope | Examples |
|-------|-------|----------|
| 1. HTTP middleware | All routes (app + admin) | Recovery, CORS, Compression |
| 2. Connect interceptor | RPC handlers only | Tracing, Auth, RateLimit, Validation, Logging |
| 3. Per-service | Specific service | Custom business logic |

### 各拦截器用法

#### Recovery

```go
import "github.com/goriller/ginny/v2/interceptor/recovery"
server.WithInterceptor(recovery.NewInterceptor(logger))
```

Catches panics, logs stack traces, converts to `CodeInternal`.

#### Logging

```go
import "github.com/goriller/ginny/v2/interceptor/logging"
server.WithInterceptor(logging.NewInterceptor(logger, logging.WithLevel(slog.LevelDebug)))
```

Logs procedure, duration, status per RPC. Errors always at Error level.

#### Tracing (OTel)

```go
import "github.com/goriller/ginny/v2/interceptor/tracing"
server.WithInterceptor(tracing.NewInterceptor())
server.WithInterceptor(tracing.NewInterceptor(tracing.WithTracerProvider(myProvider)))
```

#### Auth

```go
import "github.com/goriller/ginny/v2/interceptor/auth"

type myValidator struct{}
func (v *myValidator) Validate(ctx context.Context, token string) (auth.Identity, error) {
    return &myIdentity{userID: "123"}, nil
}

server.WithInterceptor(auth.NewInterceptor(&myValidator{},
    auth.SkipProcedures("/myapp.v1.Health/Check"),
))
```

Extracts Bearer token, validates, stores identity in context (`auth.GetIdentity(ctx)`).

#### Rate Limiting

```go
import "github.com/goriller/ginny/v2/interceptor/ratelimit"

server.WithInterceptor(ratelimit.NewInterceptor(myLimiter,
    ratelimit.WithKeyFunc(func(ctx context.Context, req connect.AnyRequest) string {
        return req.Peer().Addr
    }),
))
```

Returns `CodeResourceExhausted` when exceeded. Implement `Limiter` for local or Redis backend.

#### Validation

```go
import "github.com/goriller/ginny/v2/interceptor/validation"
server.WithInterceptor(validation.NewInterceptor())
```

Calls `Validate()` on request messages. Works with protoc-gen-validate.

---

## Configuration | 配置

```go
import "github.com/goriller/ginny/v2/config"

cfg := config.MustLoad()                                    // default
cfg := config.MustLoad(config.WithConfigPath("./config.yaml"))
cfg := config.MustLoad(config.WithEnvPrefix("MYAPP"))
```

**config.yaml:**

```yaml
app:
  name: my-service
server:
  addr: ":9090"
admin:
  addr: ":9091"
logging:
  level: debug
```

**Env overrides:**

```bash
export GINNY_SERVER_ADDR=:8080
export GINNY_LOGGING_LEVEL=info
```

---

## Error Handling | 错误处理

```go
import "github.com/goriller/ginny/v2/errs"

// Server side
return nil, errs.New(connect.CodeNotFound, 40401, "user not found")

// Client side
bizCode := errs.BizCode(err)    // 40401
bizMsg  := errs.BizMessage(err) // "user not found"

// Register codes
errs.RegisterBizCodes(map[int32]string{40401: "USER_NOT_FOUND"})
```

---

## Logging | 日志

```go
import "github.com/goriller/ginny/v2/log"

logger := log.New()
logger := log.New(log.WithLevel(slog.LevelDebug), log.WithSource(true))
logger := log.New(log.WithHandler(zapHandler)) // custom handler
```

Use `log.GetContextLogger(ctx)` / `log.SetContextLogger(ctx, logger)` for context propagation.

---

## Client | 客户端

```go
import (
    "github.com/goriller/ginny/v2/client"
    "myapp/v1/myappv1connect"
)

pool := client.NewPool(
    client.WithTimeout(10 * time.Second),
    client.WithRetry(3),
    client.WithCircuitBreaker(true),
)
defer pool.Close(context.Background())

typedClient := myappv1connect.NewMyServiceClient(pool.HTTPClient(), "http://backend:8080")
```

---

## Lifecycle Hooks | 生命周期

```go
type LifecycleHook interface {
    Name() string
    OnStart(ctx context.Context) error
    OnStop(ctx context.Context) error
    Priority() int // lower = starts earlier, stops later
}

app, _ := ginny.New(
    ginny.WithServer(srv),
    ginny.WithHook(&myDatabaseHook{}),
)
```

Built-in priority: Admin(10) → App(20) → ServiceDiscovery(30) → Readiness(40)

---

## Testing | 测试

```go
import "github.com/goriller/ginny/v2/ginnytest"

func TestMyService(t *testing.T) {
    ts, _ := ginnytest.NewTestServer(t,
        server.WithInterceptor(recovery.NewInterceptor(slog.Default())),
    )
    ts.Server().RegisterService("/myapp.v1.MyService/", myHandler)

    client := myappv1connect.NewMyServiceClient(ts.Client(), ts.BaseURL())
    resp, _ := client.GetUser(context.Background(), connect.NewRequest(&pb.GetUserReq{Id: "1"}))
}
```

---

## Migration from v1 | 从 v1 迁移

### Module Path

```
v1: github.com/goriller/ginny
v2: github.com/goriller/ginny/v2  (coexists during migration)
```

### Dependency Changes

| v1 | v2 | Reason |
|----|----|--------|
| `google.golang.org/grpc` | `connectrpc.com/connect` | Lighter, browser-native |
| `grpc-ecosystem/grpc-gateway/v2` | `connectrpc.com/vanguard` | On-the-fly REST |
| `go.uber.org/zap` | `log/slog` | Standard library |
| `github.com/spf13/viper` | `github.com/knadh/koanf/v2` | 50% fewer deps |
| `opentracing/opentracing-go` | `go.opentelemetry.io/otel` | CNCF standard |
| `go-grpc-middleware/v2` | (removed) | Connect native |
| `github.com/google/wire` | (removed) | Explicit DI |
| `github.com/pkg/errors` | `errors` | Go 1.13+ stdlib |
| `github.com/goriller/ginny-util` | Built-in | Self-contained |

### Key Code Changes

```go
// Server: Wire DI → explicit constructors
// v1
app, _ := ginny.NewApp(ginny.WithName("srv"), ginny.WithConfig(cfg))

// v2
srv := server.New(server.WithInterceptor(recovery.NewInterceptor(logger)))
app, _ := ginny.New(ginny.WithServer(srv), ginny.WithConfig(cfg))

// Registration: gRPC → Connect handler
// v1: pb.RegisterMyServiceServer(grpcServer, &impl{})
// v2: handler := myappv1connect.NewMyServiceHandler(&impl{})
//     srv.RegisterService("/myapp.v1.MyService/", handler)

// Errors: gRPC status → Connect
// v1: return nil, status.Errorf(codes.NotFound, "not found")
// v2: return nil, errs.New(connect.CodeNotFound, 40401, "not found")

// Client: grpc.Dial → http.Client
// v1: conn, _ := grpc.Dial(":9000", grpc.WithInsecure())
// v2: pool := client.NewPool()
//     typed := myappv1connect.NewMyServiceClient(pool.HTTPClient(), "http://:8080")
```

### Migration Checklist | 迁移清单

1. Update go.mod: `github.com/goriller/ginny/v2`
2. Replace protoc with buf.gen.yaml
3. Port interceptors (50% less code)
4. Rewrite server setup (no Wire)
5. Fix error returns: `status.Error` → `errs.New`
6. Update clients: `grpc.Dial` → `http.Client`
7. Remove old grpc-gateway annotations (optional)
8. Clean up old deps

---

## Performance | 性能

Unary RPC, ~1KB payload:

| Scenario | v1 | v2 |
|----------|----|----|
| Connect protocol | — | ~55,000 req/s |
| gRPC native | ~40,000 req/s | ~45,000 req/s |
| HTTP REST (JSON) | ~15,000 req/s | ~55,000 req/s |
| P99 latency | ~8ms | ~2ms |
