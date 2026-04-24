// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	archivemanager "github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
)

const archiveStatusCheckConfigKey = "server.scheduler.archive_status_check"

var archiveStatusCheckerStarted atomic.Bool

type ArchiveStatusCheckTarget struct {
	Backend string `mapstructure:"backend"`
	Prefix  string `mapstructure:"prefix"`
}

type ArchiveStatusCheckConfig struct {
	Enabled bool                       `mapstructure:"enabled"`
	Cron    string                     `mapstructure:"cron"`
	Name    string                     `mapstructure:"name"`
	Tags    []string                   `mapstructure:"tags"`
	Limit   int                        `mapstructure:"limit"`
	Targets []ArchiveStatusCheckTarget `mapstructure:"targets"`
}

type archiveStatusCheckIssue struct {
	CID       int
	Missing   []ArchiveStatusCheckTarget
	Unhealthy []ArchiveStatusCheckTarget
}

type archiveStatusCheckStats struct {
	Scanned        int
	Matched        int
	Limited        int
	Replicated     int
	Checked        int
	SkippedTargets int
	Errors         int
}

type archiveStatusCheckHooks struct {
	queryMissing   func(context.Context, ArchiveStatusCheckTarget, int) ([]int, error)
	queryUnhealthy func(context.Context, ArchiveStatusCheckTarget, int) ([]int, error)
	replicate      func(context.Context, int, ArchiveStatusCheckTarget) (bool, error)
	check          func(context.Context, int) error
}

var archiveStatusCheckRunner = runArchiveStatusCheck

func RegisterArchiveStatusChecker(ctx context.Context, sc *Scheduler) {
	if sc == nil || sc.s == nil {
		return
	}

	cfg := loadArchiveStatusCheckConfig(ctx)
	if !cfg.Enabled {
		return
	}
	if cfg.Cron == "" {
		slog.WarnContext(ctx, "scheduler ArchiveStatusChecker not registered: empty cron")
		return
	}

	targets := validateArchiveStatusCheckTargets(ctx, cfg.Targets)
	if len(targets) == 0 {
		slog.WarnContext(ctx, "scheduler ArchiveStatusChecker not registered: no valid targets")
		return
	}

	withSeconds := len(strings.Fields(cfg.Cron)) == 6
	_, err := sc.s.NewJob(
		gocron.CronJob(cfg.Cron, withSeconds),
		gocron.NewTask(func(jobCtx context.Context) {
			if !archiveStatusCheckerStarted.CompareAndSwap(false, true) {
				slog.InfoContext(jobCtx, "ArchiveStatusChecker already running, skip new start")
				return
			}
			go func() {
				defer archiveStatusCheckerStarted.Store(false)

				stats, err := archiveStatusCheckRunner(jobCtx, cfg, targets)
				if err != nil {
					slog.WarnContext(jobCtx, "ArchiveStatusChecker run failed", slog.String("err", err.Error()))
					return
				}
				slog.InfoContext(jobCtx, "ArchiveStatusChecker done",
					slog.Int("scanned", stats.Scanned),
					slog.Int("matched", stats.Matched),
					slog.Int("limited", stats.Limited),
					slog.Int("replicated", stats.Replicated),
					slog.Int("checked", stats.Checked),
					slog.Int("skipped_targets", stats.SkippedTargets),
					slog.Int("errors", stats.Errors))
			}()
		}),
		gocron.WithName(cfg.Name),
		gocron.WithTags(cfg.Tags...),
		gocron.WithContext(ctx),
	)
	if err != nil {
		slog.WarnContext(ctx, "register ArchiveStatusChecker to scheduler failed", slog.String("err", err.Error()))
	}
}

func loadArchiveStatusCheckConfig(ctx context.Context) ArchiveStatusCheckConfig {
	cfg := ArchiveStatusCheckConfig{
		Enabled: viper.GetBool(archiveStatusCheckConfigKey + ".enabled"),
		Cron:    strings.TrimSpace(viper.GetString(archiveStatusCheckConfigKey + ".cron")),
		Name:    strings.TrimSpace(viper.GetString(archiveStatusCheckConfigKey + ".name")),
		Tags:    viper.GetStringSlice(archiveStatusCheckConfigKey + ".tags"),
		Limit:   viper.GetInt(archiveStatusCheckConfigKey + ".limit"),
	}
	if cfg.Name == "" {
		cfg.Name = "ArchiveStatusChecker"
	}
	if cfg.Limit <= 0 {
		slog.WarnContext(ctx, "archive_status_check limit is invalid, use default", slog.Int("limit", cfg.Limit), slog.Int("default", 100))
		cfg.Limit = 100
	}
	if err := viper.UnmarshalKey(archiveStatusCheckConfigKey+".targets", &cfg.Targets); err != nil {
		slog.WarnContext(ctx, "archive_status_check targets decode failed", slog.String("err", err.Error()))
	}
	return cfg
}

func validateArchiveStatusCheckTargets(ctx context.Context, targets []ArchiveStatusCheckTarget) []ArchiveStatusCheckTarget {
	validTargets := make([]ArchiveStatusCheckTarget, 0, len(targets))
	seenBackends := map[string]struct{}{}
	if len(targets) == 0 {
		slog.WarnContext(ctx, "archive_status_check targets missing")
		return validTargets
	}

	for idx, target := range targets {
		target.Backend = strings.TrimSpace(target.Backend)
		target.Prefix = strings.TrimSpace(target.Prefix)

		if target.Backend == "" {
			slog.WarnContext(ctx, "archive_status_check target skipped: backend is empty", slog.Int("index", idx))
			continue
		}
		if target.Prefix == "" {
			slog.WarnContext(ctx, "archive_status_check target skipped: prefix is empty",
				slog.Int("index", idx),
				slog.String("backend", target.Backend))
			continue
		}
		if _, ok := seenBackends[target.Backend]; ok {
			slog.WarnContext(ctx, "archive_status_check target skipped: duplicate backend",
				slog.Int("index", idx),
				slog.String("backend", target.Backend))
			continue
		}
		prefix, err := storage.Path(target.Prefix)
		if err != nil {
			slog.WarnContext(ctx, "archive_status_check target skipped: invalid prefix",
				slog.Int("index", idx),
				slog.String("backend", target.Backend),
				slog.String("prefix", target.Prefix),
				slog.String("err", err.Error()))
			continue
		}
		if _, ok := storage.Get(target.Backend); !ok {
			slog.WarnContext(ctx, "archive_status_check target skipped: backend not registered",
				slog.Int("index", idx),
				slog.String("backend", target.Backend),
				slog.String("prefix", prefix))
			continue
		}
		target.Prefix = prefix
		seenBackends[target.Backend] = struct{}{}
		validTargets = append(validTargets, target)
	}

	return validTargets
}

func runArchiveStatusCheck(ctx context.Context, cfg ArchiveStatusCheckConfig, targets []ArchiveStatusCheckTarget) (archiveStatusCheckStats, error) {
	return runArchiveStatusCheckWithHooks(ctx, cfg, targets, archiveStatusCheckHooks{
		queryMissing:   listArchiveStatusCheckMissingCIDs,
		queryUnhealthy: listArchiveStatusCheckUnhealthyCIDs,
		replicate:      replicateArchiveStatusCheckTarget,
		check:          checkArchiveStatusCheckCID,
	})
}

func runArchiveStatusCheckWithHooks(ctx context.Context, cfg ArchiveStatusCheckConfig, targets []ArchiveStatusCheckTarget, hooks archiveStatusCheckHooks) (archiveStatusCheckStats, error) {
	stats := archiveStatusCheckStats{}
	issues, scanStats, err := collectArchiveStatusCheckIssues(ctx, cfg.Limit, targets, hooks)
	if err != nil {
		return stats, err
	}
	stats.Scanned = scanStats.Scanned
	stats.Matched = scanStats.Matched
	stats.Limited = scanStats.Limited

	execStats := executeArchiveStatusCheckIssues(ctx, issues, hooks)
	stats.Replicated = execStats.Replicated
	stats.Checked = execStats.Checked
	stats.SkippedTargets = execStats.SkippedTargets
	stats.Errors = execStats.Errors
	return stats, nil
}

type archiveStatusCheckCIDItem struct {
	CID int `bson:"cid"`
}

func listArchiveStatusCheckMissingCIDs(ctx context.Context, target ArchiveStatusCheckTarget, limit int) ([]int, error) {
	filter := bson.M{
		"archive": bson.M{"$exists": true},
		"archive.locators": bson.M{
			"$not": bson.M{
				"$elemMatch": bson.M{
					"backend": target.Backend,
				},
			},
		},
	}
	return listArchiveStatusCheckCIDs(ctx, filter, limit)
}

func listArchiveStatusCheckUnhealthyCIDs(ctx context.Context, target ArchiveStatusCheckTarget, limit int) ([]int, error) {
	filter := bson.M{
		"archive": bson.M{"$exists": true},
		"archive.locators": bson.M{
			"$elemMatch": bson.M{
				"backend": target.Backend,
				"healthy": false,
			},
		},
	}
	return listArchiveStatusCheckCIDs(ctx, filter, limit)
}

func listArchiveStatusCheckCIDs(ctx context.Context, filter bson.M, limit int) ([]int, error) {
	var items []archiveStatusCheckCIDItem
	builder := mongo.ComicInfoBuilder().
		SortKV("cid", 1)
	for key, value := range filter {
		builder.FilterKV(key, value)
	}
	if limit > 0 {
		builder.Limit(int64(limit))
	} else {
		builder.NoLimit()
	}
	if err := builder.All(ctx, &items); err != nil {
		return nil, err
	}
	cids := make([]int, 0, len(items))
	for _, item := range items {
		if item.CID != 0 {
			cids = append(cids, item.CID)
		}
	}
	return cids, nil
}

func replicateArchiveStatusCheckTarget(ctx context.Context, cid int, target ArchiveStatusCheckTarget) (bool, error) {
	prefix, err := storage.Path(target.Prefix)
	if err != nil {
		slog.WarnContext(ctx, "archive_status_check replicate skipped: invalid prefix",
			slog.Int("cid", cid),
			slog.String("backend", target.Backend),
			slog.String("prefix", target.Prefix),
			slog.String("err", err.Error()))
		return false, nil
	}

	dst, ok := storage.Get(target.Backend)
	if !ok {
		slog.WarnContext(ctx, "archive_status_check replicate skipped: backend not registered",
			slog.Int("cid", cid),
			slog.String("backend", target.Backend),
			slog.String("prefix", prefix))
		return false, nil
	}

	_, err = archivemanager.Replicate(ctx, dst, prefix, archivemanager.IndexFilter{ID: cid})
	if err != nil {
		return true, err
	}
	return true, nil
}

func checkArchiveStatusCheckCID(ctx context.Context, cid int) error {
	_, err := archivemanager.Check(ctx, cid, false)
	return err
}

func collectArchiveStatusCheckIssues(ctx context.Context, limit int, targets []ArchiveStatusCheckTarget, hooks archiveStatusCheckHooks) ([]archiveStatusCheckIssue, archiveStatusCheckStats, error) {
	stats := archiveStatusCheckStats{}
	issueByCID := map[int]*archiveStatusCheckIssue{}

	for _, target := range targets {
		missingCIDs, err := hooks.queryMissing(ctx, target, limit)
		if err != nil {
			return nil, stats, err
		}
		stats.Scanned += len(missingCIDs)
		for _, cid := range missingCIDs {
			appendArchiveStatusCheckIssueTarget(issueByCID, cid, target, false)
		}

		unhealthyCIDs, err := hooks.queryUnhealthy(ctx, target, limit)
		if err != nil {
			return nil, stats, err
		}
		stats.Scanned += len(unhealthyCIDs)
		for _, cid := range unhealthyCIDs {
			appendArchiveStatusCheckIssueTarget(issueByCID, cid, target, true)
		}
	}

	cids := make([]int, 0, len(issueByCID))
	for cid := range issueByCID {
		cids = append(cids, cid)
	}
	sort.Ints(cids)

	stats.Matched = len(cids)
	if limit > 0 && len(cids) > limit {
		stats.Limited = len(cids) - limit
		cids = cids[:limit]
		stats.Matched = len(cids)
	}

	issues := make([]archiveStatusCheckIssue, 0, len(cids))
	for _, cid := range cids {
		issues = append(issues, *issueByCID[cid])
	}
	return issues, stats, nil
}

func executeArchiveStatusCheckIssues(ctx context.Context, issues []archiveStatusCheckIssue, hooks archiveStatusCheckHooks) archiveStatusCheckStats {
	stats := archiveStatusCheckStats{}
	for _, issue := range issues {
		for _, target := range issue.Missing {
			executed, err := hooks.replicate(ctx, issue.CID, target)
			if err != nil {
				stats.Errors++
				slog.WarnContext(ctx, "archive_status_check replicate failed",
					slog.Int("cid", issue.CID),
					slog.String("backend", target.Backend),
					slog.String("prefix", target.Prefix),
					slog.String("err", err.Error()))
				continue
			}
			if !executed {
				stats.SkippedTargets++
				continue
			}
			stats.Replicated++
		}

		if len(issue.Unhealthy) == 0 {
			continue
		}
		if err := hooks.check(ctx, issue.CID); err != nil {
			stats.Errors++
			slog.WarnContext(ctx, "archive_status_check check failed",
				slog.Int("cid", issue.CID),
				slog.Any("backends", archiveStatusCheckBackends(issue.Unhealthy)),
				slog.String("err", err.Error()))
			continue
		}
		stats.Checked++
	}
	return stats
}

func archiveStatusCheckBackends(targets []ArchiveStatusCheckTarget) []string {
	backends := make([]string, 0, len(targets))
	for _, target := range targets {
		backends = append(backends, target.Backend)
	}
	return backends
}

func appendArchiveStatusCheckTargetUnique(targets []ArchiveStatusCheckTarget, target ArchiveStatusCheckTarget) []ArchiveStatusCheckTarget {
	for _, item := range targets {
		if item.Backend == target.Backend {
			return targets
		}
	}
	return append(targets, target)
}

func appendArchiveStatusCheckIssueTarget(issueByCID map[int]*archiveStatusCheckIssue, cid int, target ArchiveStatusCheckTarget, unhealthy bool) {
	if cid == 0 {
		return
	}
	issue := issueByCID[cid]
	if issue == nil {
		issue = &archiveStatusCheckIssue{CID: cid}
		issueByCID[cid] = issue
	}
	if unhealthy {
		issue.Unhealthy = appendArchiveStatusCheckTargetUnique(issue.Unhealthy, target)
		return
	}
	issue.Missing = appendArchiveStatusCheckTargetUnique(issue.Missing, target)
}
