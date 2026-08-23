// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build memory_storage_integration

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestGetSetting_NoType(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/setting/get?type=", nil)
	GetSetting(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for empty type get, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestSetSetting_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/setting/set", nil)
	req.Header.Set("Content-Type", "application/json")
	SetSetting(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestSetSetting_Valid(t *testing.T) {
	body := map[string]any{
		"type":     "test",
		"settings": map[string]any{"key1": "val1", "key2": float64(42)},
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/setting/set", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SetSetting(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestDelSetting_EmptyKeys(t *testing.T) {
	// 空 keys 应返回 400，避免"无 keys 误删整个 type"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/setting/del?type=", nil)
	DelSetting(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for empty keys delete, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDelSetting_WithKeys(t *testing.T) {
	// Set 两个 key，再删除其中一个，验证剩余
	setBody := map[string]any{
		"type":     "del_case",
		"settings": map[string]any{"a": 1, "b": 2},
	}
	b, _ := json.Marshal(setBody)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/setting/set", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SetSetting(w, req)
	var setResp httpwrap.ResponseInfo[any]
	_ = json.NewDecoder(w.Body).Decode(&setResp)
	if setResp.Head.Code != 0 {
		t.Fatalf("set failed: %d: %s", setResp.Head.Code, setResp.Head.Msg)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/setting/del?type=del_case&keys=a", nil)
	DelSetting(w, req)
	var delResp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&delResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if delResp.Head.Code != 0 {
		t.Errorf("expected code 0 for delete with keys, got %d: %s", delResp.Head.Code, delResp.Head.Msg)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/setting/get?type=del_case&keys=a,b", nil)
	GetSetting(w, req)
	var getResp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if _, ok := getResp.Body["a"]; ok {
		t.Error("key a should have been deleted")
	}
	if _, ok := getResp.Body["b"]; !ok {
		t.Error("key b should remain")
	}
}
