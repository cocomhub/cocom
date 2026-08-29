// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cocomaarchiver

import (
	"context"
	"testing"
)

func TestCocomaarchiver_Options(t *testing.T) {
	opts := Options{
		ScanDir:     "/tmp/scan",
		ArchiveDir:  "/tmp/archive",
		NotMatchDir: "/tmp/notmatch",
		Limit:       10,
	}
	if opts.ScanDir != "/tmp/scan" {
		t.Errorf("ScanDir assignment failed")
	}
	if opts.Limit != 10 {
		t.Errorf("expected Limit=10, got %d", opts.Limit)
	}
}

func TestCocomaarchiver_RunOnce_Validation(t *testing.T) {
	ctx := context.Background()
	_, err := RunOnce(ctx, Options{})
	if err == nil {
		t.Error("expected error for empty Options")
	} else {
		t.Logf("RunOnce validation error: %v", err)
	}
}
