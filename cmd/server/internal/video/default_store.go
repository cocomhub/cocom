// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package video

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"
)

// VideoStore 视频存储接口（测试用轻量包装）
type VideoStore interface {
	Get(ctx context.Context, vid string, info any) error
	Update(ctx context.Context, vid string, m map[string]any) error
}

var defaultStore atomic.Pointer[VideoStore]

// SetDefaultVideoStore 设置 DefaultVideoStore
func SetDefaultVideoStore(s VideoStore) { defaultStore.Store(&s) }

// GetDefaultVideoStore 获取 DefaultVideoStore
func GetDefaultVideoStore() VideoStore {
	if p := defaultStore.Load(); p != nil {
		return *p
	}
	return nil
}

// ResetDefaultVideoStore 重置 DefaultVideoStore
func ResetDefaultVideoStore() { defaultStore.Store(nil) }

// MemoryVideoStore 内存视频存储
type MemoryVideoStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

// NewMemoryVideoStore 创建 MemoryVideoStore
func NewMemoryVideoStore() *MemoryVideoStore {
	return &MemoryVideoStore{data: make(map[string]map[string]any)}
}

// Get 实现 VideoStore.Get
func (s *MemoryVideoStore) Get(_ context.Context, vid string, info any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.data[vid]
	if !ok {
		return ErrVideoNotFound
	}
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal memory video failed: %w", err)
	}
	return json.Unmarshal(data, info)
}

// Update 实现 VideoStore.Update
// 与 Mongo 路径一致：剥离 _id、附加 vid，使两路径文档形态相同。
func (s *MemoryVideoStore) Update(_ context.Context, vid string, m map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[vid] == nil {
		s.data[vid] = make(map[string]any)
	}
	doc := make(map[string]any, len(m)+1)
	maps.Copy(doc, m)
	delete(doc, "_id")
	doc["vid"] = vid
	maps.Copy(s.data[vid], doc)
	return nil
}
