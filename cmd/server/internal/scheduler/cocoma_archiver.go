// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/cocomaarchiver"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
)

var cocomaArchiverStarted atomic.Bool

func RegisterCocomaArchiver(ctx context.Context, sc *Scheduler) {
	if sc == nil || sc.s == nil {
		return
	}
	if !viper.GetBool("server.scheduler.cocoma_archiver.enabled") {
		return
	}
	cronExpr := strings.TrimSpace(viper.GetString("server.scheduler.cocoma_archiver.cron"))
	if cronExpr == "" {
		clog.Warnf(ctx, "scheduler CocomaArchiver not registered: empty cron")
		return
	}
	scanDir := strings.TrimSpace(viper.GetString("server.scheduler.cocoma_archiver.scan_dir"))
	archiveDir := strings.TrimSpace(viper.GetString("server.scheduler.cocoma_archiver.archive_dir"))
	notmatchDir := strings.TrimSpace(viper.GetString("server.scheduler.cocoma_archiver.notmatch_dir"))
	if scanDir == "" || archiveDir == "" || notmatchDir == "" {
		clog.Warnf(ctx, "scheduler CocomaArchiver not registered: missing required paths")
		return
	}
	withSeconds := len(strings.Fields(cronExpr)) == 6
	_, err := sc.s.NewJob(
		gocron.CronJob(cronExpr, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !cocomaArchiverStarted.CompareAndSwap(false, true) {
				clog.Infof(ctx, "CocomaArchiver already running, skip new start")
				return
			}
			go func() {
				defer func() { cocomaArchiverStarted.Store(false) }()
				stats, err := cocomaarchiver.RunOnce(jobCtx, cocomaarchiver.Options{
					ScanDir:     scanDir,
					ArchiveDir:  archiveDir,
					NotMatchDir: notmatchDir,
					Limit:       viper.GetInt("server.scheduler.cocoma_archiver.limit"),
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
					clog.Warnf(ctx, "CocomaArchiver run failed: %v", err)
					return
				}
				clog.Infof(ctx, "CocomaArchiver done: %s", fmt.Sprintf("scanned=%d processed=%d archived=%d notmatch=%d errors=%d",
					stats.Scanned, stats.Processed, stats.Archived, stats.NotMatch, stats.Errors))
			}()
		}),
		gocron.WithName("CocomaArchiver"),
		gocron.WithTags("archive", "cocoma"),
		gocron.WithContext(ctx),
	)
	if err != nil {
		clog.Warnf(ctx, "register CocomaArchiver to scheduler failed: %v", err)
	}
}
