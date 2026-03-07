// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/mongowrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CacheKeyFilter(filters ...interface{}) string {
	if len(filters) == 0 {
		return "total"
	}
	builder := strings.Builder{}
	builder.WriteString("filters")
	for _, v := range filters {
		builder.WriteString(fmt.Sprintf(":%v", v))
	}
	return builder.String()
}

func CacheKeyComicInfo(cid int) string {
	return fmt.Sprintf("comicInfo:%d", cid)
}

func CacheKeyRangeComicInfos(limit int64, skip int64, filters ...interface{}) string {
	return fmt.Sprintf("comicInfos:limit:%d:skip:%d:%s", limit, skip, CacheKeyFilter(filters...))
}

func CacheKeyCountTotalComicInfos(filters ...interface{}) string {
	return fmt.Sprintf("comicInfos:count:%s", CacheKeyFilter(filters...))
}

func CacheKeyTagList(tagType string, limit int64, skip int64, sortType int) string {
	return fmt.Sprintf("comicInfos:tagList:%s:limit:%d:skip:%d:sortType:%d", tagType, limit, skip, sortType)
}

func CacheKeyTagSectionIndices(tagType string, pageTagNum int) string {
	return fmt.Sprintf("comicInfos:tagSectionIndices:%s:pageTagNum:%d", tagType, pageTagNum)
}

func UpdateComicInfo(ctx context.Context, cid int, comicInfo map[string]interface{}) (err error) {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"cid": cid}
	update := bson.M{"$set": comicInfo}
	delete(comicInfo, "_id")

	_, err = mongo.ComicInfo().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return mongowrap.ErrMongoUpdateFailed.SetIErrF("mongo collection update failed. filter[%s] update[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(update), conv.JSON(opts), err.Error())
	}

	cacheKey := CacheKeyComicInfo(cid)
	errSet := cache.Delete(cacheKey)
	if errSet != nil {
		clog.Errorf(ctx, "delete comic info cache failed. key[%s] errmsg: %s", cacheKey, errSet.Error())
	} else {
		clog.Debugf(ctx, "delete comic info cache succ. key[%s]", cacheKey)
	}
	return
}

func GetComicInfo(ctx context.Context, cid int, info interface{}) (err error) {
	cacheKey := CacheKeyComicInfo(cid)
	err = cache.Get(cacheKey, info)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	opts := options.FindOne()
	filter := bson.M{"cid": cid}

	result := mongo.ComicInfo().FindOne(ctx, filter, opts)
	if result.Err() != nil {
		return mongowrap.ErrMongoFindFailed.SetIErrF("filter[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(opts), result.Err().Error())
	}

	err = result.Decode(info)
	if err != nil {
		return mongowrap.ErrMongoDecodeFailed.SetIErrF("filter[%s] opts[%s] errmsg: %s",
			conv.JSON(filter), conv.JSON(opts), err.Error())
	}

	errSet := cache.Set(cacheKey, info)
	if errSet != nil {
		clog.Errorf(ctx, "set comic info cache failed. key[%s] info[%v] errmsg: %s",
			cacheKey, conv.JSON(info), errSet.Error())
	} else {
		clog.Debugf(ctx, "set comic info cache cache succ. key[%s]", cacheKey)
	}
	return
}

func GetRangeComicInfos(ctx context.Context, limit int64, skip int64, filters ...interface{}) (infos []*api.ComicInfo, err error) {
	cacheKey := CacheKeyRangeComicInfos(limit, skip, filters...)
	infos = []*api.ComicInfo{}
	err = cache.Get(cacheKey, &infos)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	err = mongo.ComicInfoBuilder().
		Filters(filters...).
		SortKV("cid", -1).
		Limit(limit).Skip(skip).
		All(ctx, &infos)
	if err != nil {
		return
	}

	errSet := cache.Set(cacheKey, infos)
	if errSet != nil {
		clog.Errorf(ctx, "set latest comic infos cache failed. key[%s] infos[%s] errmsg: %s",
			cacheKey, conv.JSON(infos), errSet.Error())
	} else {
		clog.Debugf(ctx, "set latest comic infos cache succ. key[%s] infos[%d]", cacheKey, len(infos))
	}
	return
}

func CountTotalComicInfos(ctx context.Context, filters ...interface{}) (count int64, err error) {
	cacheKey := CacheKeyCountTotalComicInfos(filters...)
	err = cache.Get(cacheKey, &count)
	if err == nil {
		return
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	count, err = mongo.ComicInfoBuilder().
		Filters(filters...).
		NoLimit().
		Count(ctx)
	if err != nil {
		return
	}

	setErr := cache.Set(cacheKey, count)
	if setErr != nil {
		clog.Errorf(ctx, "set comic info total count cache failed. key[%s] count[%v] errmsg: %s",
			cacheKey, count, setErr.Error())
	} else {
		clog.Debugf(ctx, "set comic info total count cache succ. key[%s] count[%v]", cacheKey, count)
	}
	return
}

var (
	SortTypeByName    = 0
	SortTypeByPopular = 1
)

func aggregateTagSort(sortType int) bson.M {
	if sortType == SortTypeByName {
		return bson.M{"$sort": bson.M{"_id.name": 1}}
	}
	return bson.M{"$sort": bson.M{"count": -1}}
}

// AggregateTagList 聚合标签列表
/*
db.comicInfo.aggregate([
  // 步骤1：过滤出至少包含一个 tags.type = "artist" 的文档
  { $match: { "tags.type": "artist" } },

  // 步骤2: 展开 tags 数组（每个标签生成独立文档）
  { $unwind: "$tags" },

  // 步骤3: 筛选 artist 类型标签
  { $match: { "tags.type": "artist" } },

  // 步骤4: 按四元组分组并统计真实次数
  { $group: {
      _id: {
        id: "$tags.id",
        name: "$tags.name",
        type: "$tags.type",
        url: "$tags.url"
      },
      count: { $sum: 1 }  // 统计该组合出现的总次数
    }
  },

  // 步骤5: 分页和总数统计
  { $facet: {
      // 分页数据管道
      paginatedResults: [
        { $sort: { count: -1 } },  // 排序（按需调整）
        { $skip: 20 },             // 跳过前N条
        { $limit: 10 },            // 返回M条
        { $project: {              // 重组输出格式
            _id: 0,                // 隐藏分组ID
            id: "$_id.id",         // 提取原始字段
            name: "$_id.name",     // 提取原始字段
            type: "$_id.type",     // 提取原始字段
            url: "$_id.url",       // 提取原始字段
            count: 1               // 保留统计值
          }
        }
      ],

      // 总数统计管道
      totalCount: [
        { $count: "count" }  // 统计总记录数
      ]
    }
  },

  // 阶段6: 重组输出格式
  { $project: {
      data: "$paginatedResults",
      total: { $arrayElemAt: ["$totalCount.count", 0] }
    }
  }
])
*/
func AggregateTagList(ctx context.Context, tagType string, sortType int, skip, limit int64) (tags []*api.TagInfo, total int64, err error) {
	cacheKey := CacheKeyTagList(tagType, limit, skip, sortType)
	var results []struct {
		Data  []*api.TagInfo `bson:"data"`
		Total int            `bson:"total"`
	}
	err = cache.Get(cacheKey, &results)
	if err == nil && len(results) > 0 {
		return results[0].Data, int64(results[0].Total), nil
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	pipe := []bson.M{
		{"$match": bson.M{"tags.type": tagType}},
		{"$unwind": "$tags"},
		{"$match": bson.M{"tags.type": tagType}},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$facet": bson.M{
			"paginatedResults": []bson.M{
				aggregateTagSort(sortType),
				{"$skip": skip},
				{"$limit": limit},
				{"$project": bson.M{
					"_id":   0,
					"id":    "$_id.id",
					"name":  "$_id.name",
					"type":  "$_id.type",
					"url":   "$_id.url",
					"count": 1,
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

	err = mongo.ComicInfoBuilder().
		Aggregate(ctx, pipe, &results)
	if err != nil {
		return
	}
	if len(results) == 0 || len(results[0].Data) == 0 {
		err = errors.New("AggregateTagList result is empty")
		return
	}

	errSet := cache.Set(cacheKey, results)
	if errSet != nil {
		clog.Errorf(ctx, "set tag list cache failed. key[%s] tags[%s] total[%d] errmsg: %s",
			cacheKey, conv.JSON(results[0].Data), results[0].Total, errSet.Error())
	} else {
		clog.Debugf(ctx, "set tag list cache succ. key[%s] tags[%s] total[%d]",
			cacheKey, conv.JSON(results[0].Data), results[0].Total)
	}
	return results[0].Data, int64(results[0].Total), nil
}

// AggregateTagNameSectionIndex 聚合标签名称和索引
/*
db.comicInfo.aggregate([
  // 步骤1：过滤出至少包含一个 tags.type = "artist" 的文档
  { $match: { "tags.type": "artist" } },

  // 步骤2: 展开 tags 数组（每个标签生成独立文档）
  { $unwind: "$tags" },

  // 步骤3: 筛选 artist 类型标签
  { $match: { "tags.type": "artist" } },

  // 步骤4: 按四元组分组并统计真实次数
  { $group: {
      _id: {
        id: "$tags.id",
        name: "$tags.name",
        type: "$tags.type",
        url: "$tags.url"
      },
      count: { $sum: 1 }  // 统计该组合出现的总次数
    }
  },

  // 步骤5：计算字母分组
  { $addFields: {
      alphaGroup: {
        $cond: [
          { $regexMatch: {
              input: { $substrCP: [ "$_id.name", 0, 1 ] },
              regex: /^[A-Z]$/i
          }},
          { $toUpper: { $substrCP: [ "$_id.name", 0, 1 ] } },
          "#"
        ]
      }
    }
  },

  // 步骤6：按字母分组统计数量
  { $group: {
      _id: "$alphaGroup",
      count: { $sum: 1 }
    }
  }
])
*/
func AggregateTagSectionIndices(ctx context.Context, tagType string, pageTagNum int) ([]*api.TagSectionIndex, error) {
	cacheKey := CacheKeyTagSectionIndices(tagType, pageTagNum)
	tagSectionIndices := make([]*api.TagSectionIndex, 0, 27)
	err := cache.Get(cacheKey, &tagSectionIndices)
	if err == nil && len(tagSectionIndices) > 0 {
		return tagSectionIndices, nil
	}
	clog.Debugf(ctx, "miss cache key[%s]", cacheKey)

	pipe := []bson.M{
		{"$match": bson.M{"tags.type": tagType}},
		{"$unwind": "$tags"},
		{"$match": bson.M{"tags.type": tagType}},
		{"$group": bson.M{
			"_id":   bson.M{"id": "$tags.id", "name": "$tags.name", "type": "$tags.type", "url": "$tags.url"},
			"count": bson.M{"$sum": 1},
		}},
		{"$addFields": bson.M{
			"alphaGroup": bson.M{
				"$cond": bson.A{
					bson.M{
						"$regexMatch": bson.M{
							"input":   bson.M{"$substrCP": bson.A{"$_id.name", 0, 1}},
							"regex":   "^[A-Z]$",
							"options": "i",
						},
					},
					bson.M{"$toUpper": bson.M{"$substrCP": bson.A{"$_id.name", 0, 1}}},
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
	err = mongo.ComicInfoBuilder().
		Aggregate(ctx, pipe, &results)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("AggregateTagNameSectionIndex result is empty")
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	var sectionIndex int
	for _, result := range results {
		tagSectionIndices = append(tagSectionIndices, &api.TagSectionIndex{
			Name:  result.ID,
			Index: sectionIndex,
			Page:  sectionIndex/pageTagNum + 1,
		})
		sectionIndex += result.Count
	}

	errSet := cache.Set(cacheKey, tagSectionIndices)
	if errSet != nil {
		clog.Errorf(ctx, "set tag section indices cache failed. key[%s] tagSectionIndices[%s] total[%d] errmsg: %s",
			cacheKey, conv.JSON(tagSectionIndices), len(tagSectionIndices), errSet.Error())
	} else {
		clog.Debugf(ctx, "set tag section indices cache succ. key[%s] tagSectionIndices[%s] total[%d]",
			cacheKey, conv.JSON(tagSectionIndices), len(tagSectionIndices))
	}
	return tagSectionIndices, nil
}
