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

func TestAggregateTags_ReturnsOK(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/aggregate", nil)
	w := httptest.NewRecorder()
	AggregateTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("AggregateTags response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_DefaultType(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(default) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_WithType(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags?type=artist", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(artist) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_WithSortByName(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags?type=tag&sort=name", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(sort=name) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_WithSortByPopular(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags?type=tag&sort=popular", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(sort=popular) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_WithPage(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags?type=tag&page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(page) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestGetTags_WithLikedOnly(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags?type=tag&likedOnly=true", nil)
	w := httptest.NewRecorder()
	GetTags(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("GetTags(likedOnly) response code: %d, http: %d, msg: %s", resp.Head.Code, w.Code, resp.Head.Msg)
}

func TestSearchTags_DefaultType(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?q=test", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("SearchTags(default) response code: %d, total: %d", resp.Head.Code, resp.Body.Total)
}

func TestSearchTags_WithType(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=artist&q=a", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("SearchTags(artist) response code: %d, total: %d, tags: %d", resp.Head.Code, resp.Body.Total, len(resp.Body.Tags))
}

func TestSearchTags_LimitCap(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/comic/tags/search?type=tag&q=a&limit=200", nil)
	w := httptest.NewRecorder()
	SearchTags(w, req)

	var resp httpwrap.ResponseInfo[api.TagSearchResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 && len(resp.Body.Tags) > 100 {
		t.Errorf("expected at most 100 tags (capped limit), got %d", len(resp.Body.Tags))
	}
	t.Logf("SearchTags(limit=200) response code: %d, tags: %d, total: %d", resp.Head.Code, len(resp.Body.Tags), resp.Body.Total)
}
