// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/comic/probe"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
)

var probeComicStarted atomic.Bool

func RegisterProbeComic(ctx context.Context, sc *Scheduler) {
	if sc == nil || sc.s == nil {
		return
	}
	if !viper.GetBool("server.scheduler.probe_comic.enabled") {
		return
	}
	cronExpr := strings.TrimSpace(viper.GetString("server.scheduler.probe_comic.cron"))
	if cronExpr == "" {
		clog.Warnf(ctx, "scheduler ProbeComic not registered: empty cron")
		return
	}
	withSeconds := len(strings.Fields(cronExpr)) == 6
	tags := viper.GetStringSlice("server.scheduler.probe_comic.tags")
	name := viper.GetString("server.scheduler.probe_comic.name")
	if name == "" {
		name = "ProbeComic"
	}
	_, err := sc.s.NewJob(
		gocron.CronJob(cronExpr, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !probeComicStarted.CompareAndSwap(false, true) {
				clog.Infof(ctx, "ProbeComic already running, skip new start")
				return
			}
			go func() {
				if err := probe.ProbeComicJob(jobCtx); err != nil {
					clog.Warnf(ctx, "ProbeComic stopped: %v", err)
				}
				probeComicStarted.Store(false)
			}()
		}),
		gocron.WithName(name),
		gocron.WithTags(tags...),
		gocron.WithContext(ctx),
	)
	if err != nil {
		clog.Warnf(ctx, "register ProbeComic to scheduler failed: %v", err)
	}
}
