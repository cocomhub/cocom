// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/imaging"
)

// TestVerifyCmd_WorkersValidation 回归：`cocom image verify --workers 0` 此前绕过
// processImage 的并发数校验，BatchProcessor 不启动任何工作协程 → 任务静默丢弃并退出码 0。
func TestVerifyCmd_WorkersValidation(t *testing.T) {
	old := imageFlag.workers
	t.Cleanup(func() { imageFlag.workers = old })

	for _, bad := range []int{0, -1, maxWorkers + 1} {
		imageFlag.workers = bad
		if err := validateWorkers(); err == nil {
			t.Errorf("workers=%d 必须被拒绝", bad)
		}
	}

	imageFlag.workers = 4
	if err := validateWorkers(); err != nil {
		t.Errorf("workers=4 应通过，got %v", err)
	}
}

// TestProcessBatch_ZeroWorkersReturnsError 保证任何调用方（含 API）都不会出现「0 并发假成功」。
func TestProcessBatch_ZeroWorkersReturnsError(t *testing.T) {
	err := imaging.ProcessBatch(context.Background(), []string{"*.go"}, &imaging.BatchOptions{
		Workers: 0,
		Op:      "verify",
	}, func(h *imaging.ImageHandler) error { return nil })
	if err == nil {
		t.Fatal("Workers=0 必须返回错误")
	}
	if !strings.Contains(err.Error(), "workers") {
		t.Errorf("err = %v, 期望包含 workers 提示", err)
	}
}
