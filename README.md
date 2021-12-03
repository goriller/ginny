# [Ginny](https://github.com/gorillazer/ginny)

Ginny framework base on Gin + GRPC, more components to improve development efficiency.

## Installation

```shell
cd $GOPATH && go get github.com/gorillazer/ginny-cli/ginny
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
ginny new hellodemo --grpc
```

### 2.Create a handler
```shell
ginny handle user 
```

### 3.Create a Service

```shell
ginny service user 
```

### 3.Create a repository（Optional）


```shell
// support mysql、mongo、redis 
ginny repo user -d mysql

```
### 4. Make 

```shell
// .pb.go and service
make proto

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
                "mysql_host": "127.0.0.1:3306",
                "mysql_user": "root",
                "mysql_pass": "",
                "redis_host": "127.0.0.1:6379",
                "redis_pass": "",
                "jaeger_agent": "127.0.0.1:6831",
            },
            "args": [
                "-f","../configs/dev.yml"
                // "--remote", "etcd",
            ]
        }
    ]
}
```
Select `Launch GoPackage` to debug run. Try to call `http://localhost:9090/` or `grpc://127.0.0.1:9000/` .

## Example

Check out the [quick start example][quick-example].

[quick-example]: https://github.com/gorillazer/ginny-demo
