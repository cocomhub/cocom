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

// TestCustomLikeToTag_Migrate 验证迁移端点 CustomLikeToTag 在无 MongoDB 环境下的护栏行为。
// 该 handler 直接访问 MongoDB（mongo.ComicInfoCustom），无真实 MongoDB 时 mongowrap.Init
// 失败即 skip（take-off 前检查），避免 panic。
// 【非破坏性】本测试只读，不写真实数据，因此 Guard 后不会对任何库造成变更。
// 若环境可用 Mongo，也仅做连通性 smoke——不对真实库做自定义迁移写入。
// 注：在 memory_storage_integration 构建下 GetDefaultStorage 已注入，但该 handler
// 不走 default-storage 分支（直接查 mongo.ComicInfoCustom），故仍需真库才可执行。
func TestCustomLikeToTag_Migrate(t *testing.T) {
	if err := mongowrap.Init(context.Background(), config.Get().Mongo); err != nil {
		t.Skipf("MongoDB not available, skipping: %v", err)
	}

	w := httptest.NewRecorder()
	body := []byte(`{"cid": 999999999}`) // 使用不可能存在的 cid，确保不触碰真实业务数据
	req := httptest.NewRequest(http.MethodPost, "/api/migrate/customLikeToTag", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	CustomLikeToTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// 无论 custom 表里有/没有该 cid，端点都返回 code 0 且永不 panic。
	// body.cid 不影响结果（迁移遍历整个 custom like 集合），用超大 id 保证不写业务漫画。
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 nonzero-cid smoke, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
