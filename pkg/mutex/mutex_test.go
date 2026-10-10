// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func countLocalMutexes(provider *LocalProvider) int {
	cnt := 0
	provider.m.Range(func(key, value any) bool {
		cnt++
		return true
	})
	return cnt
}

func TestSameKeyMutexSerial(t *testing.T) {
	ctx := context.Background()
	provider := current.(*LocalProvider)
	cnt := countLocalMutexes(provider)
	if cnt != 0 {
		t.Fatalf("mutex should not be locked")
	}
	seq := make(chan int, 4)
	var wg sync.WaitGroup
	wg.Go(func() {
		With(ctx, "k", func() {
			seq <- 1
			time.Sleep(50 * time.Millisecond)
			seq <- 2
		})
	})
	wg.Go(func() {
		time.Sleep(10 * time.Millisecond)
		With(ctx, "k", func() {
			seq <- 3
			seq <- 4
		})
	})

	time.Sleep(5 * time.Millisecond)

	cnt = countLocalMutexes(provider)
	if cnt != 1 {
		t.Fatalf("provider mutex should be 1 before unlock, cnt=%d", cnt)
	}

	wg.Wait()

	cnt = countLocalMutexes(provider)
	if cnt != 0 {
		t.Fatalf("provider mutex should be 0 after unlock, cnt=%d", cnt)
	}

	close(seq)
	got := []int{}
	for v := range seq {
		got = append(got, v)
	}
	want := []int{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d, want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch got=%v want=%v", got, want)
		}
	}
}

func TestDifferentKeysParallel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		var wg sync.WaitGroup
		wg.Go(func() {
			With(ctx, "a", func() {
				time.Sleep(200 * time.Millisecond)
			})
		})
		wg.Go(func() {
			With(ctx, "b", func() {
				time.Sleep(200 * time.Millisecond)
			})
		})
		wg.Wait()
		elapsed := time.Since(start)
		if elapsed > 350*time.Millisecond {
			t.Fatalf("locks on different keys should run in parallel, elapsed=%v", elapsed)
		}
	})
}

// TestSameKeyMutexConcurrent 复现并验证 R31 竞态：多个 goroutine 交错
// Lock/Unlock 同一 key，断言临界区最大并发数为 1（互斥不失效）。
func TestSameKeyMutexConcurrent(t *testing.T) {
	ctx := context.Background()
	provider := NewLocalProvider()

	const n = 16
	var active atomic.Int32
	var violated atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := provider.Lock(ctx, "k")
			if err != nil {
				t.Errorf("lock: %v", err)
				return
			}
			// 互斥成立时 active 恒为 1；出现并发进入则 > 1。
			if active.Add(1) != 1 {
				violated.Store(true)
			}
			time.Sleep(time.Millisecond) // 放大交错窗口
			active.Add(-1)
			unlock()
		}()
	}
	wg.Wait()

	if violated.Load() {
		t.Fatalf("mutual exclusion violated: concurrent critical section detected")
	}
	if cnt := countLocalMutexes(provider); cnt != 0 {
		t.Fatalf("expected 0 lockers after all done, got %d", cnt)
	}
}

// TestLockContextCancellation 验证 R6-S6：等待资源锁的请求可被 ctx 取消，
// 且取消后正确回滚 count（不影响持锁者）。
func TestLockContextCancellation(t *testing.T) {
	provider := NewLocalProvider()
	ctx := context.Background()

	unlock, err := provider.Lock(ctx, "k")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}

	waitCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, lockErr := provider.Lock(waitCtx, "k")
		errCh <- lockErr
	}()
	cancel()

	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	// waiter 已回滚 count，持锁者仍持有 → registry 应为 1。
	if cnt := countLocalMutexes(provider); cnt != 1 {
		t.Fatalf("expected 1 held lock after waiter cancel, got %d", cnt)
	}

	unlock()
	if cnt := countLocalMutexes(provider); cnt != 0 {
		t.Fatalf("expected 0 after unlock, got %d", cnt)
	}
}

// TestUnlockIdempotent 验证 unlock 幂等：二次调用不 panic（替代 double-unlock panic）。
func TestUnlockIdempotent(t *testing.T) {
	provider := NewLocalProvider()
	ctx := context.Background()

	unlock, err := provider.Lock(ctx, "k")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	unlock()
	unlock() // 二次调用应静默忽略

	if cnt := countLocalMutexes(provider); cnt != 0 {
		t.Fatalf("expected 0 after idempotent unlock, got %d", cnt)
	}
}

func TestLockLocalCompatible(t *testing.T) {
	ctx := context.Background()
	done := make(chan struct{})
	unlock, err := Lock(ctx, "x")
	if err != nil {
		t.Fatalf("lock err: %v", err)
	}
	go func() {
		With(ctx, "x", func() {
			close(done)
		})
	}()
	select {
	case <-done:
		t.Fatalf("should not enter while locked")
	case <-time.After(50 * time.Millisecond):
	}
	unlock()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("should enter after unlock")
	}
}
