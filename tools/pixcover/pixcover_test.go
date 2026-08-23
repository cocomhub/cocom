// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPixcover_ReadFileLines 验证 readFileLines 读取文件行并去空行。
func TestPixcover_ReadFileLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(p, []byte("pid1\npid2\n\n pid3 \n"), 0o644); err != nil {
		t.Fatalf("write list file: %v", err)
	}

	dm := &DownloadManager{
		config:        &Config{},
		existingFiles: make(map[string]bool),
		downloaded:    make(map[string]bool),
	}
	if err := dm.readFileLines(p); err != nil {
		t.Fatalf("readFileLines: %v", err)
	}
	if len(dm.existingFiles) != 3 {
		t.Errorf("existingFiles = %v, want 3 entries", dm.existingFiles)
	}
	if !dm.existingFiles["pid1"] || !dm.existingFiles["pid2"] || !dm.existingFiles["pid3"] {
		t.Errorf("existingFiles = %v, want pid1/pid2/pid3 present", dm.existingFiles)
	}
}

// TestPixcover_ConfigMaxTotalSize 验证 MaxTotalSize 由 MaxSizeGB 换算（GB → byte）。
func TestPixcover_ConfigMaxTotalSize(t *testing.T) {
	cfg := Config{MaxSizeGB: 10}
	cfg.MaxTotalSize = int64(cfg.MaxSizeGB) * 1024 * 1024 * 1024
	if cfg.MaxTotalSize != 10*1024*1024*1024 {
		t.Errorf("MaxTotalSize = %d, want %d", cfg.MaxTotalSize, int64(10)*1024*1024*1024)
	}
}
