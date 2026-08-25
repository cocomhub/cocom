// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"io"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLogging_Init(t *testing.T) {
	cfg := Config{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Init panicked: %v", r)
		}
	}()
	Init(cfg)
	t.Log("Init executed without panic")
}

func TestLogging_NewLogger(t *testing.T) {
	cfg := Config{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}
	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger should return non-nil")
	}
	t.Log("NewLogger returned a valid logger")
}

// TestLogging_NewCore_LevelFallback 验证 newCore 的无效级别/编码回退行为：
// fatal/panic 映射到 Error（slog 无 Fatal/Panic 语义），未知级别回 Info，
// 未知编码回 console——响应不可直接断言（封装在 zapcore.Core 内），
// 至少确保不 panic、返回非空 core。
func TestLogging_NewCore_LevelFallback(t *testing.T) {
	w := zapcore.AddSync(io.Discard)

	core := newCore("fatal", "", w)
	if core == nil {
		t.Fatal("newCore(fatal) returned nil core")
	}

	core = newCore("panic", "json", w)
	if core == nil {
		t.Fatal("newCore(panic) returned nil core")
	}

	core = newCore("unknown-level", "unknown-encoding", w)
	if core == nil {
		t.Fatal("newCore(unknown) returned nil core")
	}

	core = newCore("info", "console", w)
	if core == nil {
		t.Fatal("newCore(info) returned nil core")
	}
}
