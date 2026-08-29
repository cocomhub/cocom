// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/comic/probe"
	"github.com/go-co-op/gocron/v2"
)

var probeComicStarted atomic.Bool

// 内层闭包统一以 jobCtx（首个参数，即 job ctx）为日志 ctx：
// 停机时与调度器持有的 sc.ctx 级联取消同步，避免日志上下文与任务生命周期不一致。
func RegisterProbeComic(ctx context.Context, sc *Scheduler) {
	if sc == nil || sc.s == nil {
		return
	}
	cfg := config.Get().Server.Scheduler.ProbeComic
	if !cfg.Enabled {
		return
	}
	cronExpr := strings.TrimSpace(cfg.Cron)
	if cronExpr == "" {
		slog.WarnContext(ctx, "scheduler ProbeComic not registered: empty cron")
		return
	}
	withSeconds := len(strings.Fields(cronExpr)) == 6
	tags := cfg.Tags
	name := cfg.Name
	if name == "" {
		name = "ProbeComic"
	}
	_, err := sc.s.NewJob(
		gocron.CronJob(cronExpr, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !probeComicStarted.CompareAndSwap(false, true) {
				slog.InfoContext(jobCtx, "ProbeComic already running, skip new start")
				return
			}
			go runJobSafely(jobCtx, name, &probeComicStarted, func(runCtx context.Context) {
				if err := probe.ProbeComicJob(runCtx); err != nil {
					if errors.Is(err, context.Canceled) {
						// 停机/取消时的正常退出路径，降级为 Info。
						slog.InfoContext(runCtx, "ProbeComic stopped", slog.String("err", err.Error()))
						return
					}
					slog.WarnContext(runCtx, "ProbeComic stopped", slog.String("err", err.Error()))
				}
			})
		}),
		gocron.WithName(name),
		gocron.WithTags(tags...),
		gocron.WithContext(sc.jobContext(ctx)),
	)
	if err != nil {
		slog.WarnContext(ctx, "register ProbeComic to scheduler failed", slog.String("err", err.Error()))
	}
}
