// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/spf13/viper"
)

func TestCollectArchiveStatusCheckIssuesAggregatesByCID(t *testing.T) {
	backends := []string{
		"backup-a",
		"backup-b",
		"backup-c",
	}
	missing := map[string][]int{
		"backup-a": {1003},
		"backup-b": {1001, 1003},
		"backup-c": {1003},
	}
	unhealthy := map[string][]int{
		"backup-a": {1001},
	}

	issues, stats, err := collectArchiveStatusCheckIssues(context.Background(), 10, backends, newArchiveStatusCheckQueryHooks(missing, unhealthy))
	if err != nil {
		t.Fatalf("collect issues err: %v", err)
	}
	if stats.Scanned != 5 {
		t.Fatalf("unexpected scanned: %d", stats.Scanned)
	}
	if stats.Matched != 2 || stats.Limited != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(issues) != 2 {
		t.Fatalf("unexpected issues length: %d", len(issues))
	}

	if issues[0].CID != 1001 {
		t.Fatalf("unexpected first cid: %+v", issues[0])
	}
	if !reflect.DeepEqual(issues[0].Missing, []string{"backup-b"}) {
		t.Fatalf("unexpected missing backends: %+v", issues[0].Missing)
	}
	if !reflect.DeepEqual(issues[0].Unhealthy, []string{"backup-a"}) {
		t.Fatalf("unexpected unhealthy backends: %+v", issues[0].Unhealthy)
	}

	if issues[1].CID != 1003 {
		t.Fatalf("unexpected second cid: %+v", issues[1])
	}
	if !reflect.DeepEqual(issues[1].Missing, []string{"backup-a", "backup-b", "backup-c"}) {
		t.Fatalf("unexpected all-missing backends: %+v", issues[1].Missing)
	}
	if len(issues[1].Unhealthy) != 0 {
		t.Fatalf("unexpected unhealthy backends: %+v", issues[1].Unhealthy)
	}
}

func TestCollectArchiveStatusCheckIssuesLimit(t *testing.T) {
	backends := []string{
		"backup-a",
	}

	issues, stats, err := collectArchiveStatusCheckIssues(context.Background(), 2, backends, newArchiveStatusCheckQueryHooks(
		map[string][]int{"backup-a": {1001, 1002, 1003}},
		nil,
	))
	if err != nil {
		t.Fatalf("collect issues err: %v", err)
	}
	if stats.Scanned != 3 || stats.Matched != 2 || stats.Limited != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(issues) != 2 {
		t.Fatalf("unexpected issues length: %d", len(issues))
	}
	if issues[0].CID != 1001 || issues[1].CID != 1002 {
		t.Fatalf("unexpected issue order: %+v", issues)
	}
}

func TestCollectArchiveStatusCheckIssuesDeduplicatesRepeatedCIDBackends(t *testing.T) {
	backends := []string{
		"backup-a",
		"backup-b",
	}

	issues, stats, err := collectArchiveStatusCheckIssues(context.Background(), 10, backends, newArchiveStatusCheckQueryHooks(
		map[string][]int{"backup-b": {1001, 1001}},
		map[string][]int{"backup-a": {1001, 1001}},
	))
	if err != nil {
		t.Fatalf("collect issues err: %v", err)
	}
	if stats.Scanned != 4 || stats.Matched != 1 || stats.Limited != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(issues) != 1 {
		t.Fatalf("unexpected issues length: %d", len(issues))
	}
	if !reflect.DeepEqual(issues[0].Missing, []string{"backup-b"}) {
		t.Fatalf("unexpected missing backends: %+v", issues[0].Missing)
	}
	if !reflect.DeepEqual(issues[0].Unhealthy, []string{"backup-a"}) {
		t.Fatalf("unexpected unhealthy backends: %+v", issues[0].Unhealthy)
	}
}

func TestRunArchiveStatusCheckUsesBackendQueriesWithLimit(t *testing.T) {
	backends := []string{
		"backup-a",
		"backup-b",
	}
	cfg := ArchiveStatusCheckConfig{Limit: 2}
	var calls []string

	stats, err := runArchiveStatusCheckWithHooks(context.Background(), cfg, backends, archiveStatusCheckHooks{
		queryMissing: func(_ context.Context, backend string, limit int) ([]int, error) {
			calls = append(calls, "missing:"+backend)
			if limit != 2 {
				t.Fatalf("unexpected missing limit: %d", limit)
			}
			if backend == "backup-a" {
				return []int{1001}, nil
			}
			return []int{1002}, nil
		},
		queryUnhealthy: func(_ context.Context, backend string, limit int) ([]int, error) {
			calls = append(calls, "unhealthy:"+backend)
			if limit != 2 {
				t.Fatalf("unexpected unhealthy limit: %d", limit)
			}
			if backend == "backup-a" {
				return []int{1001}, nil
			}
			return nil, nil
		},
		replicate: func(_ context.Context, _ int, backend string) (bool, error) { return true, nil },
		check:     func(_ context.Context, _ int) error { return nil },
	})
	if err != nil {
		t.Fatalf("runArchiveStatusCheckWithHooks err: %v", err)
	}
	wantCalls := []string{"missing:backup-a", "unhealthy:backup-a", "missing:backup-b", "unhealthy:backup-b"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("unexpected query calls: %+v", calls)
	}
	if stats.Scanned != 3 || stats.Matched != 2 || stats.Replicated != 2 || stats.Checked != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestExecuteArchiveStatusCheckIssuesReplicateThenCheckOnce(t *testing.T) {
	issues := []archiveStatusCheckIssue{
		{
			CID: 2001,
			Missing: []string{
				"backup-a",
				"backup-b",
			},
			Unhealthy: []string{
				"backup-c",
				"backup-d",
			},
		},
		{
			CID: 2002,
			Unhealthy: []string{
				"backup-e",
				"backup-f",
			},
		},
	}

	var calls []string
	stats := executeArchiveStatusCheckIssues(context.Background(), issues, archiveStatusCheckHooks{
		replicate: func(_ context.Context, cid int, backend string) (bool, error) {
			calls = append(calls, "replicate:"+backend)
			if cid != 2001 {
				t.Fatalf("unexpected replicate cid: %d", cid)
			}
			return true, nil
		},
		check: func(_ context.Context, cid int) error {
			calls = append(calls, "check")
			if cid != 2001 && cid != 2002 {
				t.Fatalf("unexpected check cid: %d", cid)
			}
			return nil
		},
	}, 2)

	wantCalls := []string{"replicate:backup-a", "replicate:backup-b", "check", "check"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("unexpected call order: %+v", calls)
	}
	if stats.Replicated != 2 || stats.Checked != 2 || stats.Errors != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestExecuteArchiveStatusCheckIssuesContinuesOnErrorAndSkip(t *testing.T) {
	issues := []archiveStatusCheckIssue{
		{
			CID: 3001,
			Missing: []string{
				"skip",
				"fail",
			},
			Unhealthy: []string{
				"broken",
			},
		},
	}

	stats := executeArchiveStatusCheckIssues(context.Background(), issues, archiveStatusCheckHooks{
		replicate: func(_ context.Context, _ int, backend string) (bool, error) {
			switch backend {
			case "skip":
				return false, nil
			case "fail":
				return true, errors.New("replicate failed")
			default:
				return true, nil
			}
		},
		check: func(_ context.Context, _ int) error {
			return errors.New("check failed")
		},
	}, 2)

	if stats.Replicated != 0 || stats.Checked != 0 {
		t.Fatalf("unexpected success stats: %+v", stats)
	}
	if stats.Skipped != 1 || stats.Errors != 2 {
		t.Fatalf("unexpected failure stats: %+v", stats)
	}
}

func TestRegisterArchiveStatusCheckerRunsThroughSchedulerEntry(t *testing.T) {
	archiveStatusCheckerStarted.Store(false)
	oldRunner := archiveStatusCheckRunner
	defer func() {
		archiveStatusCheckRunner = oldRunner
		archiveStatusCheckerStarted.Store(false)
	}()

	backendName := "archive-status-check-test-backend"
	root := t.TempDir()
	storage.Clear()
	s := localfs.New(backendName, root)
	if err := storage.Set(backendName, s); err != nil {
		t.Fatalf("set storage err: %v", err)
	}

	viper.Set("server.scheduler.archive_status_check.enabled", true)
	viper.Set("server.scheduler.archive_status_check.name", "ArchiveStatusChecker")
	viper.Set("server.scheduler.archive_status_check.cron", "*/5 * * * * *")
	viper.Set("server.scheduler.archive_status_check.tags", []string{"archive", "check"})
	viper.Set("server.scheduler.archive_status_check.limit", 3)
	viper.Set("server.scheduler.archive_status_check.backends", []string{
		backendName,
	})

	sc, err := New(context.Background())
	if err != nil {
		t.Fatalf("new scheduler err: %v", err)
	}
	defer func() { _ = sc.Stop(context.Background()) }()

	runCh := make(chan struct{}, 1)
	archiveStatusCheckRunner = func(_ context.Context, cfg ArchiveStatusCheckConfig, backends []string) (archiveStatusCheckStats, error) {
		if cfg.Limit != 3 {
			t.Fatalf("unexpected cfg limit: %d", cfg.Limit)
		}
		if len(backends) != 1 || backends[0] != backendName {
			t.Fatalf("unexpected backends: %+v", backends)
		}
		runCh <- struct{}{}
		return archiveStatusCheckStats{Scanned: 1, Matched: 1, Checked: 1}, nil
	}

	RegisterArchiveStatusChecker(context.Background(), sc)
	jobs := sc.Core().Jobs()
	if len(jobs) != 1 {
		t.Fatalf("unexpected jobs length: %d", len(jobs))
	}
	if err := sc.Start(context.Background()); err != nil {
		t.Fatalf("start scheduler err: %v", err)
	}

	if err := jobs[0].RunNow(); err != nil {
		t.Fatalf("run job now err: %v", err)
	}

	select {
	case <-runCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("archive status checker was not triggered")
	}
}

func newArchiveStatusCheckQueryHooks(missing, unhealthy map[string][]int) archiveStatusCheckHooks {
	return archiveStatusCheckHooks{
		queryMissing: func(_ context.Context, backend string, _ int) ([]int, error) {
			return append([]int(nil), missing[backend]...), nil
		},
		queryUnhealthy: func(_ context.Context, backend string, _ int) ([]int, error) {
			return append([]int(nil), unhealthy[backend]...), nil
		},
	}
}
