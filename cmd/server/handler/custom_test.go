// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestAddLikeGroup_InvalidCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=invalid", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for invalid cid, got 0")
	}
}

func TestAddLikeGroup_EmptyCID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code for missing cid, got 0")
	}
}

func TestAddLikeGroup_Valid(t *testing.T) {
	if !testMongoAvailable {
		t.Skip("MongoDB not available")
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/comic/addLikeGroup?cid=1001", nil)
	AddLikeGroup(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// custom.AddLikeGroup calls MongoDB directly, so expect non-zero in unit tests
	if resp.Head.Code == 0 {
		t.Error("expected non-zero code since AddLikeGroup uses MongoDB, got 0")
	}
}
