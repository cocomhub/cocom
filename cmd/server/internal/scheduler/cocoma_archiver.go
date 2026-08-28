// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"regexp"
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

	// 接续死键 cid_regex：注入 cfg.CIDRegex 到 Options（为空时由 cocomaarchiver
	// 回退默认 ^(\d+)\.cocoma$；编译失败时记录错误并以默认正则继续，
	// 避免配置错误导致任务被静默停用（铁律 1 不静默降级）。
	var cidRegexp *regexp.Regexp
	cidRegexStr := strings.TrimSpace(cfg.CIDRegex)
	if cidRegexStr != "" {
		re, reErr := regexp.Compile(cidRegexStr)
		if reErr != nil {
			slog.WarnContext(ctx, "CocomaArchiver using default cid_regex: invalid configured regex",
				slog.String("regex", cidRegexStr), slog.String("err", reErr.Error()))
		} else {
			cidRegexp = re
		}
	}

	_, err := sc.s.NewJob(
		gocron.CronJob(cronExpr, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !cocomaArchiverStarted.CompareAndSwap(false, true) {
				slog.InfoContext(ctx, "CocomaArchiver already running, skip new start")
				return
			}
			go runJobSafely(jobCtx, "CocomaArchiver", &cocomaArchiverStarted, func(ctx context.Context) {
				stats, err := cocomaarchiver.RunOnce(ctx, cocomaarchiver.Options{
					ScanDir:     scanDir,
					ArchiveDir:  archiveDir,
					NotMatchDir: notmatchDir,
					Limit:       cfg.Limit,
					CIDRegex:    cidRegexp,
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
			})
		}),
		gocron.WithName("CocomaArchiver"),
		gocron.WithTags("archive", "cocoma"),
		gocron.WithContext(sc.jobContext(ctx)),
	)
	if err != nil {
		slog.WarnContext(ctx, "register CocomaArchiver to scheduler failed", slog.String("err", err.Error()))
	}
}
