// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestResetCache(t *testing.T) {
	// POST empty body：TestMain 已初始化缓存，应返回成功 code 0（而非“未初始化”错误/panic）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache/reset", nil)
	ResetCache(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 (cache initialized in TestMain), got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// TestResetCache_NoPanic 与 TestResetCache 重复（后者已含 decode + code==0 断言，强于不 panic），
// 已删除。
