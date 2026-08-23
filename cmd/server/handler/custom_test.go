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

func TestAddLikeGroup_InvalidCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=invalid", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

func TestAddLikeGroup_EmptyCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestAddLikeGroup_Valid(t *testing.T) {
	// 已通过 TestMain 注入 CustomStore，应该成功
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=1001", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 with memstore, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestAddLikeGroup_ZeroCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=0", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for cid=0, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddLikeGroup_NegativeCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=-5", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for negative cid, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddLikeGroup_WritesLike(t *testing.T) {
	// 验证点赞确实写入内存 store（R11-F7）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=7777", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Fatalf("add like group failed: %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	if !testCustomStore.IsLiked(7777) {
		t.Error("IsLiked(7777) = false, want true after AddLikeGroup")
	}
}
