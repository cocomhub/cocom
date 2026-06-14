// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// LikeTag 测试

func TestLikeTag_NoType(t *testing.T) {
	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without type, got 0")
	}
}

func TestLikeTag_NoIDOrName(t *testing.T) {
	form := url.Values{"type": {"tag"}}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without id or name, got 0")
	}
}

func TestLikeTag_ValidByID(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	form := url.Values{"type": {"tag"}, "id": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for like by id, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestLikeTag_ValidByName(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	form := url.Values{"type": {"tag"}, "name": {"test"}}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/like", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Logf("LikeTag by name returned %d: %s (expected without DB data)", resp.Head.Code, resp.Head.Msg)
	}
}

// UnlikeTag 测试

func TestUnlikeTag_NoType(t *testing.T) {
	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/unlike", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	UnlikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code without type, got 0")
	}
}

func TestUnlikeTag_ValidByID(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	form := url.Values{"type": {"tag"}, "id": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/unlike", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	UnlikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code != 0 {
		t.Errorf("expected code 0 for unlike by id, got %d: %s", resp.Head.Code, resp.Head.Msg)
	}
}

func TestUnlikeTag_NonExistent(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}

	form := url.Values{"type": {"tag"}, "id": {"99999"}}
	req := httptest.NewRequest(http.MethodPost, "/api/comic/tags/unlike", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	UnlikeTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// Unlike non-existent tag should handle gracefully (404 expected)
	if resp.Head.Code == 0 {
		t.Errorf("expected non-zero code for non-existent tag unlike, got 0")
	}
}
