// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		}),
	)
	slog.SetDefault(logger)
}

var initOnce sync.Once

// Init 用配置初始化全局默认 logger，多次调用幂等（仅首次生效）。
// 先构造 logger（CoW），再 SetDefault 写入全局，失败时返回错误。
// 调用方（root.go 的 initLogging）仍沿用现有 GetE 路径处理配置错误，
// 本函数仅在真正构造失败时返回错误兜底。
// 注意：sync.Once 在同一函数内无法重复触发自我检测且不可重入，因此
// 若 NewLogger 在并发下被同时调用，只有首个调用会真正执行，其余直接返回。
func Init(cfg Config) error {
	var initErr error
	initOnce.Do(func() {
		logger := NewLogger(cfg)
		if logger == nil {
			initErr = errors.New("logging: NewLogger returned nil logger")
			return
		}
		slog.SetDefault(logger)
	})
	return initErr
}

func NewLogger(config Config) *slog.Logger {
	var core zapcore.Core

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
		fileCore := newCore(config.FileLevel, config.FileEncoding, w)

		if core != nil {
			core = zapcore.NewTee(core, fileCore)
		} else {
			core = fileCore
		}
	}

	if config.EnableConsole {
		w := zapcore.Lock(os.Stderr)
		consoleCore := newCore(config.ConsoleLevel, config.ConsoleEncoding, w)

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
			zap.PanicLevel,
		)
	}

	fields := []zap.Field{}
	if config.EnablePID {
		fields = append(fields, zap.Int("LPID", os.Getpid()))
	}

	if config.EnableSourceIP {
		fields = append(fields, zap.String("LIP", GetIP(config.SourceEth)))
	}

	core = core.With(fields)

	l := zap.New(core)
	handler := zapslog.NewHandler(l.Core(), zapslog.WithName(config.AppName), zapslog.WithCaller(config.EnableCaller))
	return slog.New(handler)
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
	case "fatal", "panic":
		// slog 只有 Error 为最高级别；配置 fatal/panic 若直接映射会"全丢日志"
		// （zap 的 Fatal/Panic 触发 os.Exit/panic，slog 无对应语义），因此
		// 降级为 Error 并告警，避免用户配置了 fatal 但一条日志都看不到的陷阱。
		l = zap.ErrorLevel
		slog.Warn("logging: level fatal/panic not supported by slog, downgraded to error",
			"configured", level)
	default:
		// 无效级别静默回 Info 是刚接手时代码的既定行为；这里加告警帮助排查拼写/配置问题。
		l = zap.InfoLevel
		slog.Warn("logging: unknown log level, fallback to info",
			"configured", level)
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
		slog.Warn("logging: unknown encoding, fallback to console",
			"configured", encoding)
	}

	core = zapcore.NewCore(e, w, l)
	return
}

func GetIP(eth string) string {
	ifi, err := net.InterfaceByName(eth)
	if err != nil {
		slog.Error("GetIPError", "err", err)
		return ""
	}

	addrs, err := ifi.Addrs()
	if err != nil {
		slog.Error("GetIPError", "err", err)
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
