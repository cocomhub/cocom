// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/mongowrap"
)

// skipIfNoMongo 在 MongoDB 不可用时跳过测试，避免 handler.Init 调用后的 mongowrap panic。
// 所有调用 BuildEngine 的测试应优先使用本函数。
func skipIfNoMongo(t *testing.T) {
	t.Helper()
	if err := mongowrap.Init(); err != nil {
		t.Skipf("MongoDB not available, skipping test: %v", err)
	}
}
