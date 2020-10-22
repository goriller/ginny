/**
 * Author: richen
 * Date: 2020-07-13 17:01:16
 * LastEditTime: 2020-07-29 10:25:36
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package loggy

import (
	"git.code.oa.com/Ginny/ginny/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Colors
//const (
//	Reset       = "\033[0m"
//	Red         = "\033[31m"
//	Green       = "\033[32m"
//	Yellow      = "\033[33m"
//	Blue        = "\033[34m"
//	Magenta     = "\033[35m"
//	Cyan        = "\033[36m"
//	White       = "\033[37m"
//	MagentaBold = "\033[35;1m"
//	RedBold     = "\033[31;1m"
//	YellowBold  = "\033[33;1m"
//)

const (
	// log key 这些字段是通用的，与具体协议无关
	KeyBenchmarkLog = "benchmarkLog"
	KeyClientIP     = "clientIP"
	KeyHandlerName  = "handlerName"
	KeyHTTPMethod   = "httpMethod"
	KeyFullPath     = "fullPath"
	KeyLatency      = "latency"
	KeyStatusCode   = "statusCode"
)

// DefaultLogger 默认的logger，使用默认opts
var DefaultLogger *zap.Logger

func init() {
	DefaultLogger = RotateFileLogger(DefaultOption())
}

func log(logger *zap.Logger, level Level, msg string, fields ...zapcore.Field) {
	switch level {
	case debugLevel:
		logger.Debug(msg, fields...)
	case warnLevel:
		logger.Warn(msg, fields...)
	case errorLevel:
		logger.Error(msg, fields...)
	case fatalLevel:
		logger.Fatal(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
}

//DebugContext debug with context
func DebugContext(ctx interface{}, msg string, fields ...zapcore.Field) {
	message := trace.MessageFromCtx(ctx)
	if message.Logger == nil {
		message.Logger = DefaultLogger
	}
	log(message.Logger, debugLevel, msg, fields...)
}

//Debug debug
func Debug(msg string, fields ...zapcore.Field) {
	log(DefaultLogger, debugLevel, msg, fields...)
}

//InfoContext info with context
func InfoContext(ctx interface{}, msg string, fields ...zapcore.Field) {
	message := trace.MessageFromCtx(ctx)
	if message.Logger == nil {
		message.Logger = DefaultLogger
	}
	log(message.Logger, infoLevel, msg, fields...)
}

//Info info
func Info(msg string, fields ...zapcore.Field) {
	log(DefaultLogger, infoLevel, msg, fields...)
}

//WarnContext warn with context
func WarnContext(ctx interface{}, msg string, fields ...zapcore.Field) {
	message := trace.MessageFromCtx(ctx)
	if message.Logger == nil {
		message.Logger = DefaultLogger
	}
	log(message.Logger, warnLevel, msg, fields...)
}

//Warn warn
func Warn(msg string, fields ...zapcore.Field) {
	log(DefaultLogger, warnLevel, msg, fields...)
}

//ErrorContext error with context
func ErrorContext(ctx interface{}, msg string, fields ...zapcore.Field) {
	message := trace.MessageFromCtx(ctx)
	if message.Logger == nil {
		message.Logger = DefaultLogger
	}
	log(message.Logger, errorLevel, msg, fields...)
}

//Error error
func Error(msg string, fields ...zapcore.Field) {
	log(DefaultLogger, errorLevel, msg, fields...)
}

//FatalContext fatal with context
func FatalContext(ctx interface{}, msg string, fields ...zapcore.Field) {
	message := trace.MessageFromCtx(ctx)
	if message.Logger == nil {
		message.Logger = DefaultLogger
	}
	log(message.Logger, fatalLevel, msg, fields...)
}

//Fatal fatal
func Fatal(msg string, fields ...zapcore.Field) {
	log(DefaultLogger, fatalLevel, msg, fields...)
}

// WithContextFields 设置自定义数据到每条log里
func WithContextFields(ctx interface{}, fields ...zapcore.Field) {
	msg := trace.MessageFromCtx(ctx)
	if msg.Logger == nil {
		msg.Logger = DefaultLogger
	}
	msg.Logger = msg.Logger.With(fields...)
}
