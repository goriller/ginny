package ginny

import (
	"context"

	"github.com/gin-gonic/gin"
)

// 定义全局上下文中的键
type (
	transCtx     struct{}
	noTransCtx   struct{}
	transLockCtx struct{}
	traceIDCtx   struct{}
)

// ginny.Context
type Context struct {
	*gin.Context
}

// NewTrans 创建事务的上下文
func (ctx *Context) NewTrans(trans interface{}) context.Context {
	return context.WithValue(ctx, transCtx{}, trans)
}

// GetTrans 从上下文中获取事务
func (ctx *Context) GetTrans() (interface{}, bool) {
	v := ctx.Value(transCtx{})
	return v, v != nil
}

// NewNoTrans 创建不使用事务的上下文
func (ctx *Context) NewNoTrans() context.Context {
	return context.WithValue(ctx, noTransCtx{}, true)
}

// GetNoTrans 从上下文中获取不使用事务标识
func (ctx *Context) GetNoTrans() bool {
	v := ctx.Value(noTransCtx{})
	return v != nil && v.(bool)
}

// NewTransLock 创建事务锁的上下文
func (ctx *Context) NewTransLock() context.Context {
	return context.WithValue(ctx, transLockCtx{}, true)
}

// GetTransLock 从上下文中获取事务锁
func (ctx *Context) GetTransLock() bool {
	v := ctx.Value(transLockCtx{})
	return v != nil && v.(bool)
}

// NewTraceID 创建追踪ID的上下文
func (ctx *Context) NewTraceID(traceID string) context.Context {
	return context.WithValue(ctx, traceIDCtx{}, traceID)
}

// GetTraceID 从上下文中获取追踪ID
func (ctx *Context) GetTraceID() (string, bool) {
	v := ctx.Value(traceIDCtx{})
	if v != nil {
		if s, ok := v.(string); ok {
			return s, s != ""
		}
	}
	return "", false
}
