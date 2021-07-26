module github.com/gorillazer/ginny

go 1.15

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.9.1

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0 // indirect
	github.com/armon/go-metrics v0.3.5-0.20201104215618-6fd5a4ddf425 // indirect
	github.com/didi/gendry v1.7.0
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-contrib/zap v0.0.1
	github.com/gin-gonic/gin v1.7.2
	github.com/go-redis/redis/v8 v8.11.0
	github.com/google/wire v0.5.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/consul/api v1.8.1
	github.com/hashicorp/go-hclog v0.14.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/mbobakov/grpc-consul-resolver v1.4.4
	github.com/miekg/dns v1.1.31 // indirect
	github.com/mitchellh/go-testing-interface v1.14.0 // indirect
	github.com/opentracing-contrib/go-gin v0.0.0-20201220185307-1dd2273433a4
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/viper v1.8.1
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/uber/jaeger-lib v2.4.0+incompatible
	go.mongodb.org/mongo-driver v1.7.0
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20200930160638-afb6bcd081ae // indirect
	google.golang.org/grpc v1.39.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gorm.io/gorm v1.21.12
)
