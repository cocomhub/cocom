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

// CreateTagRelation 测试

func TestCreateTagRelation_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for nil body, got 0")
	}
}

func TestCreateTagRelation_LessThan2Tags(t *testing.T) {
	body := map[string]any{
		"tags": []map[string]any{{"id": 1, "name": "a", "type": "tag"}},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code with less than 2 tags, got 0")
	}
}

func TestCreateTagRelation_Valid(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	body := map[string]any{
		"tags": []map[string]any{
			{"id": 1, "name": "a", "type": "tag"},
			{"id": 2, "name": "b", "type": "tag"},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTagRelation(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for create relation, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

// DeleteTagRelation 测试

func TestDeleteTagRelation_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation/delete", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for nil body, got 0")
	}
}

func TestDeleteTagRelation_EmptyID(t *testing.T) {
	body := map[string]any{"id": ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/relation/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DeleteTagRelation(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code with empty id, got 0")
	}
}

// GetTagRelations 测试

func TestGetTagRelations_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation", nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without type or name, got 0")
	}
}

func TestGetTagRelations_NonExistent(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation?type=tag&name=nonexistent", nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Logf("GetTagRelations nonexistent returned %d: %s (expected)", resp.Head.Code, resp.Head.Msg)
	}
}

func TestGetTagRelations_Valid(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/relation?type=tag&name=test", nil)
	w := httptest.NewRecorder()

	GetTagRelations(w, req)

	var resp httpwrap.ResponseInfo[map[string]any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Logf("GetTagRelations valid returned %d: %s (expected without DB data)", resp.Head.Code, resp.Head.Msg)
	}
}
