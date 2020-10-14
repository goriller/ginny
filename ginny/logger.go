/**
 * Author: richen
 * Date: 2020-07-13 17:01:16
 * LastEditTime: 2020-07-29 10:25:36
 * Description:
 * Copyright (c) - <richenlin(at)gmail.com>
 */
package ginny

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Colors
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

// LogLevel
type LogLevel int

const (
	Silent LogLevel = iota + 1
	Error
	Warn
	Info
)

var sourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	sourceDir = regexp.MustCompile(`logger\.go`).ReplaceAllString(file, "")
	fmt.Println(sourceDir)
}

// FileWithLineNum
func FileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)

		if ok && (!strings.HasPrefix(file, sourceDir) || strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.FormatInt(int64(line), 10)
		}
	}
	return ""
}

// Writer log writer interface
type Writer interface {
	Printf(string, ...interface{})
}

type LoggerConfig struct {
	SlowThreshold time.Duration
	Colorful      bool
	LogLevel      LogLevel
}

type logger struct {
	Writer
	LoggerConfig
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// ILogger logger interface
type ILogger interface {
	LogMode(LogLevel) ILogger
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Trace(begin time.Time, err error)
}

var DefaultLogger = NewLogger(log.New(os.Stdout, "", log.LstdFlags), LoggerConfig{
	SlowThreshold: time.Nanosecond,
	LogLevel:      Info,
	Colorful:      true,
})

func NewLogger(writer Writer, config LoggerConfig) ILogger {
	var (
		infoStr      = "[INFO] %s "
		warnStr      = "[WARN] %s "
		errStr       = "[ERROR] %s "
		traceStr     = "[TRACE] %s\n [%.3fms] [rows:%s] "
		traceWarnStr = "[TRACE] %s\n [%.3fms] [rows:%s] "
		traceErrStr  = "[TRACE] %+v\n [%.3fms] [rows:%s] "
	)

	if config.Colorful {
		infoStr = Green + "[INFO] " + Reset
		warnStr = Magenta + "[WARN] " + Reset
		errStr = Red + "[ERROR] " + Reset
		traceStr = Yellow + "[TRACE] %s " + Blue + "[%.3fms] " + "[rows:%s] " + Reset
		traceWarnStr = MagentaBold + "[TRACE] %s " + Yellow + "[%.3fms] " + "[rows:%s] " + Reset
		traceErrStr = RedBold + "[TRACE] " + Red + "%+v\n" + Reset + Yellow + "[%.3fms] " + Blue + "[rows:%s] " + Reset
	}

	return &logger{
		Writer:       writer,
		LoggerConfig: config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

// LogMode log mode
func (l *logger) LogMode(level LogLevel) ILogger {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l logger) Info(msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoStr+msg, append([]interface{}{}, data...)...)
	}
}

// Warn print warn messages
func (l logger) Warn(msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{}, data...)...)
	}
}

// Error print error messages
func (l logger) Error(msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errStr+msg, append([]interface{}{}, data...)...)
	}
}

// Trace print sql message
func (l logger) Trace(begin time.Time, err error) {
	if l.LogLevel > 0 {
		elapsed := time.Since(begin)
		switch {
		case err != nil && l.LogLevel >= Error:
			l.Printf(l.traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, FileWithLineNum())
		case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
			l.Printf(l.traceWarnStr, float64(elapsed.Nanoseconds())/1e6, FileWithLineNum())
		case l.LogLevel >= Info:
			l.Printf(l.traceStr, float64(elapsed.Nanoseconds())/1e6, FileWithLineNum())
		}
	}
}
