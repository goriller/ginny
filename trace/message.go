package trace

import (
	"context"
	"time"

	"git.code.oa.com/linyyyang/ginny/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// KeyMessage 存放message的key，协议无关，也会用在grpc请求的context
	KeyMessage  = "message"
	KeyUsername = "username"
	KeyReqID    = "reqID"
	KeyDeviceID = "deviceID"
)

// Message 链路跟踪上下文数据结构，http server和grpc server共用
type Message struct {
	StartTime time.Time
	Logger    *zap.Logger
	ReqID     string
	Username  string
	DeviceID  string
	Context   interface{}
	ExtData   map[string]interface{}
}

//MessageMap 将trace数据放到mapping
func (m *Message) MessageMap() map[string]string {
	mapping := make(map[string]string)
	mapping[KeyReqID] = m.ReqID
	mapping[KeyUsername] = m.Username
	mapping[KeyDeviceID] = m.DeviceID
	return mapping
}

// GinMessage 返回gin context链路追踪的msg
func GinMessage(ctx *gin.Context) *Message {
	val, ok := ctx.Get(KeyMessage)
	var msg *Message
	if ok {
		msg, ok = val.(*Message)
		if !ok {
			msg = &Message{Context: ctx}
			ctx.Set(KeyMessage, msg)
		}
	} else {
		msg = &Message{Context: ctx}
		ctx.Set(KeyMessage, msg)
	}
	return msg
}

// CtxMessage 返回context链路追踪的msg
func CtxMessage(ctx context.Context) *Message {
	val := ctx.Value(KeyMessage)
	var msg *Message
	msg, ok := val.(*Message)
	if !ok {
		msg = &Message{}
		ctx = context.WithValue(ctx, KeyMessage, msg)
		msg.Context = ctx
	}
	return msg
}

//TraceFields return trace fields
func (m *Message) TraceFields() []zapcore.Field {
	addFields := []zapcore.Field{
		zap.String(KeyReqID, m.ReqID), zap.String(KeyUsername, m.Username), zap.String(KeyDeviceID, m.DeviceID),
	}
	return addFields
}

//MessageFromCtx msg from context
func MessageFromCtx(ctx interface{}) *Message {
	var msg *Message
	switch ctx.(type) {
	case *gin.Context:
		ctx := ctx.(*gin.Context)
		msg = GinMessage(ctx)
	case context.Context:
		ctx := ctx.(context.Context)
		msg = CtxMessage(ctx)
	default:
		panic("invalid context to get message")
	}
	return msg
}

//NewNoopMsg 返回一个空msg
func NewNoopMsg() *Message {
	return &Message{}
}

//NewMessage new msg
func NewMessage(logger *zap.Logger, reqID, username, deviceID string, context interface{}) *Message {
	return &Message{Logger: logger, ReqID: reqID, Username: username, DeviceID: deviceID, Context: context}
}

//RandReqID rand reqID
func RandReqID() string {
	id, err := utils.GenerateID()
	if err != nil {
		return ""
	}
	return id
}
