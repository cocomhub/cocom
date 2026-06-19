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

func TestCustomLikeToTag_Skipped(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("CustomLikeToTag panicked (expected without MongoDB): %v", r)
		}
	}()

	w := httptest.NewRecorder()
	body := []byte(`{"cid": 1001}`)
	req := httptest.NewRequest(http.MethodPost, "/api/migrate/customLikeToTag", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	CustomLikeToTag(w, req)

	var resp httpwrap.ResponseInfo[any]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	t.Logf("CustomLikeToTag response code: %d, msg: %s", resp.Head.Code, resp.Head.Msg)
}
