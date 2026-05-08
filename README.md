<<<<<<< HEAD
# [Ginny](https://github.com/goriller/ginny)

Ginny base on gRPC + gRPC-Gateway, more components to improve development efficiency.

## Installation

```shell
cd $GOPATH && go get github.com/goriller/ginny-cli/ginny
```

### Dependencies tools

* protoc

https://github.com/protocolbuffers/protobuf/releases

* protoc-gen-go：

```shell
cd $GOPATH && go install github.com/golang/protobuf/protoc-gen-go@latest
```

* go wire:

```shell
cd $GOPATH && go get github.com/google/wire/cmd/wire
```

* protoc-gen-validate:

```shell
cd $GOPATH && go install github.com/envoyproxy/protoc-gen-validate@latest
```

* goimports：

```shell
cd $GOPATH && go get golang.org/x/tools/cmd/goimports
```

* mockgen：

```shell
cd $GOPATH && go install github.com/golang/mock/mockgen@v1.6.0
```
* make:

Mac OS and Linux systems already have the make command，

windows: [How to run "make" command in gitbash in windows?](https://gist.github.com/evanwill/0207876c3243bbb6863e65ec5dc3f058)


## Quick Start

### 1.Create Project

```shell
ginny new hellodemo
```

### 2.modify .proto and generate pb code

```shell
make protoc
```

### 4. Make 

```shell

// wire
make wire
```

## How to debug

if you use vscode , edit the `.vscode/launch.json` , like this: 
```
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch GoPackage",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd",
            "env": {
            // ...
            },
            "args": [
                "-conf","../configs/dev.yml"
                // "-remote", "etcd://127.0.0.1:1233/test",
            ]
        }
    ]
}
```
Select `Launch GoPackage` to debug run. Try to call `http://localhost:8080/` or `grpc://127.0.0.1:9000/` .

## Example

Check out the [quick start example][quick-example].

[quick-example]: https://github.com/goriller/ginny-demo
=======
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
