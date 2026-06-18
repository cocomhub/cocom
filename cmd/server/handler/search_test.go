// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mongowrap"
)

func init() {
	// 不在这里调用 cache.Init() — handler_test.go 的 TestMain 已经用 defer/recover 处理了
	// 这里只检查 MongoDB 可用性
	if err := mongowrap.Init(config.Get().Mongo); err != nil {
		slog.Warn("MongoDB not available, MongoDB-dependent tests will be skipped")
	} else {
		testMongoAvailable = true
	}
}

var testMongoAvailable bool

func TestSearchAutocomplete_EmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if resp.Head.Code != -1 {
		t.Errorf("expected code -1 (error), got %d", resp.Head.Code)
	}
}

func TestSearchAutocomplete_ShortQuery(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	// 单字符查询应返回空结果（实际是 API 仍会处理，但前端限制 2 字符）
	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=a", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.AutocompleteResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// API 不限制短查询，返回的结果可能是空的
	if resp.Body.Comics == nil {
		t.Error("expected comics field (may be empty), got nil")
	}
	if resp.Body.Tags == nil {
		t.Error("expected tags field (may be empty), got nil")
	}
}

func TestSearchAutocomplete_ResponseStructure(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/search/autocomplete?q=test&limit=3", nil)
	w := httptest.NewRecorder()
	SearchAutocomplete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.AutocompleteResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}

	// 验证类型
	if resp.Body.Comics == nil {
		t.Error("comics should be non-nil array")
	}
	if resp.Body.Tags == nil {
		t.Error("tags should be non-nil array")
	}

	// 验证 Total <= limit (3)
	if len(resp.Body.Tags) > 3 {
		t.Errorf("expected at most 3 tags, got %d", len(resp.Body.Tags))
	}
	if len(resp.Body.Comics) > 3 {
		t.Errorf("expected at most 3 comics, got %d", len(resp.Body.Comics))
	}
}
