// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/go-co-op/gocron/v2"
)

type Scheduler struct {
	s      gocron.Scheduler
	ctx    context.Context
	cancel context.CancelFunc
}

func New(ctx context.Context) (*Scheduler, error) {
	opts := []gocron.SchedulerOption{}
	tz := config.Get().Server.Scheduler.Timezone
	if tz != "" && tz != "Local" {
		if loc, err := time.LoadLocation(tz); err != nil {
			slog.WarnContext(ctx, "invalid scheduler timezone", slog.String("tz", tz), slog.String("err", err.Error()))
		} else {
			opts = append(opts, gocron.WithLocation(loc))
		}
	}
	s, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, err
	}
	// 调度器持有自己的可取消 ctx：停机时 cancel 会级联取消所有 job ctx，
	// 使后台 goroutine（配合 runJobSafely 的 recover + ctx 兜底）能真正退出。
	schedCtx, cancel := context.WithCancel(ctx)
	return &Scheduler{s: s, ctx: schedCtx, cancel: cancel}, nil
}

func (sc *Scheduler) Start(_ context.Context) error {
	if sc == nil || sc.s == nil {
		return nil
	}
	sc.s.Start()
	return nil
}

func (sc *Scheduler) Stop(_ context.Context) error {
	if sc == nil || sc.s == nil {
		return nil
	}
	// 先取消 job ctx，让后台 goroutine（配合 runJobSafely 的 recover + ctx 兜底）
	// 随 ctx.Done 退出，再停调度器。gocron.Shutdown 只停调度、不取消 job ctx。
	if sc.cancel != nil {
		sc.cancel()
	}
	return sc.s.Shutdown()
}

func (sc *Scheduler) Core() gocron.Scheduler {
	return sc.s
}

// jobContext 返回 job 应使用的父 ctx：优先调度器持有的可取消 ctx（停机时可被
// Stop 级联取消），为 nil 时回退到调用方传入的 ctx。
func (sc *Scheduler) jobContext(fallback context.Context) context.Context {
	if sc != nil && sc.ctx != nil {
		return sc.ctx
	}
	return fallback
}
