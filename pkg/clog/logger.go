// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package clog

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type CLogger struct {
	config Config
	logger *zap.Logger
}

func newCore(level string, encoding string, w zapcore.WriteSyncer) (core zapcore.Core) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.NanosDurationEncoder
	encoderConfig.TimeKey = "T"
	encoderConfig.LevelKey = "L"
	encoderConfig.MessageKey = "M"
	encoderConfig.CallerKey = "LFILE"

	var l zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		l = zap.DebugLevel
	case "info":
		l = zap.InfoLevel
	case "warn":
		l = zap.WarnLevel
	case "error":
		l = zap.ErrorLevel
	case "fatal":
		l = zap.FatalLevel
	case "panic":
		l = zap.PanicLevel
	default:
		l = zap.InfoLevel
	}

	var e zapcore.Encoder
	switch encoding {
	case "json":
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		e = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		e = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		e = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core = zapcore.NewCore(e, w, l)
	return
}

func NewLogger(config Config, opts ...Option) *CLogger {
	var core zapcore.Core

	for _, opt := range opts {
		opt(&config)
	}

	if config.EnableFile {
		hook := lumberjack.Logger{
			Filename:   config.Filename,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			LocalTime:  config.LocalTime,
			Compress:   config.Compress,
		}
		w := zapcore.AddSync(&hook)
		fileCore := newCore(config.FileLevel, config.FileEncodeing, w)

		if core != nil {
			core = zapcore.NewTee(core, fileCore)
		} else {
			core = fileCore
		}
	}

	if config.EnableConsole {
		w := zapcore.Lock(os.Stderr)
		consoleCore := newCore(config.ConsoleLevel, config.ConsoleEncodeing, w)

		if core != nil {
			core = zapcore.NewTee(core, consoleCore)
		} else {
			core = consoleCore
		}
	}

	if !config.EnableFile && !config.EnableConsole {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zap.PanicLevel)
	}

	fields := []zap.Field{zap.String("LAPP", config.AppName)}
	if config.EnablePID {
		fields = append(fields, zap.Int("LPID", os.Getpid()))
	}

	if config.EnableSourceIP {
		fields = append(fields, zap.String("LIP", GetIP(config.SourceEth)))
	}

	core = core.With(fields)

	var zapOption []zap.Option
	if config.EnableCaller {
		zapOption = append(zapOption, zap.AddCaller(), zap.AddCallerSkip(config.GlobalCallerSkip+1))
	}

	l := zap.New(core, zapOption...)

	return &CLogger{config, l}
}

func GetIP(eth string) string {
	ifi, err := net.InterfaceByName(eth)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	addrs, err := ifi.Addrs()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

func (l *CLogger) getFields(ctx context.Context) (fields []zap.Field) {
	if !l.config.DisableTraceID {
		span := trace.SpanContextFromContext(ctx)
		if span.HasTraceID() {
			fields = append(fields, zap.String("TRACE_ID", span.TraceID().String()))
			fields = append(fields, zap.String("SPAN_ID", span.SpanID().String()))
		} else {
			fields = append(fields, zap.String("TRACE_ID", GetTraceID(ctx)))
		}
	}
	return
}

func (l CLogger) AppName() string {
	return l.config.AppName
}

// Print logs a message at level Debug on the CLogger.
func (l CLogger) Print(ctx context.Context, args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Printf logs a message at level Debug on the CLogger.
func (l CLogger) Printf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Debug logs a message at level Debug on the CLogger.
func (l CLogger) Debug(ctx context.Context, args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Debugf logs a message at level Debug on the CLogger.
func (l CLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Info logs a message at level Info on the CLogger.
func (l CLogger) Info(ctx context.Context, args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Infof logs a message at level Info on the CLogger.
func (l CLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Warn logs a message at level Warn on the CLogger.
func (l CLogger) Warn(ctx context.Context, args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Warnf logs a message at level Warn on the CLogger.
func (l CLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Error logs a message at level Error on the CLogger.
func (l CLogger) Error(ctx context.Context, args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Errorf logs a message at level Error on the CLogger.
func (l CLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Fatal logs a message at level Fatal on the CLogger.
func (l CLogger) Fatal(ctx context.Context, args ...interface{}) {
	l.logger.Fatal(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Fatalf logs a message at level Fatal on the CLogger.
func (l CLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

// Panic logs a message at level Painc on the CLogger.
func (l CLogger) Panic(ctx context.Context, args ...interface{}) {
	l.logger.Panic(fmt.Sprint(args...), l.getFields(ctx)...)
}

// Panicf logs a message at level Painc on the CLogger.
func (l CLogger) Panicf(ctx context.Context, format string, args ...interface{}) {
	l.logger.Panic(fmt.Sprintf(format, args...), l.getFields(ctx)...)
}

func (l *CLogger) zapFields(fields map[string]interface{}) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return zapFields
}

// With return a logger with an extra field.
func (l *CLogger) With(key string, value interface{}) *CLogger {
	return l.WithField(zap.Any(key, value))
}

// Withs return a logger with extra fields.
func (l *CLogger) Withs(fields map[string]interface{}) *CLogger {
	return l.WithField(l.zapFields(fields)...)
}

// WithField return a logger with extra zap fields.
func (l *CLogger) WithField(fields ...zap.Field) *CLogger {
	return &CLogger{l.config, l.logger.With(fields...)}
}

// AddCallerSkip return a logger with new caller skip.
func (l *CLogger) AddCallerSkip(skip int) *CLogger {
	return &CLogger{l.config, l.logger.WithOptions(zap.AddCallerSkip(skip))}
}

// NewTraceLogger 创建带有追踪 ID 的日志记录器
func NewTraceLogger(name string) *CLogger {
	return defaultLogger.With("trace_id", name)
}
