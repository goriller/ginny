# Ginny 框架现代化改造方案

> 综合评估报告与分阶段实施计划
> 评估日期：2026-05-08
> 当前版本：Ginny v1 (Go 1.19, gRPC + grpc-gateway v2.16.2)
> 目标版本：Ginny v2 (Go 1.23+, ConnectRPC + gRPC 双协议)

---

## 目录

1. [Ginny 现状分析](#1-ginny-现状分析)
2. [同类框架对比](#2-同类框架对比)
3. [改造方案设计](#3-改造方案设计)
4. [分阶段实施计划](#4-分阶段实施计划)
5. [风险与缓解措施](#5-风险与缓解措施)

---

## 1. Ginny 现状分析

### 1.1 架构概览

```
┌──────────────────────────────────────────────────┐
│                  Application                     │
│  (Name, Version, Option, Logger, Server)         │
│  Wire DI → NewApp(Option, Logger, regFunc)       │
└──────────────┬───────────────────────────────────┘
               │
    ┌──────────▼──────────┐
    │       Server         │
    │  ┌────────────────┐  │
    │  │  gRPC Server   │  │   :9000 (gRPC)
    │  │  + interceptors│  │
    │  └────────────────┘  │
    │  ┌────────────────┐  │
    │  │  MuxServe      │  │   :8080 (HTTP/REST via grpc-gateway)
    │  │  + middlewares │  │
    │  └────────────────┘  │
    │  ┌────────────────┐  │
    │  │  Metrics       │  │   :8081 (Prometheus)
    │  └────────────────┘  │
    │  ┌────────────────┐  │
    │  │  Health Server │  │  gRPC Health Check
    │  └────────────────┘  │
    └──────────────────────┘
```

### 1.2 模块分析

| 模块 | 路径 | 职责 | 评价 |
|------|------|------|------|
| **app.go** | `app.go` | 应用生命周期管理，Wire DI 集成 | 结构清晰，但与 Server 耦合较紧 |
| **server/** | `server/server.go` | gRPC + HTTP 双协议 Server | 核心引擎，通过 grpc-gateway 自动转换 REST |
| **server/mux/** | `server/mux/` | HTTP Mux 封装 (grpc-gateway runtime) | 自定义错误处理、Body 改写、响应格式统一 |
| **server/health/** | `server/health/` | gRPC 健康检查 | 简化版，覆盖 AuthFuncOverride |
| **middleware/** | `middleware/` | HTTP 中间件 (Recover, Auth, Limit, Tracer) | 功能齐全但依赖 OpenTracing(已归档) |
| **interceptor/** | `interceptor/` | gRPC 拦截器 (Auth, Limit, Tracer, Tags, Logging) | 完备的拦截器链，但依赖 OpenTracing |
| **client/** | `client/` | gRPC + HTTP 客户端封装 | gRPC 客户端支持重试/负载均衡/超时；HTTP 客户端功能完善 |
| **errs/** | `errs/` | 错误处理 (gRPC status 封装 + Try/Catch) | 实用，支持自定义业务错误码 |
| **config/** | `config/config.go` | Viper 配置管理 | 支持远程配置(etcd/consul)、环境变量展开、热重载 |
| **logger/** | `logger/logger.go` | Zap 日志封装 | 支持文件/控制台输出、Context 传递、Tags 集成 |

### 1.3 优点

1. **gRPC Schema 驱动**：核心设计理念正确，proto 文件即 API 契约
2. **HTTP/RPC 双协议**：通过 grpc-gateway 自动从 proto 生成 REST API
3. **完备的中间件/拦截器体系**：日志、链路追踪、限流、认证、恢复、校验覆盖完整
4. **统一错误处理**：gRPC status code 到 HTTP status code 的映射，自定义业务错误码
5. **服务发现集成**：内建服务注册/注销机制
6. **配置管理成熟**：Viper 支持本地 + 远程配置 (etcd/consul)，环境变量展开
7. **响应格式统一**：`{"code": ..., "message": "...", "data": ...}` 标准化输出
8. **Wire DI**：编译时依赖注入

### 1.4 缺点与风险

| 问题 | 严重程度 | 说明 |
|------|----------|------|
| **Go 1.19 过旧** | 高 | 缺少泛型、新标准库特性，安全补丁已停止 |
| **OpenTracing 已归档** | 高 | 项目于 2022 年归档，应迁移到 OpenTelemetry |
| **go-grpc-middleware v2-rc** | 中 | 使用 rc 版本，API 可能不稳定 |
| **grpc-gateway v2 自身的局限** | 中 | HTTP/1.1 性能瓶颈，不支持 HTTP/2 服务端推送之外的 streaming |
| **无原生 Connect 协议支持** | 中 | 不支持浏览器直接调用 gRPC (Connect/gRPC-Web) |
| **无 OpenAPI 自动生成** | 低 | 缺少从 proto 自动生成 OpenAPI/Swagger 文档 |
| **无内建验证框架集成** | 低 | protoc-gen-validate 使用但未深度集成 |
| **HTTP Server 无优雅关闭超时控制** | 低 | Shutdown 缺超时保障 |
| **缺乏 CLI 工具的 schema registry 集成** | 低 | 无 BSR (Buf Schema Registry) 支持 |
| **中间件/interceptor 有大量代码重复** | 低 | HTTP middleware 和 gRPC interceptor 逻辑重复（auth/limit/tracer 各写两遍） |

---

## 2. 同类框架对比

### 2.1 评估矩阵

| 框架 | 协议支持 | Go 版本 | Schema 驱动 | 性能 | 开发效率 | 生态成熟度 | GitHub Stars |
|------|----------|---------|-------------|------|----------|------------|-------------|
| **Ginny v1** | gRPC + REST(gw) | 1.19 | Proto(gRPC) | ★★★ | ★★★ | ★★ | 内部 |
| **ConnectRPC** | gRPC + Connect + gRPC-Web | 1.21+ | Proto(gRPC) | ★★★★★ | ★★★★★ | ★★★★ | 7.5k+ |
| **buf connect-go** | Connect + gRPC | 1.21+ | Proto(gRPC) | ★★★★★ | ★★★★★ | ★★★★ | 核心库 |
| **grpc-gateway v2** | gRPC + REST | 1.19+ | Proto(gRPC) | ★★★ | ★★★★ | ★★★★★ | 18.5k+ |
| **Twirp** | Twirp + REST(JSON) | 1.18+ | Proto(Twirp) | ★★★★ | ★★★ | ★★★ | 7.5k+ |
| **Go kit** | 任意 | 1.18+ | 无 | ★★★ | ★★★ | ★★★★ | 26.5k+ |
| **Hertz** | HTTP(REST) | 1.16+ | 无(Thrift可选) | ★★★★★ | ★★★★ | ★★★★★ | 6k+ |
| **Kratos** | gRPC + REST(gw) | 1.19+ | Proto(gRPC) | ★★★★ | ★★★★ | ★★★★ | 23.5k+ |
| **Huma** | REST + GraphQL | 1.21+ | OpenAPI | ★★★★ | ★★★★★ | ★★★ | 3.5k+ |

### 2.2 各框架详评

#### ConnectRPC (buf.build/connectrpc)

- **定位**：gRPC 兼容的下一代 RPC 框架，浏览器友好
- **协议**：gRPC、Connect、gRPC-Web 三协议
- **核心优势**：
  - 无代理即可从浏览器调用 gRPC 服务
  - 使用 `net/http` 标准库，零额外依赖
  - 比 grpc-gateway 快 2-3x（无 protobuf→JSON 序列化开销）
  - Buf Schema Registry (BSR) 集成
  - 支持 streaming (client/server/bidi)
  - 优秀的 Go 1.21+ 泛型 API
- **缺点**：生态较 grpc-gateway 年轻，但增速极快
- **2026 状态**：已成为 gRPC 生态事实标准之一，buf 公司全力推动

#### buf connect-go

- **定位**：Connect 协议的 Go 参考实现
- **协议**：Connect（类 gRPC 但走 HTTP/1.1 或 HTTP/2）
- **核心优势**：
  - 极简 API，符合 Go 标准库风格
  - 与 ConnectRPC 生态无缝衔接
  - 支持 Unary + Streaming
  - 内建 Interceptor 机制
- **缺点**：不直接暴露原生 gRPC（需通过 ConnectRPC 包装）

#### grpc-gateway v2

- **定位**：gRPC 到 REST/JSON 的网关
- **协议**：gRPC → REST (JSON)
- **核心优势**：
  - 最成熟的 gRPC→REST 方案
  - 丰富的 proto annotation (google.api.http)
  - 支持 streaming 转 WebSocket
  - 大社区，文档丰富
- **缺点**：
  - 性能受 JSON 序列化限制
  - 不支持浏览器直接调用
  - 需要额外生成代码
  - 对 proto 文件的 annotation 侵入性强

#### Twirp

- **定位**：轻量级 RPC 框架（由 Twitch 创建）
- **协议**：Twirp (自有协议，支持 Protobuf + JSON)
- **核心优势**：
  - 极简单，代码量少
  - 支持 JSON 和 Protobuf 双格式
  - 路由清晰 (`/twirp/pkg.Service/Method`)
  - 对 proto 文件无侵入
- **缺点**：
  - 不支持 streaming
  - 生态较弱（非 gRPC 兼容）
  - 社区活跃度下降（已被 ConnectRPC 取代趋势）

#### Go kit

- **定位**：微服务工具包（不是框架）
- **协议**：传输无关
- **核心优势**：
  - 架构模式优秀（endpoint → transport → service）
  - 丰富的中间件生态
  - 支持多种传输协议
- **缺点**：
  - 样板代码极多
  - 学习曲线陡峭
  - 不是 Schema 驱动（需手写接口定义）

#### Hertz

- **定位**：字节跳动高性能 HTTP 框架
- **协议**：HTTP/1.1, HTTP/2, HTTP/3
- **核心优势**：
  - 极高 HTTP 性能（fasthttp 级别）
  - 内建 HTTP/3 支持
  - 丰富的中间件生态
  - 支持 Thrift 协议
- **缺点**：
  - 不是 RPC 框架
  - 无 Schema 驱动（需配合其他方案）
  - 与 gRPC 生态无关

#### Kratos

- **定位**：B站开源微服务框架
- **协议**：gRPC + REST(grpc-gateway)
- **核心优势**：
  - 完整的微服务解决方案（配置、日志、注册、追踪、限流）
  - Proto 驱动开发
  - Layout 规范
  - 活跃的中国社区
- **缺点**：
  - 框架侵入性强
  - 使用 HTTP 框架为 Gin（非标准库），非 Connect 协议
  - 重量级

#### Huma

- **定位**：OpenAPI 驱动的 Go REST/GraphQL 框架
- **协议**：REST, GraphQL
- **核心优势**：
  - OpenAPI 3.x 自动生成
  - 内建验证、文档 UI
  - 基于标准库 `net/http`
  - 开发体验极佳（代码即文档）
- **缺点**：
  - 不支持 gRPC 协议
  - 不支持 streaming（REST 方式）
  - 不适合高性能 RPC 场景

### 2.3 三维度评分总结

```
技术先进性:
  ConnectRPC ★★★★★ (2024-2026 最先进 RPC 方案)
  Hertz      ★★★★★ (HTTP 性能天花板)
  Kratos     ★★★★  (微服务全家桶)
  Ginny v1   ★★    (基于 2023 年技术栈)
  gRPC-gw v2 ★★★   (成熟但非前沿)

性能:
  Hertz      ★★★★★ (fasthttp 级别)
  ConnectRPC ★★★★★ (无 JSON 序列化开销)
  Twirp      ★★★★  (轻量)
  Ginny v1   ★★★   (JSON 序列化瓶颈)
  gRPC-gw v2 ★★★   (JSON 序列化 + 大依赖)

开发效率:
  Huma       ★★★★★ (OpenAPI 自动生成)
  ConnectRPC ★★★★★ (CLI 友好, BSR 集成)
  Ginny v1   ★★★   (手动 boilerplate 较多)
  Go kit     ★★    (大量样板代码)
```

---

## 3. 改造方案设计

### 3.1 推荐技术栈

```
┌─────────────────────────────────────────────────┐
│               Ginny v2 技术栈                     │
├─────────────────────────────────────────────────┤
│ Go 版本:        Go 1.23+ (泛型, 新标准库)        │
│ Schema 管理:    Buf CLI v2 + BSR                │
│ RPC 框架:       ConnectRPC (connectrpc.com)      │
│ 协议支持:       gRPC + Connect + gRPC-Web        │
│ HTTP 路由:      net/http (标准库)                │
│ 序列化:         Protobuf (默认) + JSON (可选)     │
│ 代码生成:       buf generate (protoc 替代)        │
│ 链路追踪:       OpenTelemetry (替代 OpenTracing)  │
│ 日志:           log/slog (标准库, 零依赖)         │
│ 配置管理:       envconfig / koanf (轻量化)        │
│ 指标:           Prometheus + OpenTelemetry        │
| 依赖注入:       显式构造函数（移除 Wire）               |
│ 验证:           protovalidate (buf 新验证方案)     │
│ CLI 工具:       ginny-v2 命令 + buf plugin 集成   │
│ 文档生成:       buf generate → OpenAPI + Docs     │
└─────────────────────────────────────────────────┘
```

### 3.2 选型理由

| 选择 | 理由 |
|------|------|
| **ConnectRPC 替代 gRPC-Gateway** | 1) 浏览器原生支持 2) 性能提升 2-3x 3) 标准库 net/http 4) BSR 生态 |
| **OpenTelemetry 替代 OpenTracing** | OpenTracing 已归档，OTel 是 CNCF 标准 |
| **log/slog 替代 Zap** | Go 1.21+ 标准库，零依赖，性能相当 |
| **Buf CLI 替代 protoc** | 无需手动管理 protoc 插件，声明式配置 |
| **protovalidate 替代 protoc-gen-validate** | buf 生态原生的新版验证方案，cel 表达式 |
| **envconfig/koanf 减少 Viper 依赖** | 轻量化，减少间接依赖 50+ |
| **移除 Wire，改为显式构造 DI** | 框架仅 4 个构造函数链（Config→Logger→Server→App），无需 DI 框架。用户可自行选用 Wire/fx/do |

### 3.3 依赖注入决策：移除 Wire

#### 当前 Wire 使用量

Ginny v1 中 Wire 仅用于 2 个文件、2 个 provider set：

```go
// app.go
AppProviderSet = wire.NewSet(logger.Default, config.ConfigProviderSet, NewOption, NewApp)

// config/config.go
ConfigProviderSet = wire.NewSet(NewConfig, NewViper)
```

总共仅 ~4 个构造函数的注入链，Wire 对于这个规模是过度设计。

#### 为什么移除

| 维度 | Wire | 显式构造 |
|------|------|----------|
| 代码量 | 需要定义 provider set + wire.go + 生成代码 | 直接写构造函数链 |
| 构建步骤 | `wire` → `go build` 两步 | `go build` 一步 |
| 可调试性 | 生成的代码需对照源文件 | 代码即文档，直接跳转 |
| 可理解性 | 新人需学习 Wire 概念 | 标准 Go，无学习成本 |
| 维护风险 | Google 半维护状态，更新缓慢 | 零依赖，永不过期 |

#### v2 推荐的 DI 方式

```go
// cmd/main.go — 清晰的直线依赖
func main() {
    cfg := config.MustLoad(config.WithEnv("dev"))
    log := log.New(cfg.LogLevel)
    srv := server.New(cfg,
        server.WithLogger(log),
        server.WithInterceptor(tracing.New()),
        server.WithInterceptor(logging.New()),
    )
    app := ginny.New(ginny.WithConfig(cfg), ginny.WithServer(srv))
    app.Start(context.Background())
}
```

4 行构造，无需 `wire gen`，无需理解 provider set。

> 用户自己的应用层想用 Wire / fx / do / samber/do 完全自由——Ginny 框架本身不强制 DI 工具。

```
┌──────────────────────────────────────────────────────────┐
│                     Application                          │
│  (Name, Version, Config, Logger, Servers)               │
│  启动入口: ginny.New(opts...) → ginny.App               │
└──────────────┬───────────────────────────────────────────┘
               │
    ┌──────────▼──────────────────────────┐
    │           Gateway Server             │
    │  ┌─────────────────────────────────┐ │
    │  │   ConnectRPC Mux                │ │ :8080 (统一端口)
    │  │   ┌───────────────────────────┐ │ │
    │  │   │ gRPC Handler     (h2c)    │ │ │ ↔ gRPC 客户端
    │  │   │ Connect Handler  (h1/h2)  │ │ │ ↔ Web/浏览器
    │  │   │ gRPC-Web Handler (h1)     │ │ │ ↔ gRPC-Web
    │  │   └───────────────────────────┘ │ │
    │  └─────────────────────────────────┘ │
    │  ┌─────────────────────────────────┐ │
    │  │   HTTP Mux (标准库)              │ │
    │  │   /healthz, /metrics, /debug    │ │
    │  └─────────────────────────────────┘ │
    │  ┌─────────────────────────────────┐ │
    │  │   Interceptors (统一)            │ │
    │  │   OTel Tracing + Logging        │ │
    │  │   Rate Limiting + Auth          │ │
    │  │   Recovery + Validation         │ │
    │  └─────────────────────────────────┘ │
    └──────────────────────────────────────┘
```

### 3.4 模块设计

```
ginny/
├── ginny.go              # 框架入口：New(), App 结构体
├── option.go             # 统一配置 Option 模式
├── app.go                # Application 生命周期
│
├── server/
│   ├── server.go         # Gateway Server (Connect + HTTP)
│   ├── option.go         # Server Option
│   └── handler.go        # 统一 Handler 注册
│
├── interceptor/          # ConnectRPC Interceptors (替代旧 interceptor 和 middleware)
│   ├── logging/          # slog 日志拦截器
│   ├── tracing/          # OTel 链路追踪
│   ├── auth/             # 认证拦截器（JWT/OAuth2）
│   ├── ratelimit/        # 限流拦截器
│   ├── recovery/         # Panic 恢复
│   └── validation/       # protovalidate 验证
│
├── client/
│   ├── client.go         # ConnectRPC 客户端（泛型 API）
│   └── option.go         # 客户端配置
│
├── config/
│   ├── config.go         # 配置加载（envconfig + koanf）
│   └── option.go
│
├── log/
│   └── log.go            # slog 封装（Context 传递 + 级别控制）
│
├── errs/
│   ├── errs.go           # Connect 错误处理
│   └── codes.go          # 自定义错误码
│
├── cmd/
│   └── ginny/
│       └── main.go       # CLI 脚手架工具
│
└── buf/
    ├── buf.gen.yaml       # buf 代码生成配置
    └── buf.yaml           # buf schema 配置
```

### 3.5 核心 API 设计

```go
// ginny.go - 框架入口
package ginny

// App 应用实例
type App struct {
    name    string
    server  *server.Server
    logger  *slog.Logger
    config  *config.Config
}

// New 创建应用 (使用泛型 Option 模式)
func New(opts ...Option) (*App, error)

// Start 启动
func (a *App) Start(ctx context.Context) error

// Stop 优雅关闭
func (a *App) Stop(ctx context.Context) error

// Option 函数式配置
type Option func(*options)

// 便捷配置函数
func WithName(name string) Option
func WithConfig(cfg *config.Config) Option
func WithLogger(logger *slog.Logger) Option
func WithServer(opts ...server.Option) Option
```

```go
// server/server.go - 统一网关
package server

type Server struct {
    mux          *http.ServeMux
    interceptors []connect.Interceptor
    services     []ServiceRegistrar
}

// ServiceRegistrar 服务注册接口
type ServiceRegistrar interface {
    Register(s *Server) error
}

// RegisterService 注册 Connect/gRPC 服务
func (s *Server) RegisterService(
    path string,
    handler http.Handler,
) error

// RegisterHTTP 注册纯 HTTP handler
func (s *Server) RegisterHTTP(
    pattern string,
    handler http.Handler,
)
```

```go
// interceptor/auth/auth.go - 统一拦截器
package auth

// NewInterceptor 创建认证拦截器 (Connect 协议)
func NewInterceptor(validator TokenValidator, opts ...Option) connect.UnaryInterceptorFunc

// 同时兼容 HTTP middleware
func (a *AuthInterceptor) HTTPMiddleware(next http.Handler) http.Handler
```

### 3.6 迁移路径

#### 从 Ginny v1 迁移到 v2

```
阶段1: Schema 迁移
  旧: protoc + grpc-gateway annotations
  新: buf.yaml + buf.gen.yaml
  影响: proto 文件可保持兼容，需移除 grpc-gateway 专用 annotation
  迁移工具: buf migration guide

阶段2: 代码生成迁移
  旧: protoc-gen-go, protoc-gen-go-grpc, protoc-gen-grpc-gateway
  新: protoc-gen-go, connectrpc.com/connect/cmd/protoc-gen-connect-go
  影响: 生成代码路径变化，接口略有不同
  迁移工具: buf generate

阶段3: 服务端代码迁移
  旧: server.RegisterService(desc, impl)
  新: server.RegisterService(path, connectHandler)
  差异: ConnectRPC handler 是 http.Handler，更灵活

阶段4: 中间件/拦截器迁移
  旧: grpc.UnaryServerInterceptor + middleware.MuxMiddleware (两套)
  新: connect.UnaryInterceptorFunc (统一)
  优势: 代码量减少 50%+

阶段5: 客户端迁移
  旧: client.NewClient(uri, pb.NewXxxClient)
  新: xxxconnect.NewXxxClient(httpClient, baseURL)
  优势: 标准 http.Client，复用连接池

阶段6: 移除旧依赖
  删除: grpc-gateway, go-grpc-middleware, opentracing, viper, zap
  新增: connectrpc.com/connect, otel, slog, koanf
```

### 3.7 CLI 工具链设计

```
ginny v2 CLI

Commands:
  ginny init [module]          创建新 Ginny 项目
  ginny generate               生成代码 (buf generate 封装)
  ginny add service [name]     添加新服务
  ginny add client [name]      添加客户端调用
  ginny dev                    本地开发模式 (热重载)
  ginny build                  构建
  ginny lint                   代码检查 (buf lint + go vet)
  ginny breaking               检查 API 兼容性 (buf breaking)

Flags:
  --schema-registry string    BSR 地址
  --config string             配置文件路径
  --env string                环境 (dev/staging/prod)
```

---

## 4. 分阶段实施计划

### Phase 0: 准备与调研 (2 周)

| 任务 | 负责人 | 输出 |
|------|--------|------|
| ConnectRPC 深度调研与 PoC | 架构组 | PoC Demo |
| 现有 Ginny 项目摸底 | 各业务线 | 使用情况报告 |
| OpenTelemetry Go SDK 调研 | 架构组 | 技术选型文档 |
| Buf CLI v2 迁移指南编写 | 架构组 | Migration Guide |
| 社区/内部意见收集 | PM | 需求文档 |

### Phase 1: 核心框架搭建 (4 周)

| 周次 | 任务 | 里程碑 |
|------|------|--------|
| W1 | 初始化 ginny-v2 仓库，Go 1.23 module | repo ready |
| W2 | 实现 `server/` 包：ConnectRPC + HTTP Mux | server 可启动 |
| W3 | 实现 `interceptor/` 层：logging(tracing/auth/recovery) | 基础拦截器可用 |
| W4 | 实现 `config/`, `log/`, `errs/` | 核心模块完成 |

**交付物**：
- `ginny.New()` 可创建最小可运行服务
- 支持 gRPC + Connect 双协议
- slog 日志 + OTel Tracing
- 单元测试覆盖率 >80%

### Phase 2: 客户端与开发者体验 (3 周)

| 周次 | 任务 | 里程碑 |
|------|------|--------|
| W5 | 实现 `client/` 包（ConnectRPC 客户端 + 泛型 API） | 客户端可用 |
| W6 | CLI 工具 `ginny` v2：init, generate, dev | CLI 可用 |
| W7 | `ginny add service/client` 脚手架命令 | 完整开发流 |

**交付物**：
- 泛型客户端 API (`client.New[XxxClient]()`)
- CLI 一键生成项目
- Example 项目 + 文档

### Phase 3: 迁移工具与兼容 (2 周)

| 周次 | 任务 | 里程碑 |
|------|------|--------|
| W8 | v1 → v2 迁移工具开发 | 迁移脚本 |
| W9 | v1 兼容层（可选）| 混合运行 |

**交付物**：
- 迁移指南文档
- 自动化迁移脚本（proto 转换 + 代码改写）
- 至少 2 个内部项目成功迁移

### Phase 4: 生态建设与推广 (持续)

| 任务 | 说明 |
|------|------|
| 内部文档站 | 使用文档 + API 参考 + Example |
| 插件市场 | 社区贡献 interceptor/client |
| BSR 集成 | 私有 BSR 部署 + 模板仓库 |
| 性能测试 | 与 v1 对比 benchmark 报告 |
| CI/CD 模板 | GitHub Actions / GitLab CI 模板 |



---

## 5. 风险与缓解措施

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| ConnectRPC 向后兼容性变更 | 低 | 中 | 锁定主版本，紧跟社区 |
| 内部项目迁移阻力 | 高 | 高 | 提供兼容层 + 自动迁移工具 + 2 周支持期 |
| 学习成本 | 中 | 中 | 编写详细迁移指南 + 内部培训 |
| OpenTelemetry 性能开销 | 低 | 低 | 采样策略 + 异步导出 |
| protoc 插件生态不兼容 | 低 | 中 | Buf 已支持主流 protoc 插件 |
| 关键依赖供应链风险 | 低 | 中 | 定期审查依赖，使用 Go 模块代理 |

---

## 附录

### A. 依赖对比

| v1 依赖 | v2 替代 | 理由 |
|---------|---------|------|
| google.golang.org/grpc | connectrpc.com/connect | 轻量 + 浏览器支持 |
| grpc-ecosystem/grpc-gateway/v2 | (移除) | ConnectRPC 原生支持 HTTP |
| grpc-ecosystem/go-grpc-middleware/v2 | (移除) | ConnectRPC 自带拦截器 |
| go.uber.org/zap | log/slog | 标准库，零依赖 |
| github.com/spf13/viper | github.com/knadh/koanf | 轻量 ~50% |
| github.com/opentracing/opentracing-go | go.opentelemetry.io/otel | OTel 为 CNCF 标准 |
| github.com/goriller/ginny-util/* | 内建入 ginny | 减少外部依赖 |
| github.com/pkg/errors | 标准库 errors | Go 1.13+ errors |
| github.com/google/wire | (移除) | 框架仅 4 个构造函数，显式 DI 即可 |

### B. 性能预估

基于 ConnectRPC 官方 benchmark 和社区数据：

```
场景: Unary RPC (小消息 ~1KB)

Ginny v1 (gRPC-gateway REST):  ~15,000 req/s  (JSON 序列化瓶颈)
Ginny v1 (gRPC 原生):         ~40,000 req/s
Ginny v2 (Connect 协议):       ~55,000 req/s  (无 JSON 开销)
Ginny v2 (gRPC 协议):         ~45,000 req/s  (兼容模式下)

场景: 延迟 (P99)

Ginny v1 (HTTP REST):  ~8ms
Ginny v2 (Connect):    ~2ms
```

### C. 参考资料

- [ConnectRPC 官方文档](https://connectrpc.com/docs/go/getting-started)
- [Buf CLI 文档](https://buf.build/docs/cli)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Go 1.23 Release Notes](https://go.dev/doc/go1.23)
- [protovalidate](https://github.com/bufbuild/protovalidate)
