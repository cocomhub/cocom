// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archivecli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestArchiveCli_EmitError_Text(t *testing.T) {
	var text bytes.Buffer
	var jsonBuf bytes.Buffer
	EmitError(&text, &jsonBuf, "text", errors.New("boom"))

	if !strings.Contains(text.String(), "boom") {
		t.Errorf("text output should contain error message, got %q", text.String())
	}
	if jsonBuf.Len() != 0 {
		t.Errorf("json writer should be empty for text mode, got %q", jsonBuf.String())
	}
}

func TestArchiveCli_EmitError_JSON(t *testing.T) {
	var text bytes.Buffer
	var jsonBuf bytes.Buffer
	EmitError(&text, &jsonBuf, "json", errors.New("boom"))

	var payload map[string]any
	if err := json.Unmarshal(jsonBuf.Bytes(), &payload); err != nil {
		t.Fatalf("decode json failed: %v", err)
	}
	if payload["ok"] != false || payload["error"] != "boom" {
		t.Errorf("unexpected payload: %+v", payload)
	}
	if text.Len() != 0 {
		t.Errorf("text writer should be empty for json mode, got %q", text.String())
	}
}

func TestArchiveCli_NormalizeMode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"json", "json"},
		{"JSON", "json"},
		{" json ", "json"},
		{"text", "text"},
		{"", "text"},
		{"xml", "text"},
	}
	for _, tt := range tests {
		if got := normalizeMode(tt.in); got != tt.want {
			t.Errorf("normalizeMode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
