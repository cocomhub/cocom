// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mongowrap"
)

// TestCustomLikeToTag_CustomLikeToTag 验证迁移端点 CustomLikeToTag 的行为。
// 该 handler 直接访问 MongoDB（mongo.ComicInfoCustom），在无真实 MongoDB 环境下
// 会 panic；因此在调用前先检查连接可用性并 skip，而不是 panic 后再跳过。
func TestCustomLikeToTag_CustomLikeToTag(t *testing.T) {
	if err := mongowrap.Init(context.Background(), config.Get().Mongo); err != nil {
		t.Skipf("MongoDB not available, skipping: %v", err)
	}

	w := httptest.NewRecorder()
	body := []byte(`{"cid": 1001}`)
	req := httptest.NewRequest(http.MethodPost, "/api/migrate/customLikeToTag", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	CustomLikeToTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// 无 custom like 数据或迁移成功时均返回 code 0
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
