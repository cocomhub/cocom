// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cocomhub/cocom/internal/archivecli"
)

func TestOutputModePrefersFlag(t *testing.T) {
	flagOutput = "json"
	t.Cleanup(func() {
		flagOutput = ""
	})
	if got := outputMode(); got != "json" {
		t.Fatalf("unexpected output mode: %s", got)
	}
}

func TestEmitErrorJSON(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	archivecli.EmitError(&stderr, &stdout, "json", fmt.Errorf("boom"))

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json failed: %v", err)
	}
	if payload["ok"] != false || payload["error"] != "boom" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %s", stderr.String())
	}
}
