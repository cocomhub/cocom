// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDocGen_ExtractPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"server.listen.http.addr", "server.*"},
		{"cocom.storage.path", "cocom.*"},
		{"single", "single.*"},
		{"", ".*"},
	}
	for _, tt := range tests {
		if got := extractPrefix(tt.in); got != tt.want {
			t.Errorf("extractPrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestConfigDocGen_GetOrCreateEntry(t *testing.T) {
	before := len(entries)

	e := getOrCreateEntry("test.doc.key")
	if e == nil {
		t.Fatal("getOrCreateEntry returned nil")
	}
	if e.Key != "test.doc.key" {
		t.Errorf("entry key = %q, want test.doc.key", e.Key)
	}
	if len(entries) != before+1 {
		t.Errorf("entries grew to %d, want %d", len(entries), before+1)
	}

	// 幂等：同一 key 返回同一指针，不重复创建
	e2 := getOrCreateEntry("test.doc.key")
	if e2 != e {
		t.Error("getOrCreateEntry should return the same pointer for the same key")
	}
}

func TestConfigDocGen_Generate_WriteError(t *testing.T) {
	// output 指向一个已存在目录 → WriteFile 必然失败（EISDIR），
	// 验证 generate 返回错误而不是静默吞掉。
	tdir := t.TempDir()
	if err := generate(tdir); err == nil {
		t.Fatal("generate on a directory path should return error")
	}
}

func TestConfigDocGen_Generate_WriteSuccess(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sub", "config.md")
	if err := generate(out); err != nil {
		t.Fatalf("generate(%q) should succeed: %v", out, err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}
