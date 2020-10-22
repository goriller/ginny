package middleware

import (
	"time"

	"git.code.oa.com/Ginny/ginny/loggy"
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

		loggy.InfoContext(ctx, loggy.KeyBenchmarkLog,
			zap.String(loggy.KeyClientIP, ctx.ClientIP()),
			zap.String(loggy.KeyHandlerName, ctx.HandlerName()),
			zap.String(loggy.KeyHTTPMethod, ctx.Request.Method),
			zap.String(loggy.KeyFullPath, ctx.FullPath()),
			zap.Int64(loggy.KeyLatency, latency.Milliseconds()),
			zap.Int(loggy.KeyStatusCode, ctx.Writer.Status()),
		)
	}
}
