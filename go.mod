module github.com/gorillazer/ginny

go 1.16

require (
	github.com/fsnotify/fsnotify v1.5.4
	github.com/google/uuid v1.1.2
	github.com/google/wire v0.5.0
	github.com/gorillazer/ginny-serve v0.1.1
	github.com/gorillazer/ginny-util v0.0.6
	github.com/gorillazer/ginny-util/graceful v0.0.0-20220701090559-95adaf70d7de
	github.com/grpc-ecosystem/go-grpc-middleware/providers/zap/v2 v2.0.0-20210710102418-709d4153d7aa
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.0-rc.2.0.20210807094637-274df5968e19
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.1-0.20200507082539-9abf3eb82b4a
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.3
	github.com/hashicorp/consul/api v1.13.0
	github.com/mbobakov/grpc-consul-resolver v1.4.4
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.12.0
	go.uber.org/zap v1.21.0
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd
	google.golang.org/grpc v1.46.2
	google.golang.org/protobuf v1.28.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
