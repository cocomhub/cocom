// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"io"
	"log/slog"
	"net"
	"os"
	"strings"

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

func Init() {
	cfg := GetConfigByViper()
	slog.SetDefault(NewLogger(cfg))
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
			zap.PanicLevel)
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
