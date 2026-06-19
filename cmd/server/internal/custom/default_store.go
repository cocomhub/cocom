// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package custom

import (
	"context"
	"sync"
)

// CustomStore 自定义存储接口（测试用轻量包装）
type CustomStore interface {
	AddLikeGroup(ctx context.Context, cid int) error
}

var defaultStore CustomStore

// SetDefaultCustomStore 设置 DefaultCustomStore
func SetDefaultCustomStore(s CustomStore) { defaultStore = s }

// GetDefaultCustomStore 获取 DefaultCustomStore
func GetDefaultCustomStore() CustomStore { return defaultStore }

// ResetDefaultCustomStore 重置 DefaultCustomStore
func ResetDefaultCustomStore() { defaultStore = nil }

// MemoryCustomStore 内存自定义存储
type MemoryCustomStore struct {
	mu   sync.Mutex
	data map[int]bool
}

// NewMemoryCustomStore 创建 MemoryCustomStore
func NewMemoryCustomStore() *MemoryCustomStore {
	return &MemoryCustomStore{data: make(map[int]bool)}
}

// AddLikeGroup 实现 CustomStore.AddLikeGroup
func (s *MemoryCustomStore) AddLikeGroup(_ context.Context, cid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cid] = true
	return nil
}

// IsLiked 检查 cid 是否已被点赞（测试辅助）
func (s *MemoryCustomStore) IsLiked(cid int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[cid]
}

// Reset 清空所有数据
func (s *MemoryCustomStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.data)
}

var _ CustomStore = (*MemoryCustomStore)(nil)
