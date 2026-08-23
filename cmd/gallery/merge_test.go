// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gallery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"", []string{}},
		{"single", []string{"single"}},
		{"  spaced  ,  around  ", []string{"spaced", "around"}},
	}
	for _, tc := range tests {
		result := splitAndTrim(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("splitAndTrim(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

// TestCreateMergeLinks 验证 createMergeLinks 不会删除真实目录（数据保护），
// 且会正确替换已存在的软链。Windows 未开启开发者模式时 os.Symlink 可能失败，
// 此时仅断言真实目录保护分支。
func TestCreateMergeLinks(t *testing.T) {
	tmp := t.TempDir()
	mergeDir := filepath.Join(tmp, "merged")
	if err := os.MkdirAll(mergeDir, 0o755); err != nil {
		t.Fatalf("mkdir mergeDir: %v", err)
	}

	// 真实目录（含一个文件），必须被保护不删除
	realDir := filepath.Join(mergeDir, "real_dir")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir realDir: %v", err)
	}
	realFile := filepath.Join(realDir, "keep.txt")
	if err := os.WriteFile(realFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("write realFile: %v", err)
	}

	src := filepath.Join(tmp, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	// 软链（模拟上次生成），应被替换
	linkPath := filepath.Join(mergeDir, "link_dir")
	linkCreated := os.Symlink(filepath.Join(tmp, "old-target"), linkPath) == nil
	if !linkCreated {
		t.Logf("os.Symlink 不可用（如 Windows 未开开发者模式），跳过软链替换断言")
	}

	statsMap := map[string]*mergeStats{
		"real_dir": {Count: 1, LatestDirectory: &directoryInfo{Path: src, Volume: "v1"}},
		"link_dir": {Count: 1, LatestDirectory: &directoryInfo{Path: src, Volume: "v1"}},
	}
	createMergeLinks(statsMap, &mergeGalleryConfig{MergeDir: mergeDir, DryRun: false})

	// 真实目录未被删除，内容保留
	if _, err := os.Stat(realDir); err != nil {
		t.Errorf("真实目录不应被删除: %v", err)
	}
	if _, err := os.Stat(realFile); err != nil {
		t.Errorf("真实目录内文件不应被删除: %v", err)
	}

	if linkCreated {
		fi, err := os.Lstat(linkPath)
		if err != nil {
			t.Fatalf("软链应存在: %v", err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("link_dir 应仍是软链")
		}
		dest, err := os.Readlink(linkPath)
		if err != nil || dest != src {
			t.Errorf("软链目标 = %q, want %q (err=%v)", dest, src, err)
		}
	}
}
