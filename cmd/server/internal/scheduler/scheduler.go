// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
)

type Scheduler struct {
	s gocron.Scheduler
}

func New(ctx context.Context) (*Scheduler, error) {
	opts := []gocron.SchedulerOption{}
	if tz := viper.GetString("server.scheduler.timezone"); tz != "" && tz != "Local" {
		if loc, err := time.LoadLocation(tz); err != nil {
			clog.Warnf(ctx, "invalid scheduler timezone: %s, use Local. err=%v", tz, err)
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
