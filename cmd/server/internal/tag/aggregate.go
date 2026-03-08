// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package tag

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
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
	cacheKey := CacheKeyTagTotal(tagType)
	var total int64
	if err := cache.Get(cacheKey, &total); err == nil {
		return total, nil
	}
	count, err := mongo.ComicTagBuilder().
		Filters("type", tagType).
		NoLimit().
		Count(ctx)
	if err != nil {
		return 0, err
	}
	if err := cache.Set(cacheKey, count); err != nil {
		clog.Warnf(ctx, "set comicTag total cache failed. key[%s] errmsg: %s", cacheKey, err.Error())
	}
	return count, nil
}

func CacheKeyTagByID(tagType string, id int) string {
	return fmt.Sprintf("comicTag:id:%s:%d", tagType, id)
}

func GetTagByID(ctx context.Context, tagType string, id int) (*ComicTagDoc, error) {
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
		clog.Warnf(ctx, "set comicTag by id cache failed. key[%s] errmsg: %s", cacheKey, err.Error())
	}
	return doc, nil
}

func AggregateTags(ctx context.Context) error {
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
	clog.Infof(ctx, "aggregate tags completed. upserted: %d", len(results))
	return nil
}

func GetTags(ctx context.Context, tagType string, limit int64, skip int64) ([]*ComicTagDoc, error) {
	cacheKey := CacheKeyTagList(tagType, limit, skip)
	var docs []*ComicTagDoc
	if err := cache.Get(cacheKey, &docs); err == nil {
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
	if err := cache.Set(cacheKey, docs); err != nil {
		clog.Warnf(ctx, "set comicTag list cache failed. key[%s] errmsg: %s", cacheKey, err.Error())
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
		clog.Errorf(ctx, "set comicTag agg list cache failed. key[%s] tags[%s] total[%d] errmsg: %s",
			cacheKey, conv.JSON(results[0].Data), results[0].Total, errSet.Error())
	} else {
		clog.Debugf(ctx, "set comicTag agg list cache succ. key[%s] total[%d]", cacheKey, results[0].Total)
	}
	return results[0].Data, int64(results[0].Total), nil
}

func AggregateTagSectionIndices(ctx context.Context, tagType string, pageTagNum int, likedOnly bool) ([]*api.TagSectionIndex, error) {
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
		clog.Errorf(ctx, "set comicTag section indices cache failed. key[%s] indices[%s] errmsg: %s",
			cacheKey, conv.JSON(indices), err.Error())
	} else {
		clog.Debugf(ctx, "set comicTag section indices cache succ. key[%s] total[%d]", cacheKey, len(indices))
	}
	return indices, nil
}
