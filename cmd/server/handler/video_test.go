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
	// GET ?id=vid123 — vid123 was saved in TestSaveVideoInfo_Valid
	req := httptest.NewRequest(http.MethodGet, "/api/video/getVideoInfo?id=vid123", nil)
	w := httptest.NewRecorder()
	GetVideoInfo(w, req)

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
