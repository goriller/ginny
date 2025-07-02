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
