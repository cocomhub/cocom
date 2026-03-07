// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package clog

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 保护 defaultLogger 的互斥锁
var defaultLoggerMu sync.RWMutex

// 全局默认日志记录器
var defaultLogger = NewLogger(NewStdConfig(), WithGlobalCallerSkip(1))

func Init() {
	cfg := GetConfigByViper()
	SetLogger(cfg)
}

func SetLogger(config Config) func() {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()

	prev := defaultLogger
	defaultLogger = NewLogger(config, WithGlobalCallerSkip(1))
	if viper.GetBool("clog.replaceStdout") {
		_, _ = ReplaceStdLog()
	}
	return func() { ReplaceLogger(prev) }
}

func ReplaceLogger(logger *CLogger) func() {
	return SetLogger(logger.config)
}

func AppName() string {
	return defaultLogger.AppName()
}

func GetPrintLogger(err error) func(context.Context, ...any) {
	if err != nil {
		return Error
	}
	return Debug
}

func GetPrintfLogger(err error) func(context.Context, string, ...any) {
	if err != nil {
		return Errorf
	}
	return Debugf
}

// Print logs a message at level Debug on the CLogger.
func Print(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Print(ctx, args...)
}

// Printf logs a message at level Debug on the CLogger.
func Printf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Printf(ctx, format, args...)
}

// Debug logs a message at level Debug on the CLogger.
func Debug(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Debug(ctx, args...)
}

// Debugf logs a message at level Debug on the CLogger.
func Debugf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Debugf(ctx, format, args...)
}

// Info logs a message at level Info on the CLogger.
func Info(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Info(ctx, args...)
}

// Infof logs a message at level Info on the CLogger.
func Infof(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Infof(ctx, format, args...)
}

// Warn logs a message at level Warn on the CLogger.
func Warn(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Warn(ctx, args...)
}

// Warnf logs a message at level Warn on the CLogger.
func Warnf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Warnf(ctx, format, args...)
}

// Error logs a message at level Error on the CLogger.
func Error(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Error(ctx, args...)
}

// Errorf logs a message at level Error on the CLogger.
func Errorf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Errorf(ctx, format, args...)
}

// Fatal logs a message at level Fatal on the CLogger.
func Fatal(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Fatal(ctx, args...)
}

// Fatalf logs a message at level Fatal on the CLogger.
func Fatalf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Fatalf(ctx, format, args...)
}

// Panic logs a message at level Panic on the CLogger.
func Panic(ctx context.Context, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Panic(ctx, args...)
}

// Panicf logs a message at level Panic on the CLogger.
func Panicf(ctx context.Context, format string, args ...any) {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	defaultLogger.Panicf(ctx, format, args...)
}

// With return a logger with an extra field.
func With(key string, value any) *CLogger {
	return defaultLogger.With(key, value)
}

// Withs return a logger with extra fields.
func Withs(fields map[string]any) *CLogger {
	return defaultLogger.Withs(fields)
}

// WithField return a logger with extra zap fields.
func WithField(fields ...zap.Field) *CLogger {
	return defaultLogger.WithField(fields...)
}

// AddCallerSkip return a logger with new caller skip.
func AddCallerSkip(skip int) *CLogger {
	return defaultLogger.AddCallerSkip(skip)
}

// RedirectStdLog redirects output from the standard library's package-global
// logger to the supplied logger at InfoLevel. Since zap already handles caller
// annotations, timestamps, etc., it automatically disables the standard
// library's annotations and prefixing.
//
// It returns a function to restore the original prefix and flags and reset the
// standard library's output to os.Stderr.
func RedirectStdLog(l *CLogger) (func(), error) {
	return redirectStdLogAt(l, "info")
}

// RedirectStdLogAt redirects output from the standard library's package-global
// logger to the supplied logger at the specified level. Since zap already
// handles caller annotations, timestamps, etc., it automatically disables the
// standard library's annotations and prefixing.
//
// It returns a function to restore the original prefix and flags and reset the
// standard library's output to os.Stderr.
func RedirectStdLogAt(l *CLogger, level string) (func(), error) {
	return redirectStdLogAt(l, level)
}

func ReplaceStdLog() (func(), error) {
	return redirectStdLogAt(defaultLogger, "info")
}

func redirectStdLogAt(l *CLogger, level string) (func(), error) {
	flags := log.Flags()
	prefix := log.Prefix()
	log.SetFlags(0)
	log.SetPrefix("")
	logFunc, err := levelToFunc(l, level)
	if err != nil {
		return nil, err
	}
	log.SetOutput(&loggerWriter{NewTraceCtx("stdLog"), logFunc})
	return func() {
		log.SetFlags(flags)
		log.SetPrefix(prefix)
		log.SetOutput(os.Stderr)
	}, nil
}

func levelToFunc(l *CLogger, lvl string) (func(context.Context, ...any), error) {
	switch strings.ToLower(lvl) {
	case "debug":
		return l.Debug, nil
	case "info":
		return l.Info, nil
	case "warn":
		return l.Warn, nil
	case "error":
		return l.Error, nil
	case "panic":
		return l.Panic, nil
	case "fatal":
		return l.Fatal, nil
	default:
		return l.Info, nil
	}
}

type loggerWriter struct {
	ctx     context.Context
	logFunc func(context.Context, ...any)
}

func (l *loggerWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	l.logFunc(l.ctx, string(p))
	return len(p), nil
}

// IsDebug 检查是否为调试模式
func IsDebug() bool {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger.Level() == zapcore.DebugLevel
}

// Level 获取日志级别
func (l *CLogger) Level() zapcore.Level {
	// 从配置中获取日志级别
	switch strings.ToLower(l.config.ConsoleLevel) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}
