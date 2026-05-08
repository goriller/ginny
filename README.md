# Ginny

**Schema-driven Go RPC framework built on ConnectRPC.**  
**基于 ConnectRPC 的 Schema 驱动 Go RPC 框架。**

Protocols: **gRPC + Connect + gRPC-Web** | 协议支持：**gRPC + Connect + gRPC-Web** 三协议

---

## Features | 功能特性

| Feature | Description |
|---------|-------------|
| **ConnectRPC** | gRPC + Connect + gRPC-Web, browser-native, no proxy |
| **Vanguard REST** | Legacy REST client compatibility, no code changes |
| **Dual Ports** | App(:8080) for RPC, Admin(:8081) for health/metrics/pprof |
| **Lifecycle Hooks** | Ordered start/stop with priority |
| **slog Logging** | Pluggable handler, inject Zap or custom backend |
| **Connect Errors** | Error details with business codes (`errs.New(code, bizCode, msg)`) |
| **OpenTelemetry** | Built-in tracing interceptor |
| **Three-layer Interceptors** | HTTP middleware → Connect → Per-service |
| **gRPC Health + Reflection** | Standard protocols, grpcurl/grpcui ready |
| **koanf Config** | Multi-source: files + env vars + remote |
| **Explicit DI** | No Wire. `Config → Logger → Server → App` |
| **ginnytest** | One-line test server with full interceptor chain |
| **Go 1.24** | Native `http.Protocols` h2c, enhanced ServeMux |

## Installation | 安装

Requires Go 1.24+. | 需要 Go 1.24+。

```
go get github.com/goriller/ginny/v2
```

Code generation:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

Optional: [Buf CLI](https://buf.build/docs/cli) replaces protoc. Templates in `buf/`.

## Quick Start | 快速开始

```go
package main

import (
    "context"
    "github.com/goriller/ginny/v2"
    "github.com/goriller/ginny/v2/config"
    "github.com/goriller/ginny/v2/server"
)

func main() {
    cfg := config.MustLoad()
    srv := server.New()
    app, _ := ginny.New(
        ginny.WithConfig(cfg),
        ginny.WithServer(srv),
    )
    app.Start(context.Background())
}
```

```bash
go run main.go
curl http://localhost:8081/healthz   # → {"status":"ok"}
```

## Documentation | 文档

- [Architecture & Design](docs/SOLUTION.md) — 架构设计与技术细节
- [Redesign Plan](docs/redesign-plan.md) — v1 → v2 改造方案
- [Review & Optimization](docs/redesign-plan-review.md) — 架构师评审与优化

## License

MIT
>>>>>>> feat/new
