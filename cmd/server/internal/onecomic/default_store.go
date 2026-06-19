// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package onecomic

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
)

// OneComicStore 单漫画存储接口（测试用轻量包装）
type OneComicStore interface {
	Get(ctx context.Context, cid string, info any) error
	Update(ctx context.Context, cid string, m map[string]any) error
}

var defaultStore OneComicStore

// SetDefaultOneComicStore 设置 DefaultOneComicStore
func SetDefaultOneComicStore(s OneComicStore) { defaultStore = s }

// GetDefaultOneComicStore 获取 DefaultOneComicStore
func GetDefaultOneComicStore() OneComicStore { return defaultStore }

// ResetDefaultOneComicStore 重置 DefaultOneComicStore
func ResetDefaultOneComicStore() { defaultStore = nil }

// MemoryOneComicStore 内存单漫画存储
type MemoryOneComicStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

// NewMemoryOneComicStore 创建 MemoryOneComicStore
func NewMemoryOneComicStore() *MemoryOneComicStore {
	return &MemoryOneComicStore{data: make(map[string]map[string]any)}
}

// Get 实现 OneComicStore.Get
func (s *MemoryOneComicStore) Get(_ context.Context, cid string, info any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.data[cid]
	if !ok {
		return fmt.Errorf("onecomic not found: %s", cid)
	}
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal memory onecomic failed: %w", err)
	}
	return json.Unmarshal(data, info)
}

// Update 实现 OneComicStore.Update
func (s *MemoryOneComicStore) Update(_ context.Context, cid string, m map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[cid] == nil {
		s.data[cid] = make(map[string]any)
	}
	maps.Copy(s.data[cid], m)
	return nil
}
