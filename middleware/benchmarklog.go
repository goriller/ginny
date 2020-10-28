package middleware

import (
	"time"

	"git.code.oa.com/Ginny/ginny/logiy"
	"git.code.oa.com/Ginny/ginny/trace"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BenchmarkLog 日志记录函数返回结果以及时延等信息，需要作为第一个中间件注入
func BenchmarkLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		msg := trace.GinMessage(ctx)
		o := trace.WithStartTime(time.Now())
		o(msg)

		ctx.Next()

		latency := time.Since(msg.StartTime)

		logiy.InfoContext(ctx, logiy.KeyBenchmarkLog,
			zap.String(logiy.KeyClientIP, ctx.ClientIP()),
			zap.String(logiy.KeyHandlerName, ctx.HandlerName()),
			zap.String(logiy.KeyHTTPMethod, ctx.Request.Method),
			zap.String(logiy.KeyFullPath, ctx.FullPath()),
			zap.Int64(logiy.KeyLatency, latency.Milliseconds()),
			zap.Int(logiy.KeyStatusCode, ctx.Writer.Status()),
		)
	}
}
