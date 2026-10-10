// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package setting

import (
	"context"
	"testing"
)

func TestMemorySettingsStore_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySettingsStore()

	if err := s.Set(ctx, "view", map[string]any{"key1": "val1", "key2": int64(42)}); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := s.Get(ctx, "view", "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(got) != 1 || got["key1"] != "val1" {
		t.Errorf("Get(key1) = %v, want {key1:val1}", got)
	}

	// Get 全部 keys
	all, err := s.Get(ctx, "view")
	if err != nil {
		t.Fatalf("Get all failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("Get(all) length = %d, want 2: %v", len(all), all)
	}

	// Del 指定 key
	deleted, err := s.Del(ctx, "view", "key1")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Del deleted = %d, want 1", deleted)
	}
	got, err = s.Get(ctx, "view")
	if err != nil {
		t.Fatalf("Get after Del failed: %v", err)
	}
	if _, ok := got["key1"]; ok {
		t.Errorf("key1 should be deleted, got %v", got)
	}
	if _, ok := got["key2"]; !ok {
		t.Errorf("key2 should remain, got %v", got)
	}
}

func TestMemorySettingsStore_SetEmptyKVs_NoOp(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySettingsStore()

	if err := s.Set(ctx, "view", nil); err != nil {
		t.Errorf("Set with nil kvs should be no-op, got error: %v", err)
	}
	if err := s.Set(ctx, "view", map[string]any{}); err != nil {
		t.Errorf("Set with empty kvs should be no-op, got error: %v", err)
	}

	got, err := s.Get(ctx, "view")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Get after empty Set = %v, want empty", got)
	}
}

func TestMemorySettingsStore_DelEmptyKeys_Error(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySettingsStore()

	if _, err := s.Del(ctx, "view"); err == nil {
		t.Error("Del with no keys should return error")
	}
	if _, err := s.Del(ctx, "view", ""); err == nil {
		t.Error("Del with empty key should return error")
	}
}

func TestMemorySettingsStore_DelNonexistentType(t *testing.T) {
	ctx := context.Background()
	s := NewMemorySettingsStore()

	deleted, err := s.Del(ctx, "nonexistent", "a")
	if err != nil {
		t.Fatalf("Del nonexistent type failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Del nonexistent type deleted = %d, want 0", deleted)
	}
}
