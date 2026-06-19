// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/cocomaarchiver"
	"github.com/go-co-op/gocron/v2"
)

var cocomaArchiverStarted atomic.Bool

func RegisterCocomaArchiver(ctx context.Context, sc *Scheduler) {
	if sc == nil || sc.s == nil {
		return
	}
	cfg := config.Get().Server.Scheduler.CocomaArchiver
	if !cfg.Enabled {
		return
	}
	cronExpr := strings.TrimSpace(cfg.Cron)
	if cronExpr == "" {
		slog.WarnContext(ctx, "scheduler CocomaArchiver not registered: empty cron")
		return
	}
	scanDir := strings.TrimSpace(cfg.ScanDir)
	archiveDir := strings.TrimSpace(cfg.ArchiveDir)
	notmatchDir := strings.TrimSpace(cfg.NotMatchDir)
	if scanDir == "" || archiveDir == "" || notmatchDir == "" {
		slog.WarnContext(ctx, "scheduler CocomaArchiver not registered: missing required paths")
		return
	}
	withSeconds := len(strings.Fields(cronExpr)) == 6
	_, err := sc.s.NewJob(
		gocron.CronJob(cronExpr, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !cocomaArchiverStarted.CompareAndSwap(false, true) {
				slog.InfoContext(ctx, "CocomaArchiver already running, skip new start")
				return
			}
			go func() {
				defer func() { cocomaArchiverStarted.Store(false) }()
				stats, err := cocomaarchiver.RunOnce(jobCtx, cocomaarchiver.Options{
					ScanDir:     scanDir,
					ArchiveDir:  archiveDir,
					NotMatchDir: notmatchDir,
					Limit:       cfg.Limit,
					LookupMD5: func(ctx context.Context, cid int) (string, error) {
						type item struct {
							Archive struct {
								MD5 string `bson:"md5"`
							} `bson:"archive"`
						}
						var list []item
						err := mongo.ComicInfoBuilder().
							Filters("cid", cid).
							Limit(1).
							All(ctx, &list)
						if err != nil {
							return "", err
						}
						if len(list) == 0 {
							return "", nil
						}
						return list[0].Archive.MD5, nil
					},
				})
				if err != nil {
					slog.WarnContext(ctx, "CocomaArchiver run failed", slog.String("err", err.Error()))
					return
				}
				slog.InfoContext(ctx, "CocomaArchiver done", slog.Group("stats",
					slog.Int("scanned", stats.Scanned),
					slog.Int("processed", stats.Processed),
					slog.Int("archived", stats.Archived),
					slog.Int("notmatch", stats.NotMatch),
					slog.Int("errors", stats.Errors)))
			}()
		}),
		gocron.WithName("CocomaArchiver"),
		gocron.WithTags("archive", "cocoma"),
		gocron.WithContext(ctx),
	)
	if err != nil {
		slog.WarnContext(ctx, "register CocomaArchiver to scheduler failed", slog.String("err", err.Error()))
	}
}
