package trace

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

// TestSetContext xxx
func TestSetContext(t *testing.T) {
	// args test struct
	type args struct {
		origin interface{}
		dst    context.Context
	}

	msg := &Message{Username: "username", ReqID: "req_id", DeviceID: "device_id"}
	//origin 1
	origin1 := &gin.Context{}
	origin1.Set(KeyMessage, msg)

	//origin 2
	ctx := context.TODO()
	origin2 := context.WithValue(ctx, KeyMessage, msg)

	var kv []string
	for k, v := range msg.MessageMap() {
		kv = append(kv, k, v)
	}
	want := metadata.AppendToOutgoingContext(context.TODO(), kv...)

	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{name: "test1", args: args{origin: origin1, dst: context.TODO()}, want: want},
		{name: "test2", args: args{origin: origin2, dst: context.TODO()}, want: want},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetContext(tt.args.origin, tt.args.dst); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetContext() = %v, want %v", got, tt.want)
				t.Log(metadata.FromOutgoingContext(got))
			} else {
				t.Log(metadata.FromOutgoingContext(got))
			}
		})
	}
}

// TestSetHTTPRequest xxx
func TestSetHTTPRequest(t *testing.T) {
	// args test struct
	type args struct {
		ctx     interface{}
		httpReq *http.Request
	}

	msg := &Message{Username: "username", ReqID: "req_id", DeviceID: "device_id"}
	//origin 1
	origin1 := &gin.Context{}
	origin1.Set(KeyMessage, msg)

	//origin 2
	ctx := context.TODO()
	origin2 := context.WithValue(ctx, KeyMessage, msg)

	tests := []struct {
		name string
		args args
	}{
		{name: "test1", args: args{ctx: origin1, httpReq: &http.Request{Header: make(http.Header)}}},
		{name: "test2", args: args{ctx: origin2, httpReq: &http.Request{Header: make(http.Header)}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.args.httpReq
			SetHTTPRequest(tt.args.ctx, req)
			//check http request header
			t.Log(req.Header)
		})
	}
}
