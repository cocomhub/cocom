// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/comic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage 实现comic.ComicStorage接口
type Storage struct{}

// NewStorage 创建存储实例
func NewStorage() *Storage {
	return &Storage{}
}

// Get 获取漫画信息
func (s *Storage) Get(ctx context.Context, id string) (comic.Comic, error) {
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
	c, err := NewComicByObject(obj)
	if err != nil {
		return err
	}

	if c == nil || c.CID == 0 {
		return fmt.Errorf("invalid comic info")
	}

	if iErr := archiveComic(ctx, c.ComicInfo); iErr != nil {
		clog.Warnf(ctx, "failed to archive comic: %s", iErr)
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

	err = UpdateComicInfo(ctx, c.CID, v)
	if err != nil {
		return fmt.Errorf("failed to save comic: %w", err)
	}
	return nil
}

// Find 列出符合条件的漫画
func (s *Storage) Find(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error) {
	cursor, err := mongo.ComicInfo().Find(ctx, s.toMongoFilter(ctx, filter), &options.FindOptions{
		Sort:  bson.M{"cid": 1},
		Limit: &filter.Limit,
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
	return mongo.ComicInfo().CountDocuments(ctx, s.toMongoFilter(ctx, filter), &options.CountOptions{
		Limit: &filter.Limit,
		Skip:  &filter.Skip,
	})
}

// FindChannel 列出符合条件的漫画，返回通道
func (s *Storage) FindChannel(ctx context.Context, filter *comic.ComicFilter) (chan comic.Comic, error) {
	comics := make(chan comic.Comic, 100)
	go func() {
		defer close(comics)
		oriLimit := filter.Limit + filter.Skip
		filter.Limit = min(100, oriLimit)
		for filter.Limit+filter.Skip <= oriLimit {
			impls, err := s.Find(ctx, filter)
			if err != nil {
				clog.Errorf(ctx, "failed to find comics: %s", err)
				return
			}
			if len(impls) == 0 {
				break
			}
			for _, c := range impls {
				comics <- c
			}
			if filter.NotArchived != nil && *filter.NotArchived {
				cid, err := strconv.Atoi(impls[len(impls)-1].GetID())
				if err != nil {
					clog.Errorf(ctx, "invalid comic id: %s", impls[len(impls)-1].GetID())
				} else {
					filter.IDRangeLeft = new(int64(cid + 1))
				}
				filter.Skip = 0
				continue
			}
			filter.Skip += int64(len(impls))
		}
	}()
	return comics, nil
}

func (s *Storage) toMongoFilter(ctx context.Context, filter *comic.ComicFilter) bson.M {
	mongoFilter := bson.M{}
	if filter == nil {
		return mongoFilter
	}

	if filter.ID != nil {
		cid, err := strconv.Atoi(*filter.ID)
		if err != nil {
			clog.Errorf(ctx, "invalid comic id: %s", *filter.ID)
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

	return mongoFilter
}

// SaveVerifyResult 保存验证结果
func (s *Storage) SaveVerifyResult(ctx context.Context, result *comic.VerifyResult) error {
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
