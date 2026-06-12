// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/pkg/mongowrap"
)

var testMongoAvailable bool

func TestMain(m *testing.M) {
	// cache.Init 可能会 panic（例如配置未加载时分片大小为 0）
	// 使用 recover 防止整个测试二进制崩溃
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("cache init panicked, continuing without cache", "recover", r)
			}
		}()
		cache.Init(context.Background())
	}()

	if err := mongowrap.Init(); err != nil {
		slog.Warn("MongoDB not available, MongoDB-dependent tests will be skipped")
	} else {
		testMongoAvailable = true
	}

	os.Exit(m.Run())
}
