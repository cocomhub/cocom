// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ar

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
)

func TestArCommand_Registration(t *testing.T) {
	if Cmd == nil {
		t.Fatal("Cmd should not be nil")
	}
	if Cmd.Use != "ar" {
		t.Errorf("expected Use 'ar', got %s", Cmd.Use)
	}
}

func TestArCommand_HasPersistentFlags(t *testing.T) {
	f := Cmd.PersistentFlags().Lookup("cid")
	if f == nil {
		t.Fatal("expected --cid flag")
	}
	f = Cmd.PersistentFlags().Lookup("output")
	if f == nil {
		t.Fatal("expected --output flag")
	}
}

func TestArCommand_OutputModeDefault(t *testing.T) {
	if arOutput != "text" {
		t.Errorf("expected default output 'text', got %s", arOutput)
	}
}

func TestArCommand_HasSubcommands(t *testing.T) {
	cmds := Cmd.Commands()
	if len(cmds) == 0 {
		t.Fatal("expected at least one subcommand")
	}
	found := make(map[string]bool)
	for _, c := range cmds {
		found[c.Name()] = true
	}
	expected := []string{"pack", "unpack", "query", "backup", "check"}
	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

// TestArGetArchiveID 覆盖 archiveIDFromCmd 的 ID 解析逻辑：
// --id 与 --cid 一致校验、单独提供任一、都缺失报错、负值/零边界。
func TestArGetArchiveID(t *testing.T) {
	origCID := archiveCID
	t.Cleanup(func() { archiveCID = origCID })

	cases := []struct {
		name string
		id   int
		cid  int
		want int
		err  bool
	}{
		{name: "only id", id: 42, want: 42},
		{name: "only cid", cid: 7, want: 7},
		{name: "matched", id: 7, cid: 7, want: 7},
		{name: "mismatch", id: 1, cid: 2, err: true},
		{name: "zero zero", err: true},
		{name: "zero negative", id: -1, cid: 0, err: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archiveCID = tc.cid
			got, err := archiveIDFromCmd(tc.id)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error, got %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("archiveIDFromCmd(%d) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}

// TestGetSourceDir_CidZero 验证默认 GetSourceDir 实现的 cid==0 守卫分支。
func TestGetSourceDir_CidZero(t *testing.T) {
	if GetSourceDir == nil {
		t.Fatal("GetSourceDir should be assigned by init")
	}
	dir, err := GetSourceDir(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for cid 0")
	}
	if dir != "" {
		t.Errorf("expected empty dir, got %q", dir)
	}
}

// TestGetSourceDir_Override 验证函数级变量注入模式（仓库既有约定）。
func TestGetSourceDir_Override(t *testing.T) {
	orig := GetSourceDir
	t.Cleanup(func() { GetSourceDir = orig })

	GetSourceDir = func(ctx context.Context, cid int) (string, error) {
		return "/tmp/savedir", nil
	}
	if dir, err := GetSourceDir(context.Background(), 100); err != nil || dir != "/tmp/savedir" {
		t.Errorf("override not effective: dir=%q err=%v", dir, err)
	}

	GetSourceDir = func(ctx context.Context, cid int) (string, error) {
		return "", errors.New("boom")
	}
	if _, err := GetSourceDir(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected boom error, got %v", err)
	}
}

// TestArArchiveFilePath_WithMeta 通过替换全局管理器注入带版本号存档的 meta，
// 验证 pack（版本递增 {id}-v{N+1}）与非 pack（复用索引路径）两条主干。
func TestArArchiveFilePath_WithMeta(t *testing.T) {
	origMgr := manager.Get()
	t.Cleanup(func() { manager.Set(origMgr) })

	mem := manager.NewMemoryIndexStore()
	if err := mem.Create(t.Context(), &manager.ArchiveMeta{ID: 7, Path: "/data/x/7-v2.cocoma"}); err != nil {
		t.Fatalf("create meta: %v", err)
	}
	manager.Set(manager.NewWithIndex(mem, archive.TypeSingle))

	ctx := context.Background()

	check := func(path string, err error, want string) {
		t.Helper()
		if err != nil {
			t.Fatalf("archiveFilePath: %v", err)
		}
		// filepath.Join 在 Windows 下产反斜杠；用 filepath.Clean 归一化维度单独断言。
		if filepath.ToSlash(path) != want {
			t.Errorf("archiveFilePath = %q, want %q", filepath.ToSlash(path), want)
		}
	}
	t.Run("pack increments version", func(t *testing.T) {
		p, err := archiveFilePath(ctx, 7, true)
		check(p, err, "/data/x/7-v3.cocoma")
	})

	t.Run("unpack reuses recorded path", func(t *testing.T) {
		p, err := archiveFilePath(ctx, 7, false)
		check(p, err, "/data/x/7-v2.cocoma")
	})
}

// TestArArchiveFilePath_Missing 验证索引不存在时回退默认路径（{id}.cocoma 且在 root 下）。
func TestArArchiveFilePath_Missing(t *testing.T) {
	origMgr := manager.Get()
	t.Cleanup(func() { manager.Set(origMgr) })
	manager.Set(manager.NewWithIndex(manager.NewMemoryIndexStore(), archive.TypeSingle))

	p, err := archiveFilePath(context.Background(), 123, true)
	if err != nil {
		t.Fatalf("archiveFilePath(missing): %v", err)
	}
	if p == "" || !strings.HasSuffix(filepath.ToSlash(p), "/123.cocoma") {
		t.Errorf("unexpected default path: %q", p)
	}
}
