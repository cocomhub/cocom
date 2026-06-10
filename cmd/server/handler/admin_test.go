// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestLinkComics_BatchSubCIDs(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	body := map[string]any{
		"main_cid": 1001,
		"sub_cids": []int{2001, 2002, 2003},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/comic/link", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 不 panic，返回合理响应（即使 comic 不存在，也返回 code 0 并附带 errors）
	LinkComics(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// 批量链接即使有失败也不应 panic，返回 code 0 表示请求处理成功
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for batch request handling, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	// 检查响应中有 sub_cids 字段
	if _, ok := resp.Body["sub_cids"]; !ok {
		t.Error("response should contain sub_cids field")
	}
	// 检查 errors 字段存在
	if _, ok := resp.Body["errors"]; !ok {
		t.Error("response should contain errors field (may be empty)")
	}
}

func TestLinkComics_EmptySubCIDs(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	body := map[string]any{
		"main_cid": 1001,
		"sub_cids": []int{},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/comic/link", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LinkComics(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for empty sub_cids, got 0")
	}
}

func TestDeleteComic_InvalidCID(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	body := map[string]any{"cid": 0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/comic/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

func TestDeleteComic_NonExistent(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	body := map[string]any{"cid": 99999}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/comic/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// 不存在应返回非零 code，但不 panic
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for non-existent comic, got 0")
	}
}

func TestLinkComics_BackwardCompatible(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	// 使用旧版 sub_cid 单字段，应该仍然工作
	body := map[string]any{
		"main_cid": 1001,
		"sub_cid":  2001,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/comic/link", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LinkComics(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for backward compat, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
