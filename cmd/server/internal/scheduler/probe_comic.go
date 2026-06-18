// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/comic/probe"
	"github.com/go-co-op/gocron/v2"
)

var probeComicStarted atomic.Bool

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
				slog.InfoContext(ctx, "ProbeComic already running, skip new start")
				return
			}
			go func() {
				if err := probe.ProbeComicJob(jobCtx); err != nil {
					slog.WarnContext(ctx, "ProbeComic stopped", slog.String("err", err.Error()))
				}
				probeComicStarted.Store(false)
			}()
		}),
		gocron.WithName(name),
		gocron.WithTags(tags...),
		gocron.WithContext(ctx),
	)
	if err != nil {
		slog.WarnContext(ctx, "register ProbeComic to scheduler failed", slog.String("err", err.Error()))
	}
}
