// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tag

import (
	"context"
	"fmt"
	"sync"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"
)

// LikeStore 标签点赞存储（测试用轻量包装）
type LikeStore interface {
	Like(ctx context.Context, tagType string, tagID int) error
	Unlike(ctx context.Context, tagType string, tagID int) error
	IsLiked(ctx context.Context, tagType string, tagID int) (bool, error)
}

// RelationStore 标签关系存储（测试用轻量包装）
type RelationStore interface {
	CreateRelation(ctx context.Context, tags []api.TagBrief) (string, error)
	DeleteRelation(ctx context.Context, groupID string) error
	GetRelationsForTag(ctx context.Context, tagType string, tagID int) ([]api.RelationGroup, error)
}

var (
	defaultComicStore    comic.Storage
	defaultLikeStore     LikeStore
	defaultRelationStore RelationStore
)

// SetDefaultComicStore 设置 DefaultComicStore
func SetDefaultComicStore(s comic.Storage) { defaultComicStore = s }

// GetDefaultComicStore 获取 DefaultComicStore
func GetDefaultComicStore() comic.Storage { return defaultComicStore }

// ResetDefaultComicStore 重置 DefaultComicStore
func ResetDefaultComicStore() { defaultComicStore = nil }

// SetDefaultLikeStore 设置 DefaultLikeStore
func SetDefaultLikeStore(s LikeStore) { defaultLikeStore = s }

// GetDefaultLikeStore 获取 DefaultLikeStore
func GetDefaultLikeStore() LikeStore { return defaultLikeStore }

// ResetDefaultLikeStore 重置 DefaultLikeStore
func ResetDefaultLikeStore() { defaultLikeStore = nil }

// SetDefaultRelationStore 设置 DefaultRelationStore
func SetDefaultRelationStore(s RelationStore) { defaultRelationStore = s }

// GetDefaultRelationStore 获取 DefaultRelationStore
func GetDefaultRelationStore() RelationStore { return defaultRelationStore }

// ResetDefaultRelationStore 重置 DefaultRelationStore
func ResetDefaultRelationStore() { defaultRelationStore = nil }

// MemoryLikeStore 内存点赞存储
type MemoryLikeStore struct {
	mu    sync.RWMutex
	likes map[string]bool
}

// NewMemoryLikeStore 创建 MemoryLikeStore
func NewMemoryLikeStore() *MemoryLikeStore {
	return &MemoryLikeStore{likes: make(map[string]bool)}
}

// Like 实现 LikeStore.Like
func (s *MemoryLikeStore) Like(_ context.Context, tagType string, tagID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likes[fmt.Sprintf("%s:%d", tagType, tagID)] = true
	return nil
}

// Unlike 实现 LikeStore.Unlike
func (s *MemoryLikeStore) Unlike(_ context.Context, tagType string, tagID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.likes[fmt.Sprintf("%s:%d", tagType, tagID)] = false
	return nil
}

// IsLiked 实现 LikeStore.IsLiked
func (s *MemoryLikeStore) IsLiked(_ context.Context, tagType string, tagID int) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.likes[fmt.Sprintf("%s:%d", tagType, tagID)], nil
}

// MemoryRelationStore 内存关系存储
type MemoryRelationStore struct {
	mu        sync.RWMutex
	relations []*relationEntry
	seq       int
}

type relationEntry struct {
	ID   string
	Tags []api.TagBrief
}

// NewMemoryRelationStore 创建 MemoryRelationStore
func NewMemoryRelationStore() *MemoryRelationStore {
	return &MemoryRelationStore{relations: make([]*relationEntry, 0)}
}

// CreateRelation 实现 RelationStore.CreateRelation
func (s *MemoryRelationStore) CreateRelation(_ context.Context, tags []api.TagBrief) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("mem-%d", s.seq)
	s.relations = append(s.relations, &relationEntry{ID: id, Tags: tags})
	return id, nil
}

// DeleteRelation 实现 RelationStore.DeleteRelation
func (s *MemoryRelationStore) DeleteRelation(_ context.Context, groupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.relations {
		if r.ID == groupID {
			s.relations = append(s.relations[:i], s.relations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("relation not found")
}

// GetRelationsForTag 实现 RelationStore.GetRelationsForTag
func (s *MemoryRelationStore) GetRelationsForTag(_ context.Context, tagType string, tagID int) ([]api.RelationGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []api.RelationGroup
	for _, r := range s.relations {
		for _, t := range r.Tags {
			if t.ID == tagID && t.Type == tagType {
				result = append(result, api.RelationGroup{
					ID:   r.ID,
					Tags: r.Tags,
				})
				break
			}
		}
	}
	if result == nil {
		return []api.RelationGroup{}, nil
	}
	return result, nil
}
