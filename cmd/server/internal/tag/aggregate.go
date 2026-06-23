// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package tag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ComicTagDoc struct {
	Type      string    `bson:"type" json:"type"`
	ID        int       `bson:"id" json:"id"`
	Name      string    `bson:"name" json:"name"`
	URL       string    `bson:"url" json:"url"`
	Count     int       `bson:"count" json:"count"`
	Like      bool      `bson:"like" json:"like"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

func CacheKeyTagList(tagType string, limit int64, skip int64) string {
	return fmt.Sprintf("comicTag:list:%s:limit:%d:skip:%d", tagType, limit, skip)
}

func CacheKeyTagTotal(tagType string) string {
	return fmt.Sprintf("comicTag:total:%s", tagType)
}

func CacheKeyTagListAgg(tagType string, limit int64, skip int64, sortType int, likedOnly bool) string {
	return fmt.Sprintf("comicTag:agg:list:%s:limit:%d:skip:%d:sort:%d:liked:%t", tagType, limit, skip, sortType, likedOnly)
}

func CacheKeyTagSectionIndices(tagType string, pageTagNum int, likedOnly bool) string {
	return fmt.Sprintf("comicTag:agg:section:%s:pageTagNum:%d:liked:%t", tagType, pageTagNum, likedOnly)
}

func CountTags(ctx context.Context, tagType string) (int64, error) {
	if s := defaultTagStore; s != nil {
		return s.CountTags(ctx, tagType)
	}
	var total int64
	if err := cache.Get(CacheKeyTagTotal(tagType), &total); err == nil {
		return total, nil
	}
	count, err := mongo.ComicTagBuilder().
		Filters("type", tagType).
		NoLimit().
		Count(ctx)
	if err != nil {
		return 0, err
	}
	if err := cache.Set(CacheKeyTagTotal(tagType), count); err != nil {
		slog.WarnContext(ctx, "set comicTag total cache failed", slog.String("key", CacheKeyTagTotal(tagType)), slog.String("err", err.Error()))
	}
	return count, nil
}

func CacheKeyTagByID(tagType string, id int) string {
	return fmt.Sprintf("comicTag:id:%s:%d", tagType, id)
}

func GetTagByID(ctx context.Context, tagType string, id int) (*ComicTagDoc, error) {
	if s := defaultTagStore; s != nil {
		return s.GetTagByID(ctx, tagType, id)
	}
	if cache.Cache() == nil {
		// 缓存未初始化（如 E2E 测试环境），跳过缓存直接走 MongoDB
		var docs []*ComicTagDoc
		if err := mongo.ComicTagBuilder().
			Filters("type", tagType).
			FilterKV("id", id).
			Limit(1).
			All(ctx, &docs); err != nil {
			return nil, err
		}
		if len(docs) == 0 {
			return nil, nil
		}
		return docs[0], nil
	}
	cacheKey := CacheKeyTagByID(tagType, id)
	var doc *ComicTagDoc
	if err := cache.Get(cacheKey, &doc); err == nil && doc != nil {
		return doc, nil
	}
	var docs []*ComicTagDoc
	if err := mongo.ComicTagBuilder().
		Filters("type", tagType).
		FilterKV("id", id).
		Limit(1).
		All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	doc = docs[0]
	if err := cache.Set(cacheKey, doc); err != nil {
		slog.WarnContext(ctx, "set comicTag by id cache failed", slog.String("key", cacheKey), slog.String("err", err.Error()))
	}
	return doc, nil
}

func AggregateTags(ctx context.Context) error {
	if s := defaultTagStore; s != nil {
		return s.AggregateTags(ctx)
	}
	var results []struct {
		ID struct {
			ID   int    `bson:"id"`
			Name string `bson:"name"`
			Type string `bson:"type"`
			URL  string `bson:"url"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	pipe := []bson.M{
		{"$unwind": "$tags"},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
	}
	if err := mongo.ComicInfoBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return err
	}
	// upsert results into comicTag collection
	upOpts := options.Update().SetUpsert(true)
	for _, r := range results {
		filter := bson.M{
			"type": r.ID.Type,
			"id":   r.ID.ID,
			"name": r.ID.Name,
			"url":  r.ID.URL,
		}
		update := bson.M{
			"$set": bson.M{
				"type":       r.ID.Type,
				"id":         r.ID.ID,
				"name":       r.ID.Name,
				"url":        r.ID.URL,
				"count":      r.Count,
				"updated_at": time.Now(),
			},
			"$setOnInsert": bson.M{
				"like": false,
			},
		}
		if _, err := mongo.ComicTag().UpdateOne(ctx, filter, update, upOpts); err != nil {
			return mongowrap.ErrMongoUpdateFailed.SetIErrF("comicTag upsert failed. filter[%s] update[%s] errmsg: %s",
				conv.JSON(filter), conv.JSON(update), err.Error())
		}
	}
	// cache reset for comicTag related keys cannot be selective; just log for visibility
	slog.InfoContext(ctx, "aggregate tags completed", slog.Int("upserted", len(results)))
	return nil
}

func GetTags(ctx context.Context, tagType string, limit int64, skip int64) ([]*ComicTagDoc, error) {
	if s := defaultTagStore; s != nil {
		return s.GetTags(ctx, tagType, limit, skip)
	}
	var docs []*ComicTagDoc
	if err := cache.Get(CacheKeyTagList(tagType, limit, skip), &docs); err == nil {
		return docs, nil
	}
	if err := mongo.ComicTagBuilder().
		Filters("type", tagType).
		SortKV("count", -1).
		Limit(limit).
		Skip(skip).
		All(ctx, &docs); err != nil {
		return nil, err
	}
	if err := cache.Set(CacheKeyTagList(tagType, limit, skip), docs); err != nil {
		slog.WarnContext(ctx, "set comicTag list cache failed", slog.String("key", CacheKeyTagList(tagType, limit, skip)), slog.String("err", err.Error()))
	}
	return docs, nil
}

func aggregateTagSort(sortType int) bson.M {
	if sortType == 0 {
		return bson.M{"$sort": bson.M{"name": 1}}
	}
	return bson.M{"$sort": bson.M{"count": -1}}
}

func AggregateTagList(ctx context.Context, tagType string, sortType int, skip, limit int64, likedOnly bool) (tags []*api.TagInfo, total int64, err error) {
	if s := GetDefaultComicStore(); s != nil {
		tagInfos, total, listErr := s.ListTags(ctx, tagType, sortType, skip, limit, likedOnly)
		if listErr != nil {
			return nil, 0, listErr
		}
		result := make([]*api.TagInfo, len(tagInfos))
		for i, t := range tagInfos {
			result[i] = &api.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like}
		}
		return result, total, nil
	}
	cacheKey := CacheKeyTagListAgg(tagType, limit, skip, sortType, likedOnly)
	var results []struct {
		Data  []*api.TagInfo `bson:"data"`
		Total int            `bson:"total"`
	}
	if err = cache.Get(cacheKey, &results); err == nil && len(results) > 0 {
		return results[0].Data, int64(results[0].Total), nil
	}
	filter := bson.M{"type": tagType}
	if likedOnly {
		filter["like"] = true
	}
	pipe := []bson.M{
		{"$match": filter},
		{"$facet": bson.M{
			"paginatedResults": []bson.M{
				aggregateTagSort(sortType),
				{"$skip": skip},
				{"$limit": limit},
				{"$project": bson.M{
					"_id":   0,
					"id":    1,
					"name":  1,
					"type":  1,
					"url":   1,
					"count": 1,
					"like":  1,
				}},
			},
			"totalCount": []bson.M{
				{"$count": "count"},
			},
		}},
		{"$project": bson.M{
			"data":  "$paginatedResults",
			"total": bson.M{"$arrayElemAt": bson.A{"$totalCount.count", 0}},
		}},
	}
	if err = mongo.ComicTagBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return
	}
	if len(results) == 0 {
		err = errors.New("AggregateTagList result is empty")
		return
	}
	if errSet := cache.Set(cacheKey, results); errSet != nil {
		slog.ErrorContext(ctx, "set comicTag agg list cache failed", slog.String("key", cacheKey), slog.String("err", errSet.Error()))
	} else {
		slog.DebugContext(ctx, "set comicTag agg list cache succ", slog.String("key", cacheKey), slog.Int("total", results[0].Total))
	}
	return results[0].Data, int64(results[0].Total), nil
}

func AggregateTagSectionIndices(ctx context.Context, tagType string, pageTagNum int, likedOnly bool) ([]*api.TagSectionIndex, error) {
	if s := defaultTagStore; s != nil {
		return s.AggregateTagSectionIndices(ctx, tagType, pageTagNum, likedOnly)
	}
	cacheKey := CacheKeyTagSectionIndices(tagType, pageTagNum, likedOnly)
	indices := make([]*api.TagSectionIndex, 0, 27)
	if err := cache.Get(cacheKey, &indices); err == nil && len(indices) > 0 {
		return indices, nil
	}
	filter := bson.M{"type": tagType}
	if likedOnly {
		filter["like"] = true
	}
	pipe := []bson.M{
		{"$match": filter},
		{"$addFields": bson.M{
			"alphaGroup": bson.M{
				"$cond": bson.A{
					bson.M{
						"$regexMatch": bson.M{
							"input":   bson.M{"$substrCP": bson.A{"$name", 0, 1}},
							"regex":   "^[A-Z]$",
							"options": "i",
						},
					},
					bson.M{"$toUpper": bson.M{"$substrCP": bson.A{"$name", 0, 1}}},
					"#",
				},
			},
		}},
		{"$group": bson.M{
			"_id":   "$alphaGroup",
			"count": bson.M{"$sum": 1},
		}},
	}
	var results []struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := mongo.ComicTagBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("AggregateTagSectionIndices result is empty")
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	var sectionIndex int
	for _, r := range results {
		indices = append(indices, &api.TagSectionIndex{
			Name:  r.ID,
			Index: sectionIndex,
			Page:  sectionIndex/pageTagNum + 1,
		})
		sectionIndex += r.Count
	}
	if err := cache.Set(cacheKey, indices); err != nil {
		slog.ErrorContext(ctx, "set comicTag section indices cache failed", slog.String("key", cacheKey), slog.String("err", err.Error()))
	} else {
		slog.DebugContext(ctx, "set comicTag section indices cache succ", slog.String("key", cacheKey), slog.Int("total", len(indices)))
	}
	return indices, nil
}

// InvalidateTagCache 删除指定 tag 的缓存
func InvalidateTagCache(ctx context.Context, tagType string, tagID int) {
	key := CacheKeyTagByID(tagType, tagID)
	if err := cache.Delete(key); err != nil {
		slog.WarnContext(ctx, "delete tag cache failed", slog.String("key", key), slog.String("err", err.Error()))
	}
}

// UpdateComicTagIncremental 增量更新 comicTag 集合的 count
// countDiff 为正表示增加（添加 tag），为负表示减少（移除 tag）
func UpdateComicTagIncremental(ctx context.Context, tagType string, tagID int, tagName string, tagURL string, countDiff int) error {
	if s := defaultTagStore; s != nil {
		return s.UpdateComicTagIncremental(ctx, tagType, tagID, tagName, tagURL, countDiff)
	}
	filter := bson.M{"type": tagType, "id": tagID, "name": tagName, "url": tagURL}

	if countDiff > 0 {
		// Tag 添加到漫画：递增 count，不存在则创建
		update := bson.M{
			"$inc": bson.M{"count": countDiff},
			"$set": bson.M{"updated_at": time.Now()},
			"$setOnInsert": bson.M{
				"type": tagType,
				"id":   tagID,
				"name": tagName,
				"url":  tagURL,
				"like": false,
			},
		}
		opts := options.Update().SetUpsert(true)
		if _, err := mongo.ComicTag().UpdateOne(ctx, filter, update, opts); err != nil {
			return mongowrap.ErrMongoUpdateFailed.SetIErrF("comicTag incremental upsert failed. tagType[%s] tagID[%d] errmsg: %s",
				tagType, tagID, err.Error())
		}
	} else if countDiff < 0 {
		// Tag 从漫画移除：递减 count
		update := bson.M{
			"$inc": bson.M{"count": countDiff},
			"$set": bson.M{"updated_at": time.Now()},
		}
		if _, err := mongo.ComicTag().UpdateOne(ctx, filter, update); err != nil {
			return mongowrap.ErrMongoUpdateFailed.SetIErrF("comicTag incremental decrement failed. tagType[%s] tagID[%d] errmsg: %s",
				tagType, tagID, err.Error())
		}
		// 如果 count <= 0，清理该文档
		if _, err := mongo.ComicTag().DeleteOne(ctx, bson.M{"type": tagType, "id": tagID, "count": bson.M{"$lte": 0}}); err != nil {
			slog.WarnContext(ctx, "comicTag delete zero-count doc failed", slog.String("err", err.Error()))
		}
	}

	InvalidateTagCache(ctx, tagType, tagID)
	return nil
}

// GetSearchUniqueTags 按搜索 query 匹配漫画，获取其中去重后的 tag 列表以及匹配的 cid 列表
func GetSearchUniqueTags(ctx context.Context, query string, limit, skip int64) (tags []*api.TagInfo, cidList []int, total int64, err error) {
	if s := defaultTagStore; s != nil {
		return s.GetSearchUniqueTags(ctx, query, limit, skip)
	}
	escapedQuery := regexp.QuoteMeta(query)
	titleFilter := bson.M{"$or": []bson.M{
		{"title.english": bson.M{"$regex": primitive.Regex{Pattern: escapedQuery, Options: "i"}}},
		{"title.japanese": bson.M{"$regex": primitive.Regex{Pattern: escapedQuery, Options: "i"}}},
		{"title.pretty": bson.M{"$regex": primitive.Regex{Pattern: escapedQuery, Options: "i"}}},
	}}

	// 第一步：获取匹配的 cid 列表
	type cidResult struct {
		CID int `bson:"cid"`
	}
	var cidResults []cidResult
	// 使用 FilterKV 设置复杂过滤器
	if err = mongo.ComicInfoBuilder().
		FilterKV("$or", titleFilter["$or"]).
		NoLimit().
		All(ctx, &cidResults); err != nil {
		return
	}
	for _, r := range cidResults {
		cidList = append(cidList, r.CID)
	}
	total = int64(len(cidList))

	if total == 0 {
		return
	}

	// 第二步：用 cid 列表过滤，unwind tags，group 获得去重标签
	type tagAggResult struct {
		ID struct {
			ID   int    `bson:"id"`
			Name string `bson:"name"`
			Type string `bson:"type"`
			URL  string `bson:"url"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}

	pipe := []bson.M{
		{"$match": bson.M{"cid": bson.M{"$in": cidList}}},
		{"$unwind": "$tags"},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"count": -1}},
		{"$skip": skip},
		{"$limit": limit},
	}

	var tagResults []tagAggResult
	if err = mongo.ComicInfoBuilder().Aggregate(ctx, pipe, &tagResults); err != nil {
		return
	}

	for _, r := range tagResults {
		tags = append(tags, &api.TagInfo{
			ID:    r.ID.ID,
			Name:  r.ID.Name,
			Type:  r.ID.Type,
			URL:   r.ID.URL,
			Count: r.Count,
		})
	}

	return
}

// GetRelatedTags 获取指定 tag 的关联 tag（合并计算关联 + 显式关系）
func GetRelatedTags(ctx context.Context, tagType, tagName string, limit int64) ([]*api.TagInfo, error) {
	if s := defaultTagStore; s != nil {
		return s.GetComputedRelatedTags(ctx, tagType, tagName, limit)
	}
	// 第一步：计算关联（co-occurrence）
	computedTags, err := getComputedRelatedTags(ctx, tagType, tagName, limit)
	if err != nil {
		return nil, err
	}

	// 第二步：显式关系（通过 tagRelation 集合）
	curTag, _ := GetTagByTypeName(ctx, tagType, tagName)
	var explicitTags []*api.TagInfo
	if curTag != nil && curTag.ID > 0 {
		explicitTags, err = GetRelatedTagsFromRelations(ctx, tagType, curTag.ID)
		if err != nil {
			slog.WarnContext(ctx, "get explicit relations failed", slog.String("errmsg", err.Error()))
		}
	}

	// 第三步：合并去重（显式关系优先）
	seen := make(map[string]bool)
	result := make([]*api.TagInfo, 0, len(explicitTags)+len(computedTags))

	for _, t := range explicitTags {
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		seen[key] = true
		result = append(result, t)
	}
	for _, t := range computedTags {
		key := fmt.Sprintf("%s:%d", t.Type, t.ID)
		if !seen[key] {
			result = append(result, t)
		}
	}

	if int64(len(result)) > limit {
		result = result[:limit]
	}
	return result, nil
}

// getComputedRelatedTags 通过 MongoDB 聚合计算共现关联
// 注：当 defaultTagStore 非 nil 时，GetRelatedTags 已委托给 TagStore，
// 此函数仅作为 MongoDB 路径的回退，不会被 tag store 路径调用。
func getComputedRelatedTags(ctx context.Context, tagType, tagName string, limit int64) ([]*api.TagInfo, error) {
	pipe := []bson.M{
		{"$match": bson.M{"tags": bson.M{"$elemMatch": bson.M{"type": tagType, "name": tagName}}}},
		{"$unwind": "$tags"},
		{"$match": bson.M{
			"$nor": []bson.M{{"tags.type": tagType, "tags.name": tagName}},
		}},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"count": -1}},
		{"$limit": limit},
		{"$project": bson.M{
			"_id":   0,
			"id":    "$_id.id",
			"name":  "$_id.name",
			"type":  "$_id.type",
			"url":   "$_id.url",
			"count": 1,
		}},
	}

	var results []struct {
		ID    int    `bson:"id"`
		Name  string `bson:"name"`
		Type  string `bson:"type"`
		URL   string `bson:"url"`
		Count int    `bson:"count"`
	}
	if err := mongo.ComicInfoBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return nil, err
	}

	tags := make([]*api.TagInfo, 0, len(results))
	for _, r := range results {
		doc, _ := GetTagByID(ctx, r.Type, r.ID)
		liked := doc != nil && doc.Like
		tags = append(tags, &api.TagInfo{
			ID:    r.ID,
			Name:  r.Name,
			Type:  r.Type,
			URL:   r.URL,
			Count: r.Count,
			Like:  liked,
		})
	}

	return tags, nil
}

// GetTagByTypeName 通过 type+name 查找 comicTag 文档
func GetTagByTypeName(ctx context.Context, tagType, tagName string) (*ComicTagDoc, error) {
	if s := defaultTagStore; s != nil {
		return s.GetTagByTypeName(ctx, tagType, tagName)
	}
	var docs []*ComicTagDoc
	if err := mongo.ComicTagBuilder().
		Filters("type", tagType, "name", tagName).
		Limit(1).
		All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	return docs[0], nil
}

// GetTagByTypeURL 通过 type+url 查找 comicTag 文档
func GetTagByTypeURL(ctx context.Context, tagType, url string) (*ComicTagDoc, error) {
	if s := defaultTagStore; s != nil {
		return s.GetTagByTypeURL(ctx, tagType, url)
	}
	var docs []*ComicTagDoc
	if err := mongo.ComicTagBuilder().
		Filters("type", tagType, "url", url).
		Limit(1).
		All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	return docs[0], nil
}

// SearchTags 按 type + name 模糊搜索 comicTag 集合中的现有标签
// query 为空时返回该 type 下按 count 降序的前 limit 条
func SearchTags(ctx context.Context, tagType string, query string, limit int64) ([]*api.TagInfo, error) {
	if s := GetDefaultComicStore(); s != nil {
		tags, _, err := s.SearchTags(ctx, tagType, query, limit)
		if err != nil {
			return nil, err
		}
		result := make([]*api.TagInfo, len(tags))
		for i, t := range tags {
			result[i] = &api.TagInfo{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL, Count: t.Count, Like: t.Like}
		}
		return result, nil
	}
	builder := mongo.ComicTagBuilder().
		SortKV("count", -1).
		Limit(limit)

	if tagType != "" {
		builder.FilterKV("type", tagType)
	}

	if query != "" {
		escapedQuery := regexp.QuoteMeta(query)
		builder.FilterKV("name", bson.M{
			"$regex": primitive.Regex{Pattern: escapedQuery, Options: "i"},
		})
	}

	var docs []*ComicTagDoc
	if err := builder.All(ctx, &docs); err != nil {
		return nil, err
	}

	tags := make([]*api.TagInfo, 0, len(docs))
	for _, d := range docs {
		tags = append(tags, &api.TagInfo{
			ID:    d.ID,
			Name:  d.Name,
			Type:  d.Type,
			URL:   d.URL,
			Count: d.Count,
			Like:  d.Like,
		})
	}
	return tags, nil
}

// GetMaxTagID 查询 comicInfo 集合中所有 tag 的最大 ID
// 返回当前最大 ID，如果没有任何 tag 则返回 0
func GetMaxTagID(ctx context.Context) (int, error) {
	if s := defaultTagStore; s != nil {
		return s.GetMaxTagID(ctx)
	}
	type maxResult struct {
		MaxID int `bson:"maxId"`
	}
	var results []maxResult
	pipe := []bson.M{
		{"$unwind": "$tags"},
		{"$group": bson.M{"_id": nil, "maxId": bson.M{"$max": "$tags.id"}}},
	}
	if err := mongo.ComicInfoBuilder().Aggregate(ctx, pipe, &results); err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0].MaxID, nil
}
