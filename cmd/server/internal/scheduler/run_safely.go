// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// runJobSafely 统一包裹调度任务的后台 goroutine：
//   - 无论 fn 正常返回还是 panic，都会复位 started 标志；
//   - panic 会被 recover 并记录日志，避免整个 server 进程崩溃。
//
// 调用方应使用 `go runJobSafely(jobCtx, name, &startedVar, fn)` 启动。
func runJobSafely(ctx context.Context, name string, started *atomic.Bool, fn func(context.Context)) {
	defer func() {
		started.Store(false)
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "scheduler job panic",
				slog.String("name", name),
				slog.Any("panic", r))
		}
	}()
	fn(ctx)
}
