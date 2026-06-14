// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestSaveComicInfo_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", nil)
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestSaveComicInfo_MissingCID(t *testing.T) {
	body := map[string]any{"title": "test"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestSaveComicInfo_Valid(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	body := map[string]any{"cid": 1001, "title": "test title"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/saveComicInfo", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	SaveComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code from memory storage, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestGetComicInfo_InvalidCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/comic/getComicInfo", nil)
	GetComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestGetComicInfo_Valid(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/comic/getComicInfo?cid=1001", nil)
	GetComicInfo(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// cid=1001 exists in testMemStorage (injected in TestMain), so expect success
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for existing comic, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestDownloadComic_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/download", nil)
	req.Header.Set("Content-Type", "application/json")
	DownloadComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestDownloadComic_Async(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	body := api.DownloadComicByIDRequest{Cid: 1001, IsSync: false}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/download", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	DownloadComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 1000 {
		t.Errorf("expected code 1000 for async download, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestRestoreComic_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/restore", nil)
	req.Header.Set("Content-Type", "application/json")
	RestoreComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for nil body, got 0")
	}
}

func TestRestoreComic_Async(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	body := api.RestoreComicByIDRequest{Cid: 1001, IsSync: false}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/restore", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	RestoreComic(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 1000 {
		t.Errorf("expected code 1000 for async restore, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
