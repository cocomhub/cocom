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

func TestSaveVideoInfo_InvalidBody(t *testing.T) {
	// POST nil body — JSON decode fails
	req := httptest.NewRequest(http.MethodPost, "/api/video/saveVideoInfo", nil)
	w := httptest.NewRecorder()
	SaveVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid body, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveVideoInfo_MissingID(t *testing.T) {
	// POST empty JSON {} — no "id" field
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/video/saveVideoInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing id, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveVideoInfo_Valid(t *testing.T) {
	// POST {"id":"vid123"} — validation passes, memory store handles save
	body := []byte(`{"id":"vid123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/video/saveVideoInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetVideoInfo_NoID(t *testing.T) {
	// GET without id param
	req := httptest.NewRequest(http.MethodGet, "/api/video/getVideoInfo", nil)
	w := httptest.NewRecorder()
	GetVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for no id, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetVideoInfo_Valid(t *testing.T) {
	// 自包含：先保存再读取，避免跨用例依赖 TestSaveVideoInfo_Valid
	saveBody := []byte(`{"id":"get_case_vid"}`)
	saveReq := httptest.NewRequest(http.MethodPost, "/api/video/saveVideoInfo", bytes.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveW := httptest.NewRecorder()
	SaveVideoInfo(saveW, saveReq)
	if saveW.Code != http.StatusOK {
		t.Fatalf("save failed, http %d", saveW.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/video/getVideoInfo?id=get_case_vid", nil)
	w := httptest.NewRecorder()
	GetVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// 自包含保存后读取，应能拿到刚写入的视频文档
	if vid, _ := resp.Body["vid"].(string); vid != "get_case_vid" {
		t.Errorf("body.vid = %q, want get_case_vid (body: %v)", vid, resp.Body)
	}
}

func TestGetVideoInfo_NotFound(t *testing.T) {
	// 不存在的 vid 应返回 404（ErrVideoNotFound 哨兵）
	req := httptest.NewRequest(http.MethodGet, "/api/video/getVideoInfo?id=not_exist_vid", nil)
	w := httptest.NewRecorder()
	GetVideoInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing video, got 0")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
