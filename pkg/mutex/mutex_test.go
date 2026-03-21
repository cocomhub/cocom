// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package mutex

import (
	"context"
	"sync"
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
