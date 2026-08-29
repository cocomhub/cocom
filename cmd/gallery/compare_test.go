// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gallery

import (
	"testing"
)

func TestExtractCID_ValidDir(t *testing.T) {
	cid, err := extractCID("[123456] Test Title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cid != "123456" {
		t.Errorf("expected 123456, got %s", cid)
	}
}

func TestExtractCID_InvalidDir(t *testing.T) {
	_, err := extractCID("Invalid Title Without Brackets")
	if err == nil {
		t.Error("expected error for invalid directory name")
	}
}

func TestExtractCID_EmptyDir(t *testing.T) {
	_, err := extractCID("")
	if err == nil {
		t.Error("expected error for empty directory name")
	}
}

func TestCountByStatus(t *testing.T) {
	diffs := []fileDiff{
		{Status: "localMissing"},
		{Status: "different"},
		{Status: "remoteMissing"},
		{Status: "localMissing"},
	}
	if n := countByStatus(diffs, "localMissing"); n != 2 {
		t.Errorf("expected 2 localMissing, got %d", n)
	}
	if n := countByStatus(diffs, "different"); n != 1 {
		t.Errorf("expected 1 different, got %d", n)
	}
	if n := countByStatus(diffs, "remoteMissing"); n != 1 {
		t.Errorf("expected 1 remoteMissing, got %d", n)
	}
	if n := countByStatus(diffs, "nonexistent"); n != 0 {
		t.Errorf("expected 0 nonexistent, got %d", n)
	}
}
