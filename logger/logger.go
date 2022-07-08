package logger

import (
	"os"

	"github.com/google/wire"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	defaultLogger *zap.Logger
	// LoggerProviderSet
	LoggerProviderSet = wire.NewSet(NewOptions, NewLogger)
)

// Options is log configuration struct
type Options struct {
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Level      string
	Stdout     bool
}

func NewOptions(v *viper.Viper) (*Options, error) {
	var (
		err error
		o   = new(Options)
	)
	if err = v.UnmarshalKey("log", o); err != nil {
		return nil, err
	}

	if o.Level == "" {
		o.Level = "debug"
	}
	if o.MaxAge == 0 {
		o.MaxAge = 3
	}
	if o.MaxSize == 0 {
		o.MaxSize = 1024
	}
	if o.MaxBackups == 0 {
		o.MaxBackups = 3
	}

	return o, err
}

// NewLogger for init zap log library
func NewLogger(o *Options) (*zap.Logger, error) {
	var (
		err   error
		level = zap.NewAtomicLevel()
	)

	err = level.UnmarshalText([]byte(o.Level))
	if err != nil {
		return nil, err
	}

	cores := make([]zapcore.Core, 0, 1)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.CallerKey = ""
	encoderCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	if o.Filename != "" {
		fw := zapcore.AddSync(&lumberjack.Logger{
			Filename:   o.Filename,
			MaxSize:    o.MaxSize, // megabytes
			MaxBackups: o.MaxBackups,
			MaxAge:     o.MaxAge, // days
		})
		// file core 采用jsonEncoder
		je := zapcore.NewJSONEncoder(encoderCfg)
		cores = append(cores, zapcore.NewCore(je, fw, level))
	}

	cw := zapcore.Lock(os.Stdout)
	// stdout core 采用 ConsoleEncoder
	if o.Stdout {
		ce := zapcore.NewConsoleEncoder(encoderCfg)
		cores = append(cores, zapcore.NewCore(ce, cw, level))
	}

	core := zapcore.NewTee(cores...)
	log := zap.New(core)

	zap.ReplaceGlobals(log)

	return log, err
}

// GetLogger
func GetLogger() *zap.Logger {
	if defaultLogger != nil {
		return defaultLogger
	}
	cfg := zap.NewProductionConfig()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.CallerKey = ""
	encoderCfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	cfg.EncoderConfig = encoderCfg

	defaultLogger, _ = cfg.Build()
	return defaultLogger
}
