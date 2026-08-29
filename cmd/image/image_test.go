// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"errors"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
)

func TestImage_CommandDefined(t *testing.T) {
	if Cmd.Use == "" {
		t.Error("Cmd.Use should not be empty")
	} else {
		t.Logf("image command: %s - %s", Cmd.Use, Cmd.Short)
	}
}

// TestImage_WorkersRange 通过替换 imageFlag.workers 直接验证 processImage 的 workers 范围校验：
// 边界（1/max）、典型、越界（0/负数/超上限）均触发同一守卫分支，无需真实文件。
func TestImage_WorkersRange(t *testing.T) {
	orig := imageFlag
	t.Cleanup(func() { imageFlag = orig })

	tests := []struct {
		name    string
		workers int
		wantErr bool
	}{
		{name: "min boundary", workers: 1, wantErr: false},
		{name: "max boundary", workers: maxWorkers, wantErr: false},
		{name: "negative", workers: -3, wantErr: true},
		{name: "too small", workers: 0, wantErr: true},
		{name: "exceeds max", workers: maxWorkers + 1, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			imageFlag.workers = tc.workers
			err := processImage(t.Context(), []string{"single.png"}, "/tmp/out", "flip", nil, func(_ *imaging.ImageHandler) error { return nil })
			// workers 守卫优先于后续逻辑：越界必 back；合法时后续可能因源文件打开失败等返回 nil? 实际是文件不存在
			// 所以这里仅断言「越界时 error 且为 ErrInvalidArgs」「合法时 error 不是 ErrInvalidArgs」。
			var imgErr *errwrap.Error
			isInvalid := errors.As(err, &imgErr) && imgErr.Code() == errwrap.ErrInvalidArgs.Code()
			if tc.wantErr != isInvalid {
				t.Errorf("processImage workers=%d: err=%v, wantErr=%v", tc.workers, err, tc.wantErr)
			}
		})
	}
}

// TestImage_ErrorInvalidWorkers 校验 workers 越界时返回 ErrInvalidArgs。
func TestImage_ErrorInvalidWorkers(t *testing.T) {
	orig := imageFlag
	t.Cleanup(func() { imageFlag = orig })
	imageFlag.workers = 0 // 0 触发越界守卫
	err := processImage(t.Context(), []string{"a.png"}, "/tmp/out", "flip", nil, func(_ *imaging.ImageHandler) error { return nil })
	if err == nil {
		t.Fatal("expected error for workers=0")
	}
	var imgErr *errwrap.Error
	if errors.As(err, &imgErr) {
		if imgErr.Code() != errwrap.ErrInvalidArgs.Code() {
			t.Errorf("code = %d, want %d", imgErr.Code(), errwrap.ErrInvalidArgs.Code())
		}
	} else {
		t.Errorf("expected *errwrap.Error, got %T", err)
	}
}

// TestImage_HasSubcommands 验证 image 下的全部子命令已注册。
func TestImage_HasSubcommands(t *testing.T) {
	names := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		names[c.Name()] = true
	}
	expected := []string{"resize", "crop", "rotate", "adjust", "blur", "sharpen", "flip", "flop", "convert", "verify"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

// TestImage_FlagBindings 验证持久化旗标已绑定到 imageFlag 结构体字段。
func TestImage_FlagBindings(t *testing.T) {
	for _, name := range []string{"format", "workers", "batch", "output"} {
		if Cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("expected persistent flag --%s", name)
		}
	}
}

// TestProcessImage_NonBatchSingle 非批量模式仅接受单个 src：多个 src 报错、
// workers 合法且单文件时进入 processing（此处用不存在源文件触发 imaging 层报错，
// 命中非批量分支本身的参数语义）。
func TestProcessImage_NonBatchSingle(t *testing.T) {
	orig := imageFlag
	t.Cleanup(func() { imageFlag = orig })
	imageFlag = imageFlags{}
	imageFlag.workers = 4

	// 非批量 + 多 src 应在 processImage 内直接报错，不触达文件层。
	t.Run("multi srcs error", func(t *testing.T) {
		err := processImage(t.Context(), []string{"a.png", "b.png"}, "/tmp/out", "flip", nil, func(_ *imaging.ImageHandler) error {
			t.Error("processor should not be called")
			return nil
		})
		if err == nil {
			t.Fatal("expected error for multiple srcs in non-batch mode")
		}
		if !strings.Contains(err.Error(), "非批量模式只能处理单个文件") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// 非批量 + 0（或 nil）src：len(srcs)!=1 分支同样报错。
	t.Run("no srcs error", func(t *testing.T) {
		err := processImage(t.Context(), nil, "/tmp/out", "flip", nil, func(_ *imaging.ImageHandler) error {
			t.Error("processor should not be called")
			return nil
		})
		if err == nil {
			t.Fatal("expected error for no srcs in non-batch mode")
		}
	})
}
