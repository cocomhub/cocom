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

	_, err = s.db.Collection("comics").UpdateOne(ctx, bson.M{"cid": comic.GetID()}, comic)
	return err
}

// Find 实现 ComicStorage 接口
func (s *MongoStorage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := s.db.Collection("comics").Find(ctx, s.toMongoFilter(filter), &options.FindOptions{
		Sort:  bson.M{"cid": 1},
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

	if filter.ID != nil {
		mongoFilter["cid"] = *filter.ID
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
