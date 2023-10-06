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
)

var (
	_globalMu sync.RWMutex
	_logger   = NewLogger(NewStdConfig(), WithGlobalCallerSkip(1))
)

func init() {
	SetLogger(GetConfigByViper())
}

func SetLogger(config Config) func() {
	_globalMu.Lock()
	defer _globalMu.Unlock()

	prev := _logger
	_logger = NewLogger(config, WithGlobalCallerSkip(1))
	if viper.GetBool("clog.replaceStdout") {
		_, _ = ReplaceStdLog()
	}
	return func() { ReplaceLogger(prev) }
}

func ReplaceLogger(logger *CLogger) func() {
	return SetLogger(logger.config)
}

func AppName() string {
	return _logger.AppName()
}

func GetPrintLogger(err error) func(context.Context, ...interface{}) {
	if err != nil {
		return Error
	}
	return Debug
}

func GetPrintfLogger(err error) func(context.Context, string, ...interface{}) {
	if err != nil {
		return Errorf
	}
	return Debugf
}

// Print logs a message at level Debug on the CLogger.
func Print(ctx context.Context, args ...interface{}) {
	_logger.Print(ctx, args...)
}

// Printf logs a message at level Debug on the CLogger.
func Printf(ctx context.Context, format string, args ...interface{}) {
	_logger.Printf(ctx, format, args...)
}

// Debug logs a message at level Debug on the CLogger.
func Debug(ctx context.Context, args ...interface{}) {
	_logger.Debug(ctx, args...)
}

// Debugf logs a message at level Debug on the CLogger.
func Debugf(ctx context.Context, format string, args ...interface{}) {
	_logger.Debugf(ctx, format, args...)
}

// Info logs a message at level Info on the CLogger.
func Info(ctx context.Context, args ...interface{}) {
	_logger.Info(ctx, args...)
}

// Infof logs a message at level Info on the CLogger.
func Infof(ctx context.Context, format string, args ...interface{}) {
	_logger.Infof(ctx, format, args...)
}

// Warn logs a message at level Warn on the CLogger.
func Warn(ctx context.Context, args ...interface{}) {
	_logger.Warn(ctx, args...)
}

// Warnf logs a message at level Warn on the CLogger.
func Warnf(ctx context.Context, format string, args ...interface{}) {
	_logger.Warnf(ctx, format, args...)
}

// Error logs a message at level Error on the CLogger.
func Error(ctx context.Context, args ...interface{}) {
	_logger.Error(ctx, args...)
}

// Errorf logs a message at level Error on the CLogger.
func Errorf(ctx context.Context, format string, args ...interface{}) {
	_logger.Errorf(ctx, format, args...)
}

// Fatal logs a message at level Fatal on the CLogger.
func Fatal(ctx context.Context, args ...interface{}) {
	_logger.Fatal(ctx, args...)
}

// Fatalf logs a message at level Fatal on the CLogger.
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	_logger.Fatalf(ctx, format, args...)
}

// Panic logs a message at level Panic on the CLogger.
func Panic(ctx context.Context, args ...interface{}) {
	_logger.Panic(ctx, args...)
}

// Panicf logs a message at level Panic on the CLogger.
func Panicf(ctx context.Context, format string, args ...interface{}) {
	_logger.Panicf(ctx, format, args...)
}

// With return a logger with an extra field.
func With(key string, value interface{}) *CLogger {
	return _logger.With(key, value)
}

// Withs return a logger with extra fields.
func Withs(fields map[string]interface{}) *CLogger {
	return _logger.Withs(fields)
}

// WithField return a logger with extra zap fields.
func WithField(fields ...zap.Field) *CLogger {
	return _logger.WithField(fields...)
}

// AddCallerSkip return a logger with new caller skip.
func AddCallerSkip(skip int) *CLogger {
	return _logger.AddCallerSkip(skip)
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
	return redirectStdLogAt(_logger, "info")
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

func levelToFunc(l *CLogger, lvl string) (func(context.Context, ...interface{}), error) {
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
	logFunc func(context.Context, ...interface{})
}

func (l *loggerWriter) Write(p []byte) (int, error) {
	p = bytes.TrimSpace(p)
	l.logFunc(l.ctx, string(p))
	return len(p), nil
}
