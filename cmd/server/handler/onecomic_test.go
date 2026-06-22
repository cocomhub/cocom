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

func TestSaveOneComicInfo_InvalidBody(t *testing.T) {
	// POST nil body — JSON decode fails
	req := httptest.NewRequest(http.MethodPost, "/api/onecomic/saveComicInfo", nil)
	w := httptest.NewRecorder()
	SaveOneComicInfo(w, req)

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

func TestSaveOneComicInfo_MissingCIDAndComicID(t *testing.T) {
	// POST empty JSON {} — no "cid", no "comicid"
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/onecomic/saveComicInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveOneComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid/comicid, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveOneComicInfo_MissingSite(t *testing.T) {
	// POST {"comicid":"123"} — has comicid but no site
	body := []byte(`{"comicid":"123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/onecomic/saveComicInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveOneComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing site, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveOneComicInfo_ValidWithCID(t *testing.T) {
	body := []byte(`{"cid":"test123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/onecomic/saveComicInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveOneComicInfo(w, req)

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

func TestSaveOneComicInfo_ValidWithComicIDAndSite(t *testing.T) {
	body := []byte(`{"comicid":"123","site":"example"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/onecomic/saveComicInfo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveOneComicInfo(w, req)

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

func TestGetOneComicInfo_NoParams(t *testing.T) {
	// GET without cid/comicid/site
	req := httptest.NewRequest(http.MethodGet, "/api/onecomic/getComicInfo", nil)
	w := httptest.NewRecorder()
	GetOneComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for no params, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetOneComicInfo_MissingSite(t *testing.T) {
	// GET ?comicid=123 without site
	req := httptest.NewRequest(http.MethodGet, "/api/onecomic/getComicInfo?comicid=123", nil)
	w := httptest.NewRecorder()
	GetOneComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing site, got 0")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetOneComicInfo_Valid(t *testing.T) {
	// GET ?cid=test123 — test123 was saved in TestSaveOneComicInfo_ValidWithCID
	req := httptest.NewRequest(http.MethodGet, "/api/onecomic/getComicInfo?cid=test123", nil)
	w := httptest.NewRecorder()
	GetOneComicInfo(w, req)

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
