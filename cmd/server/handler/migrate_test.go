// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func TestCustomLikeToTag_Skipped(t *testing.T) {
	// CustomLikeToTag directly calls mongo.ComicInfoCustom(), which panics
	// without a running MongoDB. This test verifies the handler panics
	// with a MongoDB-related error rather than a real bug.
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("%v", r)
			if !strings.Contains(msg, "mongo") && !strings.Contains(msg, "MongoDB") {
				t.Errorf("expected mongo-related panic, got: %v", r)
			}
			t.Skipf("MongoDB not available, skipping: %v", r)
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
