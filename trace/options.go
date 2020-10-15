package trace

import (
	"time"

	"go.uber.org/zap"
)

// Option 参数设置
type Option func(msg *Message)

//WithReqID set reqID
func WithReqID(id string) Option {
	return func(msg *Message) {
		msg.ReqID = id
	}
}

//WithLogger set loggy
func WithLogger(l *zap.Logger) Option {
	return func(msg *Message) {
		msg.Logger = l
	}
}

//WithUsername set username
func WithUsername(username string) Option {
	return func(msg *Message) {
		msg.Username = username
	}
}

//WithDeviceID set dev id
func WithDeviceID(id string) Option {
	return func(msg *Message) {
		msg.DeviceID = id
	}
}

//WithContext set context
func WithContext(ctx interface{}) Option {
	return func(msg *Message) {
		msg.Context = ctx
	}
}

//WithRandomReqID set rand reqID
func WithRandomReqID() Option {
	return func(msg *Message) {
		msg.ReqID = RandReqID()
	}
}

//WithStartTime set start time
func WithStartTime(start time.Time) Option {
	return func(msg *Message) {
		msg.StartTime = start
	}
}

// AddExtData append data
func AddExtData(key string, value interface{}) Option {
	return func(msg *Message) {
		if msg.ExtData == nil {
			msg.ExtData = make(map[string]interface{})
		}
		msg.ExtData[key] = value
	}
}
