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
	s gocron.Scheduler
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
	return &Scheduler{s: s}, nil
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
	return sc.s.Shutdown()
}

func (sc *Scheduler) Core() gocron.Scheduler {
	return sc.s
}
