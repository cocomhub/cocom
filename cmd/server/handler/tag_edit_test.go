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

// tagKey 纯函数测试

func TestTagKey_ByID(t *testing.T) {
	result := tagKey("tag", 123, "")
	expected := "tag:123"
	if result != expected {
		t.Errorf("tagKey('tag', 123, '') = %q, want %q", result, expected)
	}
}

func TestTagKey_ByName(t *testing.T) {
	result := tagKey("tag", 0, "test")
	expected := "tag:test"
	if result != expected {
		t.Errorf("tagKey('tag', 0, 'test') = %q, want %q", result, expected)
	}
}

// UpdateComicTags 测试

func TestUpdateComicTags_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateComicTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for invalid body, got 0")
	}
}

func TestUpdateComicTags_ZeroCID(t *testing.T) {
	body := map[string]any{"cid": 0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateComicTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for zero CID, got 0")
	}
}

func TestUpdateComicTags_NoAddedOrRemoved(t *testing.T) {

	body := map[string]any{"cid": 1001}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateComicTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code when no added or removed, got 0")
	}
}

func TestUpdateComicTags_AddTag(t *testing.T) {
	body := map[string]any{
		"cid":   1001,
		"added": []map[string]any{{"id": 10, "name": "newtag", "type": "tag"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateComicTags(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for add tag, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestUpdateComicTags_RemoveTag(t *testing.T) {
	body := map[string]any{
		"cid":     1001,
		"removed": []map[string]any{{"id": 1, "name": "test", "type": "tag"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateComicTags(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for remove tag, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// GetSearchUniqueTags 测试

func TestGetSearchUniqueTags_NoQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search-unique", nil)
	w := httptest.NewRecorder()

	GetSearchUniqueTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without query param, got 0")
	}
}

func TestGetSearchUniqueTags_Valid(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search-unique?q=test", nil)
	w := httptest.NewRecorder()

	GetSearchUniqueTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for search unique tags, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// GetRelatedTags 测试

func TestGetRelatedTags_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/related", nil)
	w := httptest.NewRecorder()

	GetRelatedTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without type/name params, got 0")
	}
}

func TestGetRelatedTags_Valid(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/related?type=tag&name=test", nil)
	w := httptest.NewRecorder()

	GetRelatedTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for related tags, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// BatchAddTagToComics 测试

func TestBatchAddTagToComics_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/batch-add", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	BatchAddTagToComics(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for nil body, got 0")
	}
}

func TestBatchAddTagToComics_EmptyCIDList(t *testing.T) {
	body := map[string]any{
		"cidList": []int{},
		"tag":     map[string]any{"id": 99, "name": "batch", "type": "tag"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/batch-add", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	BatchAddTagToComics(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for empty CID list, got 0")
	}
}

func TestBatchAddTagToComics_Valid(t *testing.T) {

	body := map[string]any{
		"cidList": []int{1001},
		"tag":     map[string]any{"id": 99, "name": "batch", "type": "tag"},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/batch-add", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	BatchAddTagToComics(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for batch add, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}
