// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tag

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"
)

// TagStore 标签聚合存储接口（测试用轻量包装）
// 覆盖原本直接读写 MongoDB comicTag/comicInfo 集合的操作
type TagStore interface {
	AggregateTagSectionIndices(ctx context.Context, tagType string, pageTagNum int, likedOnly bool) ([]*api.TagSectionIndex, error)
	CountTags(ctx context.Context, tagType string) (int64, error)
	GetTagByID(ctx context.Context, tagType string, id int) (*ComicTagDoc, error)
	GetMaxTagID(ctx context.Context) (int, error)
	GetTagByTypeName(ctx context.Context, tagType, tagName string) (*ComicTagDoc, error)
	GetTagByTypeURL(ctx context.Context, tagType, url string) (*ComicTagDoc, error)
	AggregateTags(ctx context.Context) error
	GetTags(ctx context.Context, tagType string, limit, skip int64) ([]*ComicTagDoc, error)
	UpdateComicTagIncremental(ctx context.Context, tagType string, tagID int, tagName, tagURL string, countDiff int) error
	GetSearchUniqueTags(ctx context.Context, query string, limit, skip int64) ([]*api.TagInfo, []int, int64, error)
	GetComputedRelatedTags(ctx context.Context, tagType, tagName string, limit int64) ([]*api.TagInfo, error)
}

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
	defaultTagStore      TagStore
	defaultComicStore    comic.Storage
	defaultLikeStore     LikeStore
	defaultRelationStore RelationStore
)

// SetDefaultTagStore 设置 DefaultTagStore
func SetDefaultTagStore(s TagStore) { defaultTagStore = s }

// GetDefaultTagStore 获取 DefaultTagStore
func GetDefaultTagStore() TagStore { return defaultTagStore }

// ResetDefaultTagStore 重置 DefaultTagStore
func ResetDefaultTagStore() { defaultTagStore = nil }

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

// resetAllStores 重置所有存储（测试用）
func ResetAllStores() {
	defaultTagStore = nil
	defaultComicStore = nil
	defaultLikeStore = nil
	defaultRelationStore = nil
}

// MemoryTagStore 内存标签存储
type MemoryTagStore struct {
	mu     sync.RWMutex
	tags   map[string][]*ComicTagDoc // tagType -> docs
	seq    int
	maxTag int
}

// NewMemoryTagStore 创建 MemoryTagStore
func NewMemoryTagStore() *MemoryTagStore {
	return &MemoryTagStore{
		tags:   make(map[string][]*ComicTagDoc),
		seq:    0,
		maxTag: 1000000000,
	}
}

// CountTags 实现 TagStore.CountTags
func (s *MemoryTagStore) CountTags(_ context.Context, tagType string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.tags[tagType])), nil
}

// AggregateTagSectionIndices 实现 TagStore.AggregateTagSectionIndices
func (s *MemoryTagStore) AggregateTagSectionIndices(_ context.Context, tagType string, pageTagNum int, likedOnly bool) ([]*api.TagSectionIndex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	docs := s.tags[tagType]
	// 过滤 likedOnly
	if likedOnly {
		liked := make([]*ComicTagDoc, 0, len(docs))
		for _, d := range docs {
			if d.Like {
				liked = append(liked, d)
			}
		}
		docs = liked
	}
	// 按名称首字符分组生成 section indices
	groupCounts := make(map[string]int)
	for _, d := range docs {
		if len(d.Name) == 0 {
			groupCounts["#"]++
			continue
		}
		first := string([]rune(d.Name)[0])
		if first >= "A" && first <= "Z" || first >= "a" && first <= "z" {
			groupCounts[first]++
		} else {
			groupCounts["#"]++
		}
	}
	indices := make([]*api.TagSectionIndex, 0, len(groupCounts))
	for name := range groupCounts {
		indices = append(indices, &api.TagSectionIndex{
			Name: name,
		})
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i].Name < indices[j].Name })
	// TagSectionIndex 没有 Count/Page 字段，返回按名称排序的索引列表即可
	return indices, nil
}

// GetTagByID 实现 TagStore.GetTagByID
func (s *MemoryTagStore) GetTagByID(_ context.Context, tagType string, id int) (*ComicTagDoc, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, doc := range s.tags[tagType] {
		if doc.ID == id {
			return doc, nil
		}
	}
	return nil, nil
}

// GetMaxTagID 实现 TagStore.GetMaxTagID
func (s *MemoryTagStore) GetMaxTagID(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxTag, nil
}

// GetTagByTypeName 实现 TagStore.GetTagByTypeName
func (s *MemoryTagStore) GetTagByTypeName(_ context.Context, tagType, tagName string) (*ComicTagDoc, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, doc := range s.tags[tagType] {
		if doc.Name == tagName {
			return doc, nil
		}
	}
	return nil, nil
}

// GetTagByTypeURL 实现 TagStore.GetTagByTypeURL
func (s *MemoryTagStore) GetTagByTypeURL(_ context.Context, tagType, url string) (*ComicTagDoc, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, doc := range s.tags[tagType] {
		if doc.URL == url {
			return doc, nil
		}
	}
	return nil, nil
}

// AggregateTags 实现 TagStore.AggregateTags（从 comics 聚合到 tag store）
func (s *MemoryTagStore) AggregateTags(_ context.Context) error {
	// 已在外部通过 SetDefaultComicStore 处理，这里无需操作
	return nil
}

// GetTags 实现 TagStore.GetTags
func (s *MemoryTagStore) GetTags(_ context.Context, tagType string, limit, skip int64) ([]*ComicTagDoc, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.tags[tagType]
	if int64(len(all)) < skip {
		return []*ComicTagDoc{}, nil
	}
	end := min(int64(len(all)), skip+limit)
	return all[skip:end], nil
}

// UpdateComicTagIncremental 实现 TagStore.UpdateComicTagIncremental
func (s *MemoryTagStore) UpdateComicTagIncremental(_ context.Context, tagType string, tagID int, tagName, tagURL string, countDiff int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, doc := range s.tags[tagType] {
		if doc.ID == tagID && doc.Name == tagName {
			doc.Count += countDiff
			if doc.Count <= 0 {
				// 移除
				updated := make([]*ComicTagDoc, 0, len(s.tags[tagType])-1)
				for _, d := range s.tags[tagType] {
					if d.ID != tagID {
						updated = append(updated, d)
					}
				}
				s.tags[tagType] = updated
			}
			return nil
		}
	}
	if countDiff > 0 {
		// 不存在则创建

		doc := &ComicTagDoc{
			Type:  tagType,
			ID:    tagID,
			Name:  tagName,
			URL:   tagURL,
			Count: countDiff,
		}
		s.tags[tagType] = append(s.tags[tagType], doc)
	}
	return nil
}

// GetSearchUniqueTags 实现 TagStore.GetSearchUniqueTags
func (s *MemoryTagStore) GetSearchUniqueTags(ctx context.Context, query string, limit, skip int64) ([]*api.TagInfo, []int, int64, error) {
	// 通过 comicStore 搜索
	store := GetDefaultComicStore()
	if store == nil {
		return nil, nil, 0, fmt.Errorf("comic store not set")
	}
	comics, err := store.Find(ctx, comic.NewComicFilter())
	if err != nil {
		return nil, nil, 0, err
	}
	// 收集所有 tag
	tagMap := make(map[string]*api.TagInfo)
	cidList := make([]int, 0, len(comics))
	for _, c := range comics {
		cid := 0
		_, _ = fmt.Sscanf(c.GetID(), "%d", &cid)
		cidList = append(cidList, cid)
		for _, t := range c.GetTags() {
			key := fmt.Sprintf("%s:%d", t.Type, t.ID)
			if _, ok := tagMap[key]; !ok {
				tagMap[key] = &api.TagInfo{
					ID:    t.ID,
					Name:  t.Name,
					Type:  t.Type,
					URL:   t.URL,
					Count: 0,
				}
			}
			tagMap[key].Count++
		}
	}
	// 排序
	result := make([]*api.TagInfo, 0, len(tagMap))
	for _, v := range tagMap {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	total := int64(len(result))
	// 分页
	if int64(len(result)) < skip {
		return []*api.TagInfo{}, cidList, total, nil
	}
	end := min(int64(len(result)), skip+limit)
	return result[skip:end], cidList, total, nil
}

// GetComputedRelatedTags 实现 TagStore.GetComputedRelatedTags
func (s *MemoryTagStore) GetComputedRelatedTags(ctx context.Context, tagType, tagName string, limit int64) ([]*api.TagInfo, error) {
	store := GetDefaultComicStore()
	if store == nil {
		return nil, fmt.Errorf("comic store not set")
	}
	comics, err := store.Find(ctx, comic.NewComicFilter())
	if err != nil {
		return nil, err
	}
	// 先找到包含指定 tag 的漫画
	targetIDs := make([]int, 0)
	for _, c := range comics {
		for _, t := range c.GetTags() {
			if t.Type == tagType && t.Name == tagName {
				id := 0
				_, _ = fmt.Sscanf(c.GetID(), "%d", &id)
				targetIDs = append(targetIDs, id)
				break
			}
		}
	}
	if len(targetIDs) == 0 {
		return []*api.TagInfo{}, nil
	}
	// 计算共现
	cooccur := make(map[string]*api.TagInfo)
	for _, c := range comics {
		cid := 0
		_, _ = fmt.Sscanf(c.GetID(), "%d", &cid)
		for _, t := range c.GetTags() {
			if t.Type == tagType && t.Name == tagName {
				continue
			}
			// 只取 target comic 中的 tag
			found := slices.Contains(targetIDs, cid)
			if !found {
				continue
			}
			key := fmt.Sprintf("%s:%d", t.Type, t.ID)
			entry, ok := cooccur[key]
			if !ok {
				cooccur[key] = &api.TagInfo{
					ID:    t.ID,
					Name:  t.Name,
					Type:  t.Type,
					URL:   t.URL,
					Count: 0,
				}
				entry = cooccur[key]
			}
			entry.Count++
		}
	}
	result := make([]*api.TagInfo, 0, len(cooccur))
	for _, v := range cooccur {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	if int64(len(result)) > limit {
		result = result[:limit]
	}
	return result, nil
}

// AddComicTag 手动添加一个标签（测试辅助）
func (s *MemoryTagStore) AddComicTag(tagType string, doc *ComicTagDoc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if doc.ID > s.maxTag {
		s.maxTag = doc.ID
	}
	s.tags[tagType] = append(s.tags[tagType], doc)
}

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
