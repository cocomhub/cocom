// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/comic"
	comicStorage "github.com/cocomhub/cocom/pkg/comic/storage"
	"github.com/cocomhub/cocom/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage 实现comic.ComicStorage接口
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
	cid, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comic id: %w", err)
	}

	info := &api.ComicInfo{}
	err = GetComicInfo(ctx, cid, info)
	if err != nil {
		return nil, fmt.Errorf("failed to get comic: %w", err)
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

	if c == nil || c.CID == 0 {
		return fmt.Errorf("invalid comic info")
	}

	if iErr := archiveComic(ctx, c.ComicInfo, false); iErr != nil {
		slog.WarnContext(ctx, "failed to archive comic", slog.String("err", iErr.Error()))
	}

	v, err := util.ToMap(c)
	if err != nil {
		return fmt.Errorf("failed to convert comic info to map: %w", err)
	}

	err = UpdateComicInfo(ctx, c.CID, v)
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
	cursor, err := mongo.ComicInfo().Find(ctx, s.toMongoFilter(ctx, filter), &options.FindOptions{
		Sort:  bson.M{"cid": 1},
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var infos []api.ComicInfo
	if err := cursor.All(ctx, &infos); err != nil {
		return nil, err
	}

	// 转换为接口类型
	comics := make([]comic.Comic, len(infos))
	for i := range infos {
		comics[i] = NewComic(&infos[i])
	}
	return comics, nil
}

// FindTotal 列出符合条件的漫画总数
func (s *Storage) FindTotal(ctx context.Context, filter *comic.ComicFilter) (int64, error) {
	if s.inner != nil {
		return s.inner.FindTotal(ctx, filter)
	}
	return mongo.ComicInfo().CountDocuments(ctx, s.toMongoFilter(ctx, filter), &options.CountOptions{
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *Storage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	if s.inner != nil {
		return s.inner.FindChannel(ctx, filter)
	}
	advance := func(impls []comic.Comic, f *comic.ComicFilter) {
		if filter.NotArchived != nil && *filter.NotArchived {
			cid, err := strconv.Atoi(impls[len(impls)-1].GetID())
			if err != nil {
				slog.ErrorContext(ctx, "invalid comic id", slog.String("id", impls[len(impls)-1].GetID()))
			} else {
				f.IDRangeLeft = new(int64(cid + 1))
			}
			f.Skip = 0
			return
		}
		f.Skip += int64(len(impls))
	}
	return comicStorage.FindChannelHelper(ctx, filter, s.Find, advance)
}

func (s *Storage) toMongoFilter(ctx context.Context, filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		cid, err := strconv.Atoi(*filter.ID)
		if err != nil {
			slog.ErrorContext(ctx, "invalid comic id", slog.String("id", *filter.ID))
		} else {
			mongoFilter["cid"] = cid
		}
	} else {
		idFilter := bson.M{}
		if filter.IDRangeLeft != nil {
			idFilter["$gte"] = *filter.IDRangeLeft
		}
		if filter.IDRangeRight != nil {
			idFilter["$lte"] = *filter.IDRangeRight
		}
		if len(idFilter) != 0 {
			mongoFilter["cid"] = idFilter
		}
	}
	if filter.TitlePattern != nil {
		mongoFilter["$or"] = []bson.M{
			{"title.english": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
			{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
			{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}},
		}
	}
	if filter.PageMin != nil && filter.PageMax != nil {
		mongoFilter["num_pages"] = bson.M{"$gte": *filter.PageMin, "$lte": *filter.PageMax}
	} else if filter.PageMin != nil {
		mongoFilter["num_pages"] = bson.M{"$gte": *filter.PageMin}
	} else if filter.PageMax != nil {
		mongoFilter["num_pages"] = bson.M{"$lte": *filter.PageMax}
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
	cid, err := strconv.Atoi(result.ComicID)
	if err != nil {
		return fmt.Errorf("invalid comic id: %w", err)
	}

	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	err = UpdateComicInfo(ctx, cid, map[string]any{
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
	cid, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid comic id: %w", err)
	}
	info := &api.ComicInfo{}
	if err := GetComicInfo(ctx, cid, info); err != nil {
		return fmt.Errorf("failed to get comic: %w", err)
	}
	force := false
	if v := ctx.Value("archive.force"); v != nil {
		if b, ok := v.(bool); ok && b {
			force = true
		}
	}
	if err := archiveComic(ctx, info, force); err != nil {
		return fmt.Errorf("archive comic failed: %w", err)
	}
	archiveInfo, err := util.ToMap(info.Archive)
	if err != nil {
		return fmt.Errorf("convert archive info to map failed: %w", err)
	}
	if err := UpdateComicInfo(ctx, cid, map[string]any{
		"archive": archiveInfo,
	}); err != nil {
		return fmt.Errorf("failed to update archive info: %w", err)
	}
	return nil
}

func (s *Storage) RestoreByID(ctx context.Context, id string) error {
	if s.inner != nil {
		return s.inner.RestoreByID(ctx, id)
	}
	cid, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid comic id: %w", err)
	}
	if err := RestoreComicByID(ctx, cid); err != nil {
		return fmt.Errorf("restore comic failed: %w", err)
	}
	return nil
}

// FindByTags 查找包含指定 tagType 中任意 tag ID 的其他漫画
func (s *Storage) FindByTags(ctx context.Context, tags []comic.Tag, tagType string, cid int, limit int) ([]comic.Comic, error) {
	if s.inner != nil {
		return s.inner.FindByTags(ctx, tags, tagType, cid, limit)
	}

	// 转换为 api.Tags
	apiTags := make(api.Tags, len(tags))
	for i, tag := range tags {
		apiTags[i] = api.Tag{
			Count: tag.Count,
			ID:    tag.ID,
			Name:  tag.Name,
			Type:  tag.Type,
			URL:   tag.URL,
		}
	}

	infos, err := GetByTagType(ctx, cid, apiTags, tagType, limit)
	if err != nil {
		return nil, fmt.Errorf("find by tags failed: %w", err)
	}

	comics := make([]comic.Comic, len(infos))
	for i := range infos {
		comics[i] = NewComic(infos[i])
	}
	return comics, nil
}

// SearchTags 搜索标签
func (s *Storage) SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]comic.TagInfo, int64, error) {
	if s.inner != nil {
		return s.inner.SearchTags(ctx, tagType, query, limit)
	}
	tags, err := tag.SearchTags(ctx, tagType, query, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]comic.TagInfo, len(tags))
	for i, t := range tags {
		result[i] = comic.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like}
	}
	return result, int64(len(result)), nil
}

// ListTags 列出标签
func (s *Storage) ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]comic.TagInfo, int64, error) {
	if s.inner != nil {
		return s.inner.ListTags(ctx, tagType, sortType, skip, limit, likedOnly)
	}
	tags, total, err := tag.AggregateTagList(ctx, tagType, sortType, skip, limit, likedOnly)
	if err != nil {
		return nil, 0, err
	}
	result := make([]comic.TagInfo, len(tags))
	for i, t := range tags {
		result[i] = comic.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like}
	}
	return result, total, nil
}
