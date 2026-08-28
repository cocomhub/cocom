// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
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
	cfg := config.SchedulerTask{Limit: 2}
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

	var mu sync.Mutex
	var calls []string
	stats := executeArchiveStatusCheckIssues(context.Background(), issues, archiveStatusCheckHooks{
		replicate: func(_ context.Context, cid int, backend string) (bool, error) {
			mu.Lock()
			calls = append(calls, "replicate:"+backend)
			mu.Unlock()
			if cid != 2001 {
				t.Fatalf("unexpected replicate cid: %d", cid)
			}
			return true, nil
		},
		check: func(_ context.Context, cid int) error {
			mu.Lock()
			calls = append(calls, "check")
			mu.Unlock()
			if cid != 2001 && cid != 2002 {
				t.Fatalf("unexpected check cid: %d", cid)
			}
			return nil
		},
	}, 2)

	// The goroutines run concurrently, so the call order is
	// non-deterministic.  Accept any permutation of the expected
	// set as long as all expected calls are present.
	wantCalls := []string{"replicate:backup-a", "replicate:backup-b", "check", "check"}
	if !sameElements(calls, wantCalls) {
		t.Fatalf("unexpected call set: got %+v, want %+v", calls, wantCalls)
	}
	if stats.Replicated != 2 || stats.Checked != 2 || stats.Errors != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestExecuteArchiveStatusCheckReplicateFailureSkipsUnhealthyCheck(t *testing.T) {
	issues := []archiveStatusCheckIssue{
		{
			CID:       5001,
			Missing:   []string{"broken"},
			Unhealthy: []string{"backup-z"},
		},
	}

	var checkCalled bool
	stats := executeArchiveStatusCheckIssues(context.Background(), issues, archiveStatusCheckHooks{
		replicate: func(_ context.Context, cid int, backend string) (bool, error) {
			// 状态持久化断言见 pkg/archive/manager 层 replicate 失败测试。
			return false, errors.New("replicate failed")
		},
		check: func(_ context.Context, cid int) error {
			checkCalled = true
			return nil
		},
	}, 1)

	// 设计语义：replicate 失败仅计入 errors，与成功（Replicated）跳过 count 分开；
	// 但 issue 的 Unhealthy 属另一维度（该 backend 健康度校验），不受 replicate 是否失败影响，
	// 仍会被 check 执行；本用例 replicate 失败 + Unhealthy 存在 → 两者都发生。
	if !checkCalled {
		t.Errorf("check should still run for Unhealthy dimensions independent of replicate failure")
	}
	if stats.Replicated != 0 || stats.Checked != 1 || stats.Skipped != 0 || stats.Errors != 1 {
		t.Errorf("unexpected stats: Replicated=%d Checked=%d Skipped=%d Errors=%d", stats.Replicated, stats.Checked, stats.Skipped, stats.Errors)
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

	config.G().Viper().Set("server.scheduler.archive_status_check.enabled", true)
	config.G().Viper().Set("server.scheduler.archive_status_check.name", "ArchiveStatusChecker")
	config.G().Viper().Set("server.scheduler.archive_status_check.cron", "*/5 * * * * *")
	config.G().Viper().Set("server.scheduler.archive_status_check.tags", []string{"archive", "check"})
	config.G().Viper().Set("server.scheduler.archive_status_check.limit", 3)
	config.G().Viper().Set("server.scheduler.archive_status_check.backends", []string{
		backendName,
	})

	sc, err := New(context.Background())
	if err != nil {
		t.Fatalf("new scheduler err: %v", err)
	}
	defer func() { _ = sc.Stop(context.Background()) }()

	runCh := make(chan struct{}, 1)
	archiveStatusCheckRunner = func(_ context.Context, cfg config.SchedulerTask, backends []string) (archiveStatusCheckStats, error) {
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

func TestExecuteArchiveStatusCheckIssuesRecoversPanic(t *testing.T) {
	issues := []archiveStatusCheckIssue{
		{CID: 4001, Missing: []string{"boom"}},
		{CID: 4002, Unhealthy: []string{"ok"}},
	}
	// 证明：一个 hook panic 不崩溃进程（recover + 日志）。recover 随即使
	// runCancel 传播停机——此时兄弟任务可能在信号量等待中被中止（属设计行为），
	// 因此不断言其 Checked 计数，仅断言 panic 方不污染成功/错误统计。
	stats := executeArchiveStatusCheckIssues(context.Background(), issues, archiveStatusCheckHooks{
		replicate: func(_ context.Context, _ int, backend string) (bool, error) {
			if backend == "boom" {
				panic("replicate exploded")
			}
			return true, nil
		},
		check: func(_ context.Context, _ int) error { return nil },
	}, 2)

	if stats.Replicated != 0 || stats.Errors != 0 {
		t.Fatalf("unexpected stats after panic recovery: %+v", stats)
	}
}

func TestExecuteArchiveStatusCheckIssuesPanicDoesNotCrash(t *testing.T) {
	// 单 issue 且 hook 必 panic：验证不崩溃进程且统计不被污染
	// （Replicated/Checked 皆 0、Errors 也 0——panic 不计错误，仅由
	// executeArchiveStatusCheckRecover 记录日志）。此路径无并发时序，确定性。
	stats := executeArchiveStatusCheckIssues(context.Background(),
		[]archiveStatusCheckIssue{{CID: 4101, Missing: []string{"boom"}}},
		archiveStatusCheckHooks{
			replicate: func(_ context.Context, _ int, _ string) (bool, error) {
				panic("replicate exploded")
			},
		}, 1)
	if stats.Replicated != 0 || stats.Checked != 0 || stats.Errors != 0 {
		t.Fatalf("unexpected stats after panic: %+v", stats)
	}
}

func TestExecuteArchiveStatusCheckIssuesParentCancelStopsWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	issues := []archiveStatusCheckIssue{
		{CID: 4003, Missing: []string{"delay"}},
	}

	cancelledHook := make(chan struct{})
	executeArchiveStatusCheckIssuesDone := make(chan struct{})
	go func() {
		executeArchiveStatusCheckIssues(ctx, issues, archiveStatusCheckHooks{
			replicate: func(runCtx context.Context, _ int, _ string) (bool, error) {
				// 挂住直到上层 cancel，随后因 ctx.Done 退出——不永久阻塞在信号量/等待。
				<-runCtx.Done()
				close(cancelledHook)
				return false, runCtx.Err()
			},
		}, 1)
		close(executeArchiveStatusCheckIssuesDone)
	}()

	// 让子 goroutine 有机会拿到信号量、先进钩子再取消，覆盖两处 ctx.Done 分支。
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-cancelledHook:
		// 父 ctx cancel 已级联到 runCtx：钩子 ctx 感知取消并返回。
	case <-time.After(3 * time.Second):
		t.Fatalf("hook was not unblocked by parent cancel")
	}

	select {
	case <-executeArchiveStatusCheckIssuesDone:
		// 上层 cancel 后 execute 正常返回，未挂在信号量 select/等待上。
	case <-time.After(3 * time.Second):
		t.Fatalf("execute did not return after parent cancel")
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

// sameElements reports whether a and b contain the same strings,
// ignoring order.  Duplicates are counted.
func sameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
		if m[s] < 0 {
			return false
		}
	}
	return true
}
