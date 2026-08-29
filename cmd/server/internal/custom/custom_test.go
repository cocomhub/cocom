// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package custom

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryCustomStore_AddLikeGroupAndIsLiked(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryCustomStore()

	if s.IsLiked(1) {
		t.Error("IsLiked(1) = true before AddLikeGroup, want false")
	}
	if err := s.AddLikeGroup(ctx, 1); err != nil {
		t.Fatalf("AddLikeGroup failed: %v", err)
	}
	if !s.IsLiked(1) {
		t.Error("IsLiked(1) = false after AddLikeGroup, want true")
	}
	if s.IsLiked(2) {
		t.Error("IsLiked(2) = true for never-liked cid, want false")
	}
}

func TestMemoryCustomStore_Reset(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryCustomStore()

	if err := s.AddLikeGroup(ctx, 1); err != nil {
		t.Fatalf("AddLikeGroup(1) failed: %v", err)
	}
	if err := s.AddLikeGroup(ctx, 2); err != nil {
		t.Fatalf("AddLikeGroup(2) failed: %v", err)
	}

	s.Reset()
	if s.IsLiked(1) || s.IsLiked(2) {
		t.Error("after Reset, IsLiked should be false for all cids")
	}
}

func TestAddLikeGroup_DelegatesToDefaultStore(t *testing.T) {
	ctx := context.Background()
	ms := NewMemoryCustomStore()
	SetDefaultCustomStore(ms)
	defer ResetDefaultCustomStore()

	if err := AddLikeGroup(ctx, 42); err != nil {
		t.Fatalf("AddLikeGroup failed: %v", err)
	}
	if !ms.IsLiked(42) {
		t.Error("AddLikeGroup did not delegate to the default store")
	}
}

// TestDefaultCustomStore_ConcurrentSetResetGet 回归 Batch B：包级 defaultStore
// 并发 Set/Reset/Get 无数据竞争（-race 守护）。
func TestDefaultCustomStore_ConcurrentSetResetGet(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				SetDefaultCustomStore(NewMemoryCustomStore())
			} else {
				_ = GetDefaultCustomStore()
			}
			if n == 7 {
				ResetDefaultCustomStore()
			}
		}(i)
	}
	wg.Wait()
	ResetDefaultCustomStore()
}
