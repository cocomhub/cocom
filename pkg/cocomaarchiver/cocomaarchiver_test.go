// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cocomaarchiver

import (
	"context"
	"os"
	"path/filepath"
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

// TestRunOnce_NotMatchIncrements 回归：进入 notmatch 分支的文件必须计入 Stats.NotMatch
// （此前 moveToNotMatch 只搬文件不计数，调度日志 notmatch 恒为 0）。
func TestRunOnce_NotMatchIncrements(t *testing.T) {
	ctx := context.Background()
	scanDir := t.TempDir()
	notMatchDir := filepath.Join(t.TempDir(), "notmatch")
	archiveDir := filepath.Join(t.TempDir(), "archive")

	// 文件名不匹配 CIDRegex → regex_no_match
	if err := os.WriteFile(filepath.Join(scanDir, "not-a-cid.cocoma"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// 匹配 CIDRegex 但查不到 md5 → md5_absent
	if err := os.WriteFile(filepath.Join(scanDir, "123.cocoma"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stats, err := RunOnce(ctx, Options{
		ScanDir:     scanDir,
		ArchiveDir:  archiveDir,
		NotMatchDir: notMatchDir,
		CIDRegex:    defaultCIDRegex(),
		LookupMD5:   func(context.Context, int) (string, error) { return "", nil },
	})
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if stats.Processed != 2 {
		t.Errorf("Processed = %d, want 2", stats.Processed)
	}
	if stats.NotMatch != 2 {
		t.Errorf("NotMatch = %d, want 2（notmatch 分支必须计数）", stats.NotMatch)
	}
	if stats.Archived != 0 {
		t.Errorf("Archived = %d, want 0", stats.Archived)
	}
}
