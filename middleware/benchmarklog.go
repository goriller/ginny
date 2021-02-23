package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorillazer/ginny/logg"
	"github.com/gorillazer/ginny/trace"
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

		logg.InfoContext(ctx, logg.KeyBenchmarkLog,
			zap.String(logg.KeyClientIP, ctx.ClientIP()),
			zap.String(logg.KeyHandlerName, ctx.HandlerName()),
			zap.String(logg.KeyHTTPMethod, ctx.Request.Method),
			zap.String(logg.KeyFullPath, ctx.FullPath()),
			zap.Int64(logg.KeyLatency, latency.Milliseconds()),
			zap.Int(logg.KeyStatusCode, ctx.Writer.Status()),
		)
	}
}
