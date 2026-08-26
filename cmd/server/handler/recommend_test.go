// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
)

func TestGetRecommendations_InvalidCID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/comic/recommendations?cid=invalid", nil)
	GetRecommendations(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

func TestGetRecommendations_EmptyCID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/comic/recommendations?cid=", nil)
	GetRecommendations(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for empty cid, got 0")
	}
}

func TestGetRecommendations_Valid(t *testing.T) {
	// comic 1001 带 tag id=1（type=tag），comic 1002 也带 tag id=1，应能推荐出 1002
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/comic/recommendations?cid=1001&type=tag", nil)
	GetRecommendations(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	results, _ := resp.Body["results"].([]any)
	if len(results) == 0 {
		t.Error("results is empty, want at least one recommendation for cid=1001&type=tag")
	}
	// 精确成员断言：TestMain 种入的 1001/1002 共享 tag id=1(type=tag)→ 应推出 1002。
	// 候选池（除 1001 外含 tag id=1 且 type=tag 的漫画）只有 1002，limit=5 不截断，
	// 1002 必在其中——成员断言比“非空”更稳定（不受顺序影响）。
	found1002 := false
	for _, it := range results {
		if m, ok := it.(map[string]any); ok {
			if cid, _ := m["cid"].(float64); int(cid) == 1002 {
				found1002 = true
				break
			}
		}
	}
	if !found1002 {
		t.Errorf("expected comic 1002 in results, got: %v", resp.Body["results"])
	}
}

func TestGetRecommendations_InvalidType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/comic/recommendations?cid=1001&type=invalid", nil)
	GetRecommendations(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid type, got 0")
	}
}
