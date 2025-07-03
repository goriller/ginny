package logger

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BatchLogger 批处理日志器
type BatchLogger struct {
	logger    *zap.Logger
	buffer    []logEntry
	mu        sync.Mutex
	flushSize int           // 批量大小
	flushTime time.Duration // 刷新间隔
	stopCh    chan struct{}
	wg        sync.WaitGroup

	// 性能统计
	stats BatchStats
}

// logEntry 日志条目
type logEntry struct {
	level   zapcore.Level
	message string
	fields  []zap.Field
	time    time.Time
}

// BatchStats 批处理统计
type BatchStats struct {
	TotalLogs    int64   `json:"total_logs"`
	BatchedLogs  int64   `json:"batched_logs"`
	FlushCount   int64   `json:"flush_count"`
	DroppedLogs  int64   `json:"dropped_logs"`
	AvgBatchSize float64 `json:"avg_batch_size"`
}

// BatchConfig 批处理配置
type BatchConfig struct {
	FlushSize     int           // 批量大小，默认100
	FlushInterval time.Duration // 刷新间隔，默认1秒
	BufferSize    int           // 缓冲区大小，默认1000
}

// NewBatchLogger 创建批处理日志器
func NewBatchLogger(baseLogger *zap.Logger, config BatchConfig) *BatchLogger {
	if config.FlushSize <= 0 {
		config.FlushSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = time.Second
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	bl := &BatchLogger{
		logger:    baseLogger,
		buffer:    make([]logEntry, 0, config.BufferSize),
		flushSize: config.FlushSize,
		flushTime: config.FlushInterval,
		stopCh:    make(chan struct{}),
	}

	// 启动批处理协程
	bl.wg.Add(1)
	go bl.batchProcessor()

	return bl
}

// Debug 批处理Debug日志
func (bl *BatchLogger) Debug(msg string, fields ...zap.Field) {
	bl.addEntry(zapcore.DebugLevel, msg, fields)
}

// Info 批处理Info日志
func (bl *BatchLogger) Info(msg string, fields ...zap.Field) {
	bl.addEntry(zapcore.InfoLevel, msg, fields)
}

// Warn 批处理Warn日志
func (bl *BatchLogger) Warn(msg string, fields ...zap.Field) {
	bl.addEntry(zapcore.WarnLevel, msg, fields)
}

// Error 批处理Error日志
func (bl *BatchLogger) Error(msg string, fields ...zap.Field) {
	bl.addEntry(zapcore.ErrorLevel, msg, fields)
}

// Fatal Fatal日志（立即刷新）
func (bl *BatchLogger) Fatal(msg string, fields ...zap.Field) {
	// Fatal级别立即写入，不进行批处理
	bl.logger.Fatal(msg, fields...)
}

// addEntry 添加日志条目到缓冲区
func (bl *BatchLogger) addEntry(level zapcore.Level, msg string, fields []zap.Field) {
	entry := logEntry{
		level:   level,
		message: msg,
		fields:  fields,
		time:    time.Now(),
	}

	bl.mu.Lock()
	defer bl.mu.Unlock()

	// 检查缓冲区是否已满
	if len(bl.buffer) >= cap(bl.buffer) {
		bl.stats.DroppedLogs++
		return
	}

	bl.buffer = append(bl.buffer, entry)
	bl.stats.TotalLogs++

	// 检查是否需要立即刷新
	if len(bl.buffer) >= bl.flushSize {
		bl.flushBuffer()
	}
}

// flushBuffer 刷新缓冲区（需要持有锁）
func (bl *BatchLogger) flushBuffer() {
	if len(bl.buffer) == 0 {
		return
	}

	// 复制缓冲区以避免长时间持锁
	batch := make([]logEntry, len(bl.buffer))
	copy(batch, bl.buffer)
	bl.buffer = bl.buffer[:0] // 重置缓冲区

	bl.stats.FlushCount++
	bl.stats.BatchedLogs += int64(len(batch))
	if bl.stats.FlushCount > 0 {
		bl.stats.AvgBatchSize = float64(bl.stats.BatchedLogs) / float64(bl.stats.FlushCount)
	}

	// 释放锁后写入日志
	bl.mu.Unlock()
	bl.writeBatch(batch)
	bl.mu.Lock()
}

// writeBatch 写入批量日志
func (bl *BatchLogger) writeBatch(batch []logEntry) {
	for _, entry := range batch {
		switch entry.level {
		case zapcore.DebugLevel:
			bl.logger.Debug(entry.message, entry.fields...)
		case zapcore.InfoLevel:
			bl.logger.Info(entry.message, entry.fields...)
		case zapcore.WarnLevel:
			bl.logger.Warn(entry.message, entry.fields...)
		case zapcore.ErrorLevel:
			bl.logger.Error(entry.message, entry.fields...)
		}
	}
}

// batchProcessor 批处理协程
func (bl *BatchLogger) batchProcessor() {
	defer bl.wg.Done()

	ticker := time.NewTicker(bl.flushTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bl.mu.Lock()
			bl.flushBuffer()
			bl.mu.Unlock()
		case <-bl.stopCh:
			// 最后一次刷新
			bl.mu.Lock()
			bl.flushBuffer()
			bl.mu.Unlock()
			return
		}
	}
}

// Flush 手动刷新缓冲区
func (bl *BatchLogger) Flush() {
	bl.mu.Lock()
	bl.flushBuffer()
	bl.mu.Unlock()
}

// Stats 获取批处理统计
func (bl *BatchLogger) Stats() BatchStats {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	return bl.stats
}

// Close 关闭批处理日志器
func (bl *BatchLogger) Close() error {
	close(bl.stopCh)
	bl.wg.Wait()
	return nil
}

// WithBatch 包装logger以支持批处理
type BatchWrapper struct {
	*BatchLogger
	original *zap.Logger
}

// With 实现StdLogger接口
func (bw *BatchWrapper) With(fields ...zap.Field) *zap.Logger {
	return bw.original.With(fields...)
}

// NewBatchWrapper 创建批处理包装器
func NewBatchWrapper(logger *zap.Logger, config BatchConfig) *BatchWrapper {
	batchLogger := NewBatchLogger(logger, config)
	return &BatchWrapper{
		BatchLogger: batchLogger,
		original:    logger,
	}
}

// 全局批处理日志器实例
var (
	globalBatchLogger *BatchLogger
	batchOnce         sync.Once
)

// EnableBatchLogging 启用全局批处理日志
func EnableBatchLogging(config BatchConfig) {
	batchOnce.Do(func() {
		globalBatchLogger = NewBatchLogger(std, config)
	})
}

// BatchDebug 全局批处理Debug日志
func BatchDebug(msg string, fields ...zap.Field) {
	if globalBatchLogger != nil {
		globalBatchLogger.Debug(msg, fields...)
	} else {
		std.Debug(msg, fields...)
	}
}

// BatchInfo 全局批处理Info日志
func BatchInfo(msg string, fields ...zap.Field) {
	if globalBatchLogger != nil {
		globalBatchLogger.Info(msg, fields...)
	} else {
		std.Info(msg, fields...)
	}
}

// BatchWarn 全局批处理Warn日志
func BatchWarn(msg string, fields ...zap.Field) {
	if globalBatchLogger != nil {
		globalBatchLogger.Warn(msg, fields...)
	} else {
		std.Warn(msg, fields...)
	}
}

// BatchError 全局批处理Error日志
func BatchError(msg string, fields ...zap.Field) {
	if globalBatchLogger != nil {
		globalBatchLogger.Error(msg, fields...)
	} else {
		std.Error(msg, fields...)
	}
}

// FlushBatchLogs 刷新全局批处理日志
func FlushBatchLogs() {
	if globalBatchLogger != nil {
		globalBatchLogger.Flush()
	}
}
