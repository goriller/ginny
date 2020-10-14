package middleware

import (
	"time"

	"git.code.oa.com/linyyyang/ginny/logger"
	"git.code.oa.com/linyyyang/ginny/trace"
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

		logger.InfoContext(ctx, logger.KeyBenchmarkLog,
			zap.String(logger.KeyClientIP, ctx.ClientIP()),
			zap.String(logger.KeyHandlerName, ctx.HandlerName()),
			zap.String(logger.KeyHTTPMethod, ctx.Request.Method),
			zap.String(logger.KeyFullPath, ctx.FullPath()),
			zap.Int64(logger.KeyLatency, latency.Milliseconds()),
			zap.Int(logger.KeyStatusCode, ctx.Writer.Status()),
		)
	}
}
