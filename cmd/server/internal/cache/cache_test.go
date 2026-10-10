// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/allegro/bigcache/v3"
)

// newTestCache 初始化全局缓存，并注册清理。
func newTestCache(t *testing.T) {
	t.Helper()
	Init(context.Background(), 10*time.Minute, 1*time.Minute)
	t.Cleanup(func() { _ = Close() })
}

func TestCacheRoundTrip(t *testing.T) {
	newTestCache(t)

	key := "test_roundtrip"
	val := map[string]any{"a": 1, "b": "x"}

	if err := Set(key, val); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var got map[string]any
	if err := Get(key, &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["a"] != float64(1) || got["b"] != "x" {
		t.Errorf("Get = %v, want {a:1 b:x}", got)
	}

	if err := Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	var after map[string]any
	if err := Get(key, &after); !errors.Is(err, bigcache.ErrEntryNotFound) {
		t.Errorf("Get after Delete err = %v, want ErrEntryNotFound", err)
	}
}

func TestCacheDeleteMissing_NoError(t *testing.T) {
	newTestCache(t)

	// Delete 对不存在的 key 应吞掉 ErrEntryNotFound 并返回 nil
	if err := Delete("missing_key"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

// TestCacheNotInitialized_NoPanic 验证未 Init 时所有访问器返回稳定哨兵/零值而不 panic。
func TestCacheNotInitialized_NoPanic(t *testing.T) {
	// 前置：确保全局缓存未被其他用例污染（本包用例均通过 newTestCache 配 Cleanup）。
	if cache != nil {
		if err := Close(); err != nil {
			t.Fatalf("Close residual cache: %v", err)
		}
	}
	t.Cleanup(func() {
		if cache != nil {
			_ = Close()
		}
	})

	if err := Get("k", &map[string]any{}); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("Get on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if err := Set("k", 1); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("Set on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if err := Delete("k"); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("Delete on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if err := Reset(); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("Reset on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if err := ResetStats(); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("ResetStats on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if err := Close(); !errors.Is(err, ErrCacheNotInitialized) {
		t.Errorf("Close on uninitialized cache err = %v, want ErrCacheNotInitialized", err)
	}
	if got := Cache(); got != nil {
		t.Errorf("Cache() = %v, want nil", got)
	}
	if got := Len(); got != 0 {
		t.Errorf("Len() on uninitialized cache = %d, want 0", got)
	}
	if got := Capacity(); got != 0 {
		t.Errorf("Capacity() on uninitialized cache = %d, want 0", got)
	}
	if got := Stats(); got != (bigcache.Stats{}) {
		t.Errorf("Stats() on uninitialized cache = %v, want zero-stats", got)
	}
	if got := KeyMetadata("k"); got != (bigcache.Metadata{}) {
		t.Errorf("KeyMetadata() on uninitialized cache = %v, want zero-metadata", got)
	}
	if got := Iterator(); got != nil {
		t.Errorf("Iterator() on uninitialized cache = %v, want nil", got)
	}
}

// TestCacheInitClosesOldInstance 验证 Init 重建语义：旧实例被关闭，新实例生效，
// 且旧值不再存在（保证重复 Init 不留下过期数据）。
func TestCacheInitClosesOldInstance(t *testing.T) {
	newTestCache(t)

	if err := Set("keep", "old"); err != nil {
		t.Fatalf("Set before re-init failed: %v", err)
	}

	// 再次 Init：旧实例应被关闭、替换为新实例
	Init(context.Background(), 10*time.Minute, 1*time.Minute)

	var gotAfter string
	if err := Get("keep", &gotAfter); !errors.Is(err, bigcache.ErrEntryNotFound) {
		t.Errorf("Get after re-init err = %v, want ErrEntryNotFound (旧值应随旧实例被丢弃)", err)
	}

	// 新实例维持可读写
	if err := Set("fresh", "new"); err != nil {
		t.Fatalf("Set after re-init failed: %v", err)
	}
	var v string
	if err := Get("fresh", &v); err != nil {
		t.Fatalf("Get after re-init failed: %v", err)
	}
}

func TestCacheGetMissing_ErrEntryNotFound(t *testing.T) {
	newTestCache(t)

	var out map[string]any
	if err := Get("no_such_key", &out); !errors.Is(err, bigcache.ErrEntryNotFound) {
		t.Errorf("Get(missing) err = %v, want ErrEntryNotFound", err)
	}
}

func TestCacheLenTracksEntries(t *testing.T) {
	newTestCache(t)

	if err := Set("len_a", 1); err != nil {
		t.Fatalf("Set len_a failed: %v", err)
	}
	if err := Set("len_b", 2); err != nil {
		t.Fatalf("Set len_b failed: %v", err)
	}
	if got := Len(); got < 2 {
		t.Errorf("Len() = %d, want >= 2", got)
	}

	if err := Delete("len_a"); err != nil {
		t.Fatalf("Delete len_a failed: %v", err)
	}
	if got := Len(); got < 1 {
		t.Errorf("Len() after delete = %d, want >= 1", got)
	}
}
