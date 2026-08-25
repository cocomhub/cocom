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
// 注意：onecomic 无 default-storage 注入语义（内部包级 GetOneComicInfo 走
// defaultStore 即 OneComicStore 接口，不会递归回本 Storage），因此无需自递归 guard。
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
// 注意：filter 允许为 nil。CountTotalOneComicInfos 从外部可能传 nil filter，
// toMongoFilter 已返回空 map；此处若无 nil 保护，filter.GetLimit() 会对 nil 指针取址崩溃。
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
	defer func() { _ = cursor.Close(ctx) }()

	// 解码到值类型 []api.OneComicInfo 再逐条包装，避免 BSON 解码匿名嵌入指针
	// (*api.OneComicInfo) 不分配导致 Comic.GetID/MarshalJSON 解引用 nil 指针。
	// 与 cmd/server/internal/comic/storage.go 的 Find 安全模式同构。
	var infos []api.OneComicInfo
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
// filter 允许为 nil（tolMongoFilter 已处理），与 Find 保持一致。
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

	// onecomic schema（见 cmd/server/api/onecomic.go）与主漫画集合差异适配：
	// - Status 为 string 类型（见 OneComicInfo.Status json/bson tag），ComicFilter.Status 是 *bool，
	//   用 fmt.Sprint(bool) 转换为 "true"/"false" 匹配文档中的 string 值，而非直接写入 bool（schema 失配）。
	// - comicid 是 string 类型（如 "[site]id"），其数值比较仅适用于纯数字串；
	//   范围过滤（IDRangeLeft/Right）对非数字 comicid 不生效，仅作既有兼容保留。
	// - 无 archive/redirect_to/deleted 字段：NotArchived/HasRedirect/Deleted 不作为过滤条件（下方注释 + 保留标签搜索）。
	// - TitleORPatterns 已映射到 name 字段（逻辑与 else 分支合并）。

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
	if filter.Status != nil {
		// onecomic.Status 是 string 字段，用字符串 "true"/"false" 匹配
		mongoFilter["status"] = fmt.Sprint(*filter.Status)
	}
	// onecomic schema 无 deleted 字段，Deleted 过滤不生效（保留调用方兼容，不做条件）
	if filter.NotArchived != nil {
		// onecomic schema 无 archive 字段，NotArchived 过滤不生效（保留调用方语义注释）
		_ = filter.NotArchived
	}
	// onecomic schema 无 redirect_to 字段，HasRedirect 过滤不生效（保留调用方语义注释）
	if filter.HasRedirect != nil {
		_ = filter.HasRedirect
	}
	if len(filter.TitleORPatterns) > 0 {
		orConditions := make([]bson.M, 0, len(filter.TitleORPatterns))
		for _, pattern := range filter.TitleORPatterns {
			orConditions = append(orConditions, bson.M{
				"$or": []bson.M{
					{"name": bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}},
				},
			})
		}
		// onecomic 标题字段是单字段 name，多 OR 条件全部作用于 name 字段
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
