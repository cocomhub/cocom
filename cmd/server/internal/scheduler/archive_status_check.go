// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	archivemanager "github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
)

const archiveStatusCheckConfigKey = "server.scheduler.archive_status_check"

var archiveStatusCheckerStarted atomic.Bool

type ArchiveStatusCheckConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Cron     string   `mapstructure:"cron"`
	Name     string   `mapstructure:"name"`
	Tags     []string `mapstructure:"tags"`
	Limit    int      `mapstructure:"limit"`
	MaxConn  int      `mapstructure:"max_conn"`
	Backends []string `mapstructure:"backends"`
}

type archiveStatusCheckIssue struct {
	CID       int
	Missing   []string
	Unhealthy []string
}

type archiveStatusCheckStats struct {
	Scanned    int64
	Matched    int64
	Limited    int64
	Replicated int64
	Checked    int64
	Skipped    int64
	Errors     int64
}

type archiveStatusCheckHooks struct {
	queryMissing   func(context.Context, string, int) ([]int, error)
	queryUnhealthy func(context.Context, string, int) ([]int, error)
	replicate      func(context.Context, int, string) (bool, error)
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

	backends := validateArchiveStatusCheckBackends(ctx, cfg.Backends)
	if len(backends) == 0 {
		slog.WarnContext(ctx, "scheduler ArchiveStatusChecker not registered: no valid backends")
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

				stats, err := archiveStatusCheckRunner(jobCtx, cfg, backends)
				if err != nil {
					slog.WarnContext(jobCtx, "ArchiveStatusChecker run failed", slog.String("err", err.Error()))
					return
				}
				slog.InfoContext(jobCtx, "ArchiveStatusChecker done",
					slog.Int64("scanned", stats.Scanned),
					slog.Int64("matched", stats.Matched),
					slog.Int64("limited", stats.Limited),
					slog.Int64("replicated", stats.Replicated),
					slog.Int64("checked", stats.Checked),
					slog.Int64("skipped", stats.Skipped),
					slog.Int64("errors", stats.Errors))
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
		Enabled:  viper.GetBool(archiveStatusCheckConfigKey + ".enabled"),
		Cron:     strings.TrimSpace(viper.GetString(archiveStatusCheckConfigKey + ".cron")),
		Name:     strings.TrimSpace(viper.GetString(archiveStatusCheckConfigKey + ".name")),
		Tags:     viper.GetStringSlice(archiveStatusCheckConfigKey + ".tags"),
		Limit:    viper.GetInt(archiveStatusCheckConfigKey + ".limit"),
		MaxConn:  viper.GetInt(archiveStatusCheckConfigKey + ".max_conn"),
		Backends: viper.GetStringSlice(archiveStatusCheckConfigKey + ".backends"),
	}
	if cfg.Name == "" {
		cfg.Name = "ArchiveStatusChecker"
	}
	if cfg.Limit <= 0 {
		slog.WarnContext(ctx, "archive_status_check limit is invalid, use default", slog.Int("limit", cfg.Limit), slog.Int("default", 100))
		cfg.Limit = 100
	}
	if cfg.MaxConn <= 0 {
		slog.WarnContext(ctx, "archive_status_check max_conn is invalid, use default", slog.Int("max_conn", cfg.MaxConn), slog.Int("default", 3))
		cfg.MaxConn = 3
	}
	return cfg
}

func validateArchiveStatusCheckBackends(ctx context.Context, backends []string) []string {
	validBackends := make([]string, 0, len(backends))
	seenBackends := map[string]struct{}{}
	if len(backends) == 0 {
		slog.WarnContext(ctx, "archive_status_check backends missing")
		return validBackends
	}

	for idx, backend := range backends {
		backend = strings.TrimSpace(backend)

		if backend == "" {
			slog.WarnContext(ctx, "archive_status_check backend skipped: backend is empty", slog.Int("index", idx))
			continue
		}
		if _, ok := seenBackends[backend]; ok {
			slog.WarnContext(ctx, "archive_status_check backend skipped: duplicate backend",
				slog.Int("index", idx),
				slog.String("backend", backend))
			continue
		}
		if _, ok := storage.Get(backend); !ok {
			slog.WarnContext(ctx, "archive_status_check backend skipped: backend not registered",
				slog.Int("index", idx),
				slog.String("backend", backend))
			continue
		}
		seenBackends[backend] = struct{}{}
		validBackends = append(validBackends, backend)
	}

	return validBackends
}

func runArchiveStatusCheck(ctx context.Context, cfg ArchiveStatusCheckConfig, backends []string) (archiveStatusCheckStats, error) {
	return runArchiveStatusCheckWithHooks(ctx, cfg, backends, archiveStatusCheckHooks{
		queryMissing:   listArchiveStatusCheckMissingCIDs,
		queryUnhealthy: listArchiveStatusCheckUnhealthyCIDs,
		replicate:      replicateArchiveStatusCheckBackend,
		check:          checkArchiveStatusCheckCID,
	})
}

func runArchiveStatusCheckWithHooks(ctx context.Context, cfg ArchiveStatusCheckConfig, backends []string, hooks archiveStatusCheckHooks) (archiveStatusCheckStats, error) {
	stats := archiveStatusCheckStats{}
	issues, scanStats, err := collectArchiveStatusCheckIssues(ctx, cfg.Limit, backends, hooks)
	if err != nil {
		return stats, err
	}
	stats.Scanned = scanStats.Scanned
	stats.Matched = scanStats.Matched
	stats.Limited = scanStats.Limited

	execStats := executeArchiveStatusCheckIssues(ctx, issues, hooks, cfg.MaxConn)
	stats.Replicated = execStats.Replicated
	stats.Checked = execStats.Checked
	stats.Skipped = execStats.Skipped
	stats.Errors = execStats.Errors
	return stats, nil
}

type archiveStatusCheckCIDItem struct {
	CID int `bson:"cid"`
}

func listArchiveStatusCheckMissingCIDs(ctx context.Context, backend string, limit int) ([]int, error) {
	filter := bson.M{
		"archive": bson.M{"$exists": true},
		"archive.locators": bson.M{
			"$not": bson.M{
				"$elemMatch": bson.M{
					"backend": backend,
				},
			},
		},
	}
	return listArchiveStatusCheckCIDs(ctx, filter, limit)
}

func listArchiveStatusCheckUnhealthyCIDs(ctx context.Context, backend string, limit int) ([]int, error) {
	filter := bson.M{
		"archive": bson.M{"$exists": true},
		"archive.locators": bson.M{
			"$elemMatch": bson.M{
				"backend":    backend,
				"healthy":    false,
				"checked_at": bson.M{"$lt": time.Now().Add(-time.Hour * 24 * 30)},
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

func replicateArchiveStatusCheckBackend(ctx context.Context, cid int, backend string) (bool, error) {
	prefix := api.StoragePrefix(cid)

	dst, ok := storage.Get(backend)
	if !ok {
		slog.WarnContext(ctx, "archive_status_check replicate skipped: backend not registered",
			slog.Int("cid", cid),
			slog.String("backend", backend),
			slog.String("prefix", prefix))
		return false, nil
	}

	_, err := archivemanager.ReplicateMore(ctx, dst, prefix, archivemanager.IndexFilter{ID: cid})
	if err != nil {
		return true, err
	}
	return true, nil
}

func checkArchiveStatusCheckCID(ctx context.Context, cid int) error {
	_, err := archivemanager.Check(ctx, cid, true)
	return err
}

func collectArchiveStatusCheckIssues(ctx context.Context, limit int, backends []string, hooks archiveStatusCheckHooks) ([]archiveStatusCheckIssue, archiveStatusCheckStats, error) {
	stats := archiveStatusCheckStats{}
	issueByCID := map[int]*archiveStatusCheckIssue{}

	for _, backend := range backends {
		missingCIDs, err := hooks.queryMissing(ctx, backend, limit)
		if err != nil {
			return nil, stats, err
		}
		stats.Scanned += int64(len(missingCIDs))
		for _, cid := range missingCIDs {
			appendArchiveStatusCheckIssueBackend(issueByCID, cid, backend, false)
		}

		unhealthyCIDs, err := hooks.queryUnhealthy(ctx, backend, limit)
		if err != nil {
			return nil, stats, err
		}
		stats.Scanned += int64(len(unhealthyCIDs))
		for _, cid := range unhealthyCIDs {
			appendArchiveStatusCheckIssueBackend(issueByCID, cid, backend, true)
		}
	}

	cids := make([]int, 0, len(issueByCID))
	for cid := range issueByCID {
		cids = append(cids, cid)
	}
	sort.Ints(cids)

	stats.Matched = int64(len(cids))
	if limit > 0 && len(cids) > limit {
		stats.Limited = int64(len(cids) - limit)
		cids = cids[:limit]
		stats.Matched = int64(len(cids))
	}

	issues := make([]archiveStatusCheckIssue, 0, len(cids))
	for _, cid := range cids {
		issues = append(issues, *issueByCID[cid])
	}
	return issues, stats, nil
}

func executeArchiveStatusCheckIssues(ctx context.Context, issues []archiveStatusCheckIssue, hooks archiveStatusCheckHooks, maxConn int) archiveStatusCheckStats {
	stats := archiveStatusCheckStats{}
	wg := sync.WaitGroup{}
	if maxConn <= 0 {
		maxConn = 1
	}
	ctx = context.WithoutCancel(ctx)
	maxConn = min(maxConn, len(issues))
	ch := make(chan struct{}, maxConn)
	for _, issue := range issues {
		wg.Go(func() {
			ch <- struct{}{}
			defer func() { <-ch }()

			for _, backend := range issue.Missing {
				slog.DebugContext(ctx, "archive_status_check replicate missing backend",
					slog.Int("cid", issue.CID),
					slog.String("backend", backend))
				executed, err := hooks.replicate(ctx, issue.CID, backend)
				if err != nil {
					atomic.AddInt64(&stats.Errors, 1)
					slog.WarnContext(ctx, "archive_status_check replicate failed",
						slog.Int("cid", issue.CID),
						slog.String("backend", backend),
						slog.String("err", err.Error()))
					return
				}
				if !executed {
					atomic.AddInt64(&stats.Skipped, 1)
					return
				}
				issue.Unhealthy = append(issue.Unhealthy, backend)
				atomic.AddInt64(&stats.Replicated, 1)
			}

			if len(issue.Unhealthy) == 0 {
				return
			}
			slog.DebugContext(ctx, "archive_status_check check unhealthy backend",
				slog.Int("cid", issue.CID),
				slog.Any("backends", issue.Unhealthy))
			if err := hooks.check(ctx, issue.CID); err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				slog.WarnContext(ctx, "archive_status_check check failed",
					slog.Int("cid", issue.CID),
					slog.Any("backends", issue.Unhealthy),
					slog.String("err", err.Error()))
				return
			}
			atomic.AddInt64(&stats.Checked, 1)
		})
	}
	wg.Wait()
	return stats
}

func appendArchiveStatusCheckBackendUnique(backends []string, backend string) []string {
	if slices.Contains(backends, backend) {
		return backends
	}
	return append(backends, backend)
}

func appendArchiveStatusCheckIssueBackend(issueByCID map[int]*archiveStatusCheckIssue, cid int, backend string, unhealthy bool) {
	if cid == 0 {
		return
	}
	issue := issueByCID[cid]
	if issue == nil {
		issue = &archiveStatusCheckIssue{CID: cid}
		issueByCID[cid] = issue
	}
	if unhealthy {
		issue.Unhealthy = appendArchiveStatusCheckBackendUnique(issue.Unhealthy, backend)
		return
	}
	issue.Missing = appendArchiveStatusCheckBackendUnique(issue.Missing, backend)
}
