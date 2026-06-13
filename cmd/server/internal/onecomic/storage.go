// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package onecomic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/comic"
	comicStorage "github.com/cocomhub/cocom/pkg/comic/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage 实现comic.Storage接口
type Storage struct {
	// inner 可选：不为 nil 时所有操作委托给 inner（用于测试注入 mock）
	inner comic.Storage
}

// NewStorage 创建存储实例
func NewStorage() *Storage {
	return &Storage{}
}

// NewTestStorage 创建测试用存储实例，所有操作委托给 inner
func NewTestStorage(inner comic.Storage) *Storage {
	return &Storage{inner: inner}
}

// Get 获取漫画信息
func (s *Storage) Get(ctx context.Context, id string) (comic.Comic, error) {
	if s.inner != nil {
		return s.inner.Get(ctx, id)
	}
	info := &api.OneComicInfo{}
	err := GetOneComicInfo(ctx, id, info)
	if err != nil {
		return nil, fmt.Errorf("failed to get onecomic: %w", err)
	}
	return NewComic(info), nil
}

// Update 更新漫画数据
func (s *Storage) Update(ctx context.Context, obj any) error {
	if s.inner != nil {
		return s.inner.Update(ctx, obj)
	}
	c, err := NewComicByObject(obj)
	if err != nil {
		return err
	}

	if c == nil || c.Comicid == "" {
		return fmt.Errorf("invalid comic info")
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal comic info: %w", err)
	}
	v := map[string]any{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal comic info: %w", err)
	}

	err = UpdateOneComicInfo(ctx, c.Comicid, v)
	if err != nil {
		return fmt.Errorf("failed to save comic: %w", err)
	}
	return nil
}

// Find 列出符合条件的漫画
func (s *Storage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	if s.inner != nil {
		return s.inner.Find(ctx, filter)
	}
	cursor, err := mongo.OneComicInfo().Find(ctx, s.toMongoFilter(filter), &options.FindOptions{
		Sort:  bson.M{"comicid": 1},
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var impls []Comic
	if err := cursor.All(ctx, &impls); err != nil {
		return nil, err
	}

	// 转换为接口类型
	comics := make([]comic.Comic, len(impls))
	for i := range impls {
		comics[i] = &impls[i]
	}
	return comics, nil
}

// FindTotal 列出符合条件的漫画总数
func (s *Storage) FindTotal(ctx context.Context, filter *comic.ComicFilter) (int64, error) {
	if s.inner != nil {
		return s.inner.FindTotal(ctx, filter)
	}
	return mongo.OneComicInfo().CountDocuments(ctx, s.toMongoFilter(filter), &options.CountOptions{
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *Storage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	if s.inner != nil {
		return s.inner.FindChannel(ctx, filter)
	}
	return comicStorage.FindChannelHelper(ctx, filter, s.Find, nil)
}

func (s *Storage) toMongoFilter(filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		mongoFilter["comicid"] = *filter.ID
	} else {
		idFilter := bson.M{}
		if filter.IDRangeLeft != nil {
			idFilter["$gte"] = *filter.IDRangeLeft
		}
		if filter.IDRangeRight != nil {
			idFilter["$lte"] = *filter.IDRangeRight
		}
		if len(idFilter) != 0 {
			mongoFilter["comicid"] = idFilter
		}
	}
	if filter.TitlePattern != nil {
		mongoFilter["name"] = bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}
	}
	if filter.Valid != nil {
		mongoFilter["verify.valid"] = *filter.Valid
	}
	if filter.HasValid != nil {
		if *filter.HasValid {
			mongoFilter["verify.valid"] = bson.M{"$exists": 1}
		} else {
			mongoFilter["verify.valid"] = bson.M{"$exists": 0}
		}
	}
	if filter.NotArchived != nil {
		if *filter.NotArchived {
			mongoFilter["archive.path"] = bson.M{"$exists": 0}
		} else {
			mongoFilter["archive.path"] = bson.M{"$exists": 1}
		}
	}
	if filter.Status != nil {
		mongoFilter["status"] = *filter.Status
	}
	if filter.Deleted != nil {
		mongoFilter["deleted"] = *filter.Deleted
	}
	if filter.HasRedirect != nil {
		if *filter.HasRedirect {
			mongoFilter["redirect_to"] = bson.M{"$exists": true}
		} else {
			mongoFilter["redirect_to"] = bson.M{"$exists": false}
		}
	}
	if len(filter.TitleORPatterns) > 0 {
		orConditions := make([]bson.M, 0, len(filter.TitleORPatterns))
		for _, pattern := range filter.TitleORPatterns {
			orConditions = append(orConditions, bson.M{
				"$or": []bson.M{
					{"title.english": bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}},
					{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}},
					{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}},
				},
			})
		}
		mongoFilter["$or"] = orConditions
	}

	return mongoFilter
}

// SaveVerifyResult 保存验证结果
func (s *Storage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
	if s.inner != nil {
		return s.inner.SaveVerifyResult(ctx, result)
	}
	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	err := UpdateOneComicInfo(ctx, result.ComicID, map[string]any{
		"verify": verifyInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to save verify result: %w", err)
	}
	return nil
}

func (s *Storage) ArchiveByID(ctx context.Context, id string) error {
	if s.inner != nil {
		return s.inner.ArchiveByID(ctx, id)
	}
	return fmt.Errorf("not supported")
}

func (s *Storage) RestoreByID(ctx context.Context, id string) error {
	if s.inner != nil {
		return s.inner.RestoreByID(ctx, id)
	}
	return fmt.Errorf("not supported")
}

// FindByTags 查找包含指定 tagType 中任意 tag ID 的其他漫画
func (s *Storage) FindByTags(ctx context.Context, tags []comic.Tag, tagType string, cid int, limit int) ([]comic.Comic, error) {
	if s.inner != nil {
		return s.inner.FindByTags(ctx, tags, tagType, cid, limit)
	}
	return nil, fmt.Errorf("not supported")
}

// SearchTags 搜索标签
func (s *Storage) SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]comic.TagInfo, int64, error) {
	if s.inner != nil {
		return s.inner.SearchTags(ctx, tagType, query, limit)
	}
	return nil, 0, fmt.Errorf("not supported")
}

// ListTags 列出标签
func (s *Storage) ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]comic.TagInfo, int64, error) {
	if s.inner != nil {
		return s.inner.ListTags(ctx, tagType, sortType, skip, limit, likedOnly)
	}
	return nil, 0, fmt.Errorf("not supported")
}