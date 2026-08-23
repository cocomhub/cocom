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
