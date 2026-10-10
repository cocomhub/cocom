// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestCopyDir_CopiesNestedTree(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src")
	writeFile(t, filepath.Join(src, "a.txt"), "A")
	writeFile(t, filepath.Join(src, "sub", "b.txt"), "B")

	dst := filepath.Join(t.TempDir(), "dst")
	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "A" {
		t.Errorf("a.txt = %q, want A", got)
	}
	if got := readFile(t, filepath.Join(dst, "sub", "b.txt")); got != "B" {
		t.Errorf("sub/b.txt = %q, want B", got)
	}

	info, err := os.Stat(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatalf("stat dst a.txt: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640（应保留权限位）", info.Mode().Perm())
	}
}

func TestCopyDir_Errors(t *testing.T) {
	t.Parallel()

	t.Run("源不存在", func(t *testing.T) {
		if err := CopyDir(filepath.Join(t.TempDir(), "nope"), t.TempDir()); err == nil {
			t.Fatal("源不存在时必须报错")
		}
	})

	t.Run("源不是目录", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "f.txt")
		writeFile(t, f, "x")
		if err := CopyDir(f, filepath.Join(t.TempDir(), "dst")); err == nil {
			t.Fatal("源为文件时必须报错")
		}
	})
}

func TestMove_File(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	writeFile(t, src, "hello")

	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("源文件应已不存在, err=%v", err)
	}
	if got := readFile(t, dst); got != "hello" {
		t.Errorf("dst 内容 = %q, want hello", got)
	}
}

func TestMove_Dir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "srcdir")
	writeFile(t, filepath.Join(src, "x", "y.txt"), "Y")

	dst := filepath.Join(dir, "dstdir")
	if err := Move(src, dst); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("源目录应已不存在, err=%v", err)
	}
	if got := readFile(t, filepath.Join(dst, "x", "y.txt")); got != "Y" {
		t.Errorf("dst 内容 = %q, want Y", got)
	}
}

func TestMove_MissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Move(filepath.Join(dir, "nope.txt"), filepath.Join(dir, "dst.txt")); err == nil {
		t.Fatal("源不存在时必须报错")
	}
}
