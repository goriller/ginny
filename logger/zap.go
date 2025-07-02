package logger

import (
	"context"
	"os"

	"github.com/goriller/ginny-util/graceful"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	std *zap.Logger
)

func init() {
	// you can use `export LOG_PATH=logs/log.log` to set log output to a file
	logFilePath := os.Getenv("LOG_PATH")
	level := zap.NewAtomicLevel()
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		err := level.UnmarshalText([]byte(logLevel))
		if err != nil {
			level = zap.NewAtomicLevelAt(zap.DebugLevel)
		}
	} else {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	cores := make([]zapcore.Core, 0, 1)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.CallerKey = "caller"
	encoderCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	if logFilePath != "" {
		fw := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    1024, // megabytes
			MaxBackups: 3,
			MaxAge:     3, // days
		})
		// file core 采用jsonEncoder
		je := zapcore.NewJSONEncoder(encoderCfg)
		cores = append(cores, zapcore.NewCore(je, fw, level))
	} else {
		// stdout core
		cw := zapcore.Lock(os.Stdout)
		ce := zapcore.NewJSONEncoder(encoderCfg)
		cores = append(cores, zapcore.NewCore(ce, cw, level))
	}

	core := zapcore.NewTee(cores...)
	opt := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}
	std = zap.New(core, opt...)
	zap.ReplaceGlobals(std)

	graceful.AddCloser(func(ctx context.Context) error {
		return std.Sync()
	})
}
