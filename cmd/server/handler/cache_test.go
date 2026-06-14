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

func TestResetCache(t *testing.T) {
	// POST empty body, should not panic (code 0 or non-zero both acceptable)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache/reset", nil)
	ResetCache(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	// Any response is acceptable — the test just ensures no panic
	_ = resp
}

func TestResetCache_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ResetCache panicked: %v", r)
		}
	}()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/cache/reset", nil)
	ResetCache(w, req)
}
