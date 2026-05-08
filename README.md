# Ginny v2

[English](#english) | [中文](#chinese)

Ginny is a **schema-driven Go RPC framework** built on ConnectRPC.  
Ginny 是一个基于 ConnectRPC 的 **Schema 驱动 Go RPC 框架**。

Protocols: **gRPC + Connect + gRPC-Web** (three protocols, one handler).  
协议支持：**gRPC + Connect + gRPC-Web** 三协议，单个 Handler 即插即用。

---

## English

### What's New in v2

| Feature | Description |
|---------|-------------|
| **ConnectRPC** | Three protocols unified: gRPC, Connect, gRPC-Web. No proxy, browser-native. |
| **Vanguard REST** | Transparent REST transcoding for legacy HTTP clients. Zero-cost compatibility. |
| **Dual Ports** | App(:8080) for RPC traffic, Admin(:8081) for /healthz, /readyz, /metrics, /debug/pprof. |
| **Lifecycle Hooks** | Ordered start/stop with priority. Hooks fire in order on boot, reverse on shutdown. |
| **slog Interface** | Framework uses `*slog.Logger` as the logging interface. Inject Zap handler or any custom backend. |
| **Connect Error Model** | Error details carry business codes (`errs.New(code, bizCode, msg)`). |
| **OpenTelemetry** | Built-in tracing interceptor. Drop-in OTel integration. |
| **Three-layer Interceptors** | HTTP middleware → Connect interceptor → Per-service override. No duplicated code. |
| **gRPC Health + Reflection** | Standard gRPC health protocol + reflection (grpcurl/grpcui ready). |
| **koanf Config** | Multi-source config: files (YAML/JSON/TOML) + env vars + remote providers. |
| **Explicit Constructor DI** | No Wire, no code generation. Straight-line construction: `Config → Logger → Server → App`. |
| **ginnytest** | Test helpers: one-line test server with full interceptor chain. |
| **Go 1.24** | Native `http.Protocols` h2c, enhanced ServeMux, generics throughout. |

### Architecture

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

### Installation

Requires **Go 1.24+**.

```
go get github.com/goriller/ginny/v2
```

**Dependencies for code generation:**

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

Optional: [Buf CLI](https://buf.build/docs/cli) replaces protoc entirely. See `buf/` for templates.

### Quick Start

Minimal runnable application in 5 lines:

```go
package main

import (
    "context"

    "github.com/goriller/ginny/v2"
    "github.com/goriller/ginny/v2/config"
    "github.com/goriller/ginny/v2/server"
)

func main() {
    cfg := config.MustLoad()                           // 1. load config
    srv := server.New()                                // 2. create server
    app, _ := ginny.New(                               // 3. create app
        ginny.WithConfig(cfg),
        ginny.WithServer(srv),
    )
    app.Start(context.Background())                    // 4. start (blocks until signal)
}
```

Run it:

```
go run main.go
```

Check:

```
curl http://localhost:8081/healthz   # → {"status":"ok"}
curl http://localhost:8081/metrics   # → Prometheus metrics
```

#### With Interceptors

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

    // Create structured logger (JSON to stderr, Info level)
    logger := log.New(
        log.WithLevel(slog.LevelInfo),
        log.WithSource(true),
    )

    // Create server with full interceptor chain
    srv := server.New(
        server.WithAppAddr(cfg.Server.Addr),       // default ":8080"
        server.WithAdminAddr(cfg.Admin.Addr),      // default ":8081"
        server.WithLogger(logger),

        // Layer 2: Connect interceptors (RPC only)
        server.WithInterceptor(recovery.NewInterceptor(logger)),
        server.WithInterceptor(tracing.NewInterceptor()),
        server.WithInterceptor(logging.NewInterceptor(logger,
            logging.WithLevel(slog.LevelInfo),
        )),
        server.WithInterceptor(validation.NewInterceptor()),
        // server.WithInterceptor(auth.NewInterceptor(myValidator,
        //     auth.SkipProcedures("/myapp.v1.Health/Check"),
        // )),
        // server.WithInterceptor(ratelimit.NewInterceptor(myLimiter)),

        // Dev-friendly features
        server.WithReflection(true),
        server.WithReflectionServiceNames("myapp.v1.MyService"),
        server.WithDebug(true), // pprof on admin port
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

### Registering Services

ConnectRPC service handlers are standard `http.Handler`. Register them on the server:

```go
// Generated by protoc-gen-connect-go
import "myapp/v1/myappv1connect"

handler := myappv1connect.NewMyServiceHandler(
    &myServiceImpl{},
    // Per-service interceptors (Layer 3)
)
srv.RegisterService("/myapp.v1.MyService/", handler)
```

### Interceptors

Ginny v2 uses a **three-layer interceptor model**:

| Layer | Scope | Examples |
|-------|-------|----------|
| 1. HTTP middleware | All routes (app + admin) | Recovery, CORS, Compression |
| 2. Connect interceptor | RPC handlers only | Tracing, Auth, RateLimit, Validation, Logging |
| 3. Per-service | Specific service | Custom logic per service |

#### Recovery

```go
import "github.com/goriller/ginny/v2/interceptor/recovery"

server.WithInterceptor(recovery.NewInterceptor(logger))
```

Catches panics in RPC handlers, logs stack traces, and converts them to `CodeInternal` errors.

#### Logging

```go
import "github.com/goriller/ginny/v2/interceptor/logging"

server.WithInterceptor(logging.NewInterceptor(logger,
    logging.WithLevel(slog.LevelDebug), // verbose logging
))
```

Logs every RPC: procedure name, duration, status code. Errors are always logged at Error level.

#### Tracing (OpenTelemetry)

```go
import (
    "go.opentelemetry.io/otel"
    "github.com/goriller/ginny/v2/interceptor/tracing"
)

// Option 1: default (uses global OTel TracerProvider)
server.WithInterceptor(tracing.NewInterceptor())

// Option 2: custom TracerProvider
server.WithInterceptor(tracing.NewInterceptor(
    tracing.WithTracerProvider(myProvider),
))
```

Creates a span for every RPC with `rpc.system`, `rpc.service`, `rpc.peer` attributes.

#### Auth

```go
import "github.com/goriller/ginny/v2/interceptor/auth"

// Implement TokenValidator
type myValidator struct{}
func (v *myValidator) Validate(ctx context.Context, token string) (auth.Identity, error) {
    // validate JWT / OAuth2 token
    return &myIdentity{userID: "123"}, nil
}

server.WithInterceptor(auth.NewInterceptor(&myValidator{},
    auth.SkipProcedures(
        "/myapp.v1.AuthService/Login",
        "/myapp.v1.Health/Check",
    ),
))
```

Extracts Bearer token from Authorization header, validates it, and stores identity in context (`auth.GetIdentity(ctx)`).

#### Rate Limiting

```go
import "github.com/goriller/ginny/v2/interceptor/ratelimit"

// Implement Limiter (local or distributed via Redis)
type myLimiter struct{}
func (l *myLimiter) Allow(ctx context.Context, key string) (bool, error) {
    // check rate limit
    return true, nil
}

server.WithInterceptor(ratelimit.NewInterceptor(&myLimiter{},
    ratelimit.WithKeyFunc(func(ctx context.Context, req connect.AnyRequest) string {
        // custom key: per-user or per-IP
        return req.Peer().Addr
    }),
))
```

Returns `CodeResourceExhausted` when limit is exceeded.

#### Validation

```go
import "github.com/goriller/ginny/v2/interceptor/validation"

server.WithInterceptor(validation.NewInterceptor(
    validation.SkipProcedures("/myapp.v1.Internal/NoValidation"),
))
```

Automatically calls `Validate()` on request messages that implement the `Validator` interface. Works with protoc-gen-validate or custom validation.

### Configuration

Uses **koanf** for multi-source configuration.

```go
import "github.com/goriller/ginny/v2/config"

// Default: loads ./configs/config.yaml + env vars (prefix GINNY_)
cfg := config.MustLoad()

// Custom path
cfg := config.MustLoad(config.WithConfigPath("./my-config.yaml"))

// Custom env prefix
cfg := config.MustLoad(config.WithEnvPrefix("MYAPP"))
```

**config.yaml example:**

```yaml
app:
  name: my-service
  version: 1.0.0
  env: dev
server:
  addr: ":9090"
admin:
  addr: ":9091"
logging:
  level: debug
  format: json
```

**Environment variable overrides:**

```
export GINNY_APP_NAME=my-service-prod
export GINNY_SERVER_ADDR=:8080
export GINNY_LOGGING_LEVEL=info
```

### Error Handling

Connect error model with business code details:

```go
import "github.com/goriller/ginny/v2/errs"

// Create an error with business code
func (s *MyService) GetUser(ctx context.Context, req *connect.Request[pb.GetUserReq]) (*connect.Response[pb.GetUserResp], error) {
    user, err := s.repo.Find(ctx, req.Msg.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, errs.New(connect.CodeNotFound, 40401, "user not found")
        }
        return nil, errs.New(connect.CodeInternal, 50001, "database error")
    }
    return connect.NewResponse(&pb.GetUserResp{User: user}), nil
}
```

**Extracting business codes on the client side:**

```go
resp, err := client.GetUser(ctx, req)
if err != nil {
    bizCode := errs.BizCode(err)    // 40401
    bizMsg  := errs.BizMessage(err) // "user not found"
}
```

**Register business codes for documentation:**

```go
errs.RegisterBizCodes(map[int32]string{
    40401: "USER_NOT_FOUND",
    50001: "DATABASE_ERROR",
})
```

### Logging

Slog interface with pluggable handlers:

```go
import "github.com/goriller/ginny/v2/log"

// Default: JSON to stderr, Info level
logger := log.New()

// With options
logger := log.New(
    log.WithLevel(slog.LevelDebug),
    log.WithSource(true),
)

// Inject a custom handler (e.g., Zap backend)
logger := log.New(
    log.WithHandler(zapHandler), // any slog.Handler
)
```

Use `log.GetContextLogger(ctx)` / `log.SetContextLogger(ctx, logger)` for context propagation.

### Client

Connection pool with retry and circuit breaker, built on top of ConnectRPC's typed clients:

```go
import (
    "github.com/goriller/ginny/v2/client"
    "myapp/v1/myappv1connect"
)

// Create a pool with retry and timeout
pool := client.NewPool(
    client.WithTimeout(10 * time.Second),
    client.WithRetry(3),
    client.WithCircuitBreaker(true),
)
defer pool.Close(context.Background())

// Use with generated Connect client
typedClient := myappv1connect.NewMyServiceClient(
    pool.HTTPClient(),
    "http://backend:8080",
)
resp, err := typedClient.GetUser(ctx, connect.NewRequest(&pb.GetUserReq{Id: "123"}))
```

### Lifecycle Hooks

Components that need ordered start/stop implement `LifecycleHook`:

```go
type LifecycleHook interface {
    Name() string
    OnStart(ctx context.Context) error
    OnStop(ctx context.Context) error
    Priority() int // lower = starts earlier, stops later
}

// Built-in hooks (priority order):
//  10: Admin server start
//  20: App server listen
//  30: Service discovery register
//  40: Readiness probe set ready
//
// Shutdown runs in reverse: 40 → 30 → 20 → 10
```

**Adding a custom hook:**

```go
app, _ := ginny.New(
    ginny.WithServer(srv),
    ginny.WithHook(&myDatabaseHook{}),
)
```

### Testing

Use `ginnytest` for integration tests:

```go
import "github.com/goriller/ginny/v2/ginnytest"

func TestMyService(t *testing.T) {
    // Start a test server with full interceptor chain
    ts, err := ginnytest.NewTestServer(t,
        server.WithInterceptor(recovery.NewInterceptor(slog.Default())),
        server.WithInterceptor(logging.NewInterceptor(slog.Default())),
    )
    require.NoError(t, err)
    // ts.Close() is called automatically via t.Cleanup

    // Register your service
    ts.Server().RegisterService("/myapp.v1.MyService/", myHandler)

    // Create a typed client pointing at the test server
    client := myappv1connect.NewMyServiceClient(
        ts.Client(),
        ts.BaseURL(),
    )

    // Test your RPC
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&pb.GetUserReq{Id: "1"}))
    require.NoError(t, err)
}
```

### Package Layout

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

### Migration from v1 to v2

#### Module Path

```
// v1
github.com/goriller/ginny

// v2 — both can coexist in go.mod during migration
github.com/goriller/ginny/v2
```

#### Dependency Changes

| v1 | v2 | Reason |
|----|----|--------|
| `google.golang.org/grpc` | `connectrpc.com/connect` | Lighter, browser-native |
| `grpc-ecosystem/grpc-gateway/v2` | `connectrpc.com/vanguard` | On-the-fly REST transcoding |
| `go.uber.org/zap` | `log/slog` (interfacable) | Standard library, pluggable |
| `github.com/spf13/viper` | `github.com/knadh/koanf/v2` | Lighter, ~50% fewer indirect deps |
| `opentracing/opentracing-go` | `go.opentelemetry.io/otel` | CNCF standard |
| `go-grpc-middleware/v2` | (removed) | ConnectRPC has native interceptors |
| `github.com/google/wire` | (removed) | Explicit constructor DI |
| `github.com/pkg/errors` | `errors` (stdlib) | Go 1.13+ |
| `github.com/goriller/ginny-util` | Built into ginny | Reduce external deps |

#### Code Changes

**1. Server creation — v1 vs v2:**

```go
// v1: Wire DI + grpc-gateway
app, err := ginny.NewApp(
    ginny.WithName("myservice"),
    ginny.WithConfig(cfg),
)
// ... wire.go, wire_gen.go, provider sets

// v2: Explicit construction, no code generation
srv := server.New(
    server.WithInterceptor(recovery.NewInterceptor(logger)),
    server.WithInterceptor(logging.NewInterceptor(logger)),
)
app, _ := ginny.New(
    ginny.WithName("myservice"),
    ginny.WithConfig(cfg),
    ginny.WithLogger(logger),
    ginny.WithServer(srv),
)
```

**2. Service registration — gRPC → Connect:**

```go
// v1
pb.RegisterMyServiceServer(grpcServer, &myServiceImpl{})

// v2: handler is http.Handler, compatible with any router
handler := myappv1connect.NewMyServiceHandler(&myServiceImpl{})
srv.RegisterService("/myapp.v1.MyService/", handler)
```

**3. Interceptors — two sets → one set:**

```go
// v1: two separate systems
//   grpc.UnaryServerInterceptor (for gRPC)
//   middleware.MuxMiddleware (for HTTP/gateway)
// Auth, Limit, Tracer written twice.

// v2: single connect.Interceptor type
srv := server.New(
    server.WithInterceptor(recovery.NewInterceptor(logger)),
    server.WithInterceptor(tracing.NewInterceptor()),
    server.WithInterceptor(auth.NewInterceptor(validator)),
)
```

**4. Error handling — gRPC status → Connect error:**

```go
// v1
return nil, status.Errorf(codes.NotFound, "user not found")

// v2: with business code details
return nil, errs.New(connect.CodeNotFound, 40401, "user not found")
```

**5. Client — gRPC conn → http.Client:**

```go
// v1
conn, _ := grpc.Dial("localhost:9000", grpc.WithInsecure())
client := pb.NewMyServiceClient(conn)

// v2: standard http.Client
pool := client.NewPool(client.WithTimeout(10 * time.Second))
typedClient := myappv1connect.NewMyServiceClient(
    pool.HTTPClient(),
    "http://localhost:8080",
)
```

**6. Proto generation — protoc → buf:**

```bash
# v1
protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. api.proto

# v2: single command
buf generate
```

**7. REST endpoints — grpc-gateway annotations → Vanguard:**

No proto changes required! Vanguard reads existing `google.api.http` annotations. Just enable:

```go
server.WithRESTTranscoding(true) // default on
```

Old REST clients continue working without modification.

#### Migration Steps

```
1. Update go.mod: github.com/goriller/ginny/v2
2. Replace protoc gen plugins with buf.gen.yaml
3. Port interceptors: middleware → connect.Interceptor (50% less code)
4. Rewrite server setup: Wire DI → explicit constructors
5. Fix error returns: status.Error → errs.New
6. Update client code: grpc.Dial → http.Client + pool
7. Remove old grpc-gateway annotations from proto (optional)
8. Remove old deps from go.mod
```

### Performance

Based on ConnectRPC benchmarks (Unary RPC, ~1KB payload):

| Scenario | v1 (gRPC-gateway) | v2 (Connect) |
|----------|-------------------|--------------|
| Connect protocol | — | ~55,000 req/s |
| gRPC protocol | ~40,000 req/s | ~45,000 req/s |
| HTTP REST (JSON) | ~15,000 req/s | ~55,000 req/s |
| P99 latency | ~8ms | ~2ms |

v2 removes JSON serialization overhead for RPC calls. Vanguard REST transcoding is optional and only applied to legacy HTTP clients.

---

## 中文

### v2 新特性

| 特性 | 说明 |
|------|------|
| **ConnectRPC** | gRPC + Connect + gRPC-Web 三协议统一。无需代理，浏览器原生支持。 |
| **Vanguard REST 转码** | 旧 REST 客户端零成本兼容。使用现有 `google.api.http` 注解自动转码。 |
| **双端口架构** | App(:8080) 业务流量，Admin(:8081) 健康检查/指标/pprof。安全隔离。 |
| **Lifecycle Hooks** | 有序启停。启动按优先级正序执行，关闭逆序执行。 |
| **slog 接口化日志** | 框架使用 `*slog.Logger` 接口。可注入 Zap handler 或任意自定义后端。 |
| **Connect 错误模型** | Error Detail 传递业务码，客户端可提取 `errs.BizCode(err)`。 |
| **OpenTelemetry 链路追踪** | 内置 tracing interceptor，与 OTel 生态无缝集成。 |
| **三层 Interceptor** | HTTP 中间件 → Connect 拦截器 → 单服务覆盖。各层独立，消除 v1 重复代码。 |
| **gRPC Health + Reflection** | 标准 gRPC 健康检查协议 + 反射（grpcurl/grpcui 立即可用）。 |
| **koanf 配置管理** | 多源配置：文件（YAML/JSON/TOML）+ 环境变量 + 远程 provider。 |
| **显式构造函数 DI** | 移除 Wire，无需代码生成。`Config → Logger → Server → App` 直线构造。 |
| **ginnytest 测试包** | 一行代码启动带完整拦截器链的测试服务器。 |
| **Go 1.24** | 原生 `http.Protocols` h2c，增强 ServeMux，全面泛型支持。 |

### 架构图

参见上方 [Architecture](#architecture) 部分。

### 安装

需要 **Go 1.24+**。

```
go get github.com/goriller/ginny/v2
```

**代码生成依赖：**

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

可选：[Buf CLI](https://buf.build/docs/cli) 可完全替代 protoc。模板见 `buf/` 目录。

### 快速开始

最小可运行代码（5 行）：

```go
package main

import (
    "context"

    "github.com/goriller/ginny/v2"
    "github.com/goriller/ginny/v2/config"
    "github.com/goriller/ginny/v2/server"
)

func main() {
    cfg := config.MustLoad()                           // 1. 加载配置
    srv := server.New()                                // 2. 创建服务器
    app, _ := ginny.New(                               // 3. 创建应用
        ginny.WithConfig(cfg),
        ginny.WithServer(srv),
    )
    app.Start(context.Background())                    // 4. 启动（阻塞直到收到信号）
}
```

运行：

```
go run main.go
```

验证：

```
curl http://localhost:8081/healthz   # → {"status":"ok"}
curl http://localhost:8081/metrics   # → Prometheus 指标
```

#### 带拦截器的完整示例

```go
package main

import (
    "context"
    "log/slog"

    "github.com/goriller/ginny/v2"
    "github.com/goriller/ginny/v2/config"
    "github.com/goriller/ginny/v2/interceptor/logging"
    "github.com/goriller/ginny/v2/interceptor/recovery"
    "github.com/goriller/ginny/v2/interceptor/tracing"
    "github.com/goriller/ginny/v2/interceptor/validation"
    "github.com/goriller/ginny/v2/log"
    "github.com/goriller/ginny/v2/server"
)

func main() {
    cfg := config.MustLoad()

    // 结构化日志（JSON 输出到 stderr，Info 级别）
    logger := log.New(
        log.WithLevel(slog.LevelInfo),
        log.WithSource(true),
    )

    // 带完整拦截器链的服务器
    srv := server.New(
        server.WithAppAddr(cfg.Server.Addr),       // 默认 ":8080"
        server.WithAdminAddr(cfg.Admin.Addr),      // 默认 ":8081"
        server.WithLogger(logger),

        // Layer 2: Connect 拦截器（仅作用于 RPC 请求）
        server.WithInterceptor(recovery.NewInterceptor(logger)),
        server.WithInterceptor(tracing.NewInterceptor()),
        server.WithInterceptor(logging.NewInterceptor(logger,
            logging.WithLevel(slog.LevelInfo),
        )),
        server.WithInterceptor(validation.NewInterceptor()),
        // server.WithInterceptor(auth.NewInterceptor(myValidator,
        //     auth.SkipProcedures("/myapp.v1.Health/Check"),
        // )),
        // server.WithInterceptor(ratelimit.NewInterceptor(myLimiter)),

        // 开发友好功能
        server.WithReflection(true),
        server.WithReflectionServiceNames("myapp.v1.MyService"),
        server.WithDebug(true), // Admin 端口启用 pprof
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

### 注册服务

ConnectRPC 的 handler 是标准 `http.Handler`，直接注册即可：

```go
// 由 protoc-gen-connect-go 生成
import "myapp/v1/myappv1connect"

handler := myappv1connect.NewMyServiceHandler(
    &myServiceImpl{},
    // Layer 3: 单服务级 interceptor
)
srv.RegisterService("/myapp.v1.MyService/", handler)
```

### 拦截器详解

Ginny v2 采用**三层拦截器模型**：

| 层级 | 作用范围 | 示例 |
|------|---------|------|
| 1. HTTP 中间件 | 所有路由（app + admin） | Recovery, CORS, Compression |
| 2. Connect 拦截器 | 仅 RPC handler | Tracing, Auth, RateLimit, Validation, Logging |
| 3. 单服务级 | 特定服务 | 自定义业务逻辑 |

#### Recovery（Panic 恢复）

```go
import "github.com/goriller/ginny/v2/interceptor/recovery"

server.WithInterceptor(recovery.NewInterceptor(logger))
```

捕获 RPC handler 中的 panic，记录完整堆栈，转换为 `CodeInternal` 错误。

#### Logging（请求日志）

```go
import "github.com/goriller/ginny/v2/interceptor/logging"

server.WithInterceptor(logging.NewInterceptor(logger,
    logging.WithLevel(slog.LevelDebug), // 调试模式记录所有请求
))
```

记录每次 RPC 请求的过程名、耗时、状态码。错误始终以 Error 级别输出。

#### Tracing（链路追踪）

```go
import (
    "go.opentelemetry.io/otel"
    "github.com/goriller/ginny/v2/interceptor/tracing"
)

// 方式一：使用全局 OTel TracerProvider
server.WithInterceptor(tracing.NewInterceptor())

// 方式二：自定义 TracerProvider
server.WithInterceptor(tracing.NewInterceptor(
    tracing.WithTracerProvider(myProvider),
))
```

为每次 RPC 创建 span，包含 `rpc.system`、`rpc.service`、`rpc.peer` 属性。

#### Auth（认证）

```go
import "github.com/goriller/ginny/v2/interceptor/auth"

// 实现 TokenValidator
type myValidator struct{}
func (v *myValidator) Validate(ctx context.Context, token string) (auth.Identity, error) {
    // 验证 JWT / OAuth2 token
    return &myIdentity{userID: "123"}, nil
}

server.WithInterceptor(auth.NewInterceptor(&myValidator{},
    auth.SkipProcedures(              // 跳过认证的接口
        "/myapp.v1.AuthService/Login",
        "/myapp.v1.Health/Check",
    ),
))
```

自动提取 Authorization header 中的 Bearer token，验证通过后将身份信息存入 context（`auth.GetIdentity(ctx)`）。

#### Rate Limiting（限流）

```go
import "github.com/goriller/ginny/v2/interceptor/ratelimit"

// 实现 Limiter 接口（本地或分布式 Redis 实现）
type myLimiter struct{}
func (l *myLimiter) Allow(ctx context.Context, key string) (bool, error) {
    return true, nil
}

server.WithInterceptor(ratelimit.NewInterceptor(&myLimiter{},
    ratelimit.WithKeyFunc(func(ctx context.Context, req connect.AnyRequest) string {
        return req.Peer().Addr // 按 IP 限流
    }),
))
```

超限返回 `CodeResourceExhausted`。

#### Validation（请求校验）

```go
import "github.com/goriller/ginny/v2/interceptor/validation"

server.WithInterceptor(validation.NewInterceptor(
    validation.SkipProcedures("/myapp.v1.Internal/NoValidation"),
))
```

自动调用请求消息的 `Validate()` 方法（需实现 `Validator` 接口）。配合 protoc-gen-validate 或自定义校验。

### 配置

使用 **koanf** 实现多源配置。

```go
import "github.com/goriller/ginny/v2/config"

// 默认：加载 ./configs/config.yaml + 环境变量（前缀 GINNY_）
cfg := config.MustLoad()

// 自定义路径
cfg := config.MustLoad(config.WithConfigPath("./my-config.yaml"))

// 自定义环境变量前缀
cfg := config.MustLoad(config.WithEnvPrefix("MYAPP"))
```

**config.yaml 示例：**

```yaml
app:
  name: my-service
  version: 1.0.0
  env: dev
server:
  addr: ":9090"
admin:
  addr: ":9091"
logging:
  level: debug
  format: json
```

**环境变量覆盖：**

```
export GINNY_APP_NAME=my-service-prod
export GINNY_SERVER_ADDR=:8080
export GINNY_LOGGING_LEVEL=info
```

### 错误处理

Connect 错误模型 + 业务码 Detail：

```go
import "github.com/goriller/ginny/v2/errs"

func (s *MyService) GetUser(ctx context.Context, req *connect.Request[pb.GetUserReq]) (*connect.Response[pb.GetUserResp], error) {
    user, err := s.repo.Find(ctx, req.Msg.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, errs.New(connect.CodeNotFound, 40401, "用户不存在")
        }
        return nil, errs.New(connect.CodeInternal, 50001, "数据库错误")
    }
    return connect.NewResponse(&pb.GetUserResp{User: user}), nil
}
```

**客户端提取业务码：**

```go
resp, err := client.GetUser(ctx, req)
if err != nil {
    bizCode := errs.BizCode(err)    // 40401
    bizMsg  := errs.BizMessage(err) // "用户不存在"
}
```

**注册业务码：**

```go
errs.RegisterBizCodes(map[int32]string{
    40401: "USER_NOT_FOUND",
    50001: "DATABASE_ERROR",
})
```

### 日志

slog 接口 + 可插拔 handler：

```go
import "github.com/goriller/ginny/v2/log"

// 默认：JSON 输出到 stderr，Info 级别
logger := log.New()

// 带选项
logger := log.New(
    log.WithLevel(slog.LevelDebug),
    log.WithSource(true),
)

// 注入自定义 handler（如 Zap 后端）
logger := log.New(
    log.WithHandler(zapHandler), // 任意 slog.Handler 实现
)
```

通过 `log.GetContextLogger(ctx)` / `log.SetContextLogger(ctx, logger)` 在 context 中传递 logger。

### 客户端

在 ConnectRPC typed client 之上提供连接池、重试和熔断：

```go
import (
    "github.com/goriller/ginny/v2/client"
    "myapp/v1/myappv1connect"
)

// 创建带重试和超时的连接池
pool := client.NewPool(
    client.WithTimeout(10 * time.Second),
    client.WithRetry(3),
    client.WithCircuitBreaker(true),
)
defer pool.Close(context.Background())

// 配合生成的 Connect client 使用
typedClient := myappv1connect.NewMyServiceClient(
    pool.HTTPClient(),
    "http://backend:8080",
)
resp, err := typedClient.GetUser(ctx, connect.NewRequest(&pb.GetUserReq{Id: "123"}))
```

### Lifecycle Hooks

需有序启停的组件实现 `LifecycleHook` 接口：

```go
type LifecycleHook interface {
    Name() string
    OnStart(ctx context.Context) error
    OnStop(ctx context.Context) error
    Priority() int // 数字越小越先启动、越后关闭
}

// 内建 hooks 优先级顺序：
//  10: Admin server 启动
//  20: App server 监听
//  30: 服务注册
//  40: Readiness probe 设置为就绪
//
// 关闭按逆序执行：40 → 30 → 20 → 10
```

**添加自定义 hook：**

```go
app, _ := ginny.New(
    ginny.WithServer(srv),
    ginny.WithHook(&myDatabaseHook{}),
)
```

### 测试

使用 `ginnytest` 进行集成测试：

```go
import "github.com/goriller/ginny/v2/ginnytest"

func TestMyService(t *testing.T) {
    // 一行启动带完整拦截器链的测试服务器
    ts, err := ginnytest.NewTestServer(t,
        server.WithInterceptor(recovery.NewInterceptor(slog.Default())),
        server.WithInterceptor(logging.NewInterceptor(slog.Default())),
    )
    require.NoError(t, err)
    // ts.Close() 由 t.Cleanup 自动调用

    // 注册服务
    ts.Server().RegisterService("/myapp.v1.MyService/", myHandler)

    // 创建 typed client 指向测试服务器
    client := myappv1connect.NewMyServiceClient(
        ts.Client(),
        ts.BaseURL(),
    )

    // 测试 RPC
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&pb.GetUserReq{Id: "1"}))
    require.NoError(t, err)
}
```

### 包结构

```
ginny/              # App 入口 + LifecycleHook
├── server/         # 双端口 Server（App + Admin）
├── interceptor/    # auth, tracing, recovery, ratelimit, logging, validation
├── config/         # koanf 配置管理
├── log/            # slog 接口 + 辅助函数
├── errs/           # Connect 错误 + 业务码
├── client/         # 连接池 + 重试 + 熔断
├── ginnytest/      # 测试工具包
├── example/        # 最小示例
└── buf/            # 代码生成模板
```

### v1 迁移到 v2

#### Module 路径

```
// v1
github.com/goriller/ginny

// v2 — 迁移期间两者可共存于同一 go.mod
github.com/goriller/ginny/v2
```

#### 依赖变更

| v1 | v2 | 原因 |
|----|----|------|
| `google.golang.org/grpc` | `connectrpc.com/connect` | 更轻量，浏览器原生 |
| `grpc-ecosystem/grpc-gateway/v2` | `connectrpc.com/vanguard` | 运行时 REST 转码 |
| `go.uber.org/zap` | `log/slog`（可插拔） | 标准库，零依赖 |
| `github.com/spf13/viper` | `github.com/knadh/koanf/v2` | 轻量 ~50% 间接依赖 |
| `opentracing/opentracing-go` | `go.opentelemetry.io/otel` | CNCF 标准 |
| `go-grpc-middleware/v2` | （移除） | ConnectRPC 原生拦截器 |
| `github.com/google/wire` | （移除） | 显式构造函数 DI |
| `github.com/pkg/errors` | `errors`（标准库） | Go 1.13+ |
| `github.com/goriller/ginny-util` | 内建入 ginny | 减少外部依赖 |

#### 代码变更要点

**1. 服务器创建 — v1 vs v2：**

```go
// v1: Wire DI + grpc-gateway
app, err := ginny.NewApp(
    ginny.WithName("myservice"),
    ginny.WithConfig(cfg),
)
// ... wire.go, wire_gen.go, provider sets

// v2: 显式构造，无线生成代码
srv := server.New(
    server.WithInterceptor(recovery.NewInterceptor(logger)),
    server.WithInterceptor(logging.NewInterceptor(logger)),
)
app, _ := ginny.New(
    ginny.WithName("myservice"),
    ginny.WithConfig(cfg),
    ginny.WithLogger(logger),
    ginny.WithServer(srv),
)
```

**2. 服务注册 — gRPC → Connect：**

```go
// v1
pb.RegisterMyServiceServer(grpcServer, &myServiceImpl{})

// v2: handler 是 http.Handler，兼容任意路由
handler := myappv1connect.NewMyServiceHandler(&myServiceImpl{})
srv.RegisterService("/myapp.v1.MyService/", handler)
```

**3. 拦截器 — 两套合并为统一类型：**

```go
// v1: 两套独立系统
//   grpc.UnaryServerInterceptor（gRPC 用）
//   middleware.MuxMiddleware（HTTP/gateway 用）
// Auth, Limit, Tracer 各写两遍

// v2: 单一 connect.Interceptor 类型
srv := server.New(
    server.WithInterceptor(recovery.NewInterceptor(logger)),
    server.WithInterceptor(tracing.NewInterceptor()),
    server.WithInterceptor(auth.NewInterceptor(validator)),
)
```

**4. 错误处理 — gRPC status → Connect error：**

```go
// v1
return nil, status.Errorf(codes.NotFound, "user not found")

// v2: 带业务码 Detail
return nil, errs.New(connect.CodeNotFound, 40401, "user not found")
```

**5. 客户端 — gRPC conn → http.Client：**

```go
// v1
conn, _ := grpc.Dial("localhost:9000", grpc.WithInsecure())
client := pb.NewMyServiceClient(conn)

// v2: 标准 http.Client
pool := client.NewPool(client.WithTimeout(10 * time.Second))
typedClient := myappv1connect.NewMyServiceClient(
    pool.HTTPClient(),
    "http://localhost:8080",
)
```

**6. Proto 生成 — protoc → buf：**

```bash
# v1
protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. api.proto

# v2: 一条命令
buf generate
```

**7. REST 端点 — grpc-gateway 注解 → Vanguard：**

无需修改 proto 文件！Vanguard 直接读取现有的 `google.api.http` 注解：

```go
server.WithRESTTranscoding(true) // 默认开启
```

旧 REST 客户端无需任何修改，继续正常工作。

#### 迁移步骤

```
1. 更新 go.mod: github.com/goriller/ginny/v2
2. 用 buf.gen.yaml 替换 protoc 插件配置
3. 迁移拦截器: middleware → connect.Interceptor（代码量减少 50%+）
4. 重写服务端初始化: Wire DI → 显式构造函数
5. 修正错误返回: status.Error → errs.New
6. 更新客户端代码: grpc.Dial → http.Client + pool
7. （可选）从 proto 移除旧的 grpc-gateway 注解
8. 从 go.mod 移除旧依赖
```

### 性能

基于 ConnectRPC 官方 benchmark（Unary RPC，~1KB 负载）：

| 场景 | v1 (gRPC-gateway) | v2 (Connect) |
|------|-------------------|--------------|
| Connect 协议 | — | ~55,000 req/s |
| gRPC 协议 | ~40,000 req/s | ~45,000 req/s |
| HTTP REST (JSON) | ~15,000 req/s | ~55,000 req/s |
| P99 延迟 | ~8ms | ~2ms |

v2 消除了 RPC 调用的 JSON 序列化开销。Vanguard REST 转码仅对遗留 HTTP 客户端启用，不影响 RPC 性能。

---

### Links

- [ConnectRPC 官方文档](https://connectrpc.com/docs/go/getting-started)
- [Buf CLI](https://buf.build/docs/cli)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [koanf](https://github.com/knadh/koanf)
