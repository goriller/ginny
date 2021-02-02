package logg

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// PlainTextEncoder 文本行日志
	PlainTextEncoder = 1
	// JSONEncoder JSON格式日志
	JSONEncoder = 2
)

// FileOption 设置文件日志保存选项
type FileOption struct {
	// 日志格式，PlainTextEncoder|JSONEncoder
	Encoder int
	// [Debug-Info]级别日志文件路径，默认./log/info.log
	InfoLogFilename string
	// Error及以上级别日志文件路径，默认./log/error.log
	ErrorLogFilename string
	// 日志文件最大大小，单位M
	MaxSize int
	// 保留旧文件最大天数
	MaxAge int
	// 保留旧文件最大个数
	MaxBackups int
	// 是否压缩/归档旧文件
	Compress bool
}

// DefaultOption 构造默认的FileLog Option
func DefaultOption() FileOption {
	return FileOption{
		Encoder:          JSONEncoder,
		InfoLogFilename:  "./log/info.log",
		ErrorLogFilename: "./log/error.log",
		MaxSize:          100,
		MaxAge:           30,
		MaxBackups:       100,
		Compress:         false,
	}
}

// ConsoleLogger Debug及以上Level日志输出到控制台
func ConsoleLogger() *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewConsoleEncoder(config)

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	return zap.New(core, zap.AddCaller())
}

// RotateFileLogger 文件日志，通过options设置日志文件路径、日志格式以及压缩归档等参数
func RotateFileLogger(options FileOption) *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	switch options.Encoder {
	case PlainTextEncoder:
		encoder = zapcore.NewConsoleEncoder(config)
	case JSONEncoder:
		encoder = zapcore.NewJSONEncoder(config)
	default:
		encoder = zapcore.NewJSONEncoder(config)
	}

	infoLog := lumberjack.Logger{
		Filename:   options.InfoLogFilename,
		MaxSize:    options.MaxSize,
		MaxAge:     options.MaxAge,
		MaxBackups: options.MaxBackups,
		Compress:   options.Compress,
	}

	errorLog := lumberjack.Logger{
		Filename:   options.ErrorLogFilename,
		MaxSize:    options.MaxSize,
		MaxAge:     options.MaxAge,
		MaxBackups: options.MaxBackups,
		Compress:   options.Compress,
	}

	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev >= zap.ErrorLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev >= zap.DebugLevel
	})

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(&infoLog), lowPriority),
		zapcore.NewCore(encoder, zapcore.AddSync(&errorLog), highPriority),
	)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
}
