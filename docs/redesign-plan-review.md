# Ginny v2 改造方案评审与优化建议

> 评审日期：2026-05-08
> 评审对象：`docs/redesign-plan.md`
> 结论：方案核心方向正确，需补强 5 个关键缺失 + 若干架构优化

---

## 总体评价

架构师方案的**技术选型方向正确**：ConnectRPC + OpenTelemetry + Buf CLI + 轻量化是 2026 年 Go RPC 框架的最佳组合。以下从关键缺失、架构优化、选型修正、开发体验、实施计划五个维度给出完善建议。

---

## 一、关键缺失（必须补充）

### 1. Vanguard REST 转码层 [Critical]

方案移除 grpc-gateway 但**未提供 REST 客户端向后兼容方案**。ConnectRPC 官方生态有 [`connectrpc.com/vanguard`](https://github.com/connectrpc/vanguard-go)，可在不修改 Connect handler 的前提下提供完整 REST 转码：

```go
import "connectrpc.com/vanguard"

// 现有 Connect handler 无需改动，Vanguard 自动转码
services := []*vanguard.Service{
    vanguard.NewService(rpcRoute, rpcHandler),
}
transcoder, _ := vanguard.NewTranscoder(services,
    vanguard.WithRules(httpRules...), // 支持 google.api.http 注解
)
mux.Handle("/", transcoder)
```

**价值**：
- 旧 REST 客户端零成本兼容，无需等迁移完再上线
- 使用与 grpc-gateway 相同的 `google.api.http` annotation，proto 文件可渐进移除注解
- 零额外代码生成，运行时转码
- Connect/gRPC 请求不经过转码层，无性能损失
- 由 ConnectRPC 核心团队维护，Benchmark Score 91.4/100

**建议**：将 Vanguard 作为 `server/` 包的可选层，默认启用，通过 `server.WithRESTTranscoding(false)` 关闭。

### 2. gRPC Reflection 服务 [High]

方案未提 gRPC reflection。生产环境 debug（grpcurl/grpcui/Postman）依赖此功能：

```go
import "connectrpc.com/grpcreflect"

reflector := grpcreflect.NewStaticReflector(serviceNames...)
mux.Handle(grpcreflect.NewHandlerV1(reflector))
mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
```

**建议**：开发环境默认启用，生产环境通过配置控制。

### 3. Health Check 协议标准化 [High]

v1 有 gRPC Health Check，v2 方案图中画了 `/healthz` HTTP endpoint 但未说明是否保留 gRPC Health 协议。Kubernetes 1.24+ 支持 gRPC health probe，两者都应保留：

```go
// HTTP health (Kubernetes HTTP probe, 负载均衡器)
mux.HandleFunc("GET /healthz", healthHandler)
mux.HandleFunc("GET /readyz", readinessHandler)

// gRPC health (Kubernetes gRPC probe, gRPC 客户端 LB)
mux.Handle(grpchealth.NewHandler(checker))
```

### 4. 双端口架构 [Medium]

方案将 `/healthz`、`/metrics`、`/debug/pprof` 与业务流量混在同一端口。生产环境应分离：

```
Port 8080 (App):     Connect + gRPC + gRPC-Web + REST(Vanguard)
Port 8081 (Admin):   /healthz, /readyz, /metrics, /debug/pprof
```

理由：
- 安全隔离（admin 端口只对内网/Pod 内部开放）
- Kubernetes 最佳实践（probePort != containerPort）
- 业务流量和运维流量独立，不互相影响

### 5. Client 层价值定位 [Medium]

ConnectRPC 已自动生成类型安全客户端，Ginny v2 的 `client/` 包应聚焦于 ConnectRPC 生成代码**之上**的增值能力：

- 连接池管理 + 服务发现集成
- 统一重试策略 + 熔断器（circuit breaker）
- 分布式限流客户端
- OTel 客户端拦截器预配置

而非重新封装已有的 typed client。

---

## 二、架构优化

### 1. Go 版本目标修正

方案写 "Go 1.23+"，建议调整：

| 最低版本 | 推荐版本 | 关键特性 |
|---------|---------|---------|
| Go 1.22 | Go 1.24+ | 1.22: enhanced ServeMux（方法+路径模式路由）; 1.24: `http.Protocols` 原生 h2c |

Go 1.24 的 `http.Protocols` 取代手动 h2c 配置：

```go
// Go 1.24+ 原生方式，取代 golang.org/x/net/http2/h2c
p := new(http.Protocols)
p.SetHTTP1(true)
p.SetUnencryptedHTTP2(true) // h2c for gRPC without TLS
srv := &http.Server{
    Handler:   mux,
    Protocols: p,
}
```

### 2. 统一生命周期管理

v1 的优雅关闭序列设计精良，应在 v2 中标准化为 Lifecycle Hook 模式：

```go
type LifecycleHook interface {
    OnStart(ctx context.Context) error
    OnStop(ctx context.Context) error
    Priority() int // 数字越小越先启动、越后关闭
}

// 内建 hooks 按 priority 排列:
// 10: MetricsServer.Start
// 20: AppServer.ListenAndServe
// 30: ServiceDiscovery.Register
// 40: ReadinessProbe.SetReady
//
// 关闭时逆序:
// 40: ReadinessProbe.SetNotReady
// 30: ServiceDiscovery.Deregister + drain period
// 20: AppServer.Shutdown (with timeout)
// 10: MetricsServer.Shutdown
```

### 3. Interceptor 分层设计

ConnectRPC interceptor 只作用于 RPC handler，HTTP 静态路由不经过。需明确分层：

```
Layer 1: net/http middleware (全局，包括 admin 路由)
    → Recovery, RequestID, CORS, Compression

Layer 2: Connect interceptor (仅 RPC handler)
    → OTel, Auth, RateLimit, Validation, Logging

Layer 3: Per-service interceptor (单服务覆盖)
    → 业务特定逻辑 (如特定 service 的限流策略)
```

### 4. 错误模型重设计

从 gRPC status 转向 Connect error model，通过 Error Detail 传递业务码：

```go
package errs

import "connectrpc.com/connect"

// 业务错误 → Connect Error + Detail
func New(code connect.Code, bizCode int32, msg string) *connect.Error {
    err := connect.NewError(code, errors.New(msg))
    if detail, e := connect.NewErrorDetail(&errpb.BizError{
        Code:    bizCode,
        Message: msg,
    }); e == nil {
        err.AddDetail(detail)
    }
    return err
}

// 错误码注册表（保持 v1 的自定义业务码体系）
func RegisterBizCodes(codes map[int32]connect.Code)
```

**注意**：Connect 协议本身已有标准 error JSON 格式（`{"code":"...", "message":"..."}`），不需要再包一层 `{"code":0,"message":"ok","data":{...}}`。对于 REST 兼容层（Vanguard），可通过自定义 ErrorWriter 转换。

---

## 三、技术选型修正

| 方案原选型 | 建议修正 | 理由 |
|-----------|---------|------|
| `log/slog` 替代 Zap | slog 作为**接口层**，默认用标准 handler，允许注入 Zap handler | slog 高吞吐场景有 ~20% 性能差距；接口统一、实现自由 |
| `envconfig / koanf` | **koanf 为主**，envconfig 仅作为最简场景快捷方式 | koanf 支持多源合并（文件+env+远程），envconfig 无法满足复杂配置 |
| 移除 Wire | 同意。补充：提供 `ginny.Provider` 接口供 fx/do 用户集成 | 框架不依赖 DI，不阻止用户使用 |
| `protovalidate` | 正确。通过 `connectrpc.com/validate` 包集成为**默认 interceptor** | 该包由 ConnectRPC 官方维护，API 稳定 |
| 未提及 `automaxprocs` | 容器环境默认引入 `uber-go/automaxprocs` | 容器中 GOMAXPROCS 默认取宿主机核数，导致 GC 和调度开销 |
| 未提及 `errgroup` | 使用 `golang.org/x/sync/errgroup` 管理并发启停 | 替代 v1 的自定义 graceful 包，标准且可控 |

### slog 接口化设计

```go
package log

import "log/slog"

// 框架使用标准 slog 接口，不强绑具体实现
type Logger = *slog.Logger

func New(opts ...Option) *slog.Logger {
    return slog.New(newHandler(opts...)) // 默认高性能 JSON handler
}

// 用户可注入自定义 handler（如 Zap backend、Loki direct push）
func WithHandler(h slog.Handler) Option {
    return func(o *options) { o.handler = h }
}
```

---

## 四、开发体验补充

### 1. CLI 工具优先级调低

方案 Phase 2 实现 CLI 过于前置。建议：

- **Phase 1-2**：提供 `buf.gen.yaml` 模板 + `Taskfile.yaml` 即可
- **Phase 3+**：如需 CLI，优先考虑作为 **buf plugin** 而非独立二进制
- `ginny dev`（热重载）直接推荐 `air`，不重复造轮子

### 2. 集成测试支持

ConnectRPC 测试极为简便（标准 `httptest.NewServer`），框架应内建 test helper：

```go
package ginnytest

import "testing"

// NewTestServer 一键启动带完整 interceptor 链的测试服务
func NewTestServer(t *testing.T, opts ...Option) *TestServer

func (s *TestServer) Client() *http.Client  // 预配置的 HTTP client
func (s *TestServer) BaseURL() string       // 服务地址
func (s *TestServer) Close()                // 清理
```

### 3. 推荐项目布局

```
myservice/
├── buf.gen.yaml            # buf 代码生成配置
├── buf.yaml                # buf schema 配置
├── proto/
│   └── myservice/v1/
│       └── service.proto
├── gen/                    # buf generate 输出（gitignore 或提交皆可）
│   └── myservice/v1/
│       ├── service.pb.go
│       └── servicev1connect/
├── cmd/
│   └── server/main.go     # 入口，显式 DI
├── internal/
│   ├── service/            # 业务实现（implements Connect handler）
│   ├── repository/         # 数据层
│   └── domain/             # 领域模型
├── config.yaml
├── Taskfile.yaml           # 替代 Makefile
└── Dockerfile
```

---

## 五、实施计划调整

### 原计划问题

1. Phase 0 (2 周) 调研过长——ConnectRPC 已成熟稳定
2. Phase 2 CLI 过早——应优先迁移工具
3. 缺少 Vanguard 集成节点
4. 缺少性能基准测试里程碑

### 优化后计划

```
Phase 0: PoC + 决策 (1 周)
├── ConnectRPC + Vanguard PoC（验证 REST 兼容性）
├── OTel + slog 集成验证
└── 输出: 技术决策记录 (ADR), Benchmark 初始数据

Phase 1: 核心框架 (4 周)
├── W1: module 初始化 (github.com/goriller/ginny/v2)
│       App lifecycle (hooks), Config (koanf), Logger (slog)
├── W2: Server 包 — Connect handler + Vanguard REST 转码 + 双端口
├── W3: Interceptors — OTel, Auth, Recovery, Validation, RateLimit, Logging
├── W4: Error model, Health (HTTP+gRPC), Reflection, Admin server
└── 输出: 最小可运行服务 + 单元测试 >80%

Phase 2: 客户端 + 迁移 (3 周)  ← 迁移工具前置
├── W5: Client 包 (连接池, 重试, 熔断, 服务发现)
├── W6: v1→v2 迁移工具 (proto annotation 转换, import 改写)
├── W7: 2 个内部项目试点迁移 + 问题修复
└── 输出: 迁移指南 + 自动化脚本 + 试点报告

Phase 3: 开发者体验 (2 周)
├── W8: Example 项目, 文档站, ginnytest 包, buf 模板
├── W9: 性能 Benchmark (vs v1 对比报告), CI/CD 模板
└── 输出: 完整文档 + Benchmark + 项目模板

Phase 4: 生态建设 (持续)
├── CLI 工具 / buf plugin（如确有必要）
├── 社区 interceptor 仓库
├── BSR 私有部署
└── 内部培训
```

---

## 六、补充风险项

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| **REST 客户端不兼容** | 高 | 高 | Vanguard 提供透明 REST→RPC 转码 |
| **gRPC 原生性能回退** | 低 | 中 | ConnectRPC 的 gRPC 路径性能与原生 grpc-go 相当；Phase 3 benchmark 验证 |
| **slog 高并发性能不足** | 中 | 低 | 允许 Zap handler 注入；框架层本身不做高频日志 |
| **Vanguard 成熟度** | 低 | 中 | 由 ConnectRPC 核心团队维护，已 v1 稳定版 |
| **Go 1.24 http.Protocols** | 低 | 低 | 通过 build tag 提供 1.22/1.24 双路径支持 |
| **团队对 Connect 协议不熟悉** | 中 | 中 | Phase 0 PoC + 培训；Connect 协议比 gRPC 更简单（curl 可调试） |
| **`ginny-util` 外部依赖移除** | 低 | 低 | v2 内建等价功能（graceful→errgroup, ip→net.IP） |

---

## 七、更新后架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                        ginny.App                                  │
│  Lifecycle Hooks: OnStart/OnStop (priority-ordered)              │
│  Config: koanf (file + env + remote)                             │
│  Logger: slog (pluggable handler)                                │
└──────────────────┬───────────────────────────────────────────────┘
                   │
    ┌──────────────▼──────────────────────────────────────┐
    │            App Server (:8080)                        │
    │                                                     │
    │  ┌─────────────────────────────────────────────┐   │
    │  │  net/http.ServeMux (Go 1.22+ enhanced)      │   │
    │  ├─────────────────────────────────────────────┤   │
    │  │                                             │   │
    │  │  ConnectRPC Handlers ─── Interceptors:      │   │
    │  │   ├─ gRPC    (HTTP/2 h2c)    OTel Tracing  │   │
    │  │   ├─ Connect (HTTP/1.1+2)    OTel Metrics  │   │
    │  │   └─ gRPC-Web(HTTP/1.1)      Auth (JWT)    │   │
    │  │                               Validate     │   │
    │  │  Vanguard Transcoder (可选)    RateLimit   │   │
    │  │   └─ REST ↔ RPC 自动转码      Recovery     │   │
    │  │                               Logging      │   │
    │  │  gRPC Reflection (dev 默认)                 │   │
    │  │  gRPC Health Protocol                      │   │
    │  └─────────────────────────────────────────────┘   │
    └─────────────────────────────────────────────────────┘
    ┌─────────────────────────────────────────────────────┐
    │            Admin Server (:8081)                      │
    │  GET /healthz        Liveness probe                 │
    │  GET /readyz         Readiness probe                │
    │  GET /metrics        Prometheus metrics             │
    │  GET /debug/pprof/*  Runtime profiling (dev only)   │
    └─────────────────────────────────────────────────────┘
```

---

## 八、关键 API 签名建议

```go
// === ginny.go ===
package ginny

type App struct {
    config  *config.Config
    logger  *slog.Logger
    servers []Server
    hooks   []LifecycleHook
}

func New(opts ...Option) (*App, error)
func (a *App) Start(ctx context.Context) error  // blocks until signal
func (a *App) Stop(ctx context.Context) error   // graceful shutdown

type Option func(*options)
func WithConfig(cfg *config.Config) Option
func WithLogger(logger *slog.Logger) Option
func WithServer(opts ...server.Option) Option
func WithHook(hook LifecycleHook) Option


// === server/server.go ===
package server

type Server struct {
    appMux      *http.ServeMux
    adminMux    *http.ServeMux
    interceptors []connect.Interceptor
    transcoder  *vanguard.Transcoder   // optional
    reflector   *grpcreflect.Reflector // optional
}

type Option func(*options)
func WithAddr(addr string) Option
func WithAdminAddr(addr string) Option
func WithInterceptor(i connect.Interceptor) Option
func WithRESTTranscoding(rules ...*annotations.HttpRule) Option
func WithReflection(enabled bool) Option
func WithHealthChecker(checker HealthChecker) Option
func WithH2C(enabled bool) Option  // Go <1.24 fallback

// 服务注册（ConnectRPC handler 就是 http.Handler）
func (s *Server) Register(path string, handler http.Handler)

// HealthChecker 接口
type HealthChecker interface {
    Check(ctx context.Context, service string) (Status, error)
    Watch(ctx context.Context, service string) (<-chan Status, error)
}


// === interceptor/ ===
// 每个 interceptor 同时实现 connect.Interceptor
// 避免 v1 中 middleware + interceptor 两套代码的问题

package auth
func NewInterceptor(validator TokenValidator, opts ...Option) connect.Interceptor

package ratelimit
func NewInterceptor(limiter Limiter, opts ...Option) connect.Interceptor

// Limiter 接口支持分布式（Redis）和本地两种实现
type Limiter interface {
    Allow(ctx context.Context, key string) (bool, error)
}


// === client/ ===
package client

// 不重新封装 typed client，而是提供连接管理
type Pool struct { /* ... */ }
func NewPool(opts ...Option) *Pool
func (p *Pool) HTTPClient(service string) *http.Client // 带服务发现的 HTTP client

type Option func(*options)
func WithServiceDiscovery(sd ServiceDiscovery) Option
func WithRetry(policy RetryPolicy) Option
func WithCircuitBreaker(cb CircuitBreaker) Option
func WithInterceptor(i connect.Interceptor) Option


// === errs/ ===
package errs

func New(code connect.Code, bizCode int32, msg string) *connect.Error
func Newf(code connect.Code, bizCode int32, format string, args ...any) *connect.Error
func BizCode(err error) int32  // 从 connect.Error 提取业务码
func RegisterBizCodes(codes map[int32]connect.Code)  // 业务码注册
```

---

## 九、Module 路径

遵循 Go module 版本化规范：

```
github.com/goriller/ginny/v2
```

v1 保持在 `github.com/goriller/ginny`，两者可共存于同一 `go.mod`（迁移期间）。

---

## 十、决策总结

| # | 决策项 | 建议 | 优先级 |
|---|--------|------|--------|
| 1 | Vanguard REST 兼容 | 必须集成 | P0 |
| 2 | 双端口 (app + admin) | 默认分离 | P0 |
| 3 | gRPC Reflection | 集成，配置控制 | P1 |
| 4 | Health 双协议 | HTTP + gRPC Health | P1 |
| 5 | slog 接口化 | 不锁实现 | P1 |
| 6 | Lifecycle Hooks | 有序启停 | P1 |
| 7 | Interceptor 三层分离 | 全局/RPC/服务级 | P1 |
| 8 | Error model → Connect | Detail 传业务码 | P1 |
| 9 | Go 1.22+ / 推荐 1.24 | http.Protocols | P2 |
| 10 | Client → Pool + 熔断 | 不封装 typed client | P2 |
| 11 | ginnytest 包 | 测试工具 | P2 |
| 12 | CLI 延后 | Phase 3+ | P3 |
