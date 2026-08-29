// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"

	"github.com/cocomhub/cocom/pkg/comic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStorage MongoDB 实现的存储（实现了 comic.Storage 接口）
type MongoStorage struct {
	db *mongo.Database
}

// NewMongoStorage 创建 MongoDB 存储实例
func NewMongoStorage(db *mongo.Database) *MongoStorage {
	return &MongoStorage{db: db}
}

// ComicStorage 检查 MongoStorage 是否满足 comic.Storage 接口
var _ comic.Storage = (*MongoStorage)(nil)

// 注意：本实现的存储键为 _id（ComicImpl.ID 的 bson 标签是 _id，见 pkg/comic/comic.go），
// 因此 Get/Update/Find/SaveVerifyResult 全部按 _id 过滤，文档中不存在 cid 字段。
// 业务唯一键 cid 只用于 cmd/server 主业务（comicInfo/oneComicInfo 集合，
// 见 cmd/server/internal/comic/comic_info.go / onecomic），勿将本实现的过滤键改回 cid。

// Get 实现 ComicStorage 接口
func (s *MongoStorage) Get(ctx context.Context, id string) (comic.Comic, error) {
	var comic comic.ComicImpl
	err := s.db.Collection("comics").FindOne(ctx, bson.M{"_id": id}).Decode(&comic)
	if err != nil {
		return nil, err
	}
	return &comic, nil
}

// Update 实现 ComicStorage 接口
func (s *MongoStorage) Update(ctx context.Context, obj any) error {
	comic, err := comic.NewComicImplByObject(obj)
	if err != nil {
		return err
	}

	if comic == nil || comic.GetID() == "" {
		return fmt.Errorf("invalid comic info")
	}

	// 过滤键对齐 _id（与 Get/SaveVerifyResult 一致）；ComicImpl.ID 的 bson 标签为 _id，
	// 文档并不存在 cid 字段。原 {cid: ...} 过滤永不匹配且无 upsert，导致验证结果静默丢弃。
	res, err := s.db.Collection("comics").UpdateOne(ctx, bson.M{"_id": comic.GetID()}, comic)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("comic %s not found", comic.GetID())
	}
	return nil
}

// Find 实现 ComicStorage 接口
func (s *MongoStorage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := s.db.Collection("comics").Find(ctx, s.toMongoFilter(filter), &options.FindOptions{
		Sort:  bson.M{"_id": 1},
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var impls []comic.ComicImpl
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
func (s *MongoStorage) FindTotal(ctx context.Context, filter *comic.ComicFilter) (int64, error) {
	return s.db.Collection("comics").CountDocuments(ctx, s.toMongoFilter(filter), &options.CountOptions{
		Limit: filter.GetLimit(),
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *MongoStorage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	return FindChannelHelper(ctx, filter, s.Find, nil)
}

func (s *MongoStorage) toMongoFilter(filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	// 过滤键用 _id（ComicImpl.ID 的 bson 标签为 _id，文档无 cid 字段）。
	// IDRange 数字范围对 string _id 不适用（BSON 跨类型比较：string 与 int 无法
	// 可靠 $gte/$lte，得到空结果），当前按 string 精确匹配来承载范围过滤语义；
	// 有意不改集合数据格式（此模块的存储键为 string 型 _id）。
	if filter.ID != nil {
		mongoFilter["_id"] = *filter.ID
	} else {
		idFilter := bson.M{}
		if filter.IDRangeLeft != nil {
			idFilter["$gte"] = *filter.IDRangeLeft
		}
		if filter.IDRangeRight != nil {
			idFilter["$lte"] = *filter.IDRangeRight
		}
		if len(idFilter) != 0 {
			mongoFilter["_id"] = idFilter
		}
	}
	if filter.TitlePattern != nil {
		mongoFilter["title"] = bson.M{"$regex": primitive.Regex{Pattern: *filter.TitlePattern, Options: "i"}}
	}
	if filter.PageMin != nil && filter.PageMax != nil {
		mongoFilter["$expr"] = bson.M{
			"$gte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMin},
			"$lte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMax},
		}
	} else if filter.PageMin != nil {
		mongoFilter["$expr"] = bson.M{"$gte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMin}}
	} else if filter.PageMax != nil {
		mongoFilter["$expr"] = bson.M{"$lte": bson.A{bson.M{"$size": "$tags"}, *filter.PageMax}}
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
		// 与 cmd/server/internal/comic/storage.go 的 TitleORPatterns 语义对齐：
		// 多模式 OR，每个模式内部对英文/日文/pretty 标题做 OR 匹配。
		// 此集合文档模型为扁平 title（非 {title.english} 子文档），
		// 多数模式直接用 title 字段正则即可命中同一语义。
		orConditions := make([]bson.M, 0, len(filter.TitleORPatterns))
		for _, pattern := range filter.TitleORPatterns {
			orConditions = append(orConditions, bson.M{
				"title": bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}},
			})
		}
		mongoFilter["$or"] = orConditions
	}
	// TagIDs 当前不支持：存储键为 string _id，ComicImpl.Tags 字段语义不明确
	//（此模块文档的 tags 与业务中间产物重名），不做 $in/$elemMatch 静默猜测，
	// 需要时在调用方业务层明确过滤（见 MemoryStorage.Find 的 TagIDs 语义）。

	return mongoFilter
}

// SaveVerifyResult 实现 ComicStorage 接口
func (s *MongoStorage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
	verifyInfo := comic.VerifyInfo{}
	verifyInfo.SetVerifyResult(result)
	_, err := s.db.Collection("comics").UpdateOne(
		ctx,
		bson.M{"_id": result.ComicID},
		bson.M{"$set": bson.M{
			"verify": verifyInfo,
		}},
	)
	if err != nil {
		return err
	}

	_, err = s.db.Collection("verify_results").InsertOne(ctx, result)
	return err
}

func (s *MongoStorage) ArchiveByID(ctx context.Context, id string) error {
	return fmt.Errorf("not supported")
}

func (s *MongoStorage) RestoreByID(ctx context.Context, id string) error {
	return fmt.Errorf("not supported")
}

// FindByTags 查找包含指定 tagType 中任意 tag ID 的其他漫画
func (s *MongoStorage) FindByTags(ctx context.Context, tags []comic.Tag, tagType string, cid int, limit int) ([]comic.Comic, error) {
	return nil, fmt.Errorf("not supported")
}

// SearchTags 按名称搜索标签（从漫画数据推导），MongoDB暂不支持
func (s *MongoStorage) SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]comic.TagInfo, int64, error) {
	return nil, 0, fmt.Errorf("not supported")
}

// ListTags 获取标签列表，MongoDB暂不支持
func (s *MongoStorage) ListTags(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) ([]comic.TagInfo, int64, error) {
	return nil, 0, fmt.Errorf("not supported")
}
