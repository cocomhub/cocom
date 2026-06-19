// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestSearchTags_EmptyQuery(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=tag&q=", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (empty query returns top tags), got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if resp.Body.Tags == nil {
		t.Error("expected tags field, got nil")
	}
}

func TestSearchTags_DefaultTypeAndLimit(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?q=love", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// 默认 limit=20
	if len(resp.Body.Tags) > 20 {
		t.Errorf("expected at most 20 tags (default limit), got %d", len(resp.Body.Tags))
	}
	if resp.Body.Total != len(resp.Body.Tags) {
		t.Errorf("expected total=%d == len(tags)=%d", resp.Body.Total, len(resp.Body.Tags))
	}
}

func TestSearchTags_WithLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=artist&q=a&limit=5", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	if len(resp.Body.Tags) > 5 {
		t.Errorf("expected at most 5 tags, got %d", len(resp.Body.Tags))
	}
}

func TestSearchTags_ExceedsMaxLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?q=test&limit=500", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}
	// 上限 100
	if len(resp.Body.Tags) > 100 {
		t.Errorf("expected at most 100 tags (max limit), got %d", len(resp.Body.Tags))
	}
}

func TestSearchTags_TagInfoStructure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=tag&q=love&limit=3", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %s", err)
	}

	for _, tag := range resp.Body.Tags {
		if tag.Name == "" {
			t.Error("tag name should not be empty")
		}
		if tag.Type == "" {
			t.Error("tag type should not be empty")
		}
	}
}
