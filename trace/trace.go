package trace

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

//SetHTTPRequest 设置http request trace数据，ctx可以是context或者gin context
//grpc -> http
//http -> http
func SetHTTPRequest(ctx interface{}, httpReq *http.Request) {
	msg := MessageFromCtx(ctx)
	m := msg.MessageMap()
	for k, v := range m {
		httpReq.Header.Set(k, v)
	}
}

//SetContext 初始化带trace的context，origin可以为context或者gin context
//grpc -> grpc
//http -> grpc
func SetContext(origin interface{}, ctx context.Context) context.Context {
	msg := MessageFromCtx(origin)
	m := msg.MessageMap()

	var kv []string
	for k, v := range m {
		kv = append(kv, k, v)
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}
