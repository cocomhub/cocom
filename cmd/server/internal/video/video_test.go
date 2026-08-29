// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package video

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestMemoryVideoStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryVideoStore()

	var info map[string]any
	err := s.Get(ctx, "missing", &info)
	if !errors.Is(err, ErrVideoNotFound) {
		t.Errorf("Get(missing) err = %v, want ErrVideoNotFound", err)
	}
}

func TestMemoryVideoStore_UpdateStripsIDAndAddsVid(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryVideoStore()

	// 与 Mongo 路径一致：_id 被剥离、vid 被附加到文档
	input := map[string]any{"id": "v1", "title": "T", "_id": "should_be_stripped"}
	if err := s.Update(ctx, "v1", input); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	var got map[string]any
	if err := s.Get(ctx, "v1", &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["vid"] != "v1" {
		t.Errorf("doc vid = %v, want v1", got["vid"])
	}
	if _, ok := got["_id"]; ok {
		t.Errorf("doc should not contain _id, got %v", got)
	}
	if got["title"] != "T" {
		t.Errorf("doc title = %v, want T", got["title"])
	}
}

func TestMemoryVideoStore_UpdateMergesPartial(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryVideoStore()

	if err := s.Update(ctx, "v2", map[string]any{"a": 1}); err != nil {
		t.Fatalf("Update #1 failed: %v", err)
	}
	if err := s.Update(ctx, "v2", map[string]any{"b": 2}); err != nil {
		t.Fatalf("Update #2 failed: %v", err)
	}

	var got map[string]any
	if err := s.Get(ctx, "v2", &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["a"] != float64(1) || got["b"] != float64(2) {
		t.Errorf("merged doc = %v, want {a:1 b:2}", got)
	}
}

func TestMemoryVideoStore_UpdateNilAndEmptyMap(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryVideoStore()

	// nil map 不应 panic
	if err := s.Update(ctx, "v3", nil); err != nil {
		t.Fatalf("Update(nil) failed: %v", err)
	}
	// 空 map 不应 panic
	if err := s.Update(ctx, "v3", map[string]any{}); err != nil {
		t.Fatalf("Update(empty) failed: %v", err)
	}
}

// TestDefaultVideoStore_ConcurrentSetResetGet 回归 Batch B：包级 defaultStore
// 在并发 Set/Reset/Get 下不得有数据竞争（用 -race 守护）。
func TestDefaultVideoStore_ConcurrentSetResetGet(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				SetDefaultVideoStore(NewMemoryVideoStore())
			} else {
				_ = GetDefaultVideoStore()
			}
			if n == 7 {
				ResetDefaultVideoStore()
			}
		}(i)
	}
	wg.Wait()
	ResetDefaultVideoStore()
}
